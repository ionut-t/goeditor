// editor/visual_line_mode.go
package core

type visualLineMode struct {
	startPos     Position        // Only the Row is relevant for selection extent
	currentCount *int            // Temporary count parsed within visual line mode
	charSearch   charSearchState // Character search state (f/F/t/T)
	waitingForG  bool            // true after 'g' is pressed, waiting for second key
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
	m.waitingForG = false
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
	m.waitingForG = false
}

func (m *visualLineMode) GetCurrentCount() *int {
	return m.currentCount
}

func (m *visualLineMode) SetCurrentCount(count *int) {
	m.currentCount = count
}

// HandleKey dispatches each incoming key to the appropriate handler.
func (m *visualLineMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	if key.Key == KeyEscape {
		editor.SetNormalMode()
		return nil
	}

	if m.charSearch.waitingForChar {
		if handled, err := handleVisualCharSearchInput(&m.charSearch, editor, buffer, key); handled {
			return err
		}
	}

	count, processedDigit := getMoveCount(m, editor, key)
	if processedDigit {
		return nil
	}

	if m.waitingForG {
		handleVisualGKey(&m.waitingForG, editor, buffer, key, count)
		return nil
	}

	if actionTaken, err := m.handleAction(editor, buffer, key); actionTaken {
		return err
	}

	return m.handleMovement(editor, buffer, key, count)
}

// handleAction handles keys that operate on the line selection (d/y/p/c/v/V/…).
// Returns (actionTaken, err); actionTaken=true signals HandleKey to return immediately.
func (m *visualLineMode) handleAction(editor Editor, buffer Buffer, key KeyEvent) (bool, *EditorError) {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	var err *EditorError

	switch key.Rune {
	case 'd', 'x': // Delete/Cut selected lines
		if !state.WithInsertMode {
			return true, nil
		}
		if key.Rune == 'x' {
			_ = editor.Copy(cutType)
		}
		startRow, endRow := m.selectionRows(cursor.Position.Row)
		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)
		contentDeleted, delErr := deleteLineRange(editor, buffer, startRow, endRow)
		if delErr == nil {
			editor.SaveHistory()
			editor.SetNormalMode()
			editor.DispatchSignal(DeleteSignal{content: contentDeleted})
		} else {
			err = delErr
		}
		return true, err

	case 'y': // Yank selected lines
		if copyErr := editor.Copy(yankType); copyErr != nil {
			err = &EditorError{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		return true, err

	case 'p', 'P':
		if !state.WithInsertMode {
			return true, nil
		}
		startRow, endRow := m.selectionRows(cursor.Position.Row)
		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)
		_, err = deleteLineRange(editor, buffer, startRow, endRow)
		if err == nil {
			editor.SaveHistory()
			editor.SetNormalMode()
		}
		var pastedContent string
		var pasteErr error
		if startRow < buffer.LineCount() {
			// Lines still exist at startRow — insert clipboard before that line so
			// the pasted content lands exactly where the selection was.
			pastedContent, pasteErr = editor.PasteBefore()
			if pasteErr == nil {
				cur := buffer.GetCursor()
				cur.Position.Row = startRow
				cur.Position.Col = 0
				buffer.SetCursor(cur)
			}
		} else {
			// The selection included the last line(s); append below the new last line.
			cur := buffer.GetCursor()
			cur.Position.Row = buffer.LineCount() - 1
			cur.Position.Col = 0
			buffer.SetCursor(cur)
			pastedContent, pasteErr = editor.PasteAfter()
		}
		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: pastedContent})
		}
		editor.ResetPendingCount()
		return true, err

	case 'c': // Change selected text (delete + enter insert)
		if !state.WithInsertMode {
			return true, nil
		}
		_ = editor.Copy(cutType)
		startRow, endRow := m.selectionRows(cursor.Position.Row)
		initialCursor := buffer.GetCursor()
		initialCursor.Position.Row = startRow
		buffer.SetCursor(initialCursor)
		if _, delErr := deleteLineRange(editor, buffer, startRow, endRow); delErr == nil {
			editor.SaveHistory()
			editor.SetInsertMode()
		} else {
			err = delErr
		}
		editor.ResetPendingCount()
		return true, err

	case 'v': // Switch to character-wise visual mode
		editor.SetVisualMode()
		return true, nil
	case 'V':
		editor.SetNormalMode()
		return true, nil

	}

	return false, nil
}

// selectionRows returns the normalised (startRow, endRow) of the current line selection.
func (m *visualLineMode) selectionRows(cursorRow int) (startRow, endRow int) {
	startRow, endRow = m.startPos.Row, cursorRow
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}
	return
}

// handleMovement handles motion keys that extend the line selection.
func (m *visualLineMode) handleMovement(editor Editor, buffer Buffer, key KeyEvent, count int) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	availableWidth := state.AvailableWidth
	var moveErr error
	movementAttempted := false
	moveCount := count

	switch key.Key {
	case KeyDown:
		_ = cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyUp:
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyPageDown:
		if count == 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyPageUp:
		if count == 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyCtrlD:
		moveErr = cursor.ScrollDown(buffer, state.ViewportHeight, availableWidth)
		movementAttempted = true
	case KeyCtrlU:
		moveErr = cursor.ScrollUp(buffer, state.ViewportHeight, availableWidth)
		movementAttempted = true
	default:
		col := cursor.Position.Col
		switch {
		case key.Rune == 'h' || key.Key == KeyLeft:
			moveErr = cursor.MoveLeftOrUp(buffer, 1, col)
			movementAttempted = true
		case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
			moveErr = cursor.MoveRightOrDown(buffer, 1, col)
			movementAttempted = true
		case key.Rune == 'g':
			m.waitingForG = true
			editor.UpdateCommand("g")
			return nil
		default:
			var earlyReturn bool
			movementAttempted, earlyReturn, moveErr = applyVisualMotion(&m.charSearch, editor, buffer, &cursor, key, count)
			if earlyReturn {
				return nil
			}
		}
	}

	if movementAttempted && (moveErr == nil || isBoundaryError(moveErr)) {
		buffer.SetCursor(cursor)
		return nil
	}

	return nil
}
