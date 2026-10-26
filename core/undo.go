package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Undo history is a list of undo steps, each holding the edits one command
// made. Recording, undo and redo cost the size of the change, not of the
// buffer.
//
// Boundaries follow Vim, which ends an undo step whenever it waits for a new
// typed command: dispatchKey opens a step before each key and closes it once
// the mode is idle and no mapping is still feeding keys. A whole insert
// session, a change and the insert that follows it, and a whole mapping are
// therefore each one step. Edits made outside a key (by the host, or by
// InsertCompletion between keys) collect in a step of their own that closes
// when the next key arrives.

// edit is one change recorded by recordingBuffer: starting at position at,
// removed was deleted and inserted put in its place. Positions refer to the
// buffer as it stood just before the edit.
type edit struct {
	at       Position
	removed  string
	inserted string
}

// undoStep is what one 'u' undoes.
type undoStep struct {
	edits        []edit // in the order applied
	cursorBefore Cursor // where the command started, for 'U'
	cursorAfter  Cursor // where the command ended, for 'U'
	host         bool   // opened by an edit made outside a key

	// cursor is where the change began, which undo and redo go back to, as
	// Vim's uh_cursor: the start an operator set with markChangeStart, or
	// else where the cursor was at the first edit.
	cursor Position
	marked bool // cursor was set by markChangeStart

	// While the step is open, typing grows the last edit's insertion here
	// rather than by string concatenation, which would copy it on every key.
	// typedEnd is where that insertion ends. flushTyped moves it into edits.
	typed    []byte
	typedEnd Position
}

// flushTyped stores the insertion being typed into the last edit.
func (s *undoStep) flushTyped() {
	if s.typed != nil {
		s.edits[len(s.edits)-1].inserted = string(s.typed)
		s.typed = nil
	}
}

// undo reverts the step's edits, last first, appending what it applied to
// applied. An error means the buffer no longer matches the history — it was
// edited behind the recording wrapper — and leaves the step part applied.
func (s *undoStep) undo(buf Buffer, applied []TextEdit) ([]TextEdit, error) {
	var err error
	for _, ed := range slices.Backward(s.edits) {
		if applied, err = replaceAt(buf, ed.at, ed.inserted, ed.removed, applied); err != nil {
			return applied, err
		}
	}
	return applied, nil
}

// redo re-applies the step's edits in order, as undo does.
func (s *undoStep) redo(buf Buffer, applied []TextEdit) ([]TextEdit, error) {
	var err error
	for _, ed := range s.edits {
		if applied, err = replaceAt(buf, ed.at, ed.removed, ed.inserted, applied); err != nil {
			return applied, err
		}
	}
	return applied, nil
}

// replaceAt replaces the text old, starting at position at, with text, and
// appends the replacement to applied as a TextEdit. It is the primitive undo
// and redo share. If the insert fails after the delete succeeded, the delete
// alone is appended, so applied still describes the text.
func replaceAt(buf Buffer, at Position, old, text string, applied []TextEdit) ([]TextEdit, error) {
	n := utf8.RuneCountInString(old)
	if err := buf.DeleteRunesAt(at.Row, at.Col, n); err != nil {
		return applied, fmt.Errorf("deleting %d runes at %d:%d: %w", n, at.Row, at.Col, err)
	}
	end := insertEnd(at, old)
	if text != "" {
		if err := buf.InsertRunesAt(at.Row, at.Col, []rune(text)); err != nil {
			if old != "" {
				applied = append(applied, TextEdit{Start: at, End: end})
			}
			return applied, fmt.Errorf("inserting %d runes at %d:%d: %w",
				utf8.RuneCountInString(text), at.Row, at.Col, err)
		}
	}
	return append(applied, TextEdit{Start: at, End: end, Text: text}), nil
}

// insertEnd returns where text ends once inserted at position at.
func insertEnd(at Position, text string) Position {
	i := strings.LastIndexByte(text, '\n')
	if i < 0 {
		return Position{Row: at.Row, Col: at.Col + utf8.RuneCountInString(text)}
	}
	return Position{
		Row: at.Row + strings.Count(text, "\n"),
		Col: utf8.RuneCountInString(text[i+1:]),
	}
}

// undoCorrupt handles a step that failed part way. The text no longer matches
// the history, so applying further steps would only damage it more: drop the
// history, as Vim does on E438.
func (e *editor) undoCorrupt(err error) error {
	e.resetHistory()
	e.buffer.SetCursor(e.clampCursor(e.buffer.GetCursor()))
	e.ScrollViewport()
	return fmt.Errorf("%w: %w", ErrUndoCorrupt, err)
}

