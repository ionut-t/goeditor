package core

import (
	"errors"
	"strings"
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

func (m *visualMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	if key.Key == KeyEscape {
		editor.SetNormalMode()
		return nil
	}

	cursor := buffer.GetCursor() // Get current cursor state
	var err *EditorError
	actionTaken := false // Flag if an action (delete, yank) was performed

	count, processedDigit := getMoveCount(m, editor, key)

	// If a digit was just processed, wait for the next key
	if processedDigit {
		return nil
	}

	// --- Visual Mode Actions ---
	switch key.Rune {
	case 'd', 'x': // Delete/Cut selected text
		if key.Rune == 'x' {
			_ = editor.Copy(cutType)
		}

		var finalPos Position
		var contentDeleted string
		contentDeleted, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)

		if err == nil {
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
			editor.SaveHistory()
			editor.SetNormalMode()
		}

		actionTaken = true
		editor.ResetPendingCount()
		editor.DispatchSignal(DeleteSignal{content: contentDeleted})

	case 'y': // Yank (Copy) selected text
		if copyErr := editor.Copy(yankType); copyErr != nil {
			err = &EditorError{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		actionTaken = true
		editor.ResetPendingCount()

	case 'p':
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)

		if err == nil {
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
			editor.SaveHistory()
			editor.SetNormalMode()
		}

		content, pasteErr := editor.Paste()
		count = len(content)

		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}

		actionTaken = true
		editor.ResetPendingCount()

	case 'c': // Change selected text (delete + enter insert)
		_ = editor.Copy(cutType)
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			cursor.Position = finalPos // Update cursor position based on function result
			buffer.SetCursor(cursor)   // Set cursor position in buffer
			editor.SaveHistory()
			editor.SetInsertMode()
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
		editor.UpdateCommand("")
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
	case key.Rune == '{':
		moveErr = cursor.MoveBlockBackward(buffer, count)
	case key.Rune == '}':
		moveErr = cursor.MoveBlockForward(buffer, count)
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth)
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth)
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth)
	case key.Rune == '0' || key.Key == KeyHome:
		cursor.MoveToLineStart()
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth)
	case key.Rune == '^':
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
	case key.Rune == 'g':
		cursor.MoveToBufferStart()
	case key.Rune == 'G':
		cursor.MoveToBufferEnd(buffer, availableWidth)

	case key.Key == KeyEnter:
		if count > 0 {
			cursor.Position.Row = count - 1
			buffer.SetCursor(cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}
	default:
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

	return err
}

// deleteVisualSelection handles the logic for deleting the text within a visual selection.
// It takes the buffer, the start position, and the end position of the selection.
// It returns the deleted content, the calculated cursor position after deletion (usually the start of the
// deleted area) and any error encountered during the deletion process.
func deleteVisualSelection(buffer Buffer, startPos, endPos Position) (string, Position, *EditorError) {
	var err *EditorError

	var deletedContent string
	startSel, endSel := NormalizeSelection(startPos, endPos)
	finalCursorPos := startSel // Default final position is the start of selection

	// Simple case: Single line selection
	if startSel.Row == endSel.Row {
		lineRunes := buffer.GetLineRunes(startSel.Row)
		count := endSel.Col - startSel.Col + 1 // Inclusive delete
		if count > 0 {
			endCol := min(startSel.Col+count, len(lineRunes))
			deletedContent = string(lineRunes[startSel.Col:endCol])
			err = buffer.DeleteRunesAt(startSel.Row, startSel.Col, count)
		}
	} else {
		// Multi-line selection.
		// First, gather all the content that will be deleted.
		var contentBuilder strings.Builder
		// Part of the first line
		startLineRunes := buffer.GetLineRunes(startSel.Row)
		if startSel.Col < len(startLineRunes) {
			contentBuilder.WriteString(string(startLineRunes[startSel.Col:]))
		}
		contentBuilder.WriteString("\n")

		// Intermediate lines
		for i := startSel.Row + 1; i < endSel.Row; i++ {
			lineRunes := buffer.GetLineRunes(i)
			contentBuilder.WriteString(string(lineRunes))
			contentBuilder.WriteString("\n")
		}

		// Part of the last line
		endLineRunes := buffer.GetLineRunes(endSel.Row)
		if endSel.Col+1 <= len(endLineRunes) {
			contentBuilder.WriteString(string(endLineRunes[:endSel.Col+1]))
		} else {
			contentBuilder.WriteString(string(endLineRunes))
		}
		deletedContent = contentBuilder.String()

		// 1. Delete from startCol to end of startLine
		startLine := buffer.GetLineRunes(startSel.Row)
		startLineLen := len(startLine)
		delCount1 := startLineLen - startSel.Col
		if delCount1 > 0 {
			err := buffer.DeleteRunesAt(startSel.Row, startSel.Col, delCount1)
			if err != nil {
				return "", finalCursorPos, err
			}
		}

		// 2. Delete intermediate full lines
		linesToDelete := endSel.Row - startSel.Row - 1
		for range linesToDelete {
			targetRow := startSel.Row + 1
			lineLen := buffer.LineRuneCount(targetRow)
			err = buffer.DeleteRunesAt(targetRow, 0, lineLen+1)
			if err != nil {
				return "", finalCursorPos, err
			}
		}

		// 3. Delete from beginning of the original endLine up to endCol
		currentEndRow := startSel.Row + 1
		if currentEndRow < buffer.LineCount() {
			delCount2 := endSel.Col + 1
			if delCount2 > 0 {
				err = buffer.DeleteRunesAt(currentEndRow, 0, delCount2)
				if err != nil {
					return "", finalCursorPos, err
				}
			}

			// 4. Merge lines
			startLineLenAfterDel := buffer.LineRuneCount(startSel.Row)
			if startLineLenAfterDel >= 0 && startSel.Row+1 < buffer.LineCount() {
				err = buffer.DeleteRunesAt(startSel.Row, startLineLenAfterDel, 1)
				if err != nil {
					return "", finalCursorPos, err
				}
			}
		} else {
			if startLineLen == delCount1 && buffer.LineRuneCount(startSel.Row) == 0 && startSel.Row+1 < buffer.LineCount() {
				err = buffer.DeleteRunesAt(startSel.Row, 0, 1)
				if err != nil {
					return "", finalCursorPos, err
				}
			}
		}
	}

	return deletedContent, finalCursorPos, err
}
