package core

type visualMode struct {
	startPos        Position        // Where visual selection started
	currentCount    *int            // Temporary count parsed within visual mode
	charSearch      charSearchState // Character search state (f/F/t/T)
	pendingModifier rune            // 'i' or 'a' when waiting for text object key
	waitingForG     bool            // true after 'g' is pressed, waiting for second key
}

func NewVisualMode() EditorMode {
	return &visualMode{
		startPos:     Position{-1, -1},
		currentCount: nil,
		charSearch:   charSearchState{},
	}
}
func (m *visualMode) Name() Mode { return VisualMode }

func (m *visualMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL --")
	editor.UpdateCommand("")
	// Record selection start position
	m.startPos = buffer.GetCursor().Position
	m.currentCount = nil
	m.charSearch = charSearchState{}
	m.pendingModifier = 0
	m.waitingForG = false
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
	m.waitingForG = false
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

// HandleKey dispatches each incoming key to the appropriate handler.
func (m *visualMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
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

	if m.pendingModifier != 0 {
		return m.handleTextObject(editor, buffer, key, buffer.GetCursor())
	}

	if actionTaken, err := m.handleAction(editor, buffer, key); actionTaken {
		return err
	}

	return m.handleMovement(editor, buffer, key, count)
}

// handleTextObject applies a text object (viw, vaw, vip, vap) after an 'i'/'a' modifier.
func (m *visualMode) handleTextObject(editor Editor, buffer Buffer, key KeyEvent, cursor Cursor) *EditorError {
	modifier := m.pendingModifier
	m.pendingModifier = 0
	switch key.Rune {
	case 'w': // viw / vaw — adjust selection to cover the word
		startCol, endCol, found := wordTextObjectRange(buffer, cursor.Position, modifier, editor.IsWordChar)
		if found {
			m.startPos = Position{Row: cursor.Position.Row, Col: startCol}
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position.Col = endCol
			buffer.SetCursor(cursor)
		}
	case 'p': // vip / vap — expand to paragraph and switch to visual line mode
		startRow, endRow, found := paragraphRows(buffer, cursor.Position, modifier)
		if found {
			cursor.Position = Position{Row: startRow, Col: 0}
			buffer.SetCursor(cursor)
			editor.SetVisualLineMode()
			// SetVisualLineMode.Enter() records startPos from the buffer cursor (startRow).
			// Now move cursor to endRow to define the selection end.
			cursor = buffer.GetCursor()
			cursor.Position.Row = endRow
			buffer.SetCursor(cursor)
		}
	}
	return nil
}

// handleAction handles keys that perform an operation on the visual selection (d/y/p/c/v/V/…).
// Returns (actionTaken, err); actionTaken=true signals HandleKey to return immediately.
func (m *visualMode) handleAction(editor Editor, buffer Buffer, key KeyEvent) (bool, *EditorError) {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	var err *EditorError

	switch key.Rune {
	case 'd', 'x': // Delete/Cut selected text
		if !state.WithInsertMode {
			return true, nil
		}
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
		editor.ResetPendingCount()
		editor.DispatchSignal(DeleteSignal{content: contentDeleted})
		return true, err

	case 'y': // Yank (Copy) selected text
		if copyErr := editor.Copy(yankType); copyErr != nil {
			err = &EditorError{
				id:  ErrCopyFailedId,
				err: copyErr,
			}
		}
		editor.ResetPendingCount()
		return true, err

	case 'p':
		if !state.WithInsertMode {
			return true, nil
		}
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
			editor.SaveHistory()
			editor.SetNormalMode()
		}
		content, pasteErr := editor.Paste()
		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}
		editor.ResetPendingCount()
		return true, err

	case 'c': // Change selected text (delete + enter insert)
		if !state.WithInsertMode {
			return true, nil
		}
		_ = editor.Copy(cutType)
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
			editor.SaveHistory()
			editor.SetInsertMode()
		}
		editor.ResetPendingCount()
		return true, err

	case 'i', 'a': // Text object modifier — wait for the object key (w, p, …)
		m.pendingModifier = key.Rune
		return true, nil

	case 'v':
		editor.SetNormalMode()
		return true, nil
	case 'V':
		editor.SetVisualLineMode()
		return true, nil
	}

	return false, nil
}

// handleMovement handles motion keys that extend the visual selection.
func (m *visualMode) handleMovement(editor Editor, buffer Buffer, key KeyEvent, count int) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()

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
	case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, count, col)
	case key.Rune == 'g':
		m.waitingForG = true
		editor.UpdateCommand("g")
		return nil
	default:
		var movementAttempted, earlyReturn bool
		moveErr, movementAttempted, earlyReturn = applyVisualMotion(&m.charSearch, editor, buffer, &cursor, key, count)
		if earlyReturn {
			return nil
		}
		if !movementAttempted && countWasPending {
			editor.ResetPendingCount()
		}
	}

	if moveErr == nil || isBoundaryError(moveErr) {
		buffer.SetCursor(cursor)
		return nil
	}

	if countWasPending {
		editor.ResetPendingCount()
	}
	return nil
}
