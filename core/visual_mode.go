package core

import (
	"errors"
)

type visualMode struct {
	startPos        Position        // Where visual selection started
	currentCount    *int            // Temporary count parsed within visual mode
	charSearch      charSearchState // Character search state (f/F/t/T)
	pendingModifier rune            // 'i' or 'a' when waiting for text object key
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

	// --- Text Object Dispatch (after 'i'/'a' modifier) ---
	if m.pendingModifier != 0 {
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

	state := editor.GetState()

	// --- Visual Mode Actions ---
	switch key.Rune {
	case 'd', 'x': // Delete/Cut selected text
		if !state.WithInsertMode {
			return nil
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

		actionTaken = true
		editor.ResetPendingCount()
		editor.DispatchSignal(DeleteSignal{content: contentDeleted})

	case '/':
		editor.SetSearchMode()

	case 'n':
		cursor = editor.NextSearchResult()

	case 'N':
		cursor = editor.PreviousSearchResult()

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
		if !state.WithInsertMode {
			return nil
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

	case 'i', 'a': // Text object modifier — wait for the object key (w, p, …)
		m.pendingModifier = key.Rune
		actionTaken = true

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
	case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, count, col)
	case key.Rune == 'w':
		// 'w' is an exclusive motion. In charwise visual mode the endpoint is
		// inclusive, so adjust back one column when the cursor just crossed
		// whitespace onto the first char of a new word — otherwise vw would
		// include that char but dw would not.
		// Guard: skip the adjustment when already on whitespace (the cursor was
		// placed there by a previous w adjustment). Without this guard, pressing
		// w a second time bounces between the space and the word start forever.
		preMoveLineRunes := buffer.GetLineRunes(cursor.Position.Row)
		preMoveCol := cursor.Position.Col
		startedOnWhitespace := preMoveCol < len(preMoveLineRunes) && isWhiteSpace(preMoveLineRunes[preMoveCol])
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
		if moveErr == nil && !startedOnWhitespace {
			col := cursor.Position.Col
			lineRunes := buffer.GetLineRunes(cursor.Position.Row)
			if col > 0 && col < len(lineRunes) &&
				editor.IsWordChar(lineRunes[col]) &&
				isWhiteSpace(lineRunes[col-1]) {
				cursor.Position.Col--
			}
		}
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
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
