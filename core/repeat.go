package core

import (
	"fmt"
	"slices"
	"strconv"
)

// Repeating the last change with '.' works like Vim's redo buffer: the keys of
// the last command that changed the text are kept and fed back through
// dispatchKey. They are the keys after mapping expansion, so a mapped ,d → dw
// repeats as dw, and the replay itself runs with mappings off.
//
// A command starts on a key that reaches normal mode with nothing pending and
// ends once normal mode is idle again. That takes in the insert session a
// command opens, so cwfoo<Esc> repeats as a whole. Count digits are kept out of
// the keys: the counts typed before and after an operator multiply into one
// count (2d3w is 6dw), which a count on '.' replaces.
//
// A cursor key in insert mode ends the change, as in Vim: what was typed
// before it becomes the record, and typing after it starts a new change that
// repeats as an insert at the cursor. An accepted completion is recorded as
// the keys that would type it.
//
// A change made from visual mode records only the keys from the operator on,
// with the size of the selection, which '.' selects again from the cursor.
// A command that goes through the command line or search is not recorded yet.

// repeatRecord is a change '.' can repeat.
type repeatRecord struct {
	keys   []KeyEvent // without count digits
	count  int        // 0 if none was typed
	visual visualSpan // the selection the keys apply to, if any
}

// visualSpan is the size of the selection a visual-mode change applied to.
// '.' applies the change to the same amount of text from the cursor, as Vim
// does: as many lines, and for a characterwise selection as many characters
// on one line, or up to the same column on the last of several lines.
type visualSpan struct {
	mode  Mode // VisualMode or VisualLineMode
	lines int  // 0 if the change wasn't made from visual mode
	cols  int  // characterwise: characters if one line, else last line's end column
}

// commandRecording collects the keys of the command in progress.
type commandRecording struct {
	active     bool
	keys       []KeyEvent
	count      int  // product of the counts typed so far, 0 if none
	run        int  // count being typed, while inRun
	inRun      bool // the last key was a count digit
	edits      int  // e.editCount when the command started
	repeatable bool // cleared by commands '.' must not repeat (U, '.' itself)
	visual     visualSpan
	moved      bool // a cursor key came after the last key typed in insert mode
}

// insertSession is the insert in progress, kept so that the count of the
// command that started it repeats what was typed when it ends: 3ix<Esc>
// inserts xxx, 3ofoo<Esc> opens three lines of foo.
type insertSession struct {
	count   int        // times the text goes in, the typed one included
	newLine bool       // o/O: each repeat starts on a new line
	keys    []KeyEvent // keys typed, kept only while count > 1

	cursorAt Position // where the last key left the cursor, if tracking
	tracking bool
}

// idler is implemented by modes that can tell whether a command is part way
// typed, count aside.
type idler interface {
	Idle() bool
}

// recordKeyStart runs before a key is dispatched. It starts a recording when
// the key begins a new normal-mode command, and returns State.PendingCount
// for recordKeyEnd.
func (e *editor) recordKeyStart() (pendingCount *int) {
	if e.replaying {
		return nil
	}
	rec := &e.recording
	if !rec.active {
		if !e.normalIdle() {
			return nil
		}
		*rec = commandRecording{
			active:     true,
			keys:       rec.keys[:0],
			edits:      e.editCount,
			repeatable: true,
		}
	}
	// In visual mode only the keys from the operator on are kept: each key
	// that could be an operator drops the motions before it and notes the
	// selection it would apply to.
	if span, ok := e.idleVisualSpan(); ok {
		rec.keys = rec.keys[:0]
		rec.count, rec.inRun = 0, false
		rec.visual = span
	}
	return e.state.PendingCount
}

