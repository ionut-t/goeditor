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

func yankWords(editor Editor, buffer Buffer, count int, forward bool) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	originalPos := cursor.Position
	availableWidth := state.AvailableWidth

	// Calculate end position by moving cursor
	tempCursor := cursor
	var moveErr error
	if forward {
		moveErr = tempCursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
	} else {
		moveErr = tempCursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
	}

	endPos := tempCursor.Position

	// If the cursor actually moved, we need to make the yank exclusive.
	// In Vim, 'yw' and 'yb' are exclusive motions.
	// Since editor.Copy is inclusive of both ends, we need to adjust
	// the larger of the two positions back by one character.
	var selStart, selEnd Position
	if originalPos.Row < endPos.Row || (originalPos.Row == endPos.Row && originalPos.Col < endPos.Col) {
		selStart = originalPos
		selEnd = endPos
	} else {
		selStart = endPos
		selEnd = originalPos
	}

	if selStart != selEnd {
		// Make the yank range exclusive by moving the end of the range back by one character.
		// MoveLeftOrUp correctly handles the case where the motion spanned multiple lines
		// and we want to exclude the character at the start of the next line (and potentially
		// the newline of the current line if it's an exclusive motion like 'yw' at EOL).
		endCursor := Cursor{Position: selEnd}
		_ = endCursor.MoveLeftOrUp(buffer, 1, availableWidth)
		selEnd = endCursor.Position
	}

	// Set up character-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state.VisualStart = selEnd
	state.YankSelection = SelectionCharacter // Mark as character-wise selection
	editor.SetState(state)

	// Set cursor to selStart for Copy (it uses current cursor as one end)
	cursor.Position = selStart
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		// Restore cursor to original position on error
		cursor.Position = originalPos
		buffer.SetCursor(cursor)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight
	// Handle movement errors non-fatally
	if moveErr != nil && moveErr != ErrEndOfBuffer && moveErr != ErrStartOfBuffer {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		// Restore cursor to original position on error
		cursor.Position = originalPos
		buffer.SetCursor(cursor)
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: moveErr,
		}
	}

	return nil
}

func yankWordToEnd(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	originalPos := cursor.Position
	availableWidth := state.AvailableWidth

	tempCursor := cursor
	moveErr := tempCursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
	endPos := tempCursor.Position

	// ye is inclusive — no MoveLeftOrUp adjustment unlike yw/yb.
	state.VisualStart = endPos
	state.YankSelection = SelectionCharacter
	editor.SetState(state)

	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	if err := editor.Copy(yankType); err != nil {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{id: ErrFailedToYankId, err: err}
	}

	if moveErr != nil && moveErr != ErrEndOfBuffer && moveErr != ErrStartOfBuffer {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		cursor.Position = originalPos
		buffer.SetCursor(cursor)
		return &EditorError{id: ErrInvalidMotionId, err: moveErr}
	}

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
