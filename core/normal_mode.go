package core

import (
	"errors"
	"fmt"
)

type normalMode struct {
	pendingKey      KeyEvent        // Stores the first key of a multi-key command (e.g., 'd' in 'dd')
	pendingModifier rune            // Stores text object modifier ('i' for inside, 'a' for around)
	charSearch      charSearchState // Character search state (f/F/t/T)
}

// charSearchState holds state for character search motions (f/F/t/T)
type charSearchState struct {
	lastChar       rune // The character being searched for
	searchType     rune // 'f', 'F', 't', or 'T'
	waitingForChar bool // True when waiting for character input after f/F/t/T
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
}

func (m *normalMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	var err *EditorError
	actionTaken := false // Track if the key (or sequence) resulted in an action
	state := editor.GetState()
	pendingCount := state.PendingCount
	availableWidth := state.AvailableWidth
	skipCursorUpdate := false

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
				if op == "yank" {
					err = yankTextObject(editor, buffer, modifier, 'w')
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
		case 'w': // dw = delete word, yw = yank word forward
			switch op {
			case "delete":
				err = deleteWords(editor, buffer, count)
				actionTaken = true
			case "yank":
				err = yankWords(editor, buffer, count, true) // forward
				actionTaken = true
			}
		case 'b': // yb = yank word backward
			if op == "yank" {
				err = yankWords(editor, buffer, count, false) // backward
				actionTaken = true
			}
		case '$': // d$ = delete to end of line, y$ = yank to end of line
			switch op {
			case "delete":
				var deletedContent string
				deletedContent, err = deleteToEndOfLine(editor, buffer)
				editor.DispatchSignal(DeleteSignal{content: deletedContent})
				actionTaken = true
			case "yank":
				err = yankToEndOfLine(editor, buffer)
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
				cursor := buffer.GetCursor()
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
		cursor := buffer.GetCursor()
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
	cursor := buffer.GetCursor() // Get cursor for operations
	col := cursor.Position.Col   // Use Column from cursor
	var moveErr error

	switch {
	// Movement keys
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

	case key.Rune == 'C': // Change to end of line (equivalent to c$)
		if !state.WithInsertMode {
			return nil
		}

		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen {
			delCount := lineLen - cursor.Position.Col
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, delCount)
			if err == nil {
				editor.SaveHistory()
			}
		}
		if err == nil {
			buffer.SetCursor(cursor) // Ensure cursor is at deletion point
			editor.SetInsertMode()   // Enter insert mode
		}
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
		count = len(content)

		cursor.MoveRight(buffer, count, availableWidth)

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
		// Unknown key - potentially beep or show message
		// Clear pending state if an unrecognized key is pressed
		m.pendingKey = KeyEvent{Key: KeyUnknown}
		// Keep count until non-digit command pressed
		// editor.SetMessage(fmt.Sprintf("Unknown key: %s", key.String()))

		// Reset count if it was pending and used by a command/motion executed above
		// (Excludes cases where we returned early, like starting d/y/c or parsing digits)
		if countWasPending {
			editor.ResetPendingCount()
		}
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

func deleteLines(editor Editor, buffer Buffer, count int) (string, *EditorError) {
	cursor := buffer.GetCursor()
	startLine := cursor.Position.Row
	endLine := startLine + count - 1
	availableWidth := editor.GetState().AvailableWidth

	if endLine >= buffer.LineCount() {
		endLine = buffer.LineCount() - 1
	}

	var deletedContent string
	var err *EditorError

	// Delete lines from bottom up to keep indices valid
	for i := endLine; i >= startLine; i-- {
		lineRunes := buffer.GetLineRunes(i)

		if buffer.LineCount() > 1 { // Don't delete the very last line, clear it instead
			deletedContent = string(lineRunes) + "\n" + deletedContent
			// Delete line content + newline (represented by moving to next line)
			err = buffer.DeleteRunesAt(i, 0, len(lineRunes)+1) // +1 deletes newline
		} else {
			// Clear the last line
			deletedContent = string(lineRunes) + deletedContent
			err = buffer.DeleteRunesAt(i, 0, len(lineRunes))
		}
		if err != nil {
			return "", err
		}
	}
	// Adjust cursor after deletion
	cursor = buffer.GetCursor()          // Get current cursor state
	if startLine >= buffer.LineCount() { // If deletion removed lines up to the end
		cursor.Position.Row = max(buffer.LineCount()-1, 0) // Handle empty buffer
	} else {
		cursor.Position.Row = startLine // Move to first deleted line index
	}
	// Move cursor to first non-blank of the line it landed on
	buffer.SetCursor(cursor) // Set row first
	c := buffer.GetCursor()  // Get it back to use methods
	c.MoveToFirstNonBlank(buffer, availableWidth)
	buffer.SetCursor(c) // Final position

	if err == nil {
		editor.SaveHistory()
	}

	return deletedContent, err
}

func deleteWords(editor Editor, buffer Buffer, count int) (err *EditorError) {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	// Simulate word motion to find end position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	errMove := tempCursor.MoveWordForward(buffer, count, availableWidth)
	endPos := tempCursor.Position

	// Calculate deletion range (careful with multi-line)
	// Simple version: delete from startPos.Col to endPos.Col if on same line
	if startPos.Row == endPos.Row && errMove == nil {
		deleteCount := endPos.Col - startPos.Col
		if deleteCount > 0 {
			err = buffer.DeleteRunesAt(startPos.Row, startPos.Col, deleteCount)
			if err == nil {
				editor.SaveHistory()
			}
		}
	} else {
		// TODO: Implement multi-line dw/cw/yw
	}

	return err
}

// isWordCharForTextObject checks if a rune is a word character for text objects.
// Matches Vim's default 'iskeyword' which includes letters, numbers, and underscore.
func isWordCharForTextObject(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// isWhiteSpaceForTextObject checks if a rune is whitespace.
// For Vim text objects, spaces and tabs are considered whitespace.
func isWhiteSpaceForTextObject(r rune) bool {
	return r == ' ' || r == '\t'
}

func deleteToEndOfLine(editor Editor, buffer Buffer) (string, *EditorError) {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)
	var deletedContent string
	var err *EditorError

	if cursor.Position.Col < lineLen { // Only delete if not already at/past end
		lineRunes := buffer.GetLineRunes(cursor.Position.Row)
		deletedContent = string(lineRunes[cursor.Position.Col:])

		deleteCount := lineLen - cursor.Position.Col
		err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, deleteCount)
		if err == nil {
			editor.SaveHistory()
		}
	}

	return deletedContent, err
}

func yankLines(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	originalPos := cursor.Position // Save original cursor position
	startLine := cursor.Position.Row
	endLine := startLine + count - 1

	if endLine >= buffer.LineCount() {
		endLine = buffer.LineCount() - 1
	}

	// Set up line-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state := editor.GetState()
	state.VisualStart = Position{Row: endLine, Col: max(buffer.LineRuneCount(endLine)-1, 0)}
	state.YankSelection = SelectionLine // Mark as line-wise selection
	editor.SetState(state)

	// Restore cursor to original position
	// This way cursor stays where the user had it when they pressed yy
	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight

	return nil
}

func yankWords(editor Editor, buffer Buffer, count int, forward bool) *EditorError {
	cursor := buffer.GetCursor()
	originalPos := cursor.Position
	availableWidth := editor.GetState().AvailableWidth

	// Calculate end position by moving cursor
	tempCursor := cursor
	var moveErr error
	if forward {
		moveErr = tempCursor.MoveWordForward(buffer, count, availableWidth)
	} else {
		moveErr = tempCursor.MoveWordBackward(buffer, count, availableWidth)
	}

	endPos := tempCursor.Position

	// Set up character-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state := editor.GetState()
	state.VisualStart = endPos
	state.YankSelection = SelectionCharacter // Mark as character-wise selection
	editor.SetState(state)

	// Restore cursor to original position
	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight
	// Handle movement errors non-fatally
	if moveErr != nil && moveErr != ErrEndOfBuffer && moveErr != ErrStartOfBuffer {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: moveErr,
		}
	}

	return nil
}

func yankToEndOfLine(editor Editor, buffer Buffer) *EditorError {
	cursor := buffer.GetCursor()
	originalPos := cursor.Position
	lineLen := buffer.LineRuneCount(cursor.Position.Row)

	if cursor.Position.Col >= lineLen {
		// Already at or past end of line, nothing to yank
		return nil
	}

	// Set up character-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state := editor.GetState()
	state.VisualStart = Position{Row: cursor.Position.Row, Col: lineLen - 1}
	state.YankSelection = SelectionCharacter // Mark as character-wise selection
	editor.SetState(state)

	// Restore cursor to original position
	cursor.Position = originalPos
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight

	return nil
}

// yankTextObject handles text object yanks like 'yiw' (yank inside word) and 'yaw' (yank around word).
//
// Text objects in Vim have two forms:
// - 'i' (inner): Selects just the object itself (e.g., 'iw' = just the word)
// - 'a' (around): Selects the object plus surrounding whitespace (e.g., 'aw' = word + space)
//
// Vim's text object behavior depends on what character the cursor is on:
// 1. On a word character:
//   - 'iw': Selects the entire word under cursor
//   - 'aw': Selects the word plus trailing whitespace (or leading if no trailing)
//
// 2. On whitespace:
//   - 'iw': Selects the whitespace itself
//   - 'aw': Selects the whitespace plus adjacent word (prioritizes trailing word)
//
// 3. On punctuation/other:
//   - 'iw': Selects just the character
//   - 'aw': Selects the character plus surrounding whitespace
func yankTextObject(editor Editor, buffer Buffer, modifier rune, textObject rune) *EditorError {
	cursor := buffer.GetCursor()
	originalPos := cursor.Position

	if textObject != 'w' {
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: fmt.Errorf("unsupported text object: %c", textObject),
		}
	}

	// Get the current line
	lineRunes := buffer.GetLineRunes(cursor.Position.Row)
	if len(lineRunes) == 0 {
		return nil // Nothing to yank on empty line
	}

	col := cursor.Position.Col
	if col >= len(lineRunes) {
		col = len(lineRunes) - 1
	}

	// Determine the character under the cursor and its type to apply Vim's text object rules
	startCol := col
	endCol := col

	cursorChar := lineRunes[col]
	onWord := isWordCharForTextObject(cursorChar)

	if onWord {
		// Case 1: Cursor is on a word character
		// 'iw' (inner word): Selects the word under the cursor
		// 'aw' (around word): Selects the word under the cursor and one surrounding whitespace

		// Find the start of the current word
		for startCol > 0 && isWordCharForTextObject(lineRunes[startCol-1]) {
			startCol--
		}

		// Find the end of the current word
		for endCol < len(lineRunes)-1 && isWordCharForTextObject(lineRunes[endCol+1]) {
			endCol++
		}

		// For 'aw', include trailing whitespace (or leading if no trailing)
		// This matches Vim's behavior where 'aw' prioritizes trailing, then leading whitespace
		// Examples:
		//   "word  " with cursor on 'w' -> "word  " (includes trailing spaces)
		//   "  word" with cursor on 'w' -> "  word" (includes leading spaces when no trailing)
		//   "word" with cursor on 'w' -> "word" (no surrounding whitespace to include)
		if modifier == 'a' {
			origEndCol := endCol
			// Try to include trailing whitespace
			for endCol < len(lineRunes)-1 && isWhiteSpaceForTextObject(lineRunes[endCol+1]) {
				endCol++
			}

			// If no trailing whitespace was found, look for leading whitespace
			if endCol == origEndCol {
				for startCol > 0 && isWhiteSpaceForTextObject(lineRunes[startCol-1]) {
					startCol--
				}
			}
		}
	} else if isWhiteSpaceForTextObject(cursorChar) {
		// Case 2: Cursor is on whitespace
		// 'iw' on whitespace: Selects the whitespace block itself
		// 'aw' on whitespace: Selects the whitespace block plus an adjacent word (prioritizes trailing)

		// Find the extent of the current whitespace block
		for startCol > 0 && isWhiteSpaceForTextObject(lineRunes[startCol-1]) {
			startCol--
		}
		for endCol < len(lineRunes)-1 && isWhiteSpaceForTextObject(lineRunes[endCol+1]) {
			endCol++
		}

		// For 'aw', extend to include an adjacent word
		// Examples:
		//   "word1   word2" with cursor on middle space -> "   word2" (trailing word)
		//   "word  " with cursor on trailing space -> "word  " (leading word when no trailing)
		if modifier == 'a' {
			// Try to include trailing word first
			if endCol < len(lineRunes)-1 && isWordCharForTextObject(lineRunes[endCol+1]) {
				for endCol < len(lineRunes)-1 && isWordCharForTextObject(lineRunes[endCol+1]) {
					endCol++
				}
			} else if startCol > 0 && isWordCharForTextObject(lineRunes[startCol-1]) {
				// If no trailing word, include leading word
				for startCol > 0 && isWordCharForTextObject(lineRunes[startCol-1]) {
					startCol--
				}
			}
		}
	} else {
		// Case 3: Cursor is on punctuation or other non-word, non-whitespace character
		// 'iw': Selects just the character itself
		// 'aw': Selects the character plus any surrounding whitespace
		if modifier == 'a' {
			// Expand to include leading whitespace
			for startCol > 0 && isWhiteSpaceForTextObject(lineRunes[startCol-1]) {
				startCol--
			}
			// Expand to include trailing whitespace
			for endCol < len(lineRunes)-1 && isWhiteSpaceForTextObject(lineRunes[endCol+1]) {
				endCol++
			}
		}
	}

	// Set up character-wise selection for yank highlight (stay in normal mode)
	// Do this atomically in one SetState to avoid flicker
	state := editor.GetState()
	state.VisualStart = Position{Row: cursor.Position.Row, Col: endCol}
	state.YankSelection = SelectionCharacter // Mark as character-wise selection
	editor.SetState(state)

	// Set cursor to start of selection
	cursor.Position = Position{Row: originalPos.Row, Col: startCol}
	buffer.SetCursor(cursor)

	// Copy the selection (this also dispatches the YankSignal)
	if err := editor.Copy(yankType); err != nil {
		// On error, clear the selection
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{
			id:  ErrFailedToYankId,
			err: err,
		}
	}

	// Keep the visual selection active for the yank highlight

	return nil
}

