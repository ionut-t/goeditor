package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Errors a host's write function returns: Vim's E32 for a buffer with no
// name (from the synchronous stage, as peek does) and E212 for a directory
// that doesn't exist (from the job, as the OS does).
var (
	errNoFileName  = errors.New("E32: No file name")
	errCannotWrite = errors.New("E212: Cannot open file for writing")
)

const badPath = "/nonexistent-dir/x.txt"

// newWritingEditor returns an editor on content whose writes succeed at once
// (a nil job), for tests of the commands rather than of the writing.
func newWritingEditor(content string) Editor {
	e := newTestEditor(content)
	e.SetWriteFunc(func(*string, string) (WriteJob, error) { return nil, nil })
	return e
}

// fakeWriter is a host's write function. Its synchronous stage fails a bare
// write (no file name) unless named is set; its job fails writes to badPath
// and records each write it ran.
type fakeWriter struct {
	named    bool
	prepared []writeCall // every call of the synchronous stage
	written  []writeCall // every job that ran and succeeded
	ctxs     []context.Context
	panicMsg string // the job panics with it when set
}

type writeCall struct {
	path    string // "" for nil
	content string
}

func (w *fakeWriter) write(path *string, content string) (WriteJob, error) {
	name := ""
	if path != nil {
		name = *path
	}
	w.prepared = append(w.prepared, writeCall{name, content})
	if path == nil && !w.named {
		return nil, errNoFileName
	}
	return func(ctx context.Context) error {
		w.ctxs = append(w.ctxs, ctx)
		if w.panicMsg != "" {
			panic(w.panicMsg)
		}
		if name == badPath {
			return errCannotWrite
		}
		w.written = append(w.written, writeCall{name, content})
		return nil
	}, nil
}

// newFakeWriteEditor returns an editor on "hello", modified to "ello" with x,
// writing through w, with no signals pending.
func newFakeWriteEditor(w *fakeWriter) Editor {
	e := newTestEditor("hello")
	e.SetWriteFunc(w.write)
	keys(e, 'x')
	drainSignals(e)
	return e
}

// runWrites runs the queued writes, as the host does, until none is left.
func runWrites(e Editor) {
	for {
		req, ok := e.NextWrite()
		if !ok {
			return
		}
		e.FinishWrite(req.ID(), req.Run())
	}
}

// collected sorts the signals waiting on e's channel.
type collected struct {
	saves  []SaveSignal
	errors []ErrorSignal
	quits  int
}

func collect(e Editor) collected {
	var c collected
	for s := nextSignal(e); s != nil; s = nextSignal(e) {
		switch s := s.(type) {
		case SaveSignal:
			c.saves = append(c.saves, s)
		case ErrorSignal:
			c.errors = append(c.errors, s)
		case QuitSignal:
			c.quits++
		}
	}
	return c
}

func (c collected) errorIDs() []ErrorId {
	var ids []ErrorId
	for _, s := range c.errors {
		id, _ := s.Value()
		ids = append(ids, id)
	}
	return ids
}

