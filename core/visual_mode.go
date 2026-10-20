package core

import "fmt"

type visualMode struct {
	startPos        Position        // Where visual selection started
	charSearch      charSearchState // Character search state (f/F/t/T)
	pendingModifier rune            // 'i' or 'a' when waiting for text object key
	pending         pendingPrefix   // g or z waiting for its second key
}

func NewVisualMode() EditorMode {
	return newVisualMode(&lastCharSearch{})
}

// newVisualMode builds the mode sharing last (the last f/F/t/T search) with the
// other modes.
func newVisualMode(last *lastCharSearch) *visualMode {
	return &visualMode{
		startPos:   Position{-1, -1},
		charSearch: newCharSearchState(last),
	}
}
func (m *visualMode) Name() Mode { return VisualMode }

// AwaitingLiteral reports whether the next key is the character argument of a
// character search (f/F/t/T), which mappings must not rewrite.
func (m *visualMode) AwaitingLiteral() bool { return m.charSearch.waitingForChar }

func (m *visualMode) OperatorPending() bool { return false }

// Idle reports whether no command is part way typed, count aside.
func (m *visualMode) Idle() bool {
	return !m.charSearch.waitingForChar && m.pendingModifier == 0 && m.pending.prefix == 0
}

func (m *visualMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL --")
	editor.UpdateCommand("")
	// Record selection start position
	m.startPos = buffer.GetCursor().Position
	m.charSearch.cancel() // keeps the last search for ; and ,
	m.pendingModifier = 0
	m.pending.reset()
	editor.ResetPendingCount()
	// Update editor state to reflect visual mode is active
	state := editor.GetState()
	state.VisualStart = m.startPos
	state.YankSelection = SelectionNone // a new selection, not a yank highlight
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
	m.pendingModifier = 0
	m.pending.reset()
	editor.ResetPendingCount()
}

// NormalizeSelection ensures start is before end, line by line, then column by column.
func NormalizeSelection(p1, p2 Position) (start, end Position) {
	if p1.Row < p2.Row || (p1.Row == p2.Row && p1.Col <= p2.Col) {
		return p1, p2
	}
	return p2, p1
}

