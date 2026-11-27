package core

import (
	"context"
	"fmt"
)

// Writing the buffer (:w, :wq, :x) is up to the host: a file, a database, an
// API, a field in memory. The host's WriteFunc runs synchronously, inside the
// command, on the goroutine that handles keys, so it may use host state.
//
//   - A quick write (a local file or store) is done right there: return a nil
//     job and the write's error. The write finishes inside the command, as
//     in Vim.
//   - A slow write (a network drive, a remote API) returns a WriteJob, so it
//     doesn't block typing. The host takes it with NextWrite, runs it
//     anywhere (the Bubble Tea Model runs it in a tea.Cmd) and reports the
//     result with FinishWrite.
//
// With a job, the buffer counts as saved only when FinishWrite reports
// success, and then only the content the job wrote: text typed while it ran
// stays modified. Jobs run one at a time, in the order the commands ran, so a
// slow, older job never finishes after a newer one. :wq and :x quit once every write
// finished and the buffer is unmodified; a failed write cancels the quit.
// :q! quits at once, cancelling the running job's context and dropping the
// queued ones.

// WriteFunc writes the buffer for :w, :wq and :x. path is the text typed
// after the command (a file name, or anything the host gives a meaning to),
// or nil when there is none; content is the buffer's text.
//
// It runs synchronously, inside the command. To write there, do it and return
// a nil job with the write's error: nil marks the buffer saved, an error is
// shown and leaves it modified. For a write slow enough to freeze the screen,
// return a WriteJob that does it instead, capturing what it needs. A non-nil
// error always fails the command at once, job or not.
//
// Neither the function nor the job may call back into the editor: report a
// failure by returning it. Give network calls a timeout, in either place.
//
// Use one style per target. A write done inside the function while an older
// job to the same target still runs (Editor.Writing) can be overwritten when
// that job finishes.
type WriteFunc func(path *string, content string) (WriteJob, error)

// WriteJob does the I/O of one write. It runs off the key-handling goroutine,
// so it must not read host state that changes there: capture it in the
// WriteFunc instead. ctx is cancelled when the user quits with :q! while it
// runs. A panic is recovered and reported as a failed write.
type WriteJob func(ctx context.Context) error

// WriteId identifies a write handed out by NextWrite.
type WriteId uint64

// WriteRequest is a write for the host to run: see NextWrite.
type WriteRequest struct {
	id  WriteId
	job WriteJob
	ctx context.Context
}

// ID returns the id to pass to FinishWrite.
func (r WriteRequest) ID() WriteId {
	return r.id
}

// Run runs the write's job and returns its error. It is safe to call on any
// goroutine. A panic in the job is returned as an error wrapping
// ErrWritePanicked.
func (r WriteRequest) Run() (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("%w: %v", ErrWritePanicked, p)
		}
	}()
	if r.job == nil {
		return nil
	}
	return r.job(r.ctx)
}

// pendingWrite is a write queued by :w and not finished yet.
type pendingWrite struct {
	id      WriteId
	path    *string
	content string // the text the job writes
	// savedAs is the saved content once the write succeeds: content plus
	// any text AppendBytes added since, which counts as saved.
	savedAs string
	gen     uint64 // bufferGen when queued: SetBuffer makes the write stale
	job     WriteJob
	ctx     context.Context
	cancel  context.CancelFunc
	started bool // handed out by NextWrite
}

// savedSetter is implemented by buffers that can take a saved content other
// than their current text. Buffers from hosts (SetBuffer) count as saved
// after a write only when their text is still what was written.
type savedSetter interface {
	setSavedContent(content string)
}

func (b *textBuffer) setSavedContent(content string) {
	b.savedContent = content
}

func (e *editor) SetWriteFunc(f WriteFunc) {
	e.writeFunc = f
}

// Writing reports whether a write is queued or running.
func (e *editor) Writing() bool {
	return len(e.writes) > 0
}

// NextWrite hands out the oldest queued write when no write is running. The
// host runs it (WriteRequest.Run) and reports the result with FinishWrite;
// until then NextWrite returns false. Call it after every key and after
// FinishWrite.
func (e *editor) NextWrite() (WriteRequest, bool) {
	if len(e.writes) == 0 || e.writes[0].started {
		return WriteRequest{}, false
	}
	w := e.writes[0]
	w.started = true
	return WriteRequest{id: w.id, job: w.job, ctx: w.ctx}, true
}

