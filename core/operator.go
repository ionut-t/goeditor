package core

// applyOperatorToRange applies a delete/yank/change operator to the character-wise
// range [start, end) — end exclusive, matching Vim's exclusive motions. The two
// positions may be given in either order. For inclusive motions (e.g. ge), the
// caller moves the end position one character right before calling.
func applyOperatorToRange(editor Editor, buffer Buffer, op string, start, end Position) *EditorError {
	// Normalize so start is before end
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	if start == end {
		return nil
	}

	cursor := buffer.GetCursor()

	switch op {
	case "delete", "change":
		if err := deleteRange(buffer, start, end); err != nil {
			return err
		}
		cursor.Position = start
		buffer.SetCursor(cursor)
		if op == "change" {
			editor.SetInsertMode()
		}

	case "yank":
		// editor.Copy is inclusive of both ends, so step the exclusive end back one.
		endCursor := Cursor{Position: end}
		_ = endCursor.MoveLeftOrUp(buffer, 1)
		return yankRange(editor, buffer, start, endCursor.Position)
	}

	return nil
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
// with isBigWordChar) moves over from the cursor, with Vim's rules for w after
// an operator: the last word stops at the end of its line (:h dw). An end in
// column 0 of a later line (from an empty line) goes back to the end of the
// line above, and the operation becomes linewise if it started in the indent
// (:h exclusive-linewise). A delete over several lines that starts in the
// indent and leaves only blanks on its last line is linewise too.
func applyOperatorToWords(editor Editor, buffer Buffer, op string, count int, isWordChar func(rune) bool) *EditorError {
	start := buffer.GetCursor().Position
	tempCursor := buffer.GetCursor()
	_ = tempCursor.moveWordForward(buffer, count, isWordChar, true)
	end := tempCursor.Position

	if end.Row > start.Row && end.Col == 0 {
		end = Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}
		if inIndent(buffer, start) {
			return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
		}
	} else if op == "delete" && end.Row > start.Row && inIndent(buffer, start) && onlyBlanksFrom(buffer, end) {
		return applyOperatorToLineRange(editor, buffer, op, start.Row, end.Row)
	}

	if op == "yank" && start != end {
		// The end is exclusive. At column 0 (an empty line) the last character
		// yanked is the line break above.
		last := Position{Row: end.Row, Col: end.Col - 1}
		if end.Col == 0 {
			last = Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}
		}
		return yankRange(editor, buffer, start, last)
	}
	return applyOperatorToRange(editor, buffer, op, start, end)
}

// inIndent reports whether only blanks come before pos on its line.
func inIndent(buffer Buffer, pos Position) bool {
	line := buffer.GetLineRunes(pos.Row)
	for _, r := range line[:min(pos.Col, len(line))] {
		if !isWhiteSpace(r) {
			return false
		}
	}
	return true
}

// onlyBlanksFrom reports whether only blanks come from pos to the end of its line.
func onlyBlanksFrom(buffer Buffer, pos Position) bool {
	line := buffer.GetLineRunes(pos.Row)
	for _, r := range line[min(pos.Col, len(line)):] {
		if !isWhiteSpace(r) {
			return false
		}
	}
	return true
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