// TestWriteFailure covers the rows of the Vim 9.2 table in
// goeditor-save-before-write: a failed write leaves the buffer modified, and
// :wq and :x stay open.
func TestWriteFailure(t *testing.T) {
	// The synchronous stage fails: the command returns the error at once.
	for _, cmd := range []string{"w", "wq", "wq!", "x", "xit"} {
		t.Run("bare :"+cmd, func(t *testing.T) {
			w := &fakeWriter{}
			e := newFakeWriteEditor(w)

			err := e.ExecuteCommand(cmd)

			require.NotNil(t, err)
			assert.Equal(t, ErrFailedToSaveId, err.ID())
			assert.ErrorIs(t, err, errNoFileName)
			assert.False(t, e.Writing(), "nothing queued")
			assert.True(t, e.GetBuffer().IsModified())
			assert.False(t, e.GetState().Quit)
			assert.Empty(t, collect(e).saves)
		})
	}

	// The job fails: the error is dispatched when the write finishes.
	for _, cmd := range []string{"w", "wq", "wq!", "x"} {
		t.Run(":"+cmd+" "+badPath, func(t *testing.T) {
			w := &fakeWriter{}
			e := newFakeWriteEditor(w)

			require.Nil(t, e.ExecuteCommand(cmd+" "+badPath))
			assert.True(t, e.Writing())
			runWrites(e)

			c := collect(e)
			assert.Equal(t, []ErrorId{ErrFailedToSaveId}, c.errorIDs())
			_, err := c.errors[0].Value()
			assert.ErrorIs(t, err, errCannotWrite)
			assert.Empty(t, c.saves)
			assert.Zero(t, c.quits)
			assert.False(t, e.Writing())
			assert.True(t, e.GetBuffer().IsModified())
			assert.False(t, e.GetState().Quit)
		})
	}

	t.Run("typed :w shows the error", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{})
		keys(e, ':', 'w')
		enter(e)

		c := collect(e)
		require.Len(t, c.errors, 1)
		id, err := c.errors[0].Value()
		assert.Equal(t, ErrFailedToSaveId, id)
		assert.Equal(t, errNoFileName, err)
	})

	t.Run("Save dispatches the error", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{})
		e.Save(nil)
		assert.Equal(t, ErrFailedToSaveId, nextErrorID(t, e))
		assert.True(t, e.GetBuffer().IsModified())
	})

	// Vim's E37: after a failed :w or :wq, :q still refuses to quit.
	for _, cmd := range []string{"w", "wq", "w " + badPath, "wq " + badPath} {
		t.Run(":q after a failed :"+cmd+" gives E37", func(t *testing.T) {
			e := newFakeWriteEditor(&fakeWriter{})
			_ = e.ExecuteCommand(cmd)
			runWrites(e)

			err := e.ExecuteCommand("q")

			require.NotNil(t, err)
			assert.Equal(t, ErrUnsavedChangesId, err.ID())
			assert.False(t, e.GetState().Quit)
		})
	}

	t.Run("a later write that works saves the buffer", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{})
		require.NotNil(t, e.ExecuteCommand("w"))

		require.Nil(t, e.ExecuteCommand("w ok.txt"))
		runWrites(e)

		assert.False(t, e.GetBuffer().IsModified())
		assert.Nil(t, e.ExecuteCommand("q"))
		assert.True(t, e.GetState().Quit)
	})

	t.Run("a panic in the job is a failed write", func(t *testing.T) {
		w := &fakeWriter{named: true, panicMsg: "boom"}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("wq"))

		req, ok := e.NextWrite()
		require.True(t, ok)
		err := req.Run()
		require.ErrorIs(t, err, ErrWritePanicked)
		assert.Contains(t, err.Error(), "boom")
		e.FinishWrite(req.ID(), err)

		assert.Equal(t, []ErrorId{ErrFailedToSaveId}, collect(e).errorIDs())
		assert.True(t, e.GetBuffer().IsModified())
		assert.False(t, e.GetState().Quit)
	})
}

// TestWriteNoWriteFunc checks a host that set no write function: writing
// fails and nothing counts as saved.
func TestWriteNoWriteFunc(t *testing.T) {
	for _, cmd := range []string{"w", "wq", "x"} {
		t.Run(":"+cmd, func(t *testing.T) {
			e := newTestEditor("hello")
			keys(e, 'x')
			drainSignals(e)

			err := e.ExecuteCommand(cmd)

			require.NotNil(t, err)
			assert.Equal(t, ErrNoWriteFuncId, err.ID())
			assert.ErrorIs(t, err, ErrNoWriteFunc)
			assert.True(t, e.GetBuffer().IsModified())
			assert.False(t, e.GetState().Quit)
		})
	}

	t.Run("SetWriteFunc(nil) removes it", func(t *testing.T) {
		e := newWritingEditor("hello")
		e.SetWriteFunc(nil)
		err := e.ExecuteCommand("w")
		require.NotNil(t, err)
		assert.Equal(t, ErrNoWriteFuncId, err.ID())
	})

	t.Run(":x on an unmodified buffer still quits", func(t *testing.T) {
		e := newTestEditor("hello")
		assert.Nil(t, e.ExecuteCommand("x"))
		assert.True(t, e.GetState().Quit)
	})
}

