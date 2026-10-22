package core

// applyOperatorToRange applies a delete/yank/change operator to the character-wise
// range [start, end) — end exclusive, matching Vim's exclusive motions. The two
// positions may be given in either order. For inclusive motions (e.g. ge), the
// caller moves the end position one character right before calling. Exclusive
// motions that can end on a later line go through applyOperatorToMotion.
// An empty range moves the cursor to it and goes to applyOperatorToEmptyRange.
//
// As Vim's op_delete, a delete over several lines that starts in the indent
// and leaves only blanks after its end deletes the lines instead.
func applyOperatorToRange(editor Editor, buffer Buffer, op string, start, end Position) *EditorError {
	// Normalize so start is before end
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	if start == end {
		// After a motion the cursor is already there; an empty text object
		// (di"" or ciw on an empty line) puts it on the empty range.
		cursor := buffer.GetCursor()
		cursor.Position = start
		buffer.SetCursor(cursor)
		return applyOperatorToEmptyRange(editor, op)
	}
	if op == "delete" && end.Row > start.Row && inIndent(buffer, start) && onlyBlanksFrom(buffer, end) {
		return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
	}

	cursor := buffer.GetCursor()

	switch op {
	case "delete", "change":
		if err := deleteRange(buffer, start, end); err != nil {
			return err
		}
		cursor.Position = start
		if op == "delete" {
			// A delete up to the end of the line leaves start past it.
			cursor.Position.Col = min(start.Col, lastCol(buffer, start.Row, false))
		}
		buffer.SetCursor(cursor)
		if op == "change" {
			editor.SetInsertMode()
		}

	case "yank":
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
func applyOperatorToEmptyRange(editor Editor, op string) *EditorError {
	o, ok := editor.(emptyRangeOperator)
	switch op {
	case "delete", "change":
		if ok {
			o.countChange()
		}
		if op == "change" {
			editor.SetInsertMode()
		}

	case "yank":
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
func applyOperatorToMotion(editor Editor, buffer Buffer, op string, start, end Position) *EditorError {
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	if end.Row > start.Row && end.Col == 0 {
		end = Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}
		if inIndent(buffer, start) {
			return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
		}
	}
	return applyOperatorToRange(editor, buffer, op, start, end)
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
	return nil
}

// applyOperatorToWords applies a delete/yank/change operator to what w (W
// with isBigWordChar) moves over from the cursor. After an operator the last
// word stops at the end of its line (:h dw).
func applyOperatorToWords(editor Editor, buffer Buffer, op string, count int, isWordChar func(rune) bool) *EditorError {
	cursor := buffer.GetCursor()
	start := cursor.Position
	_ = cursor.moveWordForward(buffer, count, isWordChar, true)
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

// applyOperatorToLineRange applies a delete/yank/change operator line-wise to the
// inclusive row range [startRow, endRow], used by the j/k motions.
func applyOperatorToLineRange(editor Editor, buffer Buffer, op string, startRow, endRow int) *EditorError {
	switch op {
	case "delete":
		deletedContent, err := deleteLineRange(editor, buffer, startRow, endRow)
		editor.DispatchSignal(DeleteSignal{content: deletedContent})
		return err

	case "yank":
		// yankLines starts at the cursor row, so move to the top of the range first
		// (for yk this also leaves the cursor there, matching Vim).
		cursor := buffer.GetCursor()
		if cursor.Position.Row != startRow {
			cursor.Position.Row = startRow
			cursor.clampCol(buffer)
			buffer.SetCursor(cursor)
		}
		return yankLines(editor, buffer, endRow-startRow+1)

	case "change":
		return changeLineRange(editor, buffer, startRow, endRow)
	}

	return nil
}
