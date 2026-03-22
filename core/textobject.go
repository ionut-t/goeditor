package core

import "fmt"

// wordTextObjectRange handles text object yanks like 'yiw' (yank inside word) and 'yaw' (yank around word).
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
//
// wordTextObjectRange returns the start and end column (inclusive) for a word text object.
func wordTextObjectRange(buffer Buffer, pos Position, modifier rune, isWordChar func(rune) bool) (startCol int, endCol int, found bool) {
	lineRunes := buffer.GetLineRunes(pos.Row)
	if len(lineRunes) == 0 {
		return 0, 0, false
	}

	col := pos.Col
	if col >= len(lineRunes) {
		col = len(lineRunes) - 1
	}

	startCol = col
	endCol = col

	cursorChar := lineRunes[col]
	onWord := isWordChar(cursorChar)

	if onWord {
		// Case 1: Cursor is on a word character
		for startCol > 0 && isWordChar(lineRunes[startCol-1]) {
			startCol--
		}
		for endCol < len(lineRunes)-1 && isWordChar(lineRunes[endCol+1]) {
			endCol++
		}

		if modifier == 'a' {
			origEndCol := endCol
			for endCol < len(lineRunes)-1 && isWhiteSpace(lineRunes[endCol+1]) {
				endCol++
			}
			if endCol == origEndCol {
				for startCol > 0 && isWhiteSpace(lineRunes[startCol-1]) {
					startCol--
				}
			}
		}
	} else if isWhiteSpace(cursorChar) {
		// Case 2: Cursor is on whitespace
		for startCol > 0 && isWhiteSpace(lineRunes[startCol-1]) {
			startCol--
		}
		for endCol < len(lineRunes)-1 && isWhiteSpace(lineRunes[endCol+1]) {
			endCol++
		}

		if modifier == 'a' {
			if endCol < len(lineRunes)-1 && isWordChar(lineRunes[endCol+1]) {
				for endCol < len(lineRunes)-1 && isWordChar(lineRunes[endCol+1]) {
					endCol++
				}
			} else if startCol > 0 && isWordChar(lineRunes[startCol-1]) {
				for startCol > 0 && isWordChar(lineRunes[startCol-1]) {
					startCol--
				}
			}
		}
	} else {
		// Case 3: Cursor is on punctuation or other non-word, non-whitespace character
		if modifier == 'a' {
			for startCol > 0 && isWhiteSpace(lineRunes[startCol-1]) {
				startCol--
			}
			for endCol < len(lineRunes)-1 && isWhiteSpace(lineRunes[endCol+1]) {
				endCol++
			}
		}
	}

	return startCol, endCol, true
}

func yankTextObject(editor Editor, buffer Buffer, modifier rune, textObject rune) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()

	if textObject != 'w' {
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: fmt.Errorf("unsupported text object: %c", textObject),
		}
	}

	startCol, endCol, found := wordTextObjectRange(buffer, cursor.Position, modifier, editor.IsWordChar)
	if !found {
		return nil
	}

	// Set up character-wise selection for yank highlight
	state.VisualStart = Position{Row: cursor.Position.Row, Col: endCol}
	state.YankSelection = SelectionCharacter
	editor.SetState(state)

	cursor.Position.Col = startCol
	buffer.SetCursor(cursor)

	if err := editor.Copy(yankType); err != nil {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{id: ErrFailedToYankId, err: err}
	}

	return nil
}

func deleteTextObject(editor Editor, buffer Buffer, modifier rune, textObject rune) *EditorError {
	cursor := buffer.GetCursor()

	if textObject != 'w' {
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: fmt.Errorf("unsupported text object: %c", textObject),
		}
	}

	startCol, endCol, found := wordTextObjectRange(buffer, cursor.Position, modifier, editor.IsWordChar)
	if !found {
		return nil
	}

	startPos := Position{Row: cursor.Position.Row, Col: startCol}
	endPos := Position{Row: cursor.Position.Row, Col: endCol + 1} // deleteRange is exclusive

	err := deleteRange(buffer, startPos, endPos)
	if err == nil {
		editor.SaveHistory()
		cursor.Position = startPos
		buffer.SetCursor(cursor)
	}

	return err
}

