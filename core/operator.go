package core

import "strings"

// operator is a normal-mode operator that takes a motion or text object: d, y
// or c. The case operators (gU, gu, g~) take a separate path (case.go).
type operator int

const (
	opNone operator = iota
	opDelete
	opYank
	opChange
)

// applyOperatorToRange applies a delete/yank/change operator to the character-wise
// range [start, end) — end exclusive, matching Vim's exclusive motions. The two
// positions may be given in either order. For inclusive motions (e.g. ge), the
// caller moves the end position one character right before calling. Exclusive
// motions that can end on a later line go through applyOperatorToMotion.
// An empty range moves the cursor to it and goes to applyOperatorToEmptyRange.
//
// As Vim's op_delete, a delete over several lines that starts in the indent
// and leaves only blanks after its end deletes the lines instead.
func applyOperatorToRange(editor Editor, buffer Buffer, op operator, start, end Position) *EditorError {
	// Normalize so start is before end
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	if op != opYank {
		markChangeStart(editor, start)
	}
	if start == end {
		// After a motion the cursor is already there; an empty text object
		// (di"" or ciw on an empty line) puts it on the empty range.
		cursor := buffer.GetCursor()
		cursor.Position = start
		buffer.SetCursor(cursor)
		return applyOperatorToEmptyRange(editor, op)
	}
	if op == opDelete && deletesLines(buffer, start, end) {
		return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
	}

	cursor := buffer.GetCursor()

	switch op {
	case opDelete, opChange:
		if err := deleteRange(buffer, start, end); err != nil {
			return err
		}
		cursor.Position = start
		if op == opDelete {
			// A delete up to the end of the line leaves start past it.
			cursor.Position.Col = min(start.Col, lastCol(buffer, start.Row, false))
		}
		buffer.SetCursor(cursor)
		if op == opChange {
			editor.SetInsertMode()
		}

	case opYank:
		// editor.Copy is inclusive of both ends, so step the exclusive end back
		// one. From column 0 that is the line break above, which Copy includes.
		last := Position{Row: end.Row, Col: end.Col - 1}
		if end.Col == 0 {
			last = Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}
		}
		return yankRange(editor, buffer, start, last)
	}

	return nil
}

// emptyRangeOperator is implemented by the editor so that an operator over an
// empty range can act as in Vim, without it being part of the public Editor
// interface.
type emptyRangeOperator interface {
	countChange()
	yankNothing() error
}

// applyOperatorToEmptyRange applies an operator whose motion succeeded
// without moving: d0 in column 0, dl on an empty line, $dTx with x just
// before the cursor. Vim still runs it: a delete or a change becomes the
// change '.' repeats, a change starts insert mode, and a yank empties the
// register. A motion that fails (b at the start of the buffer, % with no
// bracket) must not get here: Vim does nothing then.
func applyOperatorToEmptyRange(editor Editor, op operator) *EditorError {
	o, ok := editor.(emptyRangeOperator)
	switch op {
	case opDelete, opChange:
		if ok {
			o.countChange()
		}
		if op == opChange {
			editor.SetInsertMode()
		}

	case opYank:
		if !ok {
			return nil
		}
		if err := o.yankNothing(); err != nil {
			return &EditorError{id: ErrFailedToYankId, err: err}
		}
	}
	return nil
}

// applyOperatorToMotion applies a delete/yank/change operator to what an
// exclusive motion (w, b, {, }, ...) moves over, from either end. As Vim
// (:h exclusive-linewise), an end in column 0 of a later line goes back to
// the end of the line above, and if the operator started in the indent it
// acts on the lines instead.
func applyOperatorToMotion(editor Editor, buffer Buffer, op operator, start, end Position) *EditorError {
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	end, adjusted := exclusiveEnd(buffer, start, end)
	if adjusted && inIndent(buffer, start) {
		if op != opYank {
			markChangeStart(editor, start)
		}
		// As Vim, the cursor goes to the start, where a linewise y leaves it.
		cursor := buffer.GetCursor()
		cursor.Position = start
		buffer.SetCursor(cursor)
		return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
	}
	return applyOperatorToRange(editor, buffer, op, start, end)
}

// exclusiveEnd applies Vim's rule for an exclusive motion (:h exclusive): an
// end in column 0 of a later line than start goes back to the end of the
// line above, so the range doesn't take that line break. It reports whether
// it moved the end. start must not be after end.
func exclusiveEnd(buffer Buffer, start, end Position) (Position, bool) {
	if end.Row > start.Row && end.Col == 0 {
		return Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}, true
	}
	return end, false
}

