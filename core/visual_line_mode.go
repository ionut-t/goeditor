// editor/visual_line_mode.go
package core

import (
	"errors"
)

type visualLineMode struct {
	startPos     Position // Only the Row is relevant for selection extent
	currentCount *int
}

func NewVisualLineMode() EditorMode {
	return &visualLineMode{
		startPos:     Position{-1, -1},
		currentCount: nil,
	}
}

func (m *visualLineMode) Name() Mode { return VisualLineMode }

func (m *visualLineMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL LINE --")
	editor.UpdateCommand("")
	// Record selection start position (row matters most)
	m.startPos = buffer.GetCursor().Position
	m.currentCount = nil
	// Update editor state to reflect visual mode is active (use same flag)
	state := editor.GetState()
	state.VisualStart = m.startPos // Use VisualStart to indicate visual active
	editor.SetState(state)
}

func (m *visualLineMode) Exit(editor Editor, buffer Buffer) {
	// Clear visual selection indication in editor state
	state := editor.GetState()
	state.VisualStart = Position{Row: -1, Col: -1} // Mark inactive
	editor.SetState(state)
	editor.UpdateStatus("") // Clear status or let normal mode set it
	m.currentCount = nil
}

func (m *visualLineMode) GetCurrentCount() *int {
	return m.currentCount
}

func (m *visualLineMode) SetCurrentCount(count *int) {
	m.currentCount = count
}