// clampCursor moves cursor back inside the text: onto an existing line, and
// no further than that line's end.
func (e *editor) clampCursor(cursor Cursor) Cursor {
	cursor.Position.Row = max(0, min(cursor.Position.Row, e.buffer.LineCount()-1))
	lineLen := e.buffer.LineRuneCount(cursor.Position.Row)
	cursor.Position.Col = max(0, min(cursor.Position.Col, lineLen))
	return cursor
}

// keepCursorOnChar runs before each key. Edits keep the cursor inside the
// text (see recordingBuffer), but a host edit that shortens the cursor line
// can still leave it after the last character, where normal mode can't
// stand: put it back on that character, as Vim does after a script shortens
// the cursor line. Preferred is left as it is.
func (e *editor) keepCursorOnChar() {
	if e.state.Mode != NormalMode {
		return
	}
	cursor := e.buffer.GetCursor()
	if last := lastCol(e.buffer, cursor.Position.Row, false); cursor.Position.Col > last {
		cursor.Position.Col = last
		e.buffer.SetCursor(cursor)
	}
}

// recordingBuffer wraps the editor's buffer so that every edit, whoever makes
// it, lands in the open undo step. Undo and redo apply their edits to the
// wrapped buffer directly, so they are not recorded a second time.
//
// A delete or SetContent also keeps the cursor inside the text, so a host
// editing through GetBuffer() can't leave it past a line's end or the last
// line, whether or not the wrapped Buffer clamps it. Inserts can't do that.
type recordingBuffer struct {
	Buffer
	e *editor
}

func (b *recordingBuffer) InsertRunesAt(row, col int, runes []rune) error {
	cursor := b.GetCursor()
	if err := b.Buffer.InsertRunesAt(row, col, runes); err != nil {
		return err
	}
	if len(runes) > 0 {
		b.e.recordEdit(cursor, edit{at: Position{Row: row, Col: col}, inserted: string(runes)})
	}
	return nil
}

func (b *recordingBuffer) DeleteRunesAt(row, col int, count int) *EditorError {
	cursor := b.GetCursor()
	removed := textAt(b.Buffer, row, col, count)
	if err := b.Buffer.DeleteRunesAt(row, col, count); err != nil {
		return err
	}
	if removed != "" {
		b.e.recordEdit(cursor, edit{at: Position{Row: row, Col: col}, removed: removed})
	}
	b.keepCursorInText()
	return nil
}

// SetContent replaces the whole text, which starts a new undo history and
// forgets the change '.' repeats, as loading a file does in Vim.
func (b *recordingBuffer) SetContent(content []byte) {
	b.Buffer.SetContent(content)
	b.keepCursorInText()
	b.e.resetHistory()
	b.e.resetRepeat()
}

// keepCursorInText clamps the cursor after an edit. Preferred is left as it
// is, as Vim's curswant.
func (b *recordingBuffer) keepCursorInText() {
	cursor := b.GetCursor()
	if clamped := b.e.clampCursor(cursor); clamped != cursor {
		b.SetCursor(clamped)
	}
}

// textAt returns the text DeleteRunesAt(row, col, count) removes from buf: up
// to count runes from (row, col), a line break counting as one.
func textAt(buf Buffer, row, col, count int) string {
	if count <= 0 || row < 0 || row >= buf.LineCount() {
		return ""
	}
	line := buf.GetLineRunes(row)
	if col < 0 || col > len(line) {
		return ""
	}

	var sb strings.Builder
	for {
		n := min(count, len(line)-col)
		sb.WriteString(string(line[col : col+n]))
		count -= n
		row++
		if count == 0 || row >= buf.LineCount() {
			return sb.String()
		}
		sb.WriteByte('\n')
		count--
		line, col = buf.GetLineRunes(row), 0
	}
}

// recordEdit adds ed to the open undo step. With no step open the edit was
// made outside a key, so it opens a host step; cursor is where the cursor was
// before the edit.
func (e *editor) recordEdit(cursor Cursor, ed edit) {
	e.editCount++
	if e.openStep == nil {
		e.openStep = &undoStep{cursorBefore: cursor, host: true}
	}

	// Typing inserts one rune at a time, each where the last one ended. Merge
	// those into one edit, so an insert session costs one edit, not one per
	// key.
	step := e.openStep
	n := len(step.edits)
	if n == 0 && !step.marked {
		step.cursor = cursor.Position
	}
	if n > 0 && ed.removed == "" && step.edits[n-1].removed == "" && ed.at == step.typedEnd {
		if step.typed == nil {
			step.typed = []byte(step.edits[n-1].inserted)
		}
		step.typed = append(step.typed, ed.inserted...)
		step.typedEnd = insertEnd(ed.at, ed.inserted)
		return
	}

	step.flushTyped()
	step.edits = append(step.edits, ed)
	if ed.removed == "" {
		step.typedEnd = insertEnd(ed.at, ed.inserted)
	}
}