// applyOperatorLeft applies a delete/yank/change operator over count
// characters left of the cursor: dh, yh, ch, and X. They stay on the line
// unless wrap (the editor's MoveAcrossLines, Vim's 'h' in 'whichwrap') is on.
// Then from column 0 they go on to the line above, as Vim's nv_left: d and c
// after its last character, so its line break is deleted and the lines join,
// y onto its last character, which with the usual exclusive end in column 0
// (applyOperatorToMotion) leaves the line break out.
func applyOperatorLeft(editor Editor, buffer Buffer, op operator, count int, wrap bool) *EditorError {
	cursor := buffer.GetCursor().Position
	stop := stopOnLastChar
	if op == opDelete || op == opChange {
		stop = stopPastLineEnd
	}
	start, joins := charsLeftTarget(buffer, cursor, count, wrap, stop)
	if joins {
		// Vim doesn't move this end back to the line above (CA_NO_ADJ_OP_END).
		return applyOperatorToRange(editor, buffer, op, start, cursor)
	}
	return applyOperatorToMotion(editor, buffer, op, start, cursor)
}

// leftStop says where an operator's h lands when it steps from column 0 onto
// a non-empty line above. Vim's nv_left decides it inside its step loop, so
// it changes where every later step of the count lands, not only the last.
type leftStop int

const (
	// stopOnLastChar: y and the case operators (gU, gu, g~) land on the
	// last character, leaving the line break out.
	stopOnLastChar leftStop = iota
	// stopPastLineEnd: d, c and X land after the last character, so the
	// line break is in the range and the lines join.
	stopPastLineEnd
)

// charsLeftTarget returns where count h's go from pos after an operator (see
// applyOperatorLeft), and whether a step went past the end of a non-empty
// line above, which only stopPastLineEnd allows.
func charsLeftTarget(buffer Buffer, pos Position, count int, wrap bool, stop leftStop) (Position, bool) {
	joins := false
	for range count {
		if pos.Col > 0 {
			pos.Col, _ = charsBack(buffer.GetLineRunes(pos.Row), pos.Col, 1)
			continue
		}
		if !wrap || pos.Row == 0 {
			break
		}
		pos.Row--
		if stop == stopPastLineEnd && buffer.LineRuneCount(pos.Row) > 0 {
			pos.Col = buffer.LineRuneCount(pos.Row)
			joins = true
		} else {
			pos.Col = lastCol(buffer, pos.Row, false)
		}
	}
	return pos, joins
}

// applyOperatorRight applies a delete/yank/change operator over count
// characters from the cursor: dl, yl, cl. As Vim's nv_right, the last
// character of a line counts as one step (the motion becomes inclusive) and,
// with wrap (MoveAcrossLines), its line break as the next; an empty line's
// line break is one step. So $d2l still deletes only the last character, as
// the exclusive end in column 0 goes back to the line above, and dl on an
// empty line deletes the line.
func applyOperatorRight(editor Editor, buffer Buffer, op operator, count int, wrap bool) *EditorError {
	start := buffer.GetCursor().Position
	return applyOperatorToMotion(editor, buffer, op, start, charsRightEnd(buffer, start, count, wrap))
}

// charsRightEnd returns the exclusive end of count l's from start after an
// operator (see applyOperatorRight), before the exclusive-motion adjustment.
func charsRightEnd(buffer Buffer, start Position, count int, wrap bool) Position {
	pos, inclusive := start, false
	for range count {
		if pos.Col < lastCol(buffer, pos.Row, false) {
			pos.Col, _ = charsForward(buffer.GetLineRunes(pos.Row), pos.Col, 1)
			continue
		}
		empty := buffer.LineRuneCount(pos.Row) == 0
		if !wrap || pos.Row >= buffer.LineCount()-1 {
			inclusive = !empty
			break
		}
		if !inclusive && !empty {
			inclusive = true
		} else {
			pos = Position{Row: pos.Row + 1}
			inclusive = false
		}
	}
	if inclusive {
		pos = afterChar(buffer, pos)
	}
	return pos
}

// yankRange yanks the character-wise range [start, last], both inclusive; a
// last on a line break includes it. The cursor goes to start.
func yankRange(editor Editor, buffer Buffer, start, last Position) *EditorError {
	state := editor.GetState()
	state.VisualStart = last
	state.YankSelection = SelectionCharacter
	editor.SetState(state)

	cursor := buffer.GetCursor()
	cursor.Position = start
	buffer.SetCursor(cursor)

	if err := editor.Copy(yankType); err != nil {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}
	// T can start the range on a composing character ($yT on a target with
	// composing characters); the yank takes it, but the cursor goes back to
	// the start of its character.
	cursor = buffer.GetCursor()
	if col := charStart(buffer.GetLineRunes(cursor.Position.Row), cursor.Position.Col); col != cursor.Position.Col {
		cursor.Position.Col = col
		buffer.SetCursor(cursor)
	}
	return nil
}

