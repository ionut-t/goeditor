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

// Idle reports whether no command is part way typed, count aside.
func (m *visualLineMode) Idle() bool {
	return !m.charSearch.waitingForChar && m.pending.prefix == 0
}

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
	state.VisualStart = m.startPos      // Use VisualStart to indicate visual active
	state.YankSelection = SelectionNone // a new selection, not a yank highlight
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
		exitVisualToNormal(editor, buffer)
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
			editor.SetNormalMode()
			endVisualOperator(buffer)
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
		} else {
			// Vim leaves the cursor where it is when it is above the anchor,
			// and otherwise moves it to column 0 of the anchor's line. The
			// other end stays selected as a yank highlight, as after yy.
			other := m.startPos
			if cursor.Position.Row >= m.startPos.Row {
				other = cursor.Position
				cursor.Position = Position{Row: m.startPos.Row, Col: 0}
				cursor.Preferred = 0
				buffer.SetCursor(cursor)
			}
			editor.SetNormalMode()
			state = editor.GetState()
			state.VisualStart = other
			state.YankSelection = SelectionLine
			editor.SetState(state)
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
			editor.SetNormalMode()
		}
		afterDelete := buffer.GetCursor()
		var pastedContent string
		var pasteErr error
		if startRow < buffer.LineCount() {
			// Lines still exist at startRow — insert clipboard before that line so
			// the pasted content lands exactly where the selection was.
			pastedContent, pasteErr = editor.PasteBefore()
			if pasteErr == nil {
				cur := buffer.GetCursor()
				cur.Position.Row = startRow
				cur.MoveToFirstNonBlank(buffer)
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
			// The selection stays deleted; the cursor stays where the
			// deletion left it, not where the paste would have gone.
			buffer.SetCursor(afterDelete)
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: pastedContent})
		}
		endVisualOperator(buffer)
		editor.ResetPendingCount()
		return true, err

	case 'c': // Change selected text (delete + enter insert)
		if !state.WithInsertMode {
			return true, nil
		}
		_ = editor.Copy(cutType)
		startRow, endRow := m.selectionRows(cursor.Position.Row)
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
			// As after y: the cursor stays where it is when it is above the
			// anchor, and otherwise goes to column 0 of the anchor's line.
			if cursor.Position.Row >= m.startPos.Row {
				cursor.Position = Position{Row: m.startPos.Row, Col: 0}
			}
			buffer.SetCursor(cursor)
			editor.SetNormalMode()
			endVisualOperator(buffer)
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
	var moveErr error
	movementAttempted := false
	moveCount := max(count, 1)

	switch key.Key {
	case KeyDown:
		_ = cursor.moveDown(buffer, moveCount, true)
		movementAttempted = true
	case KeyUp:
		moveErr = cursor.moveUp(buffer, moveCount, true)
		movementAttempted = true
	case KeyPageDown:
		if count <= 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.moveDown(buffer, moveCount, true)
		movementAttempted = true
	case KeyPageUp:
		if count <= 1 {
			moveCount = state.ViewportHeight
		}
		moveErr = cursor.moveUp(buffer, moveCount, true)
		movementAttempted = true
	default:
		switch {
		case key.Rune == 'h' || key.Key == KeyLeft:
			moveErr = moveLeft(editor, &cursor, buffer, 1)
			movementAttempted = true
		case key.Rune == 'l' || key.Key == KeyRight:
			moveErr = moveRight(editor, &cursor, buffer, 1)
			movementAttempted = true
		case key.Key == KeyBackspace:
			moveErr = cursor.MoveLeftOrUp(buffer, 1)
			movementAttempted = true
		case key.Key == KeySpace:
			moveErr = cursor.MoveRightOrDown(buffer, 1)
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
