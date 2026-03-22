package core

import (
	"errors"
	"fmt"
	"strings"
)

type normalMode struct {
	pendingKey        KeyEvent        // Stores the first key of a multi-key command (e.g., 'd' in 'dd')
	pendingModifier   rune            // Stores text object modifier ('i' for inside, 'a' for around)
	charSearch        charSearchState // Character search state (f/F/t/T)
	waitingForReplace bool            // True when waiting for character input after 'r'
}

func NewNormalMode() EditorMode {
	return &normalMode{
		pendingKey:      KeyEvent{Key: KeyUnknown},
		pendingModifier: 0,
		charSearch:      charSearchState{},
	}
}

func (m *normalMode) Name() Mode { return NormalMode }

func (m *normalMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- NORMAL --")
	editor.UpdateCommand("")

	// Reset pending state on entering normal mode
	m.pendingKey = KeyEvent{Key: KeyUnknown}
	m.pendingModifier = 0
	m.charSearch = charSearchState{}
	m.waitingForReplace = false
	editor.ResetPendingCount()
	// Clear visual selection when entering normal mode
	state := editor.GetState()
	state.VisualStart = Position{-1, -1}
	editor.SetState(state)
}

func (m *normalMode) Exit(editor Editor, buffer Buffer) {
	// Clear pending state when exiting normal mode
	m.pendingKey = KeyEvent{Key: KeyUnknown}
	m.pendingModifier = 0
	m.charSearch = charSearchState{}
	m.waitingForReplace = false
}