// findCharOnLine searches for a character on the current line.
// searchType: 'f' (find forward), 'F' (find backward), 't' (till forward), 'T' (till backward)
// Returns the column position if found, -1 if not found.
func findCharOnLine(lineRunes []rune, startCol int, char rune, searchType rune, count int) int {
	if count <= 0 {
		count = 1
	}

	occurrencesFound := 0

	switch searchType {
	case 'f': // Find forward
		for col := startCol + 1; col < len(lineRunes); col++ {
			if lineRunes[col] == char {
				occurrencesFound++
				if occurrencesFound == count {
					return col
				}
			}
		}

	case 'F': // Find backward
		for col := startCol - 1; col >= 0; col-- {
			if lineRunes[col] == char {
				occurrencesFound++
				if occurrencesFound == count {
					return col
				}
			}
		}

	case 't': // Till forward (one before the character)
		for col := startCol + 1; col < len(lineRunes); col++ {
			if lineRunes[col] == char {
				occurrencesFound++
				if occurrencesFound == count {
					if col > 0 {
						return col - 1
					}
					return -1
				}
			}
		}

	case 'T': // Till backward (one after the character)
		for col := startCol - 1; col >= 0; col-- {
			if lineRunes[col] == char {
				occurrencesFound++
				if occurrencesFound == count {
					if col < len(lineRunes)-1 {
						return col + 1
					}
					return -1
				}
			}
		}
	}

	return -1 // Not found
}