// recordKeyEnd runs after key was dispatched in mode. before is
// State.PendingCount as it was before. Count digits go into the count, other
// keys into the keys, and once the command is over a command that changed
// the text becomes the record '.' repeats.
func (e *editor) recordKeyEnd(mode EditorMode, key KeyEvent, before *int) {
	if mode.Name() == InsertMode {
		e.insertKey(key)
	}
	rec := &e.recording
	if e.replaying || !rec.active {
		return
	}

	switch {
	case mode.Name() == InsertMode:
		// recorded by insertKey
	case mode.Name() == NormalMode && isCountKey(key, before, e.state.PendingCount):
		rec.run, rec.inRun = *e.state.PendingCount, true
	default:
		if rec.inRun {
			rec.count = mulCount(max(rec.count, 1), rec.run)
			rec.inRun = false
		}
		rec.keys = append(rec.keys, key)
	}

	switch {
	case e.state.Mode == CommandMode, e.state.Mode == SearchMode:
		rec.active = false
	case e.normalIdle():
		rec.active = false
		e.saveRepeat(rec.keys)
	}
}

// insertKey takes a key typed in insert mode, or one an accepted completion
// stands for: the insert count keeps it to repeat, and the recording for '.'.
// After a cursor move, the next key typed starts a new change, an insert at
// the cursor with no count, as Vim's redo buffer does.
func (e *editor) insertKey(key KeyEvent) {
	if breaksInsert(key) {
		e.breakInsert()
		return
	}
	if s := &e.insert; e.state.Mode == InsertMode && s.count > 1 {
		s.keys = append(s.keys, key)
	}

	rec := &e.recording
	if e.replaying || !rec.active {
		return
	}
	if rec.moved {
		if key.Key == KeyEscape {
			return
		}
		rec.keys = append(rec.keys, KeyEvent{Rune: 'i'})
		rec.count, rec.visual, rec.moved = 0, visualSpan{}, false
	}
	rec.keys = append(rec.keys, key)
}

// breakInsert ends the insert at a cursor move, as Vim does: the count is
// dropped, and what was typed so far becomes the change '.' repeats.
func (e *editor) breakInsert() {
	e.insert.count = 1
	rec := &e.recording
	if e.replaying || !rec.active {
		return
	}
	if !rec.moved {
		e.saveRepeat(append(rec.keys, KeyEvent{Key: KeyEscape}))
		rec.keys = rec.keys[:0]
		rec.moved = true
	}
	rec.edits = e.editCount
}

// noticeCursorMove runs before each key. In insert mode, if the cursor moved
// since the last key without a key moving it (the host set it, for a mouse
// click say), it ends the insert and its undo step, as a cursor key does.
// Vim treats a click in insert mode the same way.
func (e *editor) noticeCursorMove() {
	s := &e.insert
	if !s.tracking || e.state.Mode != InsertMode || e.buffer.GetCursor().Position == s.cursorAt {
		return
	}
	s.tracking = false
	e.breakInsert()
	if e.openStep != nil && !e.openStep.host {
		e.closeUndoStep()
	}
}

// trackInsertCursor notes where the cursor is once a key, or a completion,
// has been handled in insert mode, for noticeCursorMove.
func (e *editor) trackInsertCursor() {
	if e.state.Mode == InsertMode {
		e.insert.cursorAt = e.buffer.GetCursor().Position
		e.insert.tracking = true
	}
}

// saveRepeat makes keys, with the count and selection recorded for them, the
// change '.' repeats, if the command changed the text and may be repeated.
func (e *editor) saveRepeat(keys []KeyEvent) {
	rec := &e.recording
	if !rec.repeatable || e.editCount == rec.edits || len(keys) == 0 {
		return
	}
	e.repeat = repeatRecord{
		keys:   append(e.repeat.keys[:0], keys...),
		count:  rec.count,
		visual: rec.visual,
	}
}

// idleVisualSpan returns the size of the selection, if the editor is in a
// visual mode with no command part way typed, so that the next key may be an
// operator applied to it.
func (e *editor) idleVisualSpan() (visualSpan, bool) {
	if e.state.Mode != VisualMode && e.state.Mode != VisualLineMode {
		return visualSpan{}, false
	}
	if m, ok := e.currentMode.(idler); !ok || !m.Idle() || e.state.PendingCount != nil {
		return visualSpan{}, false
	}
	start, end := NormalizeSelection(e.state.VisualStart, e.buffer.GetCursor().Position)
	span := visualSpan{mode: e.state.Mode, lines: end.Row - start.Row + 1, cols: end.Col}
	if span.lines == 1 {
		span.cols = end.Col - start.Col + 1
	}
	return span, true
}