func (m *normalMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	var err *EditorError
	actionTaken := false // Track if the key (or sequence) resulted in an action
	state := editor.GetState()
	pendingCount := state.PendingCount
	availableWidth := state.AvailableWidth
	skipCursorUpdate := false
	cursor := buffer.GetCursor() // Get cursor for operations

	// --- Handle Character Search Input (waiting for character after f/F/t/T) ---
	if m.charSearch.waitingForChar {
		m.charSearch.waitingForChar = false
		editor.UpdateCommand("") // Clear the command display

		// Handle escape to cancel
		if key.Key == KeyEscape {
			m.clearPendingState(editor)
			return nil
		}

		// Get the character to search for
		if key.Rune == 0 {
			// Not a valid character
			m.clearPendingState(editor)
			return nil
		}

		count := 1
		if pendingCount != nil {
			count = *pendingCount
			editor.ResetPendingCount()
		}

		// Check if there's a pending operator (d/y/c)
		if m.pendingKey.Key != KeyUnknown || m.pendingKey.Rune != 0 {
			// Operator + character search motion (e.g., df,  yt;)
			firstKey := m.pendingKey
			m.pendingKey = KeyEvent{Key: KeyUnknown}

			op := ""
			switch firstKey.Rune {
			case 'd':
				op = "delete"
			case 'y':
				op = "yank"
			case 'c':
				op = "change"
			}

			if op != "" {
				err = handleCharSearchOperator(editor, buffer, op, m.charSearch.searchType, key.Rune, count)
				if err != nil {
					m.clearPendingState(editor)
				}
				return err
			}
		}

		// No pending operator - just perform the character search
		searchErr := performCharSearch(buffer, &m.charSearch, m.charSearch.searchType, key.Rune, count)
		if searchErr != nil {
			m.clearPendingState(editor)
			editor.DispatchError(ErrCharNotFoundId, searchErr)
		}
		return nil
	}

	// --- Handle Replace Character Input (waiting for character after 'r') ---
	if m.waitingForReplace {
		m.waitingForReplace = false
		editor.UpdateCommand("")

		if key.Key == KeyEscape || key.Rune == 0 {
			return nil
		}

		err = replaceCharUnderCursor(editor, buffer, key.Rune)
		return err
	}

	// --- Handle Pending Operation (e.g., after 'd') ---
	if m.pendingKey.Key != KeyUnknown || m.pendingKey.Rune != 0 {
		firstKey := m.pendingKey

		count := 1
		if pendingCount != nil {
			count = *pendingCount
			editor.ResetPendingCount()
		}

		op := ""
		switch firstKey.Rune {
		case 'd':
			op = "delete"
		case 'y':
			op = "yank"
		case 'c': // Add change later
			op = "change"
		default:
			m.pendingKey = KeyEvent{Key: KeyUnknown}
			m.pendingModifier = 0
			return &EditorError{
				id:  ErrNoPendingOperationId,
				err: ErrNoPendingOperation,
			}
		}

		// Check if we're waiting for a text object after modifier (i/a)
		if m.pendingModifier != 0 {
			modifier := m.pendingModifier
			m.pendingModifier = 0
			m.pendingKey = KeyEvent{Key: KeyUnknown}

			// Handle text objects after modifier
			switch key.Rune {
			case 'w': // iw or aw = inside/around word
				switch op {
				case "yank":
					err = yankTextObject(editor, buffer, modifier, 'w')
					actionTaken = true
				case "delete":
					err = deleteTextObject(editor, buffer, modifier, 'w')
					actionTaken = true
				case "change":
					err = changeTextObject(editor, buffer, modifier, 'w')
					actionTaken = true
				}
			case 'p': // ip or ap = inside/around paragraph
				switch op {
				case "yank":
					err = yankParagraphTextObject(editor, buffer, modifier)
					actionTaken = true
				case "delete":
					err = deleteParagraphTextObject(editor, buffer, modifier)
					actionTaken = true
				case "change":
					err = changeParagraphTextObject(editor, buffer, modifier)
					actionTaken = true
				}
			default:
				editor.DispatchError(ErrInvalidMotionId, fmt.Errorf("invalid text object '%c' after '%c'", key.Rune, modifier))
				actionTaken = true
			}

			if err != nil {
				return err
			}

			if actionTaken {
				editor.UpdateCommand("")
				return nil
			}

			return nil
		}

		// Check for text object modifiers (i/a)
		if key.Rune == 'i' || key.Rune == 'a' {
			m.pendingModifier = key.Rune
			editor.UpdateCommand(fmt.Sprintf("%s%c%c", editor.GetState().CommandLine, firstKey.Rune, key.Rune))
			return nil // Wait for the text object key
		}

		// Check for character search motions (f/F/t/T)
		if key.Rune == 'f' || key.Rune == 'F' || key.Rune == 't' || key.Rune == 'T' {
			m.charSearch.searchType = key.Rune
			m.charSearch.waitingForChar = true
			editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
			// Keep pendingKey - we'll process the operator after getting the character
			return nil
		}

		// Consume the pending key now if not waiting for text object
		m.pendingKey = KeyEvent{Key: KeyUnknown}

		// Handle motion keys after the operator
		// Supported operator-motion commands:
		//
		// Yank commands:
		//   yy  - yank current line (line-wise)
		//   yw  - yank word forward (character-wise)
		//   yb  - yank word backward (character-wise)
		//   y$  - yank to end of line (character-wise)
		//   yiw - yank inside word (character-wise)
		//   yaw - yank around word, includes surrounding whitespace (character-wise)
		//
		// Delete commands:
		//   dd  - delete current line (line-wise)
		//   dw  - delete word forward (character-wise)
		//   d$  - delete to end of line (character-wise)
		switch key.Rune {
		case 'd': // dd = delete line
			if op == "delete" {
				var deletedContent string
				deletedContent, err = deleteLines(editor, buffer, count)
				editor.DispatchSignal(DeleteSignal{content: deletedContent})
				actionTaken = true
			}
		case 'y': // yy = yank line
			if op == "yank" {
				err = yankLines(editor, buffer, count)
				actionTaken = true
			}
		case 'c': // cc = change line
			if op == "change" {
				_, err = deleteLines(editor, buffer, count)
				if err == nil {
					editor.SetInsertMode()
				}
				actionTaken = true
			}
		case 'w': // dw = delete word, yw = yank word forward, cw = change word
			switch op {
			case "delete":
				err = deleteWords(editor, buffer, count)
				actionTaken = true
			case "yank":
				err = yankWords(editor, buffer, count, true) // forward
				actionTaken = true
			case "change":
				err = changeWords(editor, buffer, count)
				actionTaken = true
			}
		case 'b': // yb = yank word backward, cb = change word backward, db = delete word backward
			switch op {
			case "delete":
				err = deleteWordsBackward(editor, buffer, count)
				actionTaken = true
			case "yank":
				err = yankWords(editor, buffer, count, false) // backward
				actionTaken = true
			case "change":
				err = changeWordsBackward(editor, buffer, count)
				actionTaken = true
			}
		case 'e': // de = delete to word end, ye = yank to word end, ce = change to word end
			switch op {
			case "delete":
				err = deleteWordToEnd(editor, buffer, count)
				actionTaken = true
			case "yank":
				err = yankWordToEnd(editor, buffer, count)
				actionTaken = true
			case "change":
				err = changeWords(editor, buffer, count) // ce and cw behave the same
				actionTaken = true
			}
		case '$': // d$ = delete to end of line, y$ = yank to end of line, c$ = change to end of line
			switch op {
			case "delete":
				var deletedContent string
				deletedContent, err = deleteToEndOfLine(editor, buffer)
				editor.DispatchSignal(DeleteSignal{content: deletedContent})
				actionTaken = true
			case "yank":
				err = yankToEndOfLine(editor, buffer)
				actionTaken = true
			case "change":
				err = changeToEndOfLine(editor, buffer)
				actionTaken = true
			}
		case 'G': // dG, yG, cG
			switch op {
			case "delete":
				count := buffer.LineCount() - cursor.Position.Row
				var deletedContent string
				deletedContent, err = deleteLines(editor, buffer, count)
				editor.DispatchSignal(DeleteSignal{content: deletedContent})
				actionTaken = true
			case "yank":
				count := buffer.LineCount() - cursor.Position.Row
				err = yankLines(editor, buffer, count)
				actionTaken = true
			case "change":
				count := buffer.LineCount() - cursor.Position.Row
				_, err = deleteLines(editor, buffer, count)
				if err == nil {
					editor.SetInsertMode()
				}
				actionTaken = true
			}

		default:
			// Invalid motion key after operator
			editor.DispatchError(ErrInvalidMotionId, fmt.Errorf("invalid motion after '%c'", firstKey.Rune))
			actionTaken = true         // Consumed the keys, even if invalid combo
			editor.ResetPendingCount() // Reset count if combo was invalid
		}

		if err != nil {
			return err // Return error from buffer operation
		}

		if actionTaken {
			editor.UpdateCommand("")
			return nil
		} // Sequence handled

		// If we fall through here, it means the second key wasn't a recognized motion
		// for the pending operator. We just discard the pending op.

		return nil
	}

	// --- Handle Numeric Input for Counts ---
	if key.Rune >= '1' && key.Rune <= '9' {
		digit := int(key.Rune - '0')
		if pendingCount == nil {
			// Start of count (unless it follows '0' motion)
			if m.pendingKey.Rune == '0' { // Check local pendingKey for '0' motion
				cursor.MoveToLineStart()
				buffer.SetCursor(cursor)                 // Update buffer cursor!
				m.pendingKey = KeyEvent{Key: KeyUnknown} // Clear pending '0' motion key
				// Start the count state with the current digit
				state.PendingCount = &digit
				editor.SetState(state) // Save state
				actionTaken = true     // '0' motion was taken
			} else {
				// Normal start of count
				state.PendingCount = &digit // Update state directly
				editor.SetState(state)      // Save state
			}
		} else {
			// Append to existing count
			newCount := (*pendingCount * 10) + digit
			state.PendingCount = &newCount // Update state
			editor.SetState(state)         // Save state
		}
		// Update command display regardless of actionTaken status for digits
		editor.UpdateCommand(fmt.Sprintf("%d", *state.PendingCount))
		return nil // Just consuming digits, wait for command

	} else if key.Rune == '0' && pendingCount == nil {
		// '0' is move-to-start-of-line command if it's the first digit pressed
		m.pendingKey = KeyEvent{Key: KeyUnknown} // Clear any other pending op (like 'd')
		editor.ResetPendingCount()               // Ensure no count is active (redundant but safe)
		cursor.MoveToLineStart()
		buffer.SetCursor(cursor) // Update buffer cursor!
		actionTaken = true
		// Don't return yet, let subsequent logic handle potential errors/updates
	} else if key.Rune == '0' && pendingCount != nil {
		// '0' as part of a multi-digit count
		digit := 0
		newCount := (*pendingCount * 10) + digit
		state.PendingCount = &newCount                               // Update state
		editor.SetState(state)                                       // Save state
		editor.UpdateCommand(fmt.Sprintf("%d", *state.PendingCount)) // Show count
		return nil                                                   // Consuming digit, wait for command
	}

	// --- Get Count or Default to 1 ---
	// This count applies to the command/motion executed *now*
	count := 1
	countWasPending := false
	if pendingCount != nil {
		count = *pendingCount
		countWasPending = true
		// Reset count state *after* using it for this command/motion
		// We do this reset *after* the switch statement below
	}

	// --- Handle Single-Key Commands or Start of Sequences ---
	col := cursor.Position.Col // Use Column from cursor
	var moveErr error

	switch {
	// Movement keys
	case key.Rune == 'h' || key.Key == KeyLeft:
		moveErr = cursor.MoveLeftOrUp(buffer, count, col)
	case key.Rune == 'j' || key.Key == KeyDown:
		moveErr = cursor.MoveDown(buffer, count, availableWidth)
	case key.Rune == 'k' || key.Key == KeyUp:
		moveErr = cursor.MoveUp(buffer, count, availableWidth)
	case key.Key == KeyCtrlD:
		moveErr = cursor.ScrollDown(buffer, state.ViewportHeight, availableWidth)
	case key.Key == KeyCtrlU:
		moveErr = cursor.ScrollUp(buffer, state.ViewportHeight, availableWidth)
	case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, count, col)
	case key.Rune == '{':
		moveErr = cursor.MoveBlockBackward(buffer, count)
	case key.Rune == '}':
		moveErr = cursor.MoveBlockForward(buffer, count)
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == '0':
		cursor.MoveToLineStart()
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth) // Move to last char
	case key.Rune == '^' || key.Key == KeyHome:
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
	case key.Rune == 'g':
		cursor.MoveToBufferStart() // Move to first line
	case key.Rune == 'G':
		cursor.MoveToBufferEnd(buffer, availableWidth) // Moves to start of last line
	case key.Key == KeyEnter: // Move down count lines to first non-blank
		if count == 0 {
			moveErr = cursor.MoveDown(buffer, count, availableWidth)
			if moveErr == nil {
				cursor.MoveToFirstNonBlank(buffer, availableWidth)
			}
		} else {
			cursor.Position.Row = count - 1
			buffer.SetCursor(cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}

	// Mode changes
	case key.Rune == 'i': // Insert before cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'I': // Insert at first non-blank
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
		buffer.SetCursor(cursor) // Update buffer's cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'a': // Insert after cursor
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveRight(buffer, 1, availableWidth) // Move one right (allows append at end of line)
		buffer.SetCursor(cursor)                    // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'A': // Insert at end of line
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Move *after* last char
		buffer.SetCursor(cursor)                          // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'o': // Open line below
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Go to end of current line
		buffer.SetCursor(cursor)
		buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune("\n")) // Insert newline
		cursor.MoveDown(buffer, 1, availableWidth)                                   // Move cursor down
		cursor.MoveToFirstNonBlank(buffer, availableWidth)                           // Go to start of new line
		buffer.SetCursor(cursor)
		editor.SaveHistory()
		editor.SetInsertMode()

	case key.Rune == 'O': // Open line above
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToLineStart()                                   // Go to start of current linavailableWidthe
		buffer.InsertRunesAt(cursor.Position.Row, 0, []rune("\n")) // Insert newline (pushes current line down)
		// Cursor stays on original line index, which is now the new blank line
		cursor.MoveToFirstNonBlank(buffer, availableWidth) // Ensure col=0 on the new line
		buffer.SetCursor(cursor)
		editor.SaveHistory()
		editor.SetInsertMode()

	case key.Rune == 'v': // Enter visual mode
		editor.SetVisualMode()

	case key.Rune == 'V': // Enter visual line mode
		editor.SetVisualLineMode()

	case key.Key == KeyEscape:
		// If pending count or op, clear them
		pendingCount = nil
		m.pendingKey = KeyEvent{Key: KeyUnknown}
		editor.UpdateCommand("") // Clear count display
		editor.SetNormalMode()

	case key.Rune == ':': // Enter command mode
		editor.SetCommandMode()

	case key.Rune == '/': // Enter search mode
		editor.SetSearchMode()

	case key.Rune == 'n': // Go to next search result
		cursor = editor.NextSearchResult()

	case key.Rune == 'N': // Go to previous search result
		cursor = editor.PreviousSearchResult()

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
		cursor = m.handleCharSearchRepeat(editor, buffer, false)

	case key.Rune == ',': // Repeat last character search in opposite direction
		cursor = m.handleCharSearchRepeat(editor, buffer, true)

	// Editing commands (single key or start of sequence)
	case key.Rune == 'x': // Delete character under cursor
		if !state.WithInsertMode {
			return nil
		}

		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen { // Only delete if cursor is on a char
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, count)
			if err == nil {
				editor.SaveHistory()
			}

			// Ensure cursor doesn't go past end of line after deletion
			cursor = buffer.GetCursor() // Refresh cursor state
			finalCol := cursor.Position.Col
			newLineLen := buffer.LineRuneCount(cursor.Position.Row)
			if finalCol >= newLineLen && newLineLen > 0 {
				cursor.Position.Col = newLineLen - 1
			} else if newLineLen == 0 {
				cursor.Position.Col = 0
			}
			buffer.SetCursor(cursor)
		}
	case key.Rune == 'X': // Delete character before cursor
		if !state.WithInsertMode {
			return nil
		}

		if cursor.Position.Col > 0 {
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col-1, count)
			if err == nil {
				cursor.MoveLeft(buffer, count, availableWidth) // Move cursor back
				buffer.SetCursor(cursor)
				editor.SaveHistory()
			}
		} else {
			err = &EditorError{
				id:  ErrStartOfLineId,
				err: ErrStartOfLine,
			}
		}

	case key.Rune == 'D': // Delete to end of line (equivalent to d$)
		if !state.WithInsertMode {
			return nil
		}

		var deletedContent string
		deletedContent, err = deleteToEndOfLine(editor, buffer)
		editor.DispatchSignal(DeleteSignal{content: deletedContent})

	case key.Rune == 'r': // Replace character under cursor
		if !state.WithInsertMode {
			return nil
		}

		m.waitingForReplace = true
		editor.UpdateCommand("r")
		return nil

	case key.Rune == 'C': // Change to end of line (equivalent to c$)
		if !state.WithInsertMode {
			return nil
		}

		err = changeToEndOfLine(editor, buffer)
	case key.Rune == 'd': // Start 'delete' operation
		if !state.WithInsertMode {
			return nil
		}

		m.pendingKey = key
		// Don't clear count yet
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))

		return nil // Wait for the next key (motion)

	case key.Rune == 'c': // Start 'change' operation
		if !state.WithInsertMode {
			return nil
		}

		m.pendingKey = key
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'y': // Start 'yank' operation
		m.pendingKey = key
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'p':
		if !state.WithInsertMode {
			return nil
		}

		content, pasteErr := editor.Paste()

		if strings.HasSuffix(content, "\n") {
			// Linewise paste: Paste() already inserted below and set cursor to (row+1, 0).
			// Repeat for any additional count (e.g. 3p pastes 3 lines).
			for i := 1; i < count; i++ {
				if _, loopErr := editor.Paste(); loopErr != nil {
					pasteErr = loopErr
					break
				}
			}
			// Cursor was set inside Paste(); refresh and skip the normal MoveRight.
			cursor = buffer.GetCursor()
			skipCursorUpdate = true
		} else {
			count = len(content)
			cursor.MoveRight(buffer, count, availableWidth)
		}

		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}

	case key.Rune == 'P':
		if !state.WithInsertMode {
			return nil
		}

		content, pasteErr := editor.PasteBefore()

		if strings.HasSuffix(content, "\n") {
			// Linewise paste above: PasteBefore() inserted above and cursor stays at the same row.
			// Repeat for any additional count (e.g. 3P pastes 3 lines above).
			for i := 1; i < count; i++ {
				if _, loopErr := editor.PasteBefore(); loopErr != nil {
					pasteErr = loopErr
					break
				}
			}
			cursor = buffer.GetCursor()
			skipCursorUpdate = true
		} else {
			count = len(content)
			cursor.MoveRight(buffer, count, availableWidth)
		}

		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}

	case key.Rune == 'u': // Undo
		if content, undoErr := editor.Undo(); undoErr != nil {
			err = &EditorError{
				id:  ErrUndoFailedId,
				err: undoErr,
			}
		} else {
			editor.DispatchSignal(UndoSignal{contentBefore: content})
		}
		skipCursorUpdate = true

	case key.Rune == 'U': // Redo
		if content, redoErr := editor.Redo(); redoErr != nil {
			err = &EditorError{
				id:  ErrRedoFailedId,
				err: redoErr,
			}
		} else {
			editor.DispatchSignal(RedoSignal{contentBefore: content})
		}
		skipCursorUpdate = true

	case key.Key == KeyBackspace: // Delete character before cursor
		moveErr = cursor.MoveLeft(buffer, count, availableWidth)

	default:
		// Unknown key - clear pending state if an unrecognized key is pressed
		m.pendingKey = KeyEvent{Key: KeyUnknown}
	}

	// Reset count after any command that consumed it (excludes early returns for
	// digit accumulation and multi-key sequences like d/y/c which return early)
	if countWasPending {
		editor.ResetPendingCount()
	}

	// Update cursor in buffer if no error or only boundary error
	// SKIP THIS IF WE JUST DID UNDO/REDO
	if !skipCursorUpdate && ((err == nil && moveErr == nil) ||
		errors.Is(moveErr, ErrEndOfBuffer) ||
		errors.Is(moveErr, ErrStartOfBuffer) ||
		errors.Is(moveErr, ErrEndOfLine) ||
		errors.Is(moveErr, ErrStartOfLine)) {
		buffer.SetCursor(cursor) // Update the buffer's cursor state
		// Don't return boundary errors as fatal errors to the editor loop
		if err != nil && !(errors.Is(moveErr, ErrEndOfBuffer) || errors.Is(moveErr, ErrStartOfBuffer) || errors.Is(moveErr, ErrEndOfLine) || errors.Is(moveErr, ErrStartOfLine)) {
			return err // Return actual errors (e.g., from delete/insert)
		}
		return nil
	}

	return err // Return other errors
}

// handleCharSearchRepeat handles repeating (;) or reversing (,) the last character search.
func (m *normalMode) handleCharSearchRepeat(editor Editor, buffer Buffer, reverse bool) Cursor {
	state := editor.GetState()
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
		editor.ResetPendingCount()
	}

	repeatCharSearch(&m.charSearch, editor, buffer, count, reverse)

	return buffer.GetCursor() // Return refreshed cursor
}

// clearPendingState resets all pending state in normal mode
func (m *normalMode) clearPendingState(editor Editor) {
	m.pendingKey = KeyEvent{Key: KeyUnknown}
	m.pendingModifier = 0
	m.charSearch = charSearchState{}
	m.waitingForReplace = false
	editor.ResetPendingCount()
}
