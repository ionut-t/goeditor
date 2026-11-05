package core

import "fmt"

type visualMode struct {
	startPos        Position        // Where visual selection started
	charSearch      charSearchState // Character search state (f/F/t/T)
	pendingModifier rune            // 'i' or 'a' when waiting for text object key
	pending         pendingPrefix   // g or z waiting for its second key
	replacing       bool            // r waiting for its character
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
// character search (f/F/t/T) or of r, which mappings must not rewrite.
func (m *visualMode) AwaitingLiteral() bool { return m.charSearch.waitingForChar || m.replacing }

func (m *visualMode) OperatorPending() bool { return false }

// Idle reports whether no command is part way typed, count aside.
func (m *visualMode) Idle() bool {
	return !m.charSearch.waitingForChar && m.pendingModifier == 0 && m.pending.prefix == 0 && !m.replacing
}

func (m *visualMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- VISUAL --")
	editor.UpdateCommand("")
	// Record selection start position
	m.startPos = buffer.GetCursor().Position
	m.charSearch.cancel() // keeps the last search for ; and ,
	m.pendingModifier = 0
	m.pending.reset()
	m.replacing = false
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
	m.replacing = false
	editor.ResetPendingCount()
}

// cancelPending drops a part typed command (i/a, g/z, f/F/t/T, r) and its count,
// as Vim does on Esc, keeping the mode and the selection.
func (m *visualMode) cancelPending(editor Editor) {
	m.charSearch.cancel() // keeps the last search for ; and ,
	m.pendingModifier = 0
	m.pending.reset()
	m.replacing = false
	editor.ResetPendingCount()
	editor.UpdateCommand("")
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
		if !m.Idle() {
			m.cancelPending(editor)
			return nil
		}
		exitVisualToNormal(editor, buffer)
		return nil
	}

	if m.replacing {
		m.replacing = false
		editor.UpdateCommand("")
		ch, ok := visualReplaceChar(key)
		if !ok {
			return nil // the selection stays, as after Esc
		}
		start, end := NormalizeSelection(m.startPos, buffer.GetCursor().Position)
		return replaceSelection(editor, buffer, start, end, false, start, ch)
	}

	if m.charSearch.waitingForChar {
		if handled, err := handleVisualCharSearchInput(&m.charSearch, editor, buffer, key); handled {
			return err
		}
	}

	if prefix := m.pending.take(); prefix != 0 {
		if prefix == 'g' && key.Rune == 'J' && key.Modifiers&ModCtrl == 0 {
			takeCount(editor) // ignored, as by J
			editor.UpdateCommand("")
			return m.join(editor, buffer, false)
		}
		handleVisualPrefix(prefix, editor, buffer, key)
		return nil
	}

	// The key after i/a is the text object, even when it is g, z or a digit
	// (an object Vim doesn't know, so it is used up rather than counted).
	if m.pendingModifier != 0 {
		return m.handleTextObject(editor, buffer, key, buffer.GetCursor(), takeCount(editor))
	}

	if accumulateCount(editor, key) {
		editor.UpdateCommand(fmt.Sprintf("%d", *editor.GetState().PendingCount))
		return nil
	}

	if (key.Rune == 'i' || key.Rune == 'a') && key.Modifiers&ModCtrl == 0 {
		m.pendingModifier = key.Rune // the count stays pending for the text object
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

	if actionTaken, err := m.handleAction(editor, buffer, key, count); actionTaken {
		return err
	}

	return m.handleMovement(editor, buffer, key, count)
}

// handleTextObject applies a text object (viw, vaw, vip, vap, vi"/va", vi(/va(, …) after an 'i'/'a' modifier.
// count is used as in visualTextObject.
func (m *visualMode) handleTextObject(editor Editor, buffer Buffer, key KeyEvent, cursor Cursor, count int) *EditorError {
	modifier := m.pendingModifier
	m.pendingModifier = 0
	sel := visualTextObject(editor, buffer, modifier, key.Rune, m.startPos, cursor.Position, count)
	m.setAnchor(editor, sel.anchor)
	setObjectCursor(buffer, sel.cursor)
	if sel.found && sel.linewise {
		editor.SetVisualLineMode() // keeps the anchor
	}
	return nil
}

// objectSelection is the selection a text object gives in visual mode.
type objectSelection struct {
	anchor, cursor Position
	linewise       bool // ip/ap: the selection is whole lines
	found          bool // false: no object; anchor and cursor are where the search left them
}

// visualTextObject finds the selection the object key (after modifier i or a)
// gives in visual mode, from the selection anchor to cursor. count extends word
// objects over additional words (v2aw), takes block objects count levels out
// (v2i() and makes i" take the quotes (v2i"); ip/ap ignore it. When the object
// isn't found the selection is unchanged, except for word objects, which, as in
// Vim, move as far as the search went (v9aw).
func visualTextObject(editor Editor, buffer Buffer, modifier, key rune, anchor, cursor Position, count int) objectSelection {
	sel := objectSelection{anchor: anchor, cursor: cursor}
	set := func(start, end Position, found bool) objectSelection {
		if found {
			sel.anchor, sel.cursor, sel.found = start, end, true
		}
		return sel
	}
	switch key {
	case 'w', 'W': // iw / aw, iW / aW — the word or WORD
		cls := editor.CharClass
		if key == 'W' {
			cls = bigWordClass
		}
		start, end, _, ok := currentWord(buffer, cursor, anchor, true, count, modifier == 'a', cls)
		sel.anchor, sel.cursor, sel.found = start, end, ok
		return sel
	case 'p': // ip / ap — whole lines of the paragraph
		startRow, endRow, found := paragraphRows(buffer, cursor, modifier)
		if !found {
			return sel
		}
		// As Vim, the anchor keeps its column when it stays on its line.
		start := Position{Row: startRow, Col: 0}
		if startRow == anchor.Row {
			start = anchor
		}
		sel.linewise = true
		return set(start, Position{Row: endRow, Col: 0}, true)
	case '"', '\'', '`': // i" / a", i' / a', i` / a` — inside/around quotes
		startCol, endCol, found := quoteTextObjectRange(buffer, cursor, modifier, key, count)
		if !found && modifier == 'i' {
			if openCol, closeCol, ok := quotePair(buffer, cursor, key); ok && closeCol == openCol+1 {
				// As Vim, i" on empty quotes selects both quotes, from
				// the closing one back to the opening one.
				startCol, endCol, found = closeCol, openCol, true
			}
		}
		return set(Position{Row: cursor.Row, Col: startCol}, Position{Row: cursor.Row, Col: endCol}, found)
	case 'q': // iq / aq — inside/around any quote
		startCol, endCol, found := anyQuoteTextObjectRange(buffer, cursor, modifier, count)
		if !found && modifier == 'i' {
			if openCol, closeCol, ok := anyQuotePair(buffer, cursor, count); ok && closeCol == openCol+1 {
				startCol, endCol, found = closeCol, openCol, true // empty quotes, as i"
			}
		}
		return set(Position{Row: cursor.Row, Col: startCol}, Position{Row: cursor.Row, Col: endCol}, found)
	case 'b': // ib / ab — inside/around the nearest bracket of any type (()/[]/{})
		if anchor != cursor {
			start, end := NormalizeSelection(anchor, cursor)
			return set(selectionBlockRange(buffer, start, end, modifier, count, anyBracketFinder(buffer)))
		}
		openPos, closePos, found := findNearestBracketBounds(buffer, cursor, count)
		if found && modifier == 'i' {
			openPos, closePos, found = bracketInnerRange(buffer, openPos, closePos)
		}
		return set(openPos, closePos, found)
	}
	var openChar, closeChar rune
	switch key {
	case '(', ')': // i( / a(, i) / a)
		openChar, closeChar = '(', ')'
	case '[', ']': // i[ / a[, i] / a]
		openChar, closeChar = '[', ']'
	case '{', '}', 'B': // i{ / a{, i} / a}, iB / aB
		openChar, closeChar = '{', '}'
	case '<', '>': // i< / a<, i> / a>
		openChar, closeChar = '<', '>'
	default:
		return sel // not an object
	}
	if anchor != cursor {
		// As Vim, a selection over more than one character is searched
		// from its start and may grow out a level (see selectionBlockRange).
		start, end := NormalizeSelection(anchor, cursor)
		return set(selectionBlockRange(buffer, start, end, modifier, count, bracketFinder(buffer, openChar, closeChar)))
	}
	start, end, found := bracketTextObjectRange(buffer, cursor, modifier, openChar, closeChar, count)
	return set(start, end, found)
}

// setObjectCursor puts the cursor where a text object left it. Vim sets curswant
// after any object, even one that isn't found, so j and k go back to that column.
func setObjectCursor(buffer Buffer, pos Position) {
	cursor := buffer.GetCursor()
	cursor.Position = pos
	cursor.Preferred = pos.Col
	buffer.SetCursor(cursor)
}

// handleAction handles keys that perform an operation on the visual selection (d/y/p/c/v/V/…).
// count is 0 when none was typed; only p and P use it, for the copies they put.
// Returns (actionTaken, err); actionTaken=true signals HandleKey to return immediately.
func (m *visualMode) handleAction(editor Editor, buffer Buffer, key KeyEvent, count int) (bool, *EditorError) {
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
		contentDeleted, finalPos, err = deleteVisualSelection(editor, buffer, m.startPos, cursor.Position)
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
			buffer.SetCursor(cursor)
			editor.SetNormalMode()
			endVisualOperator(buffer) // the start can be a line break
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
		if tooLong := checkPutCount(editor, count, false); tooLong != nil {
			return true, tooLong // nothing changes; the selection stays
		}
		repeatAsDelete(editor)
		var finalPos Position
		_, finalPos, err = deleteVisualSelection(editor, buffer, m.startPos, cursor.Position)
		if err == nil {
			// Normal mode first: entering it pulls the cursor onto the last
			// char, but the paste goes where the selection was, which can be
			// after the end of the line.
			editor.SetNormalMode()
			cursor.Position = finalPos
			buffer.SetCursor(cursor)
		}
		content, pasteErr := pasteSplit(editor, count)
		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			// The paste puts the cursor where Vim does.
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
		_, finalPos, err = deleteVisualSelection(editor, buffer, m.startPos, cursor.Position)
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

	case 'J': // Join the selected lines; the count is ignored, as in Vim
		return true, m.join(editor, buffer, true)

	case 'r': // Replace every selected character with the next key
		if state.WithInsertMode {
			m.replacing = true
			editor.UpdateCommand("r")
		}
		return true, nil

	case 'o', 'O': // Go to the other end of the selection
		m.startPos = swapVisualEnds(editor, buffer, m.startPos)
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

// join is J (spaces on) and gJ on the selection: the lines it touches are
// joined, and undo goes back to its start.
func (m *visualMode) join(editor Editor, buffer Buffer, spaces bool) *EditorError {
	if !editor.GetState().WithInsertMode {
		return nil
	}
	start, end := NormalizeSelection(m.startPos, buffer.GetCursor().Position)
	return joinSelection(editor, buffer, start, end.Row, start, spaces)
}

// handleMovement handles motion keys that extend the visual selection. count
// is 0 when none was typed.
func (m *visualMode) handleMovement(editor Editor, buffer Buffer, key KeyEvent, count int) *EditorError {
	cursor := buffer.GetCursor()
	var moveErr error

	switch {
	case key.Rune == 'h' || key.Key == KeyLeft:
		moveErr = moveLeft(editor, &cursor, buffer, max(count, 1), true)
	case key.Rune == 'l' || key.Key == KeyRight:
		moveErr = moveRight(editor, &cursor, buffer, max(count, 1), true)
	case key.Key == KeyBackspace:
		moveErr = cursor.moveLeftOrUp(buffer, max(count, 1), true)
	case key.Key == KeySpace:
		moveErr = cursor.moveRightOrDown(buffer, max(count, 1), true)
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

// pasteSplit pastes count copies of the clipboard where a deleted charwise
// selection was: lines go in between the halves of the line split there, not
// joined into it. An Editor that isn't the core one pastes one copy.
func pasteSplit(editor Editor, count int) (string, error) {
	if p, ok := editor.(paster); ok {
		content, err := p.pasteSplit(count)
		if err != nil {
			return "", err.err
		}
		return content, nil
	}
	return editor.PasteBefore()
}
