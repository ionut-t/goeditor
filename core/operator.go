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
	state := editor.GetState()

	switch op {
	case "delete", "change":
		if err := deleteRange(buffer, start, end); err != nil {
			return err
		}
		editor.SaveHistory()
		cursor.Position = start
		buffer.SetCursor(cursor)
		if op == "change" {
			editor.SetInsertMode()
		}

	case "yank":
		// editor.Copy is inclusive of both ends, so step the exclusive end back one.
		endCursor := Cursor{Position: end}
		_ = endCursor.MoveLeftOrUp(buffer, 1, state.AvailableWidth)

		state.VisualStart = endCursor.Position
		state.YankSelection = SelectionCharacter
		editor.SetState(state)

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
	}

	return nil
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
		_, err := deleteLineRange(editor, buffer, startRow, endRow)
		if err == nil {
			editor.SetInsertMode()
		}
		return err
	}

	return nil
}