// selectSpan selects span from the cursor, clamped to the text, and reports
// whether the editor is now in its visual mode.
func (e *editor) selectSpan(span visualSpan) bool {
	cursor := e.buffer.GetCursor()
	if span.mode == VisualLineMode {
		e.SetVisualLineMode()
	} else {
		e.SetVisualMode()
	}
	if e.state.Mode != span.mode {
		return false
	}

	row := min(cursor.Position.Row+span.lines-1, e.buffer.LineCount()-1)
	col := cursor.Position.Col
	if span.mode == VisualMode {
		col = span.cols
		if span.lines == 1 {
			col = cursor.Position.Col + span.cols - 1
		}
	}
	cursor.Position = Position{Row: row, Col: max(0, min(col, e.buffer.LineRuneCount(row)-1))}
	cursor.Preferred = cursor.Position.Col
	e.buffer.SetCursor(cursor)
	return true
}

// startInsert enters insert mode for a command whose count repeats the
// inserted text (i, a, I, A, o, O).
func (e *editor) startInsert(count int, newLine bool) {
	e.SetInsertMode()
	if e.state.Mode == InsertMode {
		e.insert.count, e.insert.newLine = count, newLine
	}
}

// resetInsert starts a new insert session with no count, on entering insert
// mode.
func (e *editor) resetInsert() {
	e.insert = insertSession{count: 1, keys: e.insert.keys[:0]}
}

// Limits on what an insert count may add. Vim keeps going until it is done
// or interrupted with Ctrl-C, but HandleKey can't be interrupted, so a count
// typed by mistake (999999999ix) would hang the host or exhaust its memory.
const (
	// maxRepeatedInsert caps the text added, in runes.
	maxRepeatedInsert = 10_000_000
	// maxReplayedKeys caps the keys replayed when the text can't be worked
	// out up front. Each replayed key costs as much as the line is long, so
	// this is far lower.
	maxReplayedKeys = 100_000
)

// repeatInsert adds the insert's text again for its count, as the insert
// ends, with one edit. Only when the keys delete text from before the insert
// (Backspace past its start) does each copy depend on the last, and then the
// keys are replayed one by one. A count past the limits fails with
// ErrTextTooLong and adds nothing, as Vim refuses a put that would be too
// long.
func (e *editor) repeatInsert() *EditorError {
	s := &e.insert
	repeats := s.count - 1
	s.count = 1
	if repeats <= 0 {
		return nil
	}

	text, ok := insertedText(s.keys, s.newLine)
	if !ok {
		return e.replayInsert(repeats)
	}
	if len(text) == 0 {
		return nil
	}
	if len(text) > maxRepeatedInsert/repeats {
		return textTooLong(repeats, len(text))
	}

	all := slices.Repeat(text, repeats)
	cursor := e.buffer.GetCursor()
	at := cursor.Position
	if err := e.buffer.InsertRunesAt(at.Row, at.Col, all); err != nil {
		return &EditorError{id: ErrInvalidPositionId, err: err}
	}
	cursor.Position = insertEnd(at, string(all))
	cursor.Preferred = cursor.Position.Col
	e.buffer.SetCursor(cursor)
	return nil
}

// replayInsert types the insert's keys again, repeats times.
func (e *editor) replayInsert(repeats int) *EditorError {
	s := &e.insert
	if n := len(s.keys) + 1; n > maxReplayedKeys/repeats {
		return textTooLong(repeats, n)
	}
	for range repeats {
		if s.newLine {
			_ = e.currentMode.HandleKey(e, e.buffer, KeyEvent{Key: KeyEnter})
		}
		for _, key := range s.keys {
			_ = e.currentMode.HandleKey(e, e.buffer, key)
		}
	}
	return nil
}

func textTooLong(repeats, size int) *EditorError {
	return &EditorError{
		id:  ErrTextTooLongId,
		err: fmt.Errorf("%w: %d copies of %d", ErrTextTooLong, repeats, size),
	}
}

