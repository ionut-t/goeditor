package core

import (
	"errors"
	"fmt"
)

type normalMode struct {
	pendingKey KeyEvent // Stores the first key of a multi-key command (e.g., 'd' in 'dd')
}

func NewNormalMode() EditorMode {
	return &normalMode{
		pendingKey: KeyEvent{Key: KeyUnknown},
	}
}

func (m *normalMode) Name() Mode { return NormalMode }

func (m *normalMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- NORMAL --")
	editor.UpdateCommand("")
	// Reset pending state on entering normal mode
	m.pendingKey = KeyEvent{Key: KeyUnknown}
	editor.ResetPendingCount()
	// Clear visual selection when entering normal mode
	state := editor.GetState()
	state.VisualStart = Position{-1, -1}
	editor.SetState(state)
}

func (m *normalMode) Exit(editor Editor, buffer Buffer) {
	// Clear pending state when exiting normal mode
	m.pendingKey = KeyEvent{Key: KeyUnknown}
}

func (m *normalMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *Error {
	var err *Error
	actionTaken := false // Track if the key (or sequence) resulted in an action
	state := editor.GetState()
	pendingCount := state.PendingCount
	availableWidth := state.AvailableWidth
	skipCursorUpdate := false

	// --- Handle Pending Operation (e.g., after 'd') ---
	if m.pendingKey.Key != KeyUnknown || m.pendingKey.Rune != 0 {
		firstKey := m.pendingKey
		m.pendingKey = KeyEvent{Key: KeyUnknown} // Consume the pending key

		count := 1
		if pendingCount != nil {
			count = *pendingCount
			editor.ResetPendingCount()
		}

		op := ""
		switch firstKey.Rune {
		case 'd':
			op = "delete"
		case 'y':
			op = "yank"
		case 'c': // Add change later
			op = "change"
		default:
			return &Error{
				id:  ErrNoPendingOperationId,
				err: ErrNoPendingOperation,
			}
		}

		// Handle motion keys after the operator
		switch key.Rune {
		case 'd': // dd = delete line
			if op == "delete" {
				err = deleteLines(editor, buffer, count)
				actionTaken = true
			}
		case 'w': // dw = delete word
			if op == "delete" {
				err = deleteWords(editor, buffer, count)
				actionTaken = true
			}
			// Add more motions: b, e, $, 0, ^, G, etc.
		case '$': // d$ delete to end of line
			if op == "delete" {
				err = deleteToEndOfLine(editor, buffer)
				actionTaken = true
			}

		default:
			// Invalid motion key after operator
			editor.DispatchError(ErrInvalidMotionId, fmt.Errorf("invalid motion after '%c'", firstKey.Rune))
			actionTaken = true         // Consumed the keys, even if invalid combo
			editor.ResetPendingCount() // Reset count if combo was invalid
		}

		if err != nil {
			return err // Return error from buffer operation
		}

		if actionTaken {
			editor.UpdateCommand("")
			return nil
		} // Sequence handled

		// If we fall through here, it means the second key wasn't a recognized motion
		// for the pending operator. We just discard the pending op.
		editor.DispatchMessage(EmptyMessage)

		return nil
	}

	// --- Handle Numeric Input for Counts ---
	if key.Rune >= '1' && key.Rune <= '9' {
		digit := int(key.Rune - '0')
		if pendingCount == nil {
			// Start of count (unless it follows '0' motion)
			if m.pendingKey.Rune == '0' { // Check local pendingKey for '0' motion
				cursor := buffer.GetCursor()
				cursor.MoveToLineStart()
				buffer.SetCursor(cursor)                 // Update buffer cursor!
				m.pendingKey = KeyEvent{Key: KeyUnknown} // Clear pending '0' motion key
				// Start the count state with the current digit
				state.PendingCount = &digit
				editor.SetState(state) // Save state
				actionTaken = true     // '0' motion was taken
			} else {
				// Normal start of count
				state.PendingCount = &digit // Update state directly
				editor.SetState(state)      // Save state
			}
		} else {
			// Append to existing count
			newCount := (*pendingCount * 10) + digit
			state.PendingCount = &newCount // Update state
			editor.SetState(state)         // Save state
		}
		// Update command display regardless of actionTaken status for digits
		editor.UpdateCommand(fmt.Sprintf("%d", *state.PendingCount))
		return nil // Just consuming digits, wait for command

	} else if key.Rune == '0' && pendingCount == nil {
		// '0' is move-to-start-of-line command if it's the first digit pressed
		m.pendingKey = KeyEvent{Key: KeyUnknown} // Clear any other pending op (like 'd')
		editor.ResetPendingCount()               // Ensure no count is active (redundant but safe)
		cursor := buffer.GetCursor()
		cursor.MoveToLineStart()
		buffer.SetCursor(cursor) // Update buffer cursor!
		actionTaken = true
		// Don't return yet, let subsequent logic handle potential errors/updates
	} else if key.Rune == '0' && pendingCount != nil {
		// '0' as part of a multi-digit count
		digit := 0
		newCount := (*pendingCount * 10) + digit
		state.PendingCount = &newCount                               // Update state
		editor.SetState(state)                                       // Save state
		editor.UpdateCommand(fmt.Sprintf("%d", *state.PendingCount)) // Show count
		return nil                                                   // Consuming digit, wait for command
	}

	// --- Get Count or Default to 1 ---
	// This count applies to the command/motion executed *now*
	count := 1
	countWasPending := false
	if pendingCount != nil {
		count = *pendingCount
		countWasPending = true
		// Reset count state *after* using it for this command/motion
		// We do this reset *after* the switch statement below
	}

	// --- Handle Single-Key Commands or Start of Sequences ---
	cursor := buffer.GetCursor() // Get cursor for operations
	col := cursor.Position.Col   // Use Column from cursor
	var moveErr error

	switch {
	// Movement keys
	case key.Rune == 'h' || key.Key == KeyLeft:
		moveErr = cursor.MoveLeftOrUp(buffer, count, col)
	case key.Rune == 'j' || key.Key == KeyDown:
		moveErr = cursor.MoveDown(buffer, count, availableWidth)
	case key.Rune == 'k' || key.Key == KeyUp:
		moveErr = cursor.MoveUp(buffer, count, availableWidth)
	case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, count, col)
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth)
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth)
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth)
	case key.Rune == '0':
		cursor.MoveToLineStart()
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth) // Move to last char
	case key.Rune == '^' || key.Key == KeyHome:
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
	case key.Rune == 'g':
		cursor.MoveToBufferStart() // Move to first line
	case key.Rune == 'G':
		cursor.MoveToBufferEnd(buffer, availableWidth) // Moves to start of last line
	case key.Key == KeyEnter: // Move down count lines to first non-blank
		if count == 0 {
			moveErr = cursor.MoveDown(buffer, count, availableWidth)
			if moveErr == nil {
				cursor.MoveToFirstNonBlank(buffer, availableWidth)
			}
		} else {
			cursor.Position.Row = count - 1
			buffer.SetCursor(cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}

	// Mode changes
	case key.Rune == 'i': // Insert before cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'I': // Insert at first non-blank
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
		buffer.SetCursor(cursor) // Update buffer's cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'a': // Insert after cursor
		cursor.MoveRight(buffer, 1, availableWidth) // Move one right (allows append at end of line)
		buffer.SetCursor(cursor)                    // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'A': // Insert at end of line
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Move *after* last char
		buffer.SetCursor(cursor)                          // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'o': // Open line below
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Go to end of current line
		buffer.SetCursor(cursor)
		buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune("\n")) // Insert newline
		cursor.MoveDown(buffer, 1, availableWidth)                                   // Move cursor down
		cursor.MoveToFirstNonBlank(buffer, availableWidth)                           // Go to start of new line
		buffer.SetCursor(cursor)
		editor.SaveHistory()
		editor.SetInsertMode()

	case key.Rune == 'O': // Open line above
		cursor.MoveToLineStart()                                   // Go to start of current linavailableWidthe
		buffer.InsertRunesAt(cursor.Position.Row, 0, []rune("\n")) // Insert newline (pushes current line down)
		// Cursor stays on original line index, which is now the new blank line
		cursor.MoveToFirstNonBlank(buffer, availableWidth) // Ensure col=0 on the new line
		buffer.SetCursor(cursor)
		editor.SaveHistory()
		editor.SetInsertMode()

	case key.Rune == 'v': // Enter visual mode
		editor.SetVisualMode()

	case key.Rune == 'V': // Enter visual line mode
		editor.SetVisualLineMode()

	case key.Key == KeyEscape:
		// If pending count or op, clear them
		pendingCount = nil
		m.pendingKey = KeyEvent{Key: KeyUnknown}
		editor.UpdateCommand("") // Clear count display
		editor.DispatchMessage(EmptyMessage)
		editor.SetNormalMode()

	case key.Rune == ':': // Enter command mode
		editor.SetCommandMode()

	case key.Rune == '/':
	// Enter search forward (TODO)

	case key.Rune == '?':
	// Enter search backward (TODO)

	// Editing commands (single key or start of sequence)
	case key.Rune == 'x': // Delete character under cursor
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen { // Only delete if cursor is on a char
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, count)
			if err == nil {
				editor.SaveHistory()
			}

			// Ensure cursor doesn't go past end of line after deletion
			cursor = buffer.GetCursor() // Refresh cursor state
			finalCol := cursor.Position.Col
			newLineLen := buffer.LineRuneCount(cursor.Position.Row)
			if finalCol >= newLineLen && newLineLen > 0 {
				cursor.Position.Col = newLineLen - 1
			} else if newLineLen == 0 {
				cursor.Position.Col = 0
			}
			buffer.SetCursor(cursor)
		}
	case key.Rune == 'X': // Delete character before cursor
		if cursor.Position.Col > 0 {
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col-1, count)
			if err == nil {
				cursor.MoveLeft(buffer, count, availableWidth) // Move cursor back
				buffer.SetCursor(cursor)
				editor.SaveHistory()
			}
		} else {
			err = &Error{
				id:  ErrStartOfLineId,
				err: ErrStartOfLine,
			}
		}

	case key.Rune == 'D': // Delete to end of line (equivalent to d$)
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen {
			delCount := lineLen - cursor.Position.Col
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, delCount)
			if err == nil {
				editor.SaveHistory()
			}
		}

	case key.Rune == 'C': // Change to end of line (equivalent to c$)
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen {
			delCount := lineLen - cursor.Position.Col
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, delCount)
			if err == nil {
				editor.SaveHistory()
			}
		}
		if err == nil {
			buffer.SetCursor(cursor) // Ensure cursor is at deletion point
			editor.SetInsertMode()   // Enter insert mode
		}
	case key.Rune == 'd': // Start 'delete' operation
		m.pendingKey = key
		// Don't clear count yet
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))

		return nil // Wait for the next key (motion)

	case key.Rune == 'c': // Start 'change' operation
		m.pendingKey = key
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'y': // Start 'yank' operation (TODO)
		m.pendingKey = key
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'p':
		var pasteErr error
		count, pasteErr = editor.Paste()
		cursor.MoveRight(buffer, count, availableWidth)

		if pasteErr != nil {
			err = &Error{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		}

	case key.Rune == 'u': // Undo
		if undoErr := editor.Undo(); undoErr != nil {
			err = &Error{
				id:  ErrUndoFailedId,
				err: undoErr,
			}
		}
		skipCursorUpdate = true

	case key.Rune == 'U': // Redo (Note: this is uppercase U)
		if redoErr := editor.Redo(); redoErr != nil {
			err = &Error{
				id:  ErrRedoFailedId,
				err: redoErr,
			}
		}
		skipCursorUpdate = true

	case key.Key == KeyBackspace: // Delete character before cursor
		moveErr = cursor.MoveLeft(buffer, count, availableWidth)

	default:
		// Unknown key - potentially beep or show message
		// Clear pending state if an unrecognized key is pressed
		m.pendingKey = KeyEvent{Key: KeyUnknown}
		// Keep count until non-digit command pressed
		// editor.SetMessage(fmt.Sprintf("Unknown key: %s", key.String()))

		// Reset count if it was pending and used by a command/motion executed above
		// (Excludes cases where we returned early, like starting d/y/c or parsing digits)
		if countWasPending {
			editor.ResetPendingCount()
		}
	}

	// Update cursor in buffer if no error or only boundary error
	// SKIP THIS IF WE JUST DID UNDO/REDO
	if !skipCursorUpdate && ((err == nil && moveErr == nil) ||
		errors.Is(moveErr, ErrEndOfBuffer) ||
		errors.Is(moveErr, ErrStartOfBuffer) ||
		errors.Is(moveErr, ErrEndOfLine) ||
		errors.Is(moveErr, ErrStartOfLine)) {
		buffer.SetCursor(cursor) // Update the buffer's cursor state
		// Don't return boundary errors as fatal errors to the editor loop
		if err != nil && !(errors.Is(moveErr, ErrEndOfBuffer) || errors.Is(moveErr, ErrStartOfBuffer) || errors.Is(moveErr, ErrEndOfLine) || errors.Is(moveErr, ErrStartOfLine)) {
			return err // Return actual errors (e.g., from delete/insert)
		}
		return nil
	}

	return err // Return other errors
}

