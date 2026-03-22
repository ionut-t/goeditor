// editor/visual_line_mode.go
package core

import (
	"errors"
)

type visualLineMode struct {
	startPos     Position        // Only the Row is relevant for selection extent
	currentCount *int            // Temporary count parsed within visual line mode
	charSearch   charSearchState // Character search state (f/F/t/T)
}

func NewVisualLineMode() EditorMode {
	return &visualLineMode{
		startPos:     Position{-1, -1},
		currentCount: nil,
		charSearch:   charSearchState{},
	}
}

func (m *visualLineMode) Name() Mode { return VisualLineMode }

func (m *visualLineMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL LINE --")
	editor.UpdateCommand("")
	// Record selection start position (row matters most)
	m.startPos = buffer.GetCursor().Position
	m.currentCount = nil
	m.charSearch = charSearchState{}
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

func (m *visualLineMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	if key.Key == KeyEscape {
		editor.SetNormalMode()
		return nil
	}

	cursor := buffer.GetCursor() // Get current cursor state
	var err *EditorError
	actionTaken := false // Flag if an action was performed
	availableWidth := editor.GetState().AvailableWidth

	// --- Handle Character Search Input (waiting for character after f/F/t/T) ---
	if m.charSearch.waitingForChar {
		if handled, err := handleVisualCharSearchInput(&m.charSearch, editor, buffer, key); handled {
			return err
		}
	}

	count, processedDigit := getMoveCount(m, editor, key)

	// If a digit was just processed, wait for the next key
	if processedDigit {
		return nil
	}

	state := editor.GetState()

	// --- Visual Line Mode Actions ---
	switch key.Rune {
	case 'd', 'x': // Delete/Cut selected lines
		if !state.WithInsertMode {
			return nil
		}

		if key.Rune == 'x' {
			_ = editor.Copy(cutType)
		}

		startRow, endRow := m.startPos.Row, cursor.Position.Row
		if startRow > endRow {
			startRow, endRow = endRow, startRow // Ensure start <= end
		}

		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)

		contentDeleted, err := deleteLineRange(editor, buffer, startRow, endRow)

		if err == nil {
			editor.SaveHistory()
			editor.SetNormalMode()
			editor.DispatchSignal(DeleteSignal{content: contentDeleted})
		}

		actionTaken = true

	case 'y': // Yank selected lines
		if copyErr := editor.Copy(yankType); copyErr != nil {
			err = &EditorError{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		actionTaken = true

	case 'p':
		if !state.WithInsertMode {
			return nil
		}

		startRow, endRow := m.startPos.Row, cursor.Position.Row
		if startRow > endRow {
			startRow, endRow = endRow, startRow // Ensure start <= end
		}

		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)

		_, err = deleteLineRange(editor, buffer, startRow, endRow)

		if err == nil {
			editor.SaveHistory()
			editor.SetNormalMode()
		}

		actionTaken = true

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
		if !state.WithInsertMode {
			return nil
		}

		_ = editor.Copy(cutType)
		startRow, endRow := m.startPos.Row, cursor.Position.Row
		if startRow > endRow {
			startRow, endRow = endRow, startRow // Ensure start <= end
		}

		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)

		if _, err = deleteLineRange(editor, buffer, startRow, endRow); err == nil {
			editor.SaveHistory()
			editor.SetInsertMode()
		}

		actionTaken = true
		editor.ResetPendingCount()

	// Mode Switches
	case 'v': // Switch to character-wise visual mode
		editor.SetVisualMode() // Switch to character-wise visual mode
		actionTaken = true     // Mode switch is an action
	case 'V':
		editor.SetNormalMode() // Switch to normal mode
		actionTaken = true

	case '/':
		editor.SetSearchMode()

	case 'n':
		cursor = editor.NextSearchResult()

	case 'N':
		cursor = editor.PreviousSearchResult()
	}

	if actionTaken {
		return err
	} // Return if delete/yank/paste/change/mode switch was performed

	// --- Visual Line Mode Movements (Update selection end row) ---
	// Only row movements are relevant for selection extent

	var moveErr error
	movementAttempted := false
	moveCount := count // Use 'count' for actual move amount calculation
	switch key.Key {   // Use Key for arrows/pgup/dn
	case KeyDown:
		cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyUp:
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
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

	case KeyCtrlD:
		moveErr = cursor.ScrollDown(buffer, state.ViewportHeight, availableWidth)
		movementAttempted = true
	case KeyCtrlU:
		moveErr = cursor.ScrollUp(buffer, state.ViewportHeight, availableWidth)
		movementAttempted = true

	default:
		col := cursor.Position.Col // Get Column from cursor state
		switch {                   // Allow rune based keys
		case key.Rune == 'j':
			moveErr = cursor.MoveDown(buffer, moveCount, availableWidth)
			movementAttempted = true
		case key.Rune == 'k':
			moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
			movementAttempted = true
		// Horizontal movements affect cursor position but not line selection extent
		case key.Rune == 'h' || key.Key == KeyLeft:
			moveErr = cursor.MoveLeftOrUp(buffer, 1, col)
			movementAttempted = true
		case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
			moveErr = cursor.MoveRightOrDown(buffer, 1, col)
			movementAttempted = true
		case key.Rune == '{':
			moveErr = cursor.MoveBlockBackward(buffer, count)
			movementAttempted = true
		case key.Rune == '}':
			moveErr = cursor.MoveBlockForward(buffer, count)
			movementAttempted = true
		case key.Rune == '0' || key.Key == KeyHome:
			cursor.MoveToLineStart()
			movementAttempted = true
		case key.Rune == '$' || key.Key == KeyEnd:
			cursor.MoveToLineEnd(buffer, availableWidth)
			movementAttempted = true
		case key.Rune == '^':
			cursor.MoveToFirstNonBlank(buffer, availableWidth)
			movementAttempted = true
		case key.Rune == 'g':
			cursor.MoveToBufferStart()
			movementAttempted = true
		case key.Rune == 'G':
			cursor.MoveToBufferEnd(buffer, availableWidth)
			movementAttempted = true

		case key.Key == KeyEnter:
			if count > 0 {
				cursor.Position.Row = count - 1
				buffer.SetCursor(cursor)
				editor.UpdateCommand("")
				editor.ResetPendingCount()
			}

		// Character search motions
		case key.Rune == 'f': // Find character forward
			m.charSearch.searchType = 'f'
			m.charSearch.waitingForChar = true
			editor.UpdateCommand("f")
			return nil

		case key.Rune == 'F': // Find character backward
			m.charSearch.searchType = 'F'
			m.charSearch.waitingForChar = true
			editor.UpdateCommand("F")
			return nil

		case key.Rune == 't': // Till character forward
			m.charSearch.searchType = 't'
			m.charSearch.waitingForChar = true
			editor.UpdateCommand("t")
			return nil

		case key.Rune == 'T': // Till character backward
			m.charSearch.searchType = 'T'
			m.charSearch.waitingForChar = true
			editor.UpdateCommand("T")
			return nil

		case key.Rune == ';': // Repeat last character search
			if m.charSearch.searchType != 0 && m.charSearch.lastChar != 0 {
				searchErr := performCharSearch(buffer, &m.charSearch, m.charSearch.searchType, m.charSearch.lastChar, count)
				if searchErr != nil {
					editor.DispatchError(ErrCharNotFoundId, searchErr)
				}
				cursor = buffer.GetCursor() // Refresh cursor after search
				movementAttempted = true
			}

		case key.Rune == ',': // Repeat last character search in opposite direction
			if m.charSearch.searchType != 0 && m.charSearch.lastChar != 0 {
				// Reverse the search direction
				reversedType := m.charSearch.searchType
				switch m.charSearch.searchType {
				case 'f':
					reversedType = 'F'
				case 'F':
					reversedType = 'f'
				case 't':
					reversedType = 'T'
				case 'T':
					reversedType = 't'
				}

				// Temporarily use the reversed type
				originalType := m.charSearch.searchType
				searchErr := performCharSearch(buffer, &m.charSearch, reversedType, m.charSearch.lastChar, count)
				m.charSearch.searchType = originalType // Restore original type
				if searchErr != nil {
					editor.DispatchError(ErrCharNotFoundId, searchErr)
				}
				cursor = buffer.GetCursor() // Refresh cursor after search
				movementAttempted = true
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

	return err
}

