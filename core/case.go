package core

import "unicode"

// applyCaseInRange applies a rune transform to [start, end) (exclusive end),
// processing each line segment independently using delete + insert.
func applyCaseInRange(editor Editor, buffer Buffer, start, end Position, transform func(rune) rune) *EditorError {
	if start.Row > end.Row || (start.Row == end.Row && start.Col >= end.Col) {
		return nil
	}
	markChangeStart(editor, start)

	for row := start.Row; row <= end.Row; row++ {
		lineRunes := buffer.GetLineRunes(row)
		lineLen := len(lineRunes)

		fromCol := 0
		toCol := lineLen // exclusive end of line

		if row == start.Row {
			fromCol = start.Col
		}
		if row == end.Row {
			if end.Col < toCol {
				toCol = end.Col
			}
		}

		if fromCol >= toCol || fromCol >= lineLen {
			continue
		}

		count := toCol - fromCol
		newRunes := make([]rune, count)
		for i := range count {
			newRunes[i] = transform(lineRunes[fromCol+i])
		}

		if err := buffer.DeleteRunesAt(row, fromCol, count); err != nil {
			return err
		}
		if err := buffer.InsertRunesAt(row, fromCol, newRunes); err != nil {
			return &EditorError{id: ErrInvalidPositionId, err: err}
		}
	}
	return nil
}

// caseTransformForOp returns the rune transform for a case op rune:
// 'U' = uppercase, 'u' = lowercase, '~' = toggle.
func caseTransformForOp(op rune) func(rune) rune {
	switch op {
	case 'U':
		return unicode.ToUpper
	case 'u':
		return unicode.ToLower
	default: // '~'
		return func(r rune) rune {
			if lower := unicode.ToLower(r); lower != r {
				return lower
			}
			return unicode.ToUpper(r)
		}
	}
}