// changeStarter is implemented by the editor so that a command can say where
// its change starts, without it being part of the public Editor interface.
type changeStarter interface {
	markChangeStart(pos Position)
}

// markChangeStart tells editor that the change about to be made starts at
// pos, which undo and redo go back to. Commands that edit away from the
// cursor call it: an operator with the start of its text (Vim's oap->start:
// the earlier of the cursor and where the motion goes), X with the first
// character it deletes, o and O with the cursor before they move it.
func markChangeStart(editor Editor, pos Position) {
	if s, ok := editor.(changeStarter); ok {
		s.markChangeStart(pos)
	}
}

// markChangeStart records pos as where the open step's change starts, unless
// the step has already made an edit or recorded a start: the first command to
// say where its change starts is the outermost (V before the case operator
// it runs, say).
func (e *editor) markChangeStart(pos Position) {
	if s := e.openStep; s != nil && len(s.edits) == 0 && !s.marked {
		s.cursor = pos
		s.marked = true
	}
}

// openUndoStep runs before each key is dispatched. It closes a step left open
// by host edits, then opens one for the key unless a command still in
// progress (an insert session, say) already has one.
func (e *editor) openUndoStep() {
	if e.openStep != nil && e.openStep.host {
		e.closeUndoStep()
	}
	if e.openStep == nil {
		e.openStep = &undoStep{cursorBefore: e.buffer.GetCursor()}
	}
}

// closeUndoStep ends the open step and adds it to the history, dropping any
// redo. A step with no edits is discarded.
func (e *editor) closeUndoStep() {
	step := e.openStep
	e.openStep = nil
	if step == nil || len(step.edits) == 0 {
		return
	}
	step.flushTyped()
	step.cursorAfter = e.buffer.GetCursor()

	clear(e.undoSteps[e.undoPos:])
	e.undoSteps = e.undoSteps[:e.undoPos]

	// Extend or restart the line-undo run before undoPos moves, so
	// undoLineStart records the position before the run's first step.
	e.trackUndoLineRun(step)

	e.undoSteps = append(e.undoSteps, *step)
	e.undoPos++

	if dropped := len(e.undoSteps) - int(e.maxHistory); dropped > 0 {
		clear(e.undoSteps[:dropped])
		e.undoSteps = e.undoSteps[dropped:]
		e.undoPos -= dropped

		// Shift the line-undo anchor to match; if the run's starting state has
		// aged out of the history there is nothing left to restore.
		if e.undoLineRow >= 0 {
			e.undoLineStart -= dropped
			if e.undoLineStart < 0 {
				e.undoLineRow = -1
			}
		}
	}
}

// endsUndoStep reports whether the step open for key should close once the
// key has been handled by mode (the mode current when it arrived).
func (e *editor) endsUndoStep(mode EditorMode, key KeyEvent) bool {
	// A '.' replay is one step, closed by the key that started it.
	if e.replaying {
		return false
	}
	if s, ok := mode.(undoSyncer); ok && s.syncsUndo(key) {
		return true
	}
	return e.mapDepth == 0 && e.undoIdle()
}

// undoIdle reports whether the editor is waiting for a new command: not in a
// Vim insert session and not part way through a normal-mode command.
func (e *editor) undoIdle() bool {
	if e.state.Mode == InsertMode && e.state.VimMode {
		return false
	}
	return !e.currentMode.OperatorPending() && !e.currentMode.AwaitingLiteral()
}

// undoSyncer is implemented by modes that end an undo step part way through
// a session. Insert mode does so for cursor keys, as Vim does.
type undoSyncer interface {
	syncsUndo(key KeyEvent) bool
}

// resetHistory forgets all undo steps, for a new buffer or new content.
func (e *editor) resetHistory() {
	e.undoSteps = nil
	e.undoPos = 0
	e.openStep = nil
	e.undoLineRow = -1
}

// trackUndoLineRun maintains the line-undo run used by 'U'. A change on a
// different row than the current run starts a fresh run anchored at the state
// preceding it; further changes to the same row extend the run, so a single 'U'
// undoes all of them at once — Vim's behaviour.
func (e *editor) trackUndoLineRun(step *undoStep) {
	preRow := step.cursorBefore.Position.Row
	postRow := step.cursorAfter.Position.Row

	// The run continues while changes keep landing on the same line. Both the
	// pre- and post-change rows count, because operations like 'o' create the
	// line they then edit: the command starts on one row and the change lands
	// on the next, and Vim treats that as a single run.
	if e.undoLineRow >= 0 && (preRow == e.undoLineRow || postRow == e.undoLineRow) {
		return
	}

	e.undoLineRow = postRow
	e.undoLineStart = e.undoPos
}