func (m *visualLineMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *Error {
	if key.Key == KeyEscape {
		editor.SetNormalMode()
		return nil
	}

	// if ShouldHandlePendingKey(m) {
	// 	return HandlePendingKey(m, editor, buffer, key)
	// }

	cursor := buffer.GetCursor() // Get current cursor state
	var err *Error
	actionTaken := false // Flag if an action was performed
	availableWidth := editor.GetState().AvailableWidth

	count, processedDigit := getMoveCount(m, editor, key)

	// If a digit was just processed, wait for the next key
	if processedDigit {
		return nil
	}

	// --- Visual Line Mode Actions ---
	switch key.Rune {
	case 'd', 'x': // Delete selected lines
		startRow, endRow := m.startPos.Row, cursor.Position.Row
		if startRow > endRow {
			startRow, endRow = endRow, startRow // Ensure start <= end
		}

		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)

		err = deleteLineRange(editor, buffer, startRow, endRow)

		if err == nil {
			// Cursor position adjusted within deleteLineRange
			editor.SaveHistory()
			editor.SetNormalMode() // Exit visual mode after action
		}
		actionTaken = true

	case 'y': // Yank selected lines
		if copyErr := editor.Copy(); copyErr != nil {
			err = &Error{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		actionTaken = true

	case 'c': // Change selected lines (delete + enter insert)
		// TODO: Implement Change for visual line mode
		editor.SetNormalMode() // Temp: Exit visual mode
		actionTaken = true

	// Mode Switches
	case 'v': // Switch to character-wise visual mode
		// Keep selection start, just change mode
		editor.SetVisualMode() // Switch to character-wise visual mode
		actionTaken = true     // Mode switch is an action
	case 'V':
		editor.SetNormalMode() // Switch to normal mode
		actionTaken = true
	}

	if actionTaken {
		return err
	} // Return if delete/yank/change/mode switch was performed

	// --- Visual Line Mode Movements (Update selection end row) ---
	// Only row movements are relevant for selection extent

	var moveErr error
	movementAttempted := false
	moveCount := count // Use 'count' for actual move amount calculation
	switch key.Key {   // Use Key for arrows/pgup/dn
	case KeyDown:
		cursor.MoveDown(buffer, moveCount, availableWidth) // Use count
		movementAttempted = true
	case KeyUp:
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth) // Use count
		movementAttempted = true
	case KeyPageDown:
		if count == 1 {
			moveCount = editor.GetState().ViewportHeight
		} // Use default only if no count typed
		moveErr = cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyPageUp:
		if count == 1 {
			moveCount = editor.GetState().ViewportHeight
		} // Use default only if no count typed
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
		movementAttempted = true

	default:
		col := cursor.Position.Col // Get Column from cursor state
		switch {                   // Allow rune based keys
		case key.Rune == 'j': // Allow j/k runes too
			moveErr = cursor.MoveDown(buffer, moveCount, availableWidth) // Use count
			movementAttempted = true
		case key.Rune == 'k':
			moveErr = cursor.MoveUp(buffer, moveCount, availableWidth) // Use count
			movementAttempted = true
		// Horizontal movements affect cursor position but not line selection extent
		case key.Rune == 'h' || key.Key == KeyLeft:
			moveErr = cursor.MoveLeftOrUp(buffer, 1, col) // Horizontal moves ignore count
			movementAttempted = true
		case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
			moveErr = cursor.MoveRightOrDown(buffer, 1, col) // Horizontal moves ignore count
			movementAttempted = true
		case key.Rune == '0' || key.Key == KeyHome:
			cursor.MoveToLineStart() // Ignores count
			movementAttempted = true
		case key.Rune == '$' || key.Key == KeyEnd:
			cursor.MoveToLineEnd(buffer, availableWidth) // Ignores count
			movementAttempted = true
		case key.Rune == '^':
			cursor.MoveToFirstNonBlank(buffer, availableWidth) // Ignores count
			movementAttempted = true

		case key.Key == KeyEnter:
			if count > 0 {
				cursor.Position.Row = count - 1
				buffer.SetCursor(cursor)
				editor.UpdateCommand("")
				editor.ResetPendingCount()
			}
		default:
			// Key was not a digit, action, or recognized movement
			break
		}
	}

	// Update cursor position in buffer if movement happened
	if movementAttempted &&
		(err == nil && moveErr == nil) ||
		errors.Is(moveErr, ErrEndOfBuffer) ||
		errors.Is(moveErr, ErrStartOfBuffer) ||
		errors.Is(moveErr, ErrEndOfLine) ||
		errors.Is(moveErr, ErrStartOfLine) {
		buffer.SetCursor(cursor)
		// VisualEnd is implicitly the current cursor, state.VisualStart remains fixed
		// Boundary errors are ok, just stop moving
		return nil
	}

	return err // Return actual errors from movement
}

// Helper function to delete a range of lines (inclusive)
// Similar to deleteLines from normalMode but takes range directly
func deleteLineRange(editor Editor, buffer Buffer, startRow, endRow int) *Error {
	if startRow < 0 || endRow >= buffer.LineCount() || startRow > endRow {
		return &Error{
			id:  ErrInvalidPositionId,
			err: errors.New("invalid line range for deletion"),
		}
	}

	availableWidth := editor.GetState().AvailableWidth

	linesDeleted := 0
	var firstErr *Error

	// Delete lines from bottom up to keep indices valid
	for i := endRow; i >= startRow; i-- {
		if buffer.LineCount() > 1 { // Don't delete the very last line, clear it instead
			lineRunes := buffer.GetLineRunes(i)
			err := buffer.DeleteRunesAt(i, 0, len(lineRunes)+1) // +1 deletes newline
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if err == nil {
				linesDeleted++
			}
		} else {
			// Clear the last line
			err := buffer.DeleteRunesAt(i, 0, len(buffer.GetLineRunes(i)))
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if err == nil {
				linesDeleted++ // Count clearing as deletion
			}
		}
	}

	// Adjust cursor after deletion
	cursor := buffer.GetCursor() // Get current cursor state (might have moved during deletion?)
	newRow := startRow
	if newRow >= buffer.LineCount() { // If deletion removed lines up to the end
		newRow = max(buffer.LineCount()-1, 0) // Move to last available line or 0
	}
	cursor.Position.Row = newRow
	buffer.SetCursor(cursor)    // Set row first
	cursor = buffer.GetCursor() // Get it back to use methods
	cursor.MoveToFirstNonBlank(buffer, availableWidth)
	buffer.SetCursor(cursor) // Final position

	if firstErr == nil {
		editor.SaveHistory() // Save history only if all deletions likely succeeded

	}

	return firstErr
}