// toggleCaseChar toggles the case of count characters at the cursor, then advances
// the cursor past the toggled region. This is the Vim '~' behaviour.
func toggleCaseChar(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	row := cursor.Position.Row
	lineLen := buffer.LineRuneCount(row)

	if lineLen == 0 || cursor.Position.Col >= lineLen {
		return nil
	}

	endCol, _ := charsForward(buffer.GetLineRunes(row), cursor.Position.Col, count)

	start := Position{Row: row, Col: cursor.Position.Col}
	end := Position{Row: row, Col: endCol} // exclusive

	if err := applyCaseInRange(editor, buffer, start, end, caseTransformForOp('~')); err != nil {
		return err
	}

	// Advance cursor past the toggled region (clamp to last char).
	cursor.Position.Col = min(endCol, lastCol(buffer, row, false))
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseWord applies a case transform from the cursor to the start of the
// next `count` words (guw / gUw / g~w).
func applyCaseWord(editor Editor, buffer Buffer, op rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	tempCursor := cursor

	_ = tempCursor.moveWordForward(buffer, count, editor.CharClass, true)
	endPos := tempCursor.Position // exclusive end (start of next word, or the end of its line)

	if startPos == endPos {
		return nil
	}

	if err := applyCaseInRange(editor, buffer, startPos, endPos, caseTransformForOp(op)); err != nil {
		return err
	}
	return nil
}

// applyCaseWordEnd applies a case transform from the cursor to the end of the
// current/next word (gue / gUe / g~e).
func applyCaseWordEnd(editor Editor, buffer Buffer, op rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	tempCursor := cursor

	_ = tempCursor.MoveWordToEnd(buffer, count, editor.CharClass)
	// MoveWordToEnd lands on the last character (inclusive); move one right for exclusive end.
	_ = tempCursor.MoveRight(buffer, 1)
	endPos := tempCursor.Position // exclusive

	if startPos == endPos {
		return nil
	}

	if err := applyCaseInRange(editor, buffer, startPos, endPos, caseTransformForOp(op)); err != nil {
		return err
	}
	return nil
}

// applyCaseWordBackward applies a case transform from the start of the previous
// word to the cursor position (gub / gUb / g~b), moving the cursor to the start.
func applyCaseWordBackward(editor Editor, buffer Buffer, op rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	endPos := cursor.Position // exclusive end
	tempCursor := cursor

	_ = tempCursor.MoveWordBackward(buffer, count, editor.CharClass)
	startPos := tempCursor.Position

	if startPos == endPos {
		return nil
	}

	if err := applyCaseInRange(editor, buffer, startPos, endPos, caseTransformForOp(op)); err != nil {
		return err
	}

	// Cursor moves to the start of the affected range.
	cursor.Position = startPos
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToLines applies a case transform to `count` lines starting from the
// cursor row (guu / gUU / g~~).
func applyCaseToLines(editor Editor, buffer Buffer, op rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	startRow := cursor.Position.Row
	endRow := startRow + count - 1
	if endRow >= buffer.LineCount() {
		endRow = buffer.LineCount() - 1
	}

	transform := caseTransformForOp(op)
	for r := startRow; r <= endRow; r++ {
		lineLen := buffer.LineRuneCount(r)
		if lineLen == 0 {
			continue
		}
		start := Position{Row: r, Col: 0}
		end := Position{Row: r, Col: lineLen} // exclusive
		if err := applyCaseInRange(editor, buffer, start, end, transform); err != nil {
			return err
		}
	}

	return nil
}

// applyCaseToBufferEnd applies a case transform from the cursor to the end of
// the buffer (guG / gUG / g~G).
func applyCaseToBufferEnd(editor Editor, buffer Buffer, op rune) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	lastRow := buffer.LineCount() - 1
	endPos := Position{Row: lastRow, Col: buffer.LineRuneCount(lastRow)} // exclusive

	if err := applyCaseInRange(editor, buffer, startPos, endPos, caseTransformForOp(op)); err != nil {
		return err
	}
	return nil
}

// applyCaseToLineStart applies a case transform from the beginning of the current line to
// the cursor position (gu0 / gU0 / g~0), then moves the cursor to column 0.
func applyCaseToLineStart(editor Editor, buffer Buffer, op rune) *EditorError {
	cursor := buffer.GetCursor()
	row := cursor.Position.Row

	if cursor.Position.Col == 0 {
		return nil
	}

	start := Position{Row: row, Col: 0}
	end := Position{Row: row, Col: cursor.Position.Col} // exclusive

	if err := applyCaseInRange(editor, buffer, start, end, caseTransformForOp(op)); err != nil {
		return err
	}

	cursor.Position.Col = 0
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToFirstNonBlank applies a case transform between the cursor and the first
// non-blank character on the current line (gu^ / gU^ / g~^), then moves the cursor
// to that first non-blank position.
func applyCaseToFirstNonBlank(editor Editor, buffer Buffer, op rune) *EditorError {
	cursor := buffer.GetCursor()

	tempCursor := cursor
	tempCursor.MoveToFirstNonBlank(buffer)
	firstNonBlankCol := tempCursor.Position.Col

	curCol := cursor.Position.Col
	if curCol == firstNonBlankCol {
		return nil
	}

	startCol := min(curCol, firstNonBlankCol)
	endCol := max(curCol, firstNonBlankCol) // exclusive

	start := Position{Row: cursor.Position.Row, Col: startCol}
	end := Position{Row: cursor.Position.Row, Col: endCol}

	if err := applyCaseInRange(editor, buffer, start, end, caseTransformForOp(op)); err != nil {
		return err
	}

	cursor.Position.Col = firstNonBlankCol
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToBufferStart applies a case transform from the beginning of the buffer to
// the end of the cursor's current line (gugg / gUgg / g~gg), then moves the cursor to (0, 0).
func applyCaseToBufferStart(editor Editor, buffer Buffer, op rune) *EditorError {
	cursor := buffer.GetCursor()
	endRow := cursor.Position.Row
	endCol := buffer.LineRuneCount(endRow) // exclusive: full lines up to and including cursor row

	start := Position{Row: 0, Col: 0}
	end := Position{Row: endRow, Col: endCol}

	if err := applyCaseInRange(editor, buffer, start, end, caseTransformForOp(op)); err != nil {
		return err
	}

	cursor.Position = Position{Row: 0, Col: 0}
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToTargetLine applies a case transform to all lines between the cursor row and
// targetRow (inclusive), then moves the cursor to targetRow column 0. Used for gu{n}G.
func applyCaseToTargetLine(editor Editor, buffer Buffer, op rune, targetRow int) *EditorError {
	if targetRow < 0 {
		targetRow = 0
	}
	if targetRow >= buffer.LineCount() {
		targetRow = buffer.LineCount() - 1
	}

	cursorRow := buffer.GetCursor().Position.Row
	startRow := min(cursorRow, targetRow)
	endRow := max(cursorRow, targetRow)

	transform := caseTransformForOp(op)
	for r := startRow; r <= endRow; r++ {
		lineLen := buffer.LineRuneCount(r)
		if lineLen == 0 {
			continue
		}
		start := Position{Row: r, Col: 0}
		end := Position{Row: r, Col: lineLen}
		if err := applyCaseInRange(editor, buffer, start, end, transform); err != nil {
			return err
		}
	}

	cursor := buffer.GetCursor()
	cursor.Position = Position{Row: targetRow, Col: 0}
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToVisualSelection applies a case transform to a charwise visual selection
// [startPos, endPos] (inclusive on both ends, as used by visual mode).
func applyCaseToVisualSelection(editor Editor, buffer Buffer, op rune, startPos, endPos Position) *EditorError {
	startSel, endSel := NormalizeSelection(startPos, endPos)
	// Visual selection end is inclusive; convert to exclusive.
	exclusiveEnd := afterChar(buffer, endSel)

	if err := applyCaseInRange(editor, buffer, startSel, exclusiveEnd, caseTransformForOp(op)); err != nil {
		return err
	}
	return nil
}

// applyCaseToTextObject applies a case transform to the text object at the cursor.
// modifier is 'i' (inner) or 'a' (around); objectKey selects the object type
// (w, W, ", ', `, q, (, ), b, [, ], {, }, B, <, >). count extends word
// objects over additional words (gU2aw); other objects ignore it.
// The cursor is moved to the start of the affected range, matching Vim behaviour.
func applyCaseToTextObject(editor Editor, buffer Buffer, op, modifier, objectKey rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	var startPos, endPos Position // inclusive [start, end]
	found := false

	switch objectKey {
	case 'w', 'W':
		cls := editor.CharClass
		if objectKey == 'W' {
			cls = bigWordClass
		}
		return applyCaseToWordObject(editor, buffer, op, modifier == 'a', count, cls)

	case '"', '\'', '`':
		startCol, endCol, ok := quoteTextObjectRange(buffer, cursor.Position, modifier, objectKey)
		if !ok {
			return nil
		}
		startPos = Position{Row: cursor.Position.Row, Col: startCol}
		endPos = Position{Row: cursor.Position.Row, Col: endCol}
		found = true

	case 'q':
		startCol, endCol, ok := anyQuoteTextObjectRange(buffer, cursor.Position, modifier)
		if !ok {
			return nil
		}
		startPos = Position{Row: cursor.Position.Row, Col: startCol}
		endPos = Position{Row: cursor.Position.Row, Col: endCol}
		found = true

	case '(', ')', 'b', '[', ']', '{', '}', 'B', '<', '>':
		openPos, closePos, ok := caseBlockBounds(buffer, cursor.Position, objectKey)
		if !ok {
			return nil
		}
		if modifier == 'a' {
			startPos, endPos, found = openPos, closePos, true
			break
		}
		startPos, endPos, found = bracketInnerRange(buffer, openPos, closePos)
		if !found {
			// As d and c, an empty block changes nothing but moves the
			// cursor to it, just after the open bracket, and is the change
			// '.' repeats.
			cursor.Position, _ = innerBlock(buffer, openPos, closePos)
			buffer.SetCursor(cursor)
			if o, ok := editor.(emptyRangeOperator); ok {
				o.countChange()
			}
			return nil
		}
	}

	if !found {
		return nil
	}

	// endPos is inclusive; convert to exclusive for applyCaseInRange.
	exclusiveEnd := afterChar(buffer, endPos)
	if err := applyCaseInRange(editor, buffer, startPos, exclusiveEnd, caseTransformForOp(op)); err != nil {
		return err
	}

	// Move cursor to the start of the transformed range (matches Vim).
	cursor.Position = startPos
	buffer.SetCursor(cursor)
	return nil
}

// caseBlockBounds returns the bracket pair of the block object objectKey
// (one of ( ) b [ ] { } B < >) at pos; see findBlock. b is the nearest pair
// of any of ( [ {. Any other key finds no block.
func caseBlockBounds(buffer Buffer, pos Position, objectKey rune) (openPos, closePos Position, found bool) {
	var openChar, closeChar rune
	switch objectKey {
	case 'b':
		openPos, closePos, _, _, found = findNearestBracketBounds(buffer, pos)
		return openPos, closePos, found
	case '(', ')':
		openChar, closeChar = '(', ')'
	case '[', ']':
		openChar, closeChar = '[', ']'
	case '{', '}', 'B':
		openChar, closeChar = '{', '}'
	case '<', '>':
		openChar, closeChar = '<', '>'
	default:
		return Position{}, Position{}, false
	}
	openPos, closePos, _, found = findBlock(buffer, blockSearchStart(buffer, pos, openChar), openChar, closeChar)
	return openPos, closePos, found
}

// applyCaseToWordObject applies a case transform to the iw/aw text object at
// the cursor (iW/aW when cls is bigWordClass), count times, and moves
// the cursor to its start. As with the other operators, an object that runs
// out at the end of the buffer changes nothing, and the cursor stays where the
// search ended.
func applyCaseToWordObject(editor Editor, buffer Buffer, op rune, include bool, count int, cls func(rune) int) *EditorError {
	cursor := buffer.GetCursor()
	start, end, inclusive, ok := currentWord(buffer, cursor.Position, cursor.Position, false, count, include, cls)
	if !ok {
		cursor.Position = Position{Row: end.Row, Col: min(end.Col, lastCol(buffer, end.Row, false))}
		buffer.SetCursor(cursor)
		return nil
	}
	if inclusive {
		end = afterWordObject(buffer, end)
	} else if end.Col == 0 && end.Row > start.Row {
		// The end is column 0 of the next line: stop at the line break before it.
		end = Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}
	}
	if err := applyCaseInRange(editor, buffer, start, end, caseTransformForOp(op)); err != nil {
		return err
	}
	cursor.Position = start
	buffer.SetCursor(cursor)
	return nil
}

// applyCaseToVisualLineSelection applies a case transform to a visual line selection
// covering rows [startRow, endRow] (inclusive).
func applyCaseToVisualLineSelection(editor Editor, buffer Buffer, op rune, startRow, endRow int) *EditorError {
	if startRow > endRow {
		startRow, endRow = endRow, startRow
	}

	transform := caseTransformForOp(op)
	for r := startRow; r <= endRow; r++ {
		lineLen := buffer.LineRuneCount(r)
		if lineLen == 0 {
			continue
		}
		start := Position{Row: r, Col: 0}
		end := Position{Row: r, Col: lineLen} // exclusive
		if err := applyCaseInRange(editor, buffer, start, end, transform); err != nil {
			return err
		}
	}

	return nil
}