// insertedText returns the text keys type in insert mode, after a line break
// if newLine is set. It follows insertMode.HandleKey: Backspace deletes the
// rune before, and Ctrl keys and unknown special keys type nothing. It
// returns false if a Backspace would delete text from before the insert.
func insertedText(keys []KeyEvent, newLine bool) ([]rune, bool) {
	text := make([]rune, 0, len(keys)+1)
	if newLine {
		text = append(text, '\n')
	}
	for _, key := range keys {
		switch key.Key {
		case KeyEnter:
			text = append(text, '\n')
		case KeyTab:
			text = append(text, '\t')
		case KeyBackspace:
			if len(text) == 0 {
				return nil, false
			}
			text = text[:len(text)-1]
		case KeyEscape, KeyLeft, KeyRight, KeyUp, KeyDown:
			return nil, false // they end the insert or drop its count
		default:
			if key.Rune != 0 && key.Modifiers&ModCtrl == 0 {
				text = append(text, key.Rune)
			}
		}
	}
	return text, true
}

// recordCompletion records an accepted completion as the keys that would type
// it, as Vim's redo buffer does: a Backspace for each character of the word
// it completed, then its text. '.' and an insert count then repeat the
// completed text. Text the completion removed after the cursor isn't
// recorded; Vim's completion never removes any.
func (e *editor) recordCompletion(replaced int, text string) {
	if e.state.Mode != InsertMode {
		return
	}
	for range replaced {
		e.insertKey(KeyEvent{Key: KeyBackspace})
	}
	for _, r := range text {
		if r == '\n' {
			e.insertKey(KeyEvent{Key: KeyEnter})
		} else {
			e.insertKey(KeyEvent{Rune: r})
		}
	}
}

// isCountKey reports whether key was taken as a count digit: State.PendingCount
// went from before to after by appending it.
func isCountKey(key KeyEvent, before, after *int) bool {
	if after == nil || key.Rune < '0' || key.Rune > '9' || key.Modifiers != 0 {
		return false
	}
	n := 0
	if before != nil {
		n = *before
	}
	return *after == appendCountDigit(n, int(key.Rune-'0'))
}

// normalIdle reports whether the editor is in normal mode with no command
// part way typed, count included.
func (e *editor) normalIdle() bool {
	m, ok := e.currentMode.(*normalMode)
	return ok && e.state.Mode == NormalMode && m.Idle() && e.state.PendingCount == nil
}

// noRepeat keeps the command in progress from becoming the record '.'
// repeats, though it changes the text.
func (e *editor) noRepeat() {
	e.recording.repeatable = false
}

// repeatChange replays the last change. A count other than 0 replaces the
// recorded one, for this and later repeats, as in Vim; a change made from
// visual mode ignores it. The replay is one undo step. If a key fails, or the command ends before its last key (a motion
// that found nothing ends it early), the rest is dropped and the editor goes
// back to normal mode; the record is kept.
func (e *editor) repeatChange(count int) *EditorError {
	e.noRepeat()
	if len(e.repeat.keys) == 0 {
		return nil
	}
	visual := e.repeat.visual.lines > 0
	if count > 0 && !visual {
		e.repeat.count = count
	}

	e.replaying = true
	defer func() { e.replaying = false }()

	if visual && !e.selectSpan(e.repeat.visual) {
		return nil
	}

	keys := make([]KeyEvent, 0, len(e.repeat.keys)+4)
	if e.repeat.count > 0 {
		for _, r := range strconv.Itoa(e.repeat.count) {
			keys = append(keys, KeyEvent{Rune: r})
		}
	}
	keys = append(keys, e.repeat.keys...)

	for i, key := range keys {
		if err := e.dispatchKey(key); err != nil {
			e.abortReplay()
			return err
		}
		if i < len(keys)-1 && e.normalIdle() {
			break
		}
	}
	return nil
}

// abortReplay drops whatever command a failed replay left part way.
func (e *editor) abortReplay() {
	if m, ok := e.currentMode.(*normalMode); ok {
		m.clearPendingState(e)
		return
	}
	e.SetNormalMode()
}

// resetRepeat forgets the record, for new content.
func (e *editor) resetRepeat() {
	e.repeat = repeatRecord{}
	e.recording = commandRecording{}
}