// performCharSearch executes a character search and moves the cursor.
// This is the shared implementation used by normal, visual, and visual line modes.
// Returns error if character not found.
func performCharSearch(buffer Buffer, cs *charSearchState, searchType rune, char rune, count int) error {
	cursor := buffer.GetCursor()
	lineRunes := buffer.GetLineRunes(cursor.Position.Row)

	newCol := findCharOnLine(lineRunes, cursor.Position.Col, char, searchType, count)

	if newCol == -1 {
		return fmt.Errorf("character '%c' not found", char)
	}

	// Update cursor position
	cursor.Position.Col = newCol
	buffer.SetCursor(cursor)

	// Save search state for repeat with ; and ,
	cs.lastChar = char
	cs.searchType = searchType

	return nil
}

// handleCharSearchOperator handles operator + character search motion combinations
// like df, (delete until comma), yt; (yank till semicolon), etc.
func handleCharSearchOperator(editor Editor, buffer Buffer, op string, searchType rune, char rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	lineRunes := buffer.GetLineRunes(cursor.Position.Row)

	// Find the target position
	targetCol := findCharOnLine(lineRunes, cursor.Position.Col, char, searchType, count)

	if targetCol == -1 {
		// Character not found
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: fmt.Errorf("character '%c' not found", char),
		}
	}

	// For 'f' and 't', we need to include the character under cursor up to (and possibly including) target
	// For 'F' and 'T', we go backwards
	var startCol, endCol int

	switch searchType {
	case 'f', 't': // Forward search
		startCol = startPos.Col
		endCol = targetCol
		if searchType == 'f' {
			endCol++ // Include the found character for 'f'
		} else {
			endCol++ // For 't', we stopped one before, so include up to that position
		}
	case 'F', 'T': // Backward search
		startCol = targetCol
		endCol = startPos.Col
		if searchType == 'F' {
			// For 'F', targetCol is the found character, start from there
		} else {
			// For 'T', we stopped one after the character, so adjust
			startCol++
		}
	}

	// Ensure we don't go out of bounds
	if startCol < 0 {
		startCol = 0
	}
	if endCol > len(lineRunes) {
		endCol = len(lineRunes)
	}

	deleteCount := endCol - startCol

	switch op {
	case "delete":
		if deleteCount > 0 {
			err := buffer.DeleteRunesAt(startPos.Row, startCol, deleteCount)
			if err != nil {
				return err
			}
			editor.SaveHistory()

			// Update cursor position after delete
			cursor.Position.Col = startCol
			buffer.SetCursor(cursor)
		}

	case "yank":
		if deleteCount > 0 {
			// Set up visual selection for yank
			state := editor.GetState()
			state.VisualStart = Position{Row: startPos.Row, Col: endCol - 1}
			state.YankSelection = SelectionCharacter
			editor.SetState(state)

			// Move cursor to start of selection
			cursor.Position.Col = startCol
			buffer.SetCursor(cursor)

			// Perform yank
			if err := editor.Copy(yankType); err != nil {
				state.VisualStart = Position{-1, -1}
				state.YankSelection = SelectionNone
				editor.SetState(state)
				return &EditorError{
					id:  ErrFailedToYankId,
					err: err,
				}
			}
		}

	case "change":
		if deleteCount > 0 {
			err := buffer.DeleteRunesAt(startPos.Row, startCol, deleteCount)
			if err != nil {
				return err
			}
			editor.SaveHistory()

			// Update cursor position and enter insert mode
			cursor.Position.Col = startCol
			buffer.SetCursor(cursor)
			editor.SetInsertMode()
		}
	}

	return nil
}

