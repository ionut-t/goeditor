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

// quoteTextObjectRange finds the quote pair surrounding or nearest to pos on the same line.
// It collects all positions of quote, pairs them up ([0,1], [2,3], …), and returns the first
// pair whose closing quote is at or after pos.Col. This matches Vim's behaviour: if the cursor
// is inside a pair or to the left of a pair, that pair is selected.
//
// modifier 'i' → columns of the content between the quotes (exclusive of the quote chars).
// modifier 'a' → columns of the opening and closing quote chars themselves.
func quoteTextObjectRange(buffer Buffer, pos Position, modifier rune, quote rune) (startCol, endCol int, found bool) {
	lineRunes := buffer.GetLineRunes(pos.Row)

	var positions []int
	for i, r := range lineRunes {
		if r == quote {
			positions = append(positions, i)
		}
	}

	if len(positions) < 2 {
		return 0, 0, false
	}

	col := pos.Col
	for i := 0; i+1 < len(positions); i += 2 {
		open := positions[i]
		close := positions[i+1]
		if close < col {
			continue // cursor is past this pair; try the next one
		}
		if modifier == 'i' {
			if close-open <= 1 {
				return 0, 0, false // empty quotes: nothing inside
			}
			return open + 1, close - 1, true
		}
		return open, close, true
	}
	return 0, 0, false
}

// anyQuoteTextObjectRange tries each standard quote character (", ', `) and returns the range
// for the innermost pair (smallest span) that contains or follows the cursor.
func anyQuoteTextObjectRange(buffer Buffer, pos Position, modifier rune) (startCol, endCol int, found bool) {
	bestSpan := -1
	bestStart, bestEnd := 0, 0

	for _, q := range []rune{'"', '\'', '`'} {
		s, e, ok := quoteTextObjectRange(buffer, pos, 'a', q)
		if !ok {
			continue
		}
		span := e - s
		if bestSpan == -1 || span < bestSpan {
			bestSpan = span
			bestStart, bestEnd = s, e
		}
	}

	if bestSpan == -1 {
		return 0, 0, false
	}

	if modifier == 'i' {
		if bestEnd-bestStart < 2 {
			return 0, 0, false
		}
		return bestStart + 1, bestEnd - 1, true
	}
	return bestStart, bestEnd, true
}

// findBracketBounds locates the innermost bracket pair (openChar/closeChar) enclosing pos,
// handling nesting and multi-line spans. Returns the positions of the open and close bracket chars.
//
// If the cursor sits on openChar it is used directly. If it sits on closeChar, the backward scan
// starts from col-1 with depth 0 so the matching open is found correctly. Otherwise a backward
// depth-counting scan finds the enclosing open, and a forward scan finds the close.
func findBracketBounds(buffer Buffer, pos Position, openChar, closeChar rune) (openPos, closePos Position, found bool) {
	lineRunes := buffer.GetLineRunes(pos.Row)
	col := pos.Col

	var op Position
	openFound := false

	if col < len(lineRunes) && lineRunes[col] == openChar {
		op = pos
		openFound = true
	} else {
		depth := 0
		startC := col
		// If cursor is ON the close bracket, scan backward from just before it (depth stays 0
		// so we find its direct match, not an enclosing one).
		if col < len(lineRunes) && lineRunes[col] == closeChar {
			startC = col - 1
		}

	findOpen:
		for r := pos.Row; r >= 0; r-- {
			line := buffer.GetLineRunes(r)
			endC := len(line) - 1
			if r == pos.Row {
				endC = startC
			}
			// Clamp to avoid OOB on empty lines or when startC < 0.
			if endC >= len(line) {
				endC = len(line) - 1
			}
			for c := endC; c >= 0; c-- {
				switch line[c] {
				case closeChar:
					depth++
				case openChar:
					if depth == 0 {
						op = Position{r, c}
						openFound = true
						break findOpen
					}
					depth--
				}
			}
		}
	}

	if !openFound {
		return Position{}, Position{}, false
	}

	// Scan forward from just past the opening bracket to find the matching close.
	depth := 0
	var cp Position
	closeFound := false

findClose:
	for r := op.Row; r < buffer.LineCount(); r++ {
		line := buffer.GetLineRunes(r)
		startC := 0
		if r == op.Row {
			startC = op.Col + 1
		}
		for c := startC; c < len(line); c++ {
			switch line[c] {
			case openChar:
				depth++
			case closeChar:
				if depth == 0 {
					cp = Position{r, c}
					closeFound = true
					break findClose
				}
				depth--
			}
		}
	}

	if !closeFound {
		return Position{}, Position{}, false
	}

	return op, cp, true
}