// FinishWrite reports the result of the write NextWrite handed out. On
// success the written content counts as saved (unless SetBuffer replaced the
// buffer since) and a SaveSignal follows; an error is dispatched and cancels
// a pending :wq or :x. An unknown or stale id is ignored.
func (e *editor) FinishWrite(id WriteId, err error) {
	if len(e.writes) == 0 || !e.writes[0].started || e.writes[0].id != id {
		return
	}
	w := e.writes[0]
	e.writes = e.writes[1:]
	w.cancel()

	if err != nil {
		e.quitWhenIdle = false
		e.DispatchError(ErrFailedToSaveId, err)
		return
	}
	e.completeWrite(w)
	e.quitIfIdle()
}

// Save writes the buffer as :w does, to path or, when nil, the current file.
// An error is dispatched, not returned.
func (e *editor) Save(path *string) {
	if err := e.write(path); err != nil {
		e.DispatchError(err.id, err.err)
	}
}

// write queues a write of the buffer for :w, :wq and :x. A nil job with no
// write queued completes at once.
func (e *editor) write(path *string) *EditorError {
	if e.writeFunc == nil {
		return &EditorError{id: ErrNoWriteFuncId, err: ErrNoWriteFunc}
	}
	if path != nil {
		p := *path // the caller's variable may change
		path = &p
	}
	content := e.buffer.GetCurrentContent()
	job, err := e.writeFunc(path, content)
	if err != nil {
		return &EditorError{id: ErrFailedToSaveId, err: err}
	}

	e.lastWriteID++
	w := &pendingWrite{
		id:      e.lastWriteID,
		path:    path,
		content: content,
		savedAs: content,
		gen:     e.bufferGen,
		job:     job,
	}
	if job == nil && len(e.writes) == 0 {
		e.completeWrite(w)
		return nil
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	e.writes = append(e.writes, w)
	return nil
}

// completeWrite marks a successful write's content saved and dispatches the
// SaveSignal. A write queued before SetBuffer replaced the buffer leaves the
// new buffer's saved state alone.
func (e *editor) completeWrite(w *pendingWrite) {
	if w.gen == e.bufferGen {
		buf := e.buffer.Buffer
		if s, ok := buf.(savedSetter); ok {
			s.setSavedContent(w.savedAs)
		} else if buf.GetCurrentContent() == w.savedAs {
			buf.SaveContent()
		}
	}
	e.DispatchSignal(SaveSignal{path: w.path, content: w.content})
}

// writeQueued reports whether the newest queued write is of the current
// buffer's text to path, so :x can wait for it instead of writing again.
func (e *editor) writeQueued(path *string) bool {
	if len(e.writes) == 0 {
		return false
	}
	w := e.writes[len(e.writes)-1]
	samePath := (w.path == nil) == (path == nil) && (path == nil || *w.path == *path)
	return samePath && w.gen == e.bufferGen && w.savedAs == e.buffer.GetCurrentContent()
}

// requestQuit quits for :wq and :x: now when no write is pending, otherwise
// once the last one finished (see quitIfIdle).
func (e *editor) requestQuit() *EditorError {
	if len(e.writes) > 0 {
		e.quitWhenIdle = true
		return nil
	}
	if e.buffer.IsModified() {
		return &EditorError{id: ErrUnsavedChangesId, err: ErrUnsavedChanges}
	}
	e.Quit()
	return nil
}

// quitIfIdle runs a quit requestQuit deferred, once no write is pending. Text
// typed meanwhile keeps the editor open with E37.
func (e *editor) quitIfIdle() {
	if !e.quitWhenIdle || len(e.writes) > 0 {
		return
	}
	e.quitWhenIdle = false
	if e.buffer.IsModified() {
		e.DispatchError(ErrUnsavedChangesId, ErrUnsavedChanges)
		return
	}
	e.Quit()
}

// abandonWrites cancels the running write's context and drops the queued
// ones, for :q!.
func (e *editor) abandonWrites() {
	for _, w := range e.writes {
		if w.cancel != nil {
			w.cancel()
		}
	}
	e.writes = nil
	e.quitWhenIdle = false
}

// appendToPendingWrites adds text that AppendBytes added to the buffer to the
// content each pending write of this buffer will mark saved, since appended
// text counts as saved. replace is the append replacing an empty buffer.
func (e *editor) appendToPendingWrites(text string, replace bool) {
	for _, w := range e.writes {
		if w.gen != e.bufferGen {
			continue
		}
		if replace && w.savedAs == "" {
			w.savedAs = text
		} else {
			w.savedAs += "\n" + text
		}
	}
}
