// editor/visual_line_mode.go
package core

import "fmt"

type visualLineMode struct {
	startPos   Position        // Only the Row is relevant for selection extent
	charSearch charSearchState // Character search state (f/F/t/T)
	pending    pendingPrefix   // g or z waiting for its second key
}

func NewVisualLineMode() EditorMode {
	return newVisualLineMode(&lastCharSearch{})
}

// newVisualLineMode builds the mode sharing last (the last f/F/t/T search) with the
// other modes.
func newVisualLineMode(last *lastCharSearch) *visualLineMode {
	return &visualLineMode{
		startPos:   Position{-1, -1},
		charSearch: newCharSearchState(last),
	}
}

func (m *visualLineMode) Name() Mode { return VisualLineMode }

// AwaitingLiteral reports whether the next key is the character argument of a
// character search (f/F/t/T), which mappings must not rewrite.
func (m *visualLineMode) AwaitingLiteral() bool { return m.charSearch.waitingForChar }

func (m *visualLineMode) OperatorPending() bool { return false }

func (m *visualLineMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL LINE --")
	editor.UpdateCommand("")
	// Record selection start position (row matters most)
	m.startPos = buffer.GetCursor().Position
	m.charSearch.cancel() // keeps the last search for ; and ,
	m.pending.reset()
	editor.ResetPendingCount()
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
	m.pending.reset()
	editor.ResetPendingCount()
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

	if prefix := m.pending.take(); prefix != 0 {
		handleVisualPrefix(prefix, editor, buffer, key)
		return nil
	}

	if accumulateCount(editor, key) {
		editor.UpdateCommand(fmt.Sprintf("%d", *editor.GetState().PendingCount))
		return nil
	}

	if isPrefixKey(key) {
		m.pending.start(key.Rune) // the count stays pending for the prefix handler
		showPrefix(editor, key.Rune)
		return nil
	}

	if isCharSearchKey(key) {
		startCharSearch(&m.charSearch, editor, key.Rune) // the count stays pending for the target
		return nil
	}

	count := takeCount(editor)

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

	// Ctrl-modified keys carry their letter in Rune after normalisation; none of
	// them are actions, so let them fall through to the motion handler.
	if key.Modifiers&ModCtrl != 0 {
		return false, nil
	}

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
		err = changeLineRange(editor, buffer, startRow, endRow)
		editor.ResetPendingCount()
		return true, err

	case 'u', 'U', '~': // Lowercase / uppercase / toggle case of line selection
		if !state.WithInsertMode {
			return true, nil
		}
		startRow, endRow := m.selectionRows(cursor.Position.Row)
		err = applyCaseToVisualLineSelection(editor, buffer, key.Rune, startRow, endRow)
		if err == nil {
			cur := buffer.GetCursor()
			cur.Position.Row = startRow
			cur.Position.Col = 0
			buffer.SetCursor(cur)
			editor.SetNormalMode()
		}
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

// handleMovement handles motion keys that extend the line selection. count
// is 0 when none was typed.
func (m *visualLineMode) handleMovement(editor Editor, buffer Buffer, key KeyEvent, count int) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()
	availableWidth := state.AvailableWidth
	var moveErr error
	movementAttempted := false
	moveCount := max(count, 1)

	switch key.Key {
	case KeyDown:
		_ = cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyUp:
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyPageDown:
		if count <= 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.MoveDown(buffer, moveCount, availableWidth)
		movementAttempted = true
	case KeyPageUp:
		if count <= 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.MoveUp(buffer, moveCount, availableWidth)
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
