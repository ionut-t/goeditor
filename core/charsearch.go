package core

import (
	"errors"
	"fmt"
)

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
func performCharSearch(buffer Buffer, cs *charSearchState, char rune, count int) error {
	cursor := buffer.GetCursor()
	lineRunes := buffer.GetLineRunes(cursor.Position.Row)

	newCol := findCharOnLine(lineRunes, cursor.Position.Col, char, cs.searchType, count)

	if newCol == -1 {
		return fmt.Errorf("character '%c' not found", char)
	}

	cursor.Position.Col = newCol
	buffer.SetCursor(cursor)
	cs.lastChar = char

	return nil
}

// handleCharSearchOperator handles operator + character search motion combinations
// like df, (delete until comma), yt; (yank till semicolon), etc.
// skipAdjacent starts a t/T search one character further along, so a repeated
// till-search (d;) doesn't re-match the character the cursor already sits next to.
func handleCharSearchOperator(editor Editor, buffer Buffer, op string, searchType rune, char rune, count int, skipAdjacent bool) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	lineRunes := buffer.GetLineRunes(cursor.Position.Row)

	searchCol := startPos.Col
	if skipAdjacent {
		switch searchType {
		case 't':
			searchCol++
		case 'T':
			searchCol--
		}
	}

	// Find the target position
	targetCol := findCharOnLine(lineRunes, searchCol, char, searchType, count)

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

	return applyOperatorToRange(editor, buffer, op,
		Position{Row: startPos.Row, Col: startCol},
		Position{Row: startPos.Row, Col: endCol})
}

// repeatCharSearchOperator implements d;/y;/c; and d,/y,/c, — apply an operator
// over a repeat of the last f/F/t/T search (reversed for ',').
func repeatCharSearchOperator(cs *charSearchState, editor Editor, buffer Buffer, op string, count int, reverse bool) *EditorError {
	if cs.searchType == 0 || cs.lastChar == 0 {
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: errors.New("no previous character search to repeat"),
		}
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

	return handleCharSearchOperator(editor, buffer, op, searchType, cs.lastChar, count, true)
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

	if err := performCharSearch(buffer, cs, key.Rune, count); err != nil {
		*cs = charSearchState{}
		editor.DispatchError(ErrCharNotFoundId, err)
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
	cs.searchType = searchType

	// For t/T the cursor sits one col before/after the matched char. Nudge it
	// one step further so findCharOnLine skips that char and finds the next one.
	original := buffer.GetCursor()
	cursor := original
	switch searchType {
	case 't':
		cursor.Position.Col++
		buffer.SetCursor(cursor)
	case 'T':
		cursor.Position.Col--
		buffer.SetCursor(cursor)
	}

	if err := performCharSearch(buffer, cs, cs.lastChar, count); err != nil {
		buffer.SetCursor(original) // undo the nudge if the search found nothing
		editor.DispatchError(ErrCharNotFoundId, err)
	}
	cs.searchType = originalType
}
