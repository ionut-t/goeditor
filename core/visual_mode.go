package core

import (
	"errors"
)

type visualMode struct {
	startPos     Position // Where visual selection started
	currentCount *int     // Temporary count parsed within visual mode
}

func NewVisualMode() EditorMode {
	return &visualMode{
		startPos:     Position{-1, -1},
		currentCount: nil,
	}
}
func (m *visualMode) Name() Mode { return VisualMode }

func (m *visualMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL --")
	editor.UpdateCommand("")
	// Record selection start position
	m.startPos = buffer.GetCursor().Position
	m.currentCount = nil
	// Update editor state to reflect visual mode is active
	state := editor.GetState()
	state.VisualStart = m.startPos
	// VisualEnd is implicitly the current cursor position
	editor.SetState(state)
}

func (m *visualMode) Exit(editor Editor, buffer Buffer) {
	// Clear visual selection indication in editor state
	state := editor.GetState()
	state.VisualStart = Position{Row: -1, Col: -1} // Mark inactive
	editor.SetState(state)
	editor.UpdateStatus("")  // Clear status or let normal mode set it
	editor.UpdateCommand("") // Clear command display
}

// NormalizeSelection ensures start is before end, line by line, then column by column.
func NormalizeSelection(p1, p2 Position) (start, end Position) {
	if p1.Row < p2.Row || (p1.Row == p2.Row && p1.Col <= p2.Col) {
		return p1, p2
	}
	return p2, p1
}

func (m *visualMode) GetCurrentCount() *int {
	return m.currentCount
}

func (m *visualMode) SetCurrentCount(count *int) {
	m.currentCount = count
}

func (m *visualMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *Error {
	if key.Key == KeyEscape {
		editor.SetNormalMode()
		return nil
	}

	cursor := buffer.GetCursor() // Get current cursor state
	var err *Error
	actionTaken := false // Flag if an action (delete, yank) was performed

	count, processedDigit := getMoveCount(m, editor, key)

	// If a digit was just processed, wait for the next key
	if processedDigit {
		return nil
	}

	// --- Visual Mode Actions ---
	switch key.Rune {
	case 'd', 'x': // Delete selected text
		var finalPos Position
		finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			cursor.Position = finalPos // Update cursor position based on function result
			buffer.SetCursor(cursor)   // Set cursor position in buffer
			editor.SaveHistory()
			editor.SetNormalMode() // Exit visual mode after action
		}
		actionTaken = true
		editor.ResetPendingCount()

	case 'y': // Yank (Copy) selected text
		if copyErr := editor.Copy(); copyErr != nil {
			err = &Error{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		actionTaken = true
		editor.ResetPendingCount()

	case 'c': // Change selected text (delete + enter insert)
		var finalPos Position
		finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			cursor.Position = finalPos // Update cursor position based on function result
			buffer.SetCursor(cursor)   // Set cursor position in buffer
			editor.SaveHistory()
			editor.SetNormalMode()
		}

		actionTaken = true
		editor.ResetPendingCount()

	case 'v':
		editor.SetNormalMode()
		actionTaken = true
	case 'V':
		editor.SetVisualLineMode()
		actionTaken = true
	}

	if actionTaken {
		return err
	} // Return if delete/yank/change was performed

	// --- Visual Mode Movements (Update selection end) ---
	// Allow regular normal mode movements, they just extend the selection
	state := editor.GetState()
	availableWidth := state.AvailableWidth

	countWasPending := false

	if state.PendingCount != nil {
		count = *state.PendingCount
		countWasPending = true
		editor.SetState(state)
		editor.UpdateCommand("") // Clear count display
	}

	col := cursor.Position.Col

	var moveErr error

	switch {
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
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth)
	case key.Rune == '0' || key.Key == KeyHome:
		cursor.MoveToLineStart()
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth)
	case key.Rune == '^':
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
	case key.Rune == 'G':
		moveErr = cursor.MoveToBufferEnd(buffer, availableWidth)

	case key.Key == KeyEnter:
		if count > 0 {
			cursor.Position.Row = count - 1
			buffer.SetCursor(cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}
	default:
		// Ignore other keys or beep?
		if countWasPending {
			editor.ResetPendingCount()
		}
	}

	// Update cursor position in buffer if movement happened
	if (err == nil && moveErr == nil) ||
		errors.Is(moveErr, ErrEndOfBuffer) ||
		errors.Is(moveErr, ErrStartOfBuffer) ||
		errors.Is(moveErr, ErrEndOfLine) ||
		errors.Is(moveErr, ErrStartOfLine) {
		buffer.SetCursor(cursor)
		// VisualEnd is implicitly the current cursor, no need to update state explicitly here
		// Boundary errors are ok, just stop moving
		return nil
	}

	// If there was a real error during movement, reset any pending count
	if countWasPending {
		editor.ResetPendingCount()
	}

	return err // Return actual errors from movement
}