func changeTextObject(editor Editor, buffer Buffer, modifier rune, textObject rune) *EditorError {
	cursor := buffer.GetCursor()

	if textObject != 'w' {
		return &EditorError{
			id:  ErrInvalidMotionId,
			err: fmt.Errorf("unsupported text object: %c", textObject),
		}
	}

	startCol, endCol, found := wordTextObjectRange(buffer, cursor.Position, modifier, editor.IsWordChar)
	if !found {
		return nil
	}

	startPos := Position{Row: cursor.Position.Row, Col: startCol}
	endPos := Position{Row: cursor.Position.Row, Col: endCol + 1} // deleteRange is exclusive

	err := deleteRange(buffer, startPos, endPos)
	if err == nil {
		editor.SaveHistory()
		cursor.Position = startPos
		buffer.SetCursor(cursor)
		editor.SetInsertMode()
	}

	return err
}

// paragraphRows returns the inclusive [startRow, endRow] of the paragraph block under pos.
//
// Cursor on a non-blank line:
//   - 'i': the contiguous block of non-blank lines.
//   - 'a': same block plus trailing blank lines (or leading ones when no trailing exist).
//
// Cursor on a blank line:
//   - 'i': the contiguous run of blank lines.
//   - 'a': blank lines plus the adjacent paragraph below (or above if none below).
func paragraphRows(buffer Buffer, pos Position, modifier rune) (startRow, endRow int, found bool) {
	lineCount := buffer.LineCount()

	if len(buffer.GetLineRunes(pos.Row)) == 0 {
		// Cursor is on a blank line: find the contiguous blank-line block.
		startRow = pos.Row
		for startRow > 0 && len(buffer.GetLineRunes(startRow-1)) == 0 {
			startRow--
		}
		endRow = pos.Row
		for endRow < lineCount-1 && len(buffer.GetLineRunes(endRow+1)) == 0 {
			endRow++
		}

		if modifier == 'a' {
			// Prefer extending into the paragraph below.
			if endRow < lineCount-1 {
				endRow++ // first line of the next paragraph
				for endRow < lineCount-1 && len(buffer.GetLineRunes(endRow+1)) > 0 {
					endRow++
				}
			} else if startRow > 0 {
				// No paragraph below; extend into the paragraph above.
				startRow-- // last line of the previous paragraph
				for startRow > 0 && len(buffer.GetLineRunes(startRow-1)) > 0 {
					startRow--
				}
			}
		}

		return startRow, endRow, true
	}

	// Scan upward to find the first line of the contiguous non-blank block.
	startRow = pos.Row
	for startRow > 0 && len(buffer.GetLineRunes(startRow-1)) > 0 {
		startRow--
	}

	// Scan downward to find the last line of the block.
	endRow = pos.Row
	for endRow < lineCount-1 && len(buffer.GetLineRunes(endRow+1)) > 0 {
		endRow++
	}

	if modifier == 'a' {
		// Prefer trailing blank lines.
		newEnd := endRow
		for newEnd < lineCount-1 && len(buffer.GetLineRunes(newEnd+1)) == 0 {
			newEnd++
		}
		if newEnd > endRow {
			endRow = newEnd
		} else if startRow > 0 {
			// No trailing blanks; absorb leading blank lines instead.
			for startRow > 0 && len(buffer.GetLineRunes(startRow-1)) == 0 {
				startRow--
			}
		}
	}

	return startRow, endRow, true
}

// paragraphDeleteRange translates an inclusive [startRow, endRow] paragraph range into
// the (start, end) Position pair expected by deleteRange (end is exclusive).
// It ensures the surrounding newline is included so the rows are fully removed.
func paragraphDeleteRange(buffer Buffer, startRow, endRow int) (start, end Position) {
	lineCount := buffer.LineCount()

	if endRow < lineCount-1 {
		// There is a line below: delete up to the start of the next line (removes trailing newline).
		return Position{Row: startRow, Col: 0}, Position{Row: endRow + 1, Col: 0}
	}
	if startRow > 0 {
		// Last line(s): delete backward from end of the preceding line (removes leading newline).
		return Position{Row: startRow - 1, Col: buffer.LineRuneCount(startRow - 1)},
			Position{Row: endRow, Col: buffer.LineRuneCount(endRow)}
	}
	// Only content in the buffer.
	return Position{Row: 0, Col: 0}, Position{Row: endRow, Col: buffer.LineRuneCount(endRow)}
}