// HandleKey dispatches each incoming key to the appropriate handler.
func (m *visualMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
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

	// The key after i/a is the text object, even when it is g or z.
	if m.pendingModifier != 0 {
		editor.ResetPendingCount()
		return m.handleTextObject(editor, buffer, key, buffer.GetCursor())
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

// handleTextObject applies a text object (viw, vaw, vip, vap, vi"/va", vi(/va(, …) after an 'i'/'a' modifier.
func (m *visualMode) handleTextObject(editor Editor, buffer Buffer, key KeyEvent, cursor Cursor) *EditorError {
	modifier := m.pendingModifier
	m.pendingModifier = 0
	switch key.Rune {
	case 'w': // viw / vaw — adjust selection to cover the word
		startCol, endCol, found := wordTextObjectRange(buffer, cursor.Position, modifier, 1, editor.IsWordChar)
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
	case '"', '\'', '`': // vi"/va", vi'/va', vi`/va` — select inside/around quote
		startCol, endCol, found := quoteTextObjectRange(buffer, cursor.Position, modifier, key.Rune)
		if found {
			m.startPos = Position{Row: cursor.Position.Row, Col: startCol}
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position.Col = endCol
			buffer.SetCursor(cursor)
		}
	case 'q': // viq/vaq — select inside/around any quote
		startCol, endCol, found := anyQuoteTextObjectRange(buffer, cursor.Position, modifier)
		if found {
			m.startPos = Position{Row: cursor.Position.Row, Col: startCol}
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position.Col = endCol
			buffer.SetCursor(cursor)
		}
	case '(', ')': // vi(/va(, vi)/va) — select inside/around parentheses
		startPos, endPos, found := bracketTextObjectRange(buffer, cursor.Position, modifier, '(', ')')
		if found {
			m.startPos = startPos
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position = endPos
			buffer.SetCursor(cursor)
		}
	case 'b': // vib/vab — select inside/around nearest bracket of any type (()/[]/{}/)
		openPos, closePos, _, _, found := findNearestBracketBounds(buffer, cursor.Position)
		if found {
			startPos, endPos := openPos, closePos
			if modifier == 'i' {
				if s, e, ok := bracketInnerRange(buffer, openPos, closePos); ok {
					startPos, endPos = s, e
				} else {
					found = false
				}
			}
			if found {
				m.startPos = startPos
				state := editor.GetState()
				state.VisualStart = m.startPos
				editor.SetState(state)
				cursor.Position = endPos
				buffer.SetCursor(cursor)
			}
		}
	case '[', ']': // vi[/va[, vi]/va] — select inside/around square brackets
		startPos, endPos, found := bracketTextObjectRange(buffer, cursor.Position, modifier, '[', ']')
		if found {
			m.startPos = startPos
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position = endPos
			buffer.SetCursor(cursor)
		}
	case '{', '}', 'B': // vi{/va{, vi}/va}, viB/vaB — select inside/around curly braces
		startPos, endPos, found := bracketTextObjectRange(buffer, cursor.Position, modifier, '{', '}')
		if found {
			m.startPos = startPos
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position = endPos
			buffer.SetCursor(cursor)
		}
	case '<', '>': // vi</va<, vi>/va> — select inside/around angle brackets
		startPos, endPos, found := bracketTextObjectRange(buffer, cursor.Position, modifier, '<', '>')
		if found {
			m.startPos = startPos
			state := editor.GetState()
			state.VisualStart = m.startPos
			editor.SetState(state)
			cursor.Position = endPos
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

	// Ctrl-modified keys carry their letter in Rune after normalisation; none of
	// them are actions, so let them fall through to the motion handler.
	if key.Modifiers&ModCtrl != 0 {
		return false, nil
	}

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
			editor.SetNormalMode()
			endVisualOperator(buffer)
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
		} else {
			// Vim leaves the cursor at the start of the selection. The rest
			// stays selected as a yank highlight, as after yw.
			start, end := NormalizeSelection(m.startPos, cursor.Position)
			cursor.Position = start
			cursor.Preferred = start.Col
			buffer.SetCursor(cursor)
			editor.SetNormalMode()
			state = editor.GetState()
			state.VisualStart = end
			state.YankSelection = SelectionCharacter
			editor.SetState(state)
		}
		editor.ResetPendingCount()
		return true, err

	case 'p', 'P':
		if !state.WithInsertMode {
			return true, nil
		}
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(buffer, m.startPos, cursor.Position)
		if err == nil {
			// Normal mode first: entering it pulls the cursor onto the last
			// char, but the paste goes where the selection was, which can be
			// after the end of the line.
			editor.SetNormalMode()
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
		}
		content, pasteErr := editor.PasteBefore()
		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			runes := []rune(content)
			if runeLen := len(runes); runeLen > 0 && runes[runeLen-1] != '\n' {
				cursor.Position.Col = finalPos.Col + runeLen - 1
				buffer.SetCursor(cursor)
			}
			editor.DispatchSignal(PasteSignal{content: content})
		}
		endVisualOperator(buffer)
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
			editor.SetInsertMode()
		}
		editor.ResetPendingCount()
		return true, err

	case 'u', 'U', '~': // Lowercase / uppercase / toggle case of selection
		if !state.WithInsertMode {
			return true, nil
		}
		err = applyCaseToVisualSelection(editor, buffer, key.Rune, m.startPos, cursor.Position)
		if err == nil {
			startSel, _ := NormalizeSelection(m.startPos, cursor.Position)
			cursor.Position = startSel
			buffer.SetCursor(cursor)
			editor.SetNormalMode()
			endVisualOperator(buffer)
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

// handleMovement handles motion keys that extend the visual selection. count
// is 0 when none was typed.
func (m *visualMode) handleMovement(editor Editor, buffer Buffer, key KeyEvent, count int) *EditorError {
	cursor := buffer.GetCursor()
	var moveErr error

	switch {
	case key.Rune == 'h' || key.Key == KeyLeft:
		moveErr = moveLeft(editor, &cursor, buffer, max(count, 1))
	case key.Rune == 'l' || key.Key == KeyRight:
		moveErr = moveRight(editor, &cursor, buffer, max(count, 1))
	case key.Key == KeyBackspace:
		moveErr = cursor.MoveLeftOrUp(buffer, max(count, 1))
	case key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, max(count, 1))
	default:
		var earlyReturn bool
		_, earlyReturn, moveErr = applyVisualMotion(&m.charSearch, editor, buffer, &cursor, key, count)
		if earlyReturn {
			return nil
		}
	}

	if moveErr == nil || isBoundaryError(moveErr) {
		buffer.SetCursor(cursor)
	}
	return nil
}
