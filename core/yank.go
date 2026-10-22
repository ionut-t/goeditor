package core

func yankLines(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	originalPos := cursor.Position
	startLine := cursor.Position.Row
	endLine := startLine + count - 1

	if endLine >= buffer.LineCount() {
		endLine = buffer.LineCount() - 1
	}

	// Set up line-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state.VisualStart = Position{Row: endLine, Col: max(buffer.LineRuneCount(endLine)-1, 0)}
	state.YankSelection = SelectionLine // Mark as line-wise selection
	editor.SetState(state)

	// Restore cursor to original position
	// This way cursor stays where the user had it when they pressed yy
	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight

	return nil
}

func yankToEndOfLine(editor Editor, buffer Buffer) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	originalPos := cursor.Position
	lineLen := buffer.LineRuneCount(cursor.Position.Row)

	if cursor.Position.Col >= lineLen {
		// Already at or past end of line, nothing to yank
		return nil
	}

	// Set up character-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state.VisualStart = Position{Row: cursor.Position.Row, Col: lineLen - 1}
	state.YankSelection = SelectionCharacter // Mark as character-wise selection
	editor.SetState(state)

	// Restore cursor to original position
	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight

	return nil
}