// bracketTextObjectRange wraps findBracketBounds and returns the text object range.
// It is used by visual mode, which only needs the inclusive [startPos, endPos] pair.
//
// modifier 'i' → content between the brackets (exclusive of the bracket chars).
// modifier 'a' → the bracket chars themselves.
func bracketTextObjectRange(buffer Buffer, pos Position, modifier rune, openChar, closeChar rune) (startPos, endPos Position, found bool) {
	openPos, closePos, ok := findBracketBounds(buffer, pos, openChar, closeChar)
	if !ok {
		return Position{}, Position{}, false
	}

	if modifier == 'a' {
		return openPos, closePos, true
	}

	// modifier == 'i': content between the brackets.
	innerStart := Position{openPos.Row, openPos.Col + 1}
	innerEnd := Position{closePos.Row, closePos.Col - 1}

	// Open bracket at end of line → content starts at the next line.
	if innerStart.Col >= buffer.LineRuneCount(openPos.Row) {
		if openPos.Row+1 >= buffer.LineCount() {
			return Position{}, Position{}, false
		}
		innerStart = Position{openPos.Row + 1, 0}
	}

	// Close bracket at start of line → content ends at end of previous line.
	if closePos.Col == 0 {
		if closePos.Row == 0 {
			return Position{}, Position{}, false
		}
		prevLen := buffer.LineRuneCount(closePos.Row - 1)
		innerEnd = Position{closePos.Row - 1, max(prevLen-1, 0)}
	}

	// Empty brackets: no content.
	if innerStart.Row > innerEnd.Row || (innerStart.Row == innerEnd.Row && innerStart.Col > innerEnd.Col) {
		return Position{}, Position{}, false
	}

	return innerStart, innerEnd, true
}

// applyCharRangeOp performs a yank, delete, or change on the inclusive character range
// [startPos, endPos]. It is the shared implementation used by quote text objects.
func applyCharRangeOp(editor Editor, buffer Buffer, startPos, endPos Position, op string) *EditorError {
	switch op {
	case "yank":
		state := editor.GetState()
		state.VisualStart = endPos
		state.YankSelection = SelectionCharacter
		editor.SetState(state)

		cursor := buffer.GetCursor()
		cursor.Position = startPos
		buffer.SetCursor(cursor)

		if err := editor.Copy(yankType); err != nil {
			state.VisualStart = Position{-1, -1}
			state.YankSelection = SelectionNone
			editor.SetState(state)
			return &EditorError{id: ErrFailedToYankId, err: err}
		}

	case "delete", "change":
		// deleteRange end is exclusive; convert inclusive endPos.
		exclusiveEnd := Position{endPos.Row, endPos.Col + 1}
		lineLen := buffer.LineRuneCount(endPos.Row)
		if exclusiveEnd.Col > lineLen {
			exclusiveEnd = Position{endPos.Row, lineLen}
		}

		if err := deleteRange(buffer, startPos, exclusiveEnd); err != nil {
			return err
		}
		editor.SaveHistory()

		cursor := buffer.GetCursor()
		cursor.Position = startPos
		lineCount := buffer.LineCount()
		if cursor.Position.Row >= lineCount {
			cursor.Position.Row = lineCount - 1
		}
		lineLen = buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col >= lineLen && lineLen > 0 {
			cursor.Position.Col = lineLen - 1
		}
		buffer.SetCursor(cursor)

		if op == "change" {
			editor.SetInsertMode()
		}
	}
	return nil
}

// applyQuoteOp applies op (yank/delete/change) to the i/a quote text object for the given quote char.
func applyQuoteOp(editor Editor, buffer Buffer, modifier rune, quote rune, op string) *EditorError {
	cursor := buffer.GetCursor()
	startCol, endCol, found := quoteTextObjectRange(buffer, cursor.Position, modifier, quote)
	if !found {
		return nil
	}
	return applyCharRangeOp(editor, buffer,
		Position{Row: cursor.Position.Row, Col: startCol},
		Position{Row: cursor.Position.Row, Col: endCol},
		op)
}

// applyAnyQuoteOp applies op to the innermost quote pair of any type (", ', `).
func applyAnyQuoteOp(editor Editor, buffer Buffer, modifier rune, op string) *EditorError {
	cursor := buffer.GetCursor()
	startCol, endCol, found := anyQuoteTextObjectRange(buffer, cursor.Position, modifier)
	if !found {
		return nil
	}
	return applyCharRangeOp(editor, buffer,
		Position{Row: cursor.Position.Row, Col: startCol},
		Position{Row: cursor.Position.Row, Col: endCol},
		op)
}

// applyBracketRange is the shared implementation for bracket text object operations.
// It computes the yank/delete/change range from already-found openPos/closePos.
func applyBracketRange(editor Editor, buffer Buffer, modifier rune, openPos, closePos Position, op string) *EditorError {
	if modifier == 'a' {
		exclusiveEnd := Position{closePos.Row, closePos.Col + 1}
		if exclusiveEnd.Col > buffer.LineRuneCount(closePos.Row) {
			exclusiveEnd = Position{closePos.Row, buffer.LineRuneCount(closePos.Row)}
		}
		return bracketApplyRange(editor, buffer, openPos, closePos, exclusiveEnd, op)
	}

	innerStart, innerEnd, found := bracketInnerRange(buffer, openPos, closePos)
	if !found {
		return nil
	}
	return bracketApplyRange(editor, buffer, innerStart, innerEnd, closePos, op)
}