// deleteVisualSelection handles the logic for deleting the text within a visual selection.
// It takes the buffer, the start position, and the end position of the selection.
// It returns the calculated cursor position after deletion (usually the start of the
// deleted area) and any error encountered during the deletion process.
func deleteVisualSelection(buffer Buffer, startPos, endPos Position) (Position, *Error) {
	var err *Error

	startSel, endSel := NormalizeSelection(startPos, endPos)
	finalCursorPos := startSel // Default final position is the start of selection

	// Simple case: Single line selection
	if startSel.Row == endSel.Row {
		count := endSel.Col - startSel.Col + 1 // Inclusive delete
		if count > 0 {
			err = buffer.DeleteRunesAt(startSel.Row, startSel.Col, count)
			// errorMessage = delErr.Error()
			// err = ErrDeleteRunes
			// finalCursorPos is already startSel, which is correct here
		}
	} else {
		// Multi-line selection deletion (more complex)

		// 1. Delete from startCol to end of startLine
		startLine := buffer.GetLineRunes(startSel.Row)
		startLineLen := len(startLine) // Use rune count if buffer stores runes directly
		// Note: If buffer.GetLineRunes includes newline, adjust calculation
		delCount1 := startLineLen - startSel.Col
		if delCount1 > 0 {
			err := buffer.DeleteRunesAt(startSel.Row, startSel.Col, delCount1)

			if err != nil {
				return finalCursorPos, err
			}
		}

		// 2. Delete intermediate full lines (from bottom up to handle shifting indices)
		// The number of lines *between* start and end (exclusive start, exclusive end)
		linesToDelete := endSel.Row - startSel.Row - 1
		for range linesToDelete {
			// The row index to delete is always startSel.Row + 1 because
			// deleting a line shifts subsequent lines up.
			targetRow := startSel.Row + 1
			lineLen := buffer.LineRuneCount(targetRow)
			// Delete the line content plus the newline character
			err = buffer.DeleteRunesAt(targetRow, 0, lineLen+1)
			if err != nil {
				return finalCursorPos, err // Return immediately on error
			}
		}

		// 3. Delete from beginning of the original endLine up to endCol (inclusive)
		//    AND merge the remaining part of startLine with remaining part of endLine
		currentEndRow := startSel.Row + 1 // Line index where original endSel content now resides
		// after intermediate line deletions.

		if currentEndRow < buffer.LineCount() { // Check if the line wasn't deleted entirely
			delCount2 := endSel.Col + 1 // Delete from column 0 up to endCol (inclusive)
			if delCount2 > 0 {
				err = buffer.DeleteRunesAt(currentEndRow, 0, delCount2)
				if err != nil {
					return finalCursorPos, err // Return immediately on error
				}
			}

			// 4. Merge lines: Delete the newline character between the modified startRow
			//    and the modified endRow. This newline is now at the end of startRow.
			startLineLenAfterDel := buffer.LineRuneCount(startSel.Row)
			if startLineLenAfterDel >= 0 && startSel.Row+1 < buffer.LineCount() { // Ensure there's a newline to delete
				err = buffer.DeleteRunesAt(startSel.Row, startLineLenAfterDel, 1) // Delete newline
				if err != nil {
					// This error might occur if the start line became empty etc.
					// Depending on desired behavior, might log or handle differently.
					return finalCursorPos, err
				}
			}
		} else {
			// The original end line was one of the intermediate lines deleted entirely.
			// Or it was the line immediately after startLine and got fully consumed
			// by step 1 & potentially merging.
			// The merge step (4) might still be needed if startLine had content remaining
			// and the line below it (originally endLine) was deleted. Check if startLine
			// ends with a newline that shouldn't be there.

			// Safest approach might be to just rely on the cursor position update below.
			// However, ensure step 1 didn't leave a dangling newline if it deleted the whole line content.
			// If startLineLen == delCount1, step 1 deleted the whole content. Check if a newline remains.
			if startLineLen == delCount1 && buffer.LineRuneCount(startSel.Row) == 0 && startSel.Row+1 < buffer.LineCount() {
				// Delete the newline that originally belonged to startSel.Row
				err = buffer.DeleteRunesAt(startSel.Row, 0, 1)
				if err != nil {
					return finalCursorPos, err
				}
			}

		}
	}

	return finalCursorPos, err
}