// applyOperatorToWords applies a delete/yank/change operator to what w (W
// with bigWordClass) moves over from the cursor. After an operator the last
// word stops at the end of its line (:h dw).
func applyOperatorToWords(editor Editor, buffer Buffer, op operator, count int, cls func(rune) int) *EditorError {
	cursor := buffer.GetCursor()
	start := cursor.Position
	_ = cursor.moveWordForward(buffer, count, cls, true)
	return applyOperatorToMotion(editor, buffer, op, start, cursor.Position)
}

// inIndent reports whether only blanks come before pos on its line.
func inIndent(buffer Buffer, pos Position) bool {
	return pos.Col <= firstNonBlankCol(buffer.GetLineRunes(pos.Row))
}

// onlyBlanksFrom reports whether only blanks come from pos to the end of its line.
func onlyBlanksFrom(buffer Buffer, pos Position) bool {
	line := buffer.GetLineRunes(pos.Row)
	col := min(pos.Col, len(line))
	return col+firstNonBlankCol(line[col:]) == len(line)
}

// deletesLines reports whether a delete of [start, end) over several lines
// deletes the lines instead (Vim's op_delete): it starts in the indent and
// leaves only blanks after its end.
func deletesLines(buffer Buffer, start, end Position) bool {
	return end.Row > start.Row && inIndent(buffer, start) && onlyBlanksFrom(buffer, end)
}

// lineEndTarget returns where [count]$ from the cursor ends, after the last
// character of the line count-1 lines down (the last line if there are
// fewer), for an operator's exclusive end. On the last line a count above 1
// fails as $ does: the operator does nothing, and j and k go to the end of
// every line.
func (m *normalMode) lineEndTarget(buffer Buffer, count int) (Position, bool) {
	cursor := buffer.GetCursor()
	target := cursor
	if target.moveToLineEnd(buffer, count, true) != nil {
		cursor.Preferred = maxCol
		buffer.SetCursor(cursor)
		m.keepColumn = true
		return Position{}, false
	}
	return target.Position, true
}

// applyOperatorToLineEnd applies a delete/yank/change operator to [count]$
// (d$, y$, c$, D, C): charwise through the end of the line count-1 lines
// down, or linewise for a delete from the indent over several lines.
func (m *normalMode) applyOperatorToLineEnd(editor Editor, buffer Buffer, op operator, count int) *EditorError {
	end, ok := m.lineEndTarget(buffer, count)
	if !ok {
		return nil
	}
	start := buffer.GetCursor().Position
	if op != opDelete || deletesLines(buffer, start, end) { // the line delete sends its own DeleteSignal
		return applyOperatorToRange(editor, buffer, op, start, end)
	}
	deleted := rangeText(buffer, start, end)
	err := applyOperatorToRange(editor, buffer, op, start, end)
	editor.DispatchSignal(DeleteSignal{content: deleted})
	return err
}

// rangeText returns the text of the character-wise range [start, end), with
// a line break between lines.
func rangeText(buffer Buffer, start, end Position) string {
	var b strings.Builder
	for row := start.Row; row <= end.Row; row++ {
		line := buffer.GetLineRunes(row)
		from, to := 0, len(line)
		if row == start.Row {
			from = min(start.Col, to)
		}
		if row == end.Row {
			to = min(end.Col, to)
		}
		b.WriteString(string(line[from:max(from, to)]))
		if row < end.Row {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// applyOperatorToLineRange applies a delete/yank/change operator line-wise to the
// inclusive row range [startRow, endRow], used by the j/k motions.
func applyOperatorToLineRange(editor Editor, buffer Buffer, op operator, startRow, endRow int) *EditorError {
	switch op {
	case opDelete:
		deletedContent, err := deleteLineRange(editor, buffer, startRow, endRow)
		editor.DispatchSignal(DeleteSignal{content: deletedContent})
		return err

	case opYank:
		// yankLines starts at the cursor row, so move to the top of the range first
		// (for yk this also leaves the cursor there, matching Vim).
		cursor := buffer.GetCursor()
		if cursor.Position.Row != startRow {
			cursor.Position.Row = startRow
			cursor.clampCol(buffer)
			buffer.SetCursor(cursor)
		}
		return yankLines(editor, buffer, endRow-startRow+1)

	case opChange:
		return changeLineRange(editor, buffer, startRow, endRow)
	}

	return nil
}