func yankParagraphTextObject(editor Editor, buffer Buffer, modifier rune) *EditorError {
	cursor := buffer.GetCursor()
	state := editor.GetState()

	startRow, endRow, found := paragraphRows(buffer, cursor.Position, modifier)
	if !found {
		return nil
	}

	lastCol := buffer.LineRuneCount(endRow)
	if lastCol > 0 {
		lastCol-- // make inclusive for VisualStart / cursor position used by Copy
	}

	state.VisualStart = Position{Row: startRow, Col: 0}
	state.YankSelection = SelectionLine
	editor.SetState(state)

	cursor.Position = Position{Row: endRow, Col: lastCol}
	buffer.SetCursor(cursor)

	if err := editor.Copy(yankType); err != nil {
		state.VisualStart = Position{-1, -1}
		state.YankSelection = SelectionNone
		editor.SetState(state)
		return &EditorError{id: ErrFailedToYankId, err: err}
	}

	return nil
}

func deleteParagraphTextObject(editor Editor, buffer Buffer, modifier rune) *EditorError {
	cursor := buffer.GetCursor()

	startRow, endRow, found := paragraphRows(buffer, cursor.Position, modifier)
	if !found {
		return nil
	}

	start, end := paragraphDeleteRange(buffer, startRow, endRow)

	if err := deleteRange(buffer, start, end); err != nil {
		return err
	}

	editor.SaveHistory()

	newRow := startRow
	if newRow >= buffer.LineCount() {
		newRow = buffer.LineCount() - 1
	}
	cursor.Position = Position{Row: newRow, Col: 0}
	buffer.SetCursor(cursor)

	return nil
}

func changeParagraphTextObject(editor Editor, buffer Buffer, modifier rune) *EditorError {
	cursor := buffer.GetCursor()

	startRow, endRow, found := paragraphRows(buffer, cursor.Position, modifier)
	if !found {
		return nil
	}

	if modifier == 'i' {
		// cip: clear startRow content, then delete rows startRow+1..endRow (bottom-up).
		// This keeps exactly one empty line at startRow. For blank-line cursors
		// (startRow == endRow) the loop is a no-op, preserving the existing blank line.

		// Clear startRow content (no-op if already blank).
		if lineLen := buffer.LineRuneCount(startRow); lineLen > 0 {
			if err := buffer.DeleteRunesAt(startRow, 0, lineLen); err != nil {
				return err
			}
		}

		// Delete rows startRow+1..endRow from bottom to top.
		for r := endRow; r > startRow; r-- {
			lineLen := buffer.LineRuneCount(r)
			if r == buffer.LineCount()-1 {
				// Last line in the buffer: clear its content then remove it by
				// deleting the newline at the end of the preceding row.
				if lineLen > 0 {
					if err := buffer.DeleteRunesAt(r, 0, lineLen); err != nil {
						return err
					}
				}
				prevLen := buffer.LineRuneCount(r - 1)
				if err := buffer.DeleteRunesAt(r-1, prevLen, 1); err != nil {
					return err
				}
			} else {
				// Non-last line: delete content + its newline to remove the row.
				if err := buffer.DeleteRunesAt(r, 0, lineLen+1); err != nil {
					return err
				}
			}
		}

		cursor.Position = Position{Row: startRow, Col: 0}
		buffer.SetCursor(cursor)
		editor.SaveHistory()
		editor.SetInsertMode()
		return nil
	}

	// modifier == 'a': cap
	deleteStart, deleteEnd := paragraphDeleteRange(buffer, startRow, endRow)

	if err := deleteRange(buffer, deleteStart, deleteEnd); err != nil {
		return err
	}

	// cap: dap removed the paragraph + surrounding blank lines.
	// Re-open exactly one blank line at the original paragraph position so the
	// user has a clean line to type the replacement content — matching Vim's behaviour.
	if deleteStart.Col > 0 {
		// Deletion started mid-line (preceding content exists on deleteStart.Row).
		// Append a newline after that line; cursor lands on the new blank line below it.
		lineEnd := buffer.LineRuneCount(deleteStart.Row)
		if err := buffer.InsertRunesAt(deleteStart.Row, lineEnd, []rune("\n")); err != nil {
			return &EditorError{id: ErrInvalidMotionId, err: err}
		}
		cursor.Position = Position{Row: deleteStart.Row + 1, Col: 0}
	} else {
		// Deletion started at col 0. If the row now holds other content (the next
		// paragraph), push it down with a blank line; cursor stays on the new blank row.
		if deleteStart.Row < buffer.LineCount() && len(buffer.GetLineRunes(deleteStart.Row)) > 0 {
			if err := buffer.InsertRunesAt(deleteStart.Row, 0, []rune("\n")); err != nil {
				return &EditorError{id: ErrInvalidMotionId, err: err}
			}
		}
		cursor.Position = Position{Row: deleteStart.Row, Col: 0}
	}

	buffer.SetCursor(cursor)
	editor.SaveHistory()
	editor.SetInsertMode()
	return nil
}