// applyBracketOp applies op to the i/a bracket text object for the given open/close chars.
func applyBracketOp(editor Editor, buffer Buffer, modifier rune, openChar, closeChar rune, op string) *EditorError {
	cursor := buffer.GetCursor()
	openPos, closePos, found := findBracketBounds(buffer, cursor.Position, openChar, closeChar)
	if !found {
		return nil
	}
	return applyBracketRange(editor, buffer, modifier, openPos, closePos, op)
}

// bracketApplyRange performs the actual yank/delete/change for bracket text objects.
// yankEnd is the inclusive end used for the yank highlight; exclusiveDeleteEnd is the
// exclusive end passed to deleteRange for delete/change operations.
func bracketApplyRange(editor Editor, buffer Buffer, startPos, yankEnd, exclusiveDeleteEnd Position, op string) *EditorError {
	switch op {
	case "yank":
		state := editor.GetState()
		state.VisualStart = yankEnd
		state.YankSelection = SelectionCharacter
		editor.SetState(state)

		cursor := buffer.GetCursor()
		cursor.Position = startPos
		buffer.SetCursor(cursor)

		if err := editor.Copy(yankType); err != nil {
			state.VisualStart = Position{-1, -1}
			state.YankSelection = SelectionNone
			editor.SetState(state)
			return &EditorError{id: ErrFailedToYankId, err: err}
		}

	case "delete", "change":
		if err := deleteRange(buffer, startPos, exclusiveDeleteEnd); err != nil {
			return err
		}
		editor.SaveHistory()

		cursor := buffer.GetCursor()
		cursor.Position = startPos
		lineCount := buffer.LineCount()
		if cursor.Position.Row >= lineCount {
			cursor.Position.Row = lineCount - 1
		}
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col >= lineLen && lineLen > 0 {
			cursor.Position.Col = lineLen - 1
		}
		buffer.SetCursor(cursor)

		if op == "change" {
			editor.SetInsertMode()
		}
	}
	return nil
}

// bracketInnerRange computes the inclusive [innerStart, innerEnd] of the content between
// openPos and closePos. It is a shared helper for applyAnyBracketOp and visual mode.
func bracketInnerRange(buffer Buffer, openPos, closePos Position) (innerStart, innerEnd Position, found bool) {
	innerStart = Position{openPos.Row, openPos.Col + 1}
	innerEnd = Position{closePos.Row, closePos.Col - 1}

	if innerStart.Col >= buffer.LineRuneCount(openPos.Row) {
		if openPos.Row+1 >= buffer.LineCount() {
			return Position{}, Position{}, false
		}
		innerStart = Position{openPos.Row + 1, 0}
	}
	if closePos.Col == 0 {
		if closePos.Row == 0 {
			return Position{}, Position{}, false
		}
		prevLen := buffer.LineRuneCount(closePos.Row - 1)
		innerEnd = Position{closePos.Row - 1, max(prevLen-1, 0)}
	}
	if innerStart.Row > innerEnd.Row || (innerStart.Row == innerEnd.Row && innerStart.Col > innerEnd.Col) {
		return Position{}, Position{}, false
	}
	return innerStart, innerEnd, true
}

// findNearestBracketBounds finds the innermost bracket pair of any standard type — (), [], {} —
// that encloses pos. "Innermost" is the pair whose opening bracket appears closest to the cursor
// when scanning backward (i.e. largest position). Returns the open/close positions and the
// bracket characters used.
func findNearestBracketBounds(buffer Buffer, pos Position) (openPos, closePos Position, openChar, closeChar rune, found bool) {
	type candidate struct {
		openPos  Position
		closePos Position
		open     rune
		close    rune
	}

	var best *candidate
	for _, pair := range [][2]rune{{'(', ')'}, {'[', ']'}, {'{', '}'}} {
		op, cp, ok := findBracketBounds(buffer, pos, pair[0], pair[1])
		if !ok {
			continue
		}
		if best == nil ||
			op.Row > best.openPos.Row ||
			(op.Row == best.openPos.Row && op.Col > best.openPos.Col) {
			best = &candidate{op, cp, pair[0], pair[1]}
		}
	}

	if best == nil {
		return Position{}, Position{}, 0, 0, false
	}
	return best.openPos, best.closePos, best.open, best.close, true
}

// applyAnyBracketOp applies op to the innermost bracket pair of any type (()/[]/{}/)
// enclosing the cursor. Used by the 'b' text object alias.
func applyAnyBracketOp(editor Editor, buffer Buffer, modifier rune, op string) *EditorError {
	cursor := buffer.GetCursor()
	openPos, closePos, _, _, found := findNearestBracketBounds(buffer, cursor.Position)
	if !found {
		return nil
	}
	return applyBracketRange(editor, buffer, modifier, openPos, closePos, op)
}