// TestWriteSuccess checks a write that succeeds: the buffer counts as saved
// only once the job finished, SaveSignal follows, and :wq and :x quit after.
func TestWriteSuccess(t *testing.T) {
	t.Run(":w", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)

		require.Nil(t, e.ExecuteCommand("w"))

		assert.True(t, e.Writing())
		assert.True(t, e.GetBuffer().IsModified(), "not saved before the job ran")
		assert.Empty(t, collect(e).saves)

		runWrites(e)

		assert.False(t, e.Writing())
		assert.False(t, e.GetBuffer().IsModified())
		assert.Equal(t, []writeCall{{"", "ello"}}, w.written)
		c := collect(e)
		require.Len(t, c.saves, 1)
		path, content := c.saves[0].Value()
		assert.Nil(t, path)
		assert.Equal(t, "ello", content)
		assert.False(t, e.GetState().Quit)
	})

	t.Run(":w path passes the path", func(t *testing.T) {
		w := &fakeWriter{}
		e := newFakeWriteEditor(w)

		require.Nil(t, e.ExecuteCommand("w my file.txt"))
		runWrites(e)

		assert.Equal(t, []writeCall{{"my file.txt", "ello"}}, w.written)
		c := collect(e)
		require.Len(t, c.saves, 1)
		path, _ := c.saves[0].Value()
		require.NotNil(t, path)
		assert.Equal(t, "my file.txt", *path)
	})

	for _, cmd := range []string{"wq", "wq!", "x", "xit"} {
		t.Run(":"+cmd+" quits only after the write", func(t *testing.T) {
			w := &fakeWriter{named: true}
			e := newFakeWriteEditor(w)

			require.Nil(t, e.ExecuteCommand(cmd))
			assert.False(t, e.GetState().Quit, "still writing")

			runWrites(e)

			assert.Len(t, w.written, 1)
			assert.True(t, e.GetState().Quit)
			// SaveSignal comes before QuitSignal, so the host sees the save.
			saw := []string{}
			for s := nextSignal(e); s != nil; s = nextSignal(e) {
				switch s.(type) {
				case SaveSignal:
					saw = append(saw, "save")
				case QuitSignal:
					saw = append(saw, "quit")
				}
			}
			assert.Equal(t, []string{"save", "quit"}, saw)
		})
	}

	// Vim writes an unmodified buffer on :w and :wq, but not on :x.
	t.Run(":wq on an unmodified buffer writes", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newTestEditor("hello")
		e.SetWriteFunc(w.write)

		require.Nil(t, e.ExecuteCommand("wq"))
		runWrites(e)

		assert.Equal(t, []writeCall{{"", "hello"}}, w.written)
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":x on an unmodified buffer quits without writing", func(t *testing.T) {
		w := &fakeWriter{}
		e := newTestEditor("hello")
		e.SetWriteFunc(w.write)

		require.Nil(t, e.ExecuteCommand("x"))

		assert.Empty(t, w.prepared)
		assert.True(t, e.GetState().Quit)
	})

	t.Run("a nil job succeeds at once", func(t *testing.T) {
		e := newWritingEditor("hello")
		keys(e, 'x')
		drainSignals(e)

		require.Nil(t, e.ExecuteCommand("wq"))

		assert.False(t, e.Writing())
		assert.True(t, e.GetState().Quit)
		assert.Len(t, collect(e).saves, 1)
	})
}

