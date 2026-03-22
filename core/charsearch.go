package core

import "fmt"

// charSearchState holds state for character search motions (f/F/t/T)
type charSearchState struct {
	lastChar       rune // The character being searched for
	searchType     rune // 'f', 'F', 't', or 'T'
	waitingForChar bool // True when waiting for character input after f/F/t/T
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
	state := editor.GetState()
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
		endCol = startPos.Col + 1 // include the cursor char (matching Vim dF/dT behaviour)
		if searchType == 'T' {
			// findCharOnLine for 'T' already returns col+1 (one after the found char);
			// no further adjustment needed.
			endCol-- // 'T' excludes the cursor char itself
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

// handleVisualCharSearchInput encapsulates the repeated waitingForChar block used
// Returns (true, err) if the event was handled, (false, nil) if not.
func handleVisualCharSearchInput(cs *charSearchState, editor Editor, buffer Buffer, key KeyEvent) (bool, *EditorError) {
	cs.waitingForChar = false
	editor.UpdateCommand("")

	if key.Key == KeyEscape {
		*cs = charSearchState{}
		editor.SetNormalMode()
		return true, nil
	}

	if key.Rune == 0 {
		*cs = charSearchState{}
		return true, nil
	}

	state := editor.GetState()
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
		editor.ResetPendingCount()
	}

	searchErr := performCharSearch(buffer, cs, cs.searchType, key.Rune, count)
	if searchErr != nil {
		*cs = charSearchState{}
		editor.DispatchError(ErrCharNotFoundId, searchErr)
	}

	return true, nil
}

// repeatCharSearch repeats (reverse=false) or reverses (reverse=true) the last
// character search stored in cs.
func repeatCharSearch(cs *charSearchState, editor Editor, buffer Buffer, count int, reverse bool) {
	if cs.searchType == 0 || cs.lastChar == 0 {
		return
	}

	searchType := cs.searchType
	if reverse {
		switch cs.searchType {
		case 'f':
			searchType = 'F'
		case 'F':
			searchType = 'f'
		case 't':
			searchType = 'T'
		case 'T':
			searchType = 't'
		}
	}

	originalType := cs.searchType
	if err := performCharSearch(buffer, cs, searchType, cs.lastChar, count); err != nil {
		editor.DispatchError(ErrCharNotFoundId, err)
	}
	if reverse {
		cs.searchType = originalType
	}
}