func deleteLines(editor Editor, buffer Buffer, count int) (err *Error) {
	cursor := buffer.GetCursor()
	startLine := cursor.Position.Row
	endLine := startLine + count - 1
	availableWidth := editor.GetState().AvailableWidth

	if endLine >= buffer.LineCount() {
		endLine = buffer.LineCount() - 1
	}

	linesDeleted := 0
	// Delete lines from bottom up to keep indices valid
	for i := endLine; i >= startLine; i-- {

		if buffer.LineCount() > 1 { // Don't delete the very last line, clear it instead
			lineRunes := buffer.GetLineRunes(i)
			// Delete line content + newline (represented by moving to next line)
			err = buffer.DeleteRunesAt(i, 0, len(lineRunes)+1) // +1 deletes newline
			linesDeleted++
		} else {
			// Clear the last line
			err = buffer.DeleteRunesAt(i, 0, len(buffer.GetLineRunes(i)))
			linesDeleted++ // Count clearing as deletion
		}
		if err != nil {
			break
		}
	}
	// Adjust cursor after deletion
	cursor = buffer.GetCursor()          // Get current cursor state
	if startLine >= buffer.LineCount() { // If deletion removed lines up to the end
		cursor.Position.Row = max(buffer.LineCount()-1, 0) // Handle empty buffer
	} else {
		cursor.Position.Row = startLine // Move to first deleted line index
	}
	// Move cursor to first non-blank of the line it landed on
	buffer.SetCursor(cursor) // Set row first
	c := buffer.GetCursor()  // Get it back to use methods
	c.MoveToFirstNonBlank(buffer, availableWidth)
	buffer.SetCursor(c) // Final position

	if err == nil {
		editor.SaveHistory()
	}

	return err
}

func deleteWords(editor Editor, buffer Buffer, count int) (err *Error) {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	// Simulate word motion to find end position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	errMove := tempCursor.MoveWordForward(buffer, count, availableWidth)
	endPos := tempCursor.Position

	// Calculate deletion range (careful with multi-line)
	// Simple version: delete from startPos.Col to endPos.Col if on same line
	if startPos.Row == endPos.Row && errMove == nil {
		deleteCount := endPos.Col - startPos.Col
		if deleteCount > 0 {
			err = buffer.DeleteRunesAt(startPos.Row, startPos.Col, deleteCount)
			if err == nil {
				editor.SaveHistory()
			}
		}
	} else {
		// TODO: Implement multi-line dw/cw/yw
	}

	return err
}

func deleteToEndOfLine(editor Editor, buffer Buffer) (err *Error) {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)
	if cursor.Position.Col < lineLen { // Only delete if not already at/past end
		deleteCount := lineLen - cursor.Position.Col
		err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, deleteCount)
		if err == nil {
			editor.SaveHistory()
		}
	}

	return err
}