// TestWriteInFlight covers what can happen while a write runs.
func TestWriteInFlight(t *testing.T) {
	t.Run("text typed meanwhile stays modified", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w"))
		req, _ := e.NextWrite()

		keys(e, 'x') // "llo"
		e.FinishWrite(req.ID(), req.Run())

		assert.Equal(t, []writeCall{{"", "ello"}}, w.written)
		assert.True(t, e.GetBuffer().IsModified())
		keys(e, 'u') // back to the text written
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run(":wq with text typed meanwhile stays open with E37", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{named: true})
		require.Nil(t, e.ExecuteCommand("wq"))
		keys(e, 'x')
		drainSignals(e)

		runWrites(e)

		c := collect(e)
		assert.Len(t, c.saves, 1)
		assert.Equal(t, []ErrorId{ErrUnsavedChangesId}, c.errorIDs())
		assert.Zero(t, c.quits)
		assert.False(t, e.GetState().Quit)
	})

	t.Run("writes run one at a time, in order", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w a.txt"))
		keys(e, 'x') // "llo"
		require.Nil(t, e.ExecuteCommand("w b.txt"))

		first, ok := e.NextWrite()
		require.True(t, ok)
		_, ok = e.NextWrite()
		assert.False(t, ok, "the second waits for the first")

		e.FinishWrite(first.ID(), first.Run())
		assert.True(t, e.GetBuffer().IsModified(), "a.txt has the older text")
		second, ok := e.NextWrite()
		require.True(t, ok)
		e.FinishWrite(second.ID(), second.Run())

		assert.Equal(t, []writeCall{{"a.txt", "ello"}, {"b.txt", "llo"}}, w.written)
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run("a nil job behind a running write waits its turn", func(t *testing.T) {
		calls := 0
		e := newTestEditor("hello")
		e.SetWriteFunc(func(*string, string) (WriteJob, error) {
			calls++
			if calls == 1 {
				return func(context.Context) error { return nil }, nil
			}
			return nil, nil
		})
		keys(e, 'x')
		require.Nil(t, e.ExecuteCommand("w"))
		keys(e, 'x')
		require.Nil(t, e.ExecuteCommand("w"))

		assert.True(t, e.GetBuffer().IsModified(), "the second didn't jump the queue")
		runWrites(e)
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run("a failed write cancels a pending quit", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w "+badPath))
		require.Nil(t, e.ExecuteCommand("wq"))

		runWrites(e)

		assert.Equal(t, []writeCall{{"", "ello"}}, w.written, "the second write still ran")
		assert.False(t, e.GetBuffer().IsModified())
		assert.False(t, e.GetState().Quit)
	})

	t.Run(":x waits for the write of the same text", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w"))

		require.Nil(t, e.ExecuteCommand("x"))
		runWrites(e)

		assert.Len(t, w.prepared, 1, "no second write")
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":x writes again after more typing", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w"))
		keys(e, 'x')

		require.Nil(t, e.ExecuteCommand("x"))
		runWrites(e)

		assert.Equal(t, []writeCall{{"", "ello"}, {"", "llo"}}, w.written)
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":x to another path writes", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w"))

		require.Nil(t, e.ExecuteCommand("x other.txt"))
		runWrites(e)

		assert.Equal(t, []writeCall{{"", "ello"}, {"other.txt", "ello"}}, w.written)
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":q on the modified buffer gives E37", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{named: true})
		require.Nil(t, e.ExecuteCommand("w"))

		err := e.ExecuteCommand("q")

		require.NotNil(t, err)
		assert.Equal(t, ErrUnsavedChangesId, err.ID())
		runWrites(e)
		assert.False(t, e.GetState().Quit, "the refused :q isn't run later")
	})

	t.Run(":q waits for a write of the unmodified buffer", func(t *testing.T) {
		w := &fakeWriter{}
		e := newTestEditor("hello")
		e.SetWriteFunc(w.write)
		require.Nil(t, e.ExecuteCommand("w copy.txt"))

		require.Nil(t, e.ExecuteCommand("q"))
		assert.False(t, e.GetState().Quit, "the write isn't cut off")

		runWrites(e)
		assert.Equal(t, []writeCall{{"copy.txt", "hello"}}, w.written)
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":q! quits at once and abandons the writes", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w a.txt"))
		require.Nil(t, e.ExecuteCommand("w b.txt"))
		running, ok := e.NextWrite()
		require.True(t, ok)

		require.Nil(t, e.ExecuteCommand("q!"))

		assert.True(t, e.GetState().Quit)
		assert.False(t, e.Writing())
		_ = running.Run()
		require.Len(t, w.ctxs, 1)
		assert.ErrorIs(t, w.ctxs[0].Err(), context.Canceled)
		e.FinishWrite(running.ID(), nil) // the result comes in late
		_, ok = e.NextWrite()
		assert.False(t, ok, "b.txt was dropped")
		assert.Equal(t, []writeCall{{"a.txt", "ello"}}, w.written)
	})

	t.Run("a finished write's context is cancelled", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newFakeWriteEditor(w)
		require.Nil(t, e.ExecuteCommand("w"))
		runWrites(e)
		require.Len(t, w.ctxs, 1)
		assert.Error(t, w.ctxs[0].Err())
	})

	t.Run("unknown, stale and early results are ignored", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{named: true})
		require.Nil(t, e.ExecuteCommand("w"))

		e.FinishWrite(1, nil) // not handed out yet
		assert.True(t, e.Writing())
		req, _ := e.NextWrite()
		e.FinishWrite(req.ID()+1, nil) // unknown
		assert.True(t, e.Writing())
		e.FinishWrite(req.ID(), nil)
		assert.False(t, e.Writing())
		e.FinishWrite(req.ID(), errCannotWrite) // stale
		assert.Empty(t, collect(e).errors)
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run("SetContent meanwhile: the new buffer isn't marked saved", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{named: true})
		require.Nil(t, e.ExecuteCommand("w"))
		req, _ := e.NextWrite()

		// A new buffer, edited to exactly the text being written.
		e.SetContent([]byte("hello"))
		keys(e, 'x')
		drainSignals(e)
		e.FinishWrite(req.ID(), req.Run())

		assert.Equal(t, "ello", content(e))
		assert.True(t, e.GetBuffer().IsModified(), "the write was of the old buffer")
		assert.Len(t, collect(e).saves, 1, "the write itself still happened")
	})

	t.Run("AppendBytes meanwhile counts as saved", func(t *testing.T) {
		e := newFakeWriteEditor(&fakeWriter{named: true})
		require.Nil(t, e.ExecuteCommand("w"))
		req, _ := e.NextWrite()

		e.AppendBytes([]byte("more\n"))
		e.FinishWrite(req.ID(), req.Run())

		assert.Equal(t, "ello\nmore", content(e))
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run("AppendBytes replacing an empty buffer meanwhile", func(t *testing.T) {
		w := &fakeWriter{named: true}
		e := newTestEditor("")
		e.SetWriteFunc(w.write)
		require.Nil(t, e.ExecuteCommand("w"))
		req, _ := e.NextWrite()

		e.AppendBytes([]byte("first\n"))
		e.FinishWrite(req.ID(), req.Run())

		assert.Equal(t, "first", content(e))
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run("a host buffer counts as saved only if unchanged", func(t *testing.T) {
		for _, typed := range []bool{false, true} {
			e := newTestEditor("")
			e.SetBuffer(&hostBuffer{inner: NewBufferFromBytes([]byte("hello"))})
			e.SetWriteFunc((&fakeWriter{named: true}).write)
			keys(e, 'x')
			require.Nil(t, e.ExecuteCommand("w"))
			req, _ := e.NextWrite()
			if typed {
				keys(e, 'x')
			}
			e.FinishWrite(req.ID(), req.Run())
			assert.Equal(t, typed, e.GetBuffer().IsModified(), "typed %v", typed)
		}
	})

	t.Run("the path given to the write function is a copy", func(t *testing.T) {
		var got *string
		e := newTestEditor("hello")
		e.SetWriteFunc(func(path *string, _ string) (WriteJob, error) {
			got = path
			return nil, nil
		})
		name := "a.txt"
		e.Save(&name)
		name = "changed"
		require.NotNil(t, got)
		assert.Equal(t, "a.txt", *got)
	})
}