// resetCharSearch clears the character search state
func (m *normalMode) resetCharSearch() {
	m.charSearch = charSearchState{}
}

// handleCharSearchRepeat handles repeating (;) or reversing (,) the last character search.
func (m *normalMode) handleCharSearchRepeat(editor Editor, buffer Buffer, reverse bool) Cursor {
	cursor := buffer.GetCursor()

	if m.charSearch.searchType == 0 || m.charSearch.lastChar == 0 {
		return cursor // No previous search to repeat
	}

	state := editor.GetState()
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
		editor.ResetPendingCount()
	}

	effectiveSearchType := m.charSearch.searchType
	if reverse {
		// Reverse the search direction
		switch m.charSearch.searchType {
		case 'f':
			effectiveSearchType = 'F'
		case 'F':
			effectiveSearchType = 'f'
		case 't':
			effectiveSearchType = 'T'
		case 'T':
			effectiveSearchType = 't'
		}
	}

	// Save original type to restore after reverse search
	originalType := m.charSearch.searchType

	searchErr := performCharSearch(buffer, &m.charSearch, effectiveSearchType, m.charSearch.lastChar, count)

	// Restore original type if reversed search was temporary
	if reverse {
		m.charSearch.searchType = originalType
	}

	if searchErr != nil {
		editor.DispatchError(ErrCharNotFoundId, searchErr)
	}

	return buffer.GetCursor() // Return refreshed cursor
}

// clearPendingState resets all pending state in normal mode
func (m *normalMode) clearPendingState(editor Editor) {
	m.pendingKey = KeyEvent{Key: KeyUnknown}
	m.pendingModifier = 0
	m.charSearch = charSearchState{}
	editor.ResetPendingCount()
}