// Undo reverts the latest undo step and returns the edits it applied. If the
// step no longer fits the text it fails with ErrUndoCorrupt, returning the
// edits it applied before the failure, which still describe the text.
func (e *editor) Undo() ([]TextEdit, error) {
	e.closeUndoStep()
	if e.undoPos == 0 {
		return nil, errors.New("already at oldest change")
	}

	// Moving through history invalidates the current line-undo run: its anchor
	// no longer describes a trailing sequence of changes.
	e.undoLineRow = -1

	e.undoPos--
	step := &e.undoSteps[e.undoPos]
	edits, err := step.undo(e.buffer.Buffer, nil)
	if err != nil {
		return edits, e.undoCorrupt(err)
	}

	e.buffer.SetCursor(e.undoCursor(step, edits))

	e.ScrollViewport()

	return edits, nil
}

// Redo re-applies the undo step after the current one and returns the edits
// it applied, failing as Undo does.
func (e *editor) Redo() ([]TextEdit, error) {
	e.closeUndoStep()
	if e.undoPos >= len(e.undoSteps) {
		return nil, errors.New("already at newest change")
	}

	e.undoLineRow = -1

	step := &e.undoSteps[e.undoPos]
	e.undoPos++
	edits, err := step.redo(e.buffer.Buffer, nil)
	if err != nil {
		return edits, e.undoCorrupt(err)
	}
	e.buffer.SetCursor(e.undoCursor(step, edits))

	e.ScrollViewport()

	return edits, nil
}

// undoCursor returns where undo and redo of step, which applied edits, leave
// the cursor. As Vim's u_undoredo, both go back to where the change began
// (step.cursor) when that is on or next to the changed lines, and otherwise
// to the first changed line. On any other row than step.cursor's, which the
// change may have removed, the cursor goes to the first non-blank. In normal
// mode the column stays on a character.
func (e *editor) undoCursor(step *undoStep, edits []TextEdit) Cursor {
	at := step.cursor
	if first, last, ok := changedRows(edits); ok && (at.Row < first-1 || at.Row > last+1) {
		at.Row = first
	}

	cursor := e.buffer.GetCursor()
	cursor.Position.Row = max(0, min(at.Row, e.buffer.LineCount()-1))
	if cursor.Position.Row == step.cursor.Row {
		cursor.Position.Col = min(at.Col, lastCol(e.buffer, cursor.Position.Row, !e.IsNormalMode()))
		cursor.Preferred = cursor.Position.Col
	} else {
		cursor.MoveToFirstNonBlank(e.buffer)
	}
	return cursor
}

// changedRows returns the first and last rows that edits, applied in order,
// changed in the text they leave. ok is false when there are no edits.
func changedRows(edits []TextEdit) (first, last int, ok bool) {
	for i, ed := range edits {
		end := insertEnd(ed.Start, ed.Text).Row
		if i == 0 {
			first, last = ed.Start.Row, end
			continue
		}
		// Rows below the edit move by the lines it added or removed.
		if last >= ed.End.Row {
			last += end - ed.End.Row
		}
		first, last = min(first, ed.Start.Row), max(last, end)
	}
	return first, last, len(edits) > 0
}

// UndoLine implements Vim's 'U': undo all the latest changes made to a single
// line — the line where the most recent change happened. It returns the edits
// it applied, failing as Undo does.
//
// The restore is applied as a *new* step rather than by moving undoPos, which
// gives two Vim behaviours for free: 'u' undoes a 'U', and a second 'U'
// toggles back to the changed version, because the run is re-anchored at the
// position just before this restore.
func (e *editor) UndoLine() ([]TextEdit, error) {
	e.closeUndoStep()
	if e.undoLineRow < 0 || e.undoLineStart < 0 || e.undoLineStart >= e.undoPos {
		return nil, ErrNoLineToRestore
	}

	row := e.undoLineRow

	// U changes the text through the recording buffer, but '.' doesn't
	// repeat it.
	e.noRepeat()

	// Revert the run through the recording buffer, so the reverting edits
	// become the new step.
	e.openStep = &undoStep{cursorBefore: e.buffer.GetCursor()}
	var edits []TextEdit
	for i := e.undoPos - 1; i >= e.undoLineStart; i-- {
		var err error
		if edits, err = e.undoSteps[i].undo(e.buffer, edits); err != nil {
			return edits, e.undoCorrupt(err)
		}
	}

	// Park the cursor on the restored line, clamped to the restored content.
	cursor := e.buffer.GetCursor()
	cursor.Position.Row = min(row, max(0, e.buffer.LineCount()-1))
	cursor.Position.Col = 0
	cursor.Preferred = 0
	e.buffer.SetCursor(cursor)

	e.closeUndoStep()

	// closeUndoStep re-derives the run from the step's cursors; override it so
	// the next 'U' reverts this step.
	if e.undoPos > 0 {
		e.undoLineRow = row
		e.undoLineStart = e.undoPos - 1
	}

	e.ScrollViewport()

	return edits, nil
}
