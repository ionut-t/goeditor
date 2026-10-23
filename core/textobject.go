package core

// wordWalker steps through the buffer as Vim's inc() and dec() do, for the
// word text objects. The line break of each line is a position of its own
// (column = line length), and reads as a blank, so a word ends at the end of
// its line and an empty line is one blank.
type wordWalker struct {
	buffer     Buffer
	pos        Position
	line       []rune
	isWordChar func(rune) bool
}

func newWordWalker(buffer Buffer, pos Position, isWordChar func(rune) bool) *wordWalker {
	w := &wordWalker{buffer: buffer, isWordChar: isWordChar}
	w.moveTo(pos)
	return w
}

func (w *wordWalker) moveTo(pos Position) {
	w.pos = pos
	w.line = w.buffer.GetLineRunes(pos.Row)
	w.pos.Col = min(pos.Col, len(w.line))
}

// cls is Vim's cls(): the class of the character under the walker.
func (w *wordWalker) cls() int {
	if w.pos.Col >= len(w.line) {
		return blankClass
	}
	return wordClass(w.line[w.pos.Col], w.isWordChar)
}

func (w *wordWalker) onEmptyLine() bool { return len(w.line) == 0 }

// inc moves one position forward: it returns 0 within the line, 2 onto the
// line break, 1 onto the next line and -1 at the end of the buffer.
func (w *wordWalker) inc() int {
	if w.pos.Col < len(w.line) {
		w.pos.Col++
		if w.pos.Col < len(w.line) {
			return 0
		}
		return 2
	}
	if w.pos.Row < w.buffer.LineCount()-1 {
		w.moveTo(Position{Row: w.pos.Row + 1})
		return 1
	}
	return -1
}

// dec moves one position back: it returns 0 within the line, 1 onto the line
// break of the line above and -1 at the start of the buffer.
func (w *wordWalker) dec() int {
	if w.pos.Col > 0 {
		w.pos.Col--
		return 0
	}
	if w.pos.Row > 0 {
		w.moveTo(Position{Row: w.pos.Row - 1, Col: maxCol})
		return 1
	}
	return -1
}

// incl is inc that skips the line break of a non-empty line.
func (w *wordWalker) incl() int {
	r := w.inc()
	if r >= 1 && w.pos.Col > 0 {
		r = w.inc()
	}
	return r
}

// decl is dec that skips the line break of a non-empty line.
func (w *wordWalker) decl() int {
	r := w.dec()
	if r == 1 && w.pos.Col > 0 {
		r = w.dec()
	}
	return r
}

// oneLeft moves one character left on the line; false in column 0.
func (w *wordWalker) oneLeft() bool {
	if w.pos.Col == 0 {
		return false
	}
	w.pos.Col--
	return true
}

// backInLine goes back to the start of the run under the walker, on its line.
func (w *wordWalker) backInLine() {
	class := w.cls()
	for w.pos.Col > 0 {
		w.dec()
		if w.cls() != class {
			w.inc()
			return
		}
	}
}

// skipRun moves over the run of class, forward or back; false at the end of
// the buffer.
func (w *wordWalker) skipRun(class int, forward bool) bool {
	for w.cls() == class {
		if forward && w.inc() == -1 || !forward && w.dec() == -1 {
			return false
		}
	}
	return true
}

// fwdWord is Vim's fwd_word(1, bigword, TRUE): to the start of the next word,
// but not past the end of the line. It fails on the end of the last line.
func (w *wordWalker) fwdWord() bool {
	class := w.cls()
	lastLine := w.pos.Row == w.buffer.LineCount()-1
	i := w.inc()
	if i == -1 || i >= 1 && lastLine {
		return false
	}
	if i >= 1 {
		return true
	}
	if class != blankClass {
		for w.cls() == class {
			if i := w.inc(); i == -1 || i >= 1 {
				return true
			}
		}
	}
	for w.cls() == blankClass {
		if w.pos.Col == 0 && w.onEmptyLine() {
			break
		}
		if i := w.inc(); i == -1 || i >= 1 {
			return true
		}
	}
	return true
}

// endWord is Vim's end_word(1, bigword, TRUE, TRUE): to the end of the word,
// or of the next one from its last character or from blanks, stopping on an
// empty line.
func (w *wordWalker) endWord() bool {
	class := w.cls()
	if w.inc() == -1 {
		return false
	}
	if w.cls() == class && class != blankClass {
		if !w.skipRun(class, true) {
			return false
		}
	} else if class == blankClass {
		for w.cls() == blankClass {
			if w.pos.Col == 0 && w.onEmptyLine() {
				return true
			}
			if w.inc() == -1 {
				return false
			}
		}
		if !w.skipRun(w.cls(), true) {
			return false
		}
	}
	w.dec()
	return true
}

// bckWord is Vim's bck_word(1, bigword, TRUE): to the start of the word, or
// of the one before from its first character or from blanks.
func (w *wordWalker) bckWord() bool {
	class := w.cls()
	if w.dec() == -1 {
		return false
	}
	if class == w.cls() || class == blankClass {
		for w.cls() == blankClass {
			if w.pos.Col == 0 && w.onEmptyLine() {
				return true
			}
			if w.dec() == -1 {
				return true
			}
		}
		if !w.skipRun(w.cls(), false) {
			return true
		}
	}
	w.inc()
	return true
}

// bckendWord is Vim's bckend_word(1, bigword, TRUE): back to the end of the
// word before, but not onto the line above.
func (w *wordWalker) bckendWord() bool {
	class := w.cls()
	i := w.dec()
	if i == -1 {
		return false
	}
	if i == 1 {
		return true
	}
	if class != blankClass {
		for w.cls() == class {
			if i := w.dec(); i == -1 || i == 1 {
				return true
			}
		}
	}
	for w.cls() == blankClass {
		if w.pos.Col == 0 && w.onEmptyLine() {
			break
		}
		if i := w.dec(); i == -1 || i == 1 {
			return true
		}
	}
	return true
}

// currentWord is Vim's current_word: the iw/aw text object at cursor (iW/aW
// when isWordChar is isBigWordChar), count times, with include for aw. A word
// is a run of one class (blanks, word characters, other characters; see
// wordClass), so a WORD is a run of non-blanks, and a line break is a blank.
//
//   - i: the run under the cursor; a count adds the runs after it.
//   - a on blanks: the blanks and the word after them.
//   - a on a word: the word and the blanks after it; when none follow, the
//     blanks before it, unless they are the indent.
//
// Objects after the first can cross lines; an empty line is a run of its own.
// end is the last position, inclusive, or with inclusive false, the position
// after the object (column 0 of the next line). The end can be a line break
// (column = line length). In visual mode (visual true), a selection of more
// than one character is extended from the cursor, away from visualStart.
//
// On failure (a count that runs out at the end of the buffer), end is where
// Vim leaves the cursor, and in visual mode, start where it leaves the start.
func currentWord(buffer Buffer, cursor, visualStart Position, visual bool, count int, include bool, isWordChar func(rune) bool) (start, end Position, inclusive, ok bool) {
	w := newWordWalker(buffer, cursor, isWordChar)
	start = visualStart
	inclusive = true
	includeWhite := false
	count = max(count, 1)

	if !visual || cursor == visualStart {
		w.backInLine()
		start = w.pos
		if (w.cls() == blankClass) == include {
			if !w.endWord() {
				return visualStart, w.pos, inclusive, false
			}
		} else {
			// The blanks before the next word, or the word before the blanks.
			switch {
			case !w.fwdWord() && w.pos == start && w.onEmptyLine():
				// An empty last line. Vim's decl reaches into the line above
				// (Gdiw on abc\n deletes c); this keeps the empty range.
			case w.pos.Col == 0:
				w.decl()
			default:
				w.oneLeft()
			}
			includeWhite = include
		}
		count--
	}

	for ; count > 0; count-- {
		inclusive = true
		if visual && (w.pos.Row < start.Row || w.pos.Row == start.Row && w.pos.Col < start.Col) {
			// Selecting backwards: extend the start of the selection.
			if w.decl() == -1 {
				return start, w.pos, inclusive, false
			}
			if include != (w.cls() != blankClass) {
				if !w.bckWord() {
					return start, w.pos, inclusive, false
				}
			} else {
				if !w.bckendWord() {
					return start, w.pos, inclusive, false
				}
				w.incl()
			}
			continue
		}
		if w.incl() == -1 {
			return start, w.pos, inclusive, false
		}
		if include != (w.cls() == blankClass) {
			if !w.fwdWord() && count > 1 {
				return start, w.pos, inclusive, false
			}
			// An end just past a line break stays before it.
			if !w.oneLeft() {
				inclusive = false
			}
		} else if !w.endWord() {
			return start, w.pos, inclusive, false
		}
	}

	// With no blanks after the word, aw takes the blanks before it instead,
	// unless they are the indent.
	if includeWhite && (w.cls() != blankClass || w.pos.Col == 0 && !inclusive) {
		b := newWordWalker(buffer, start, isWordChar)
		if b.oneLeft() {
			b.backInLine()
			if b.cls() == blankClass && b.pos.Col > 0 {
				start = b.pos
			}
		}
	}
	return start, w.pos, inclusive, true
}

// applyWordTextObject applies a delete/yank/change operator to the iw/aw text
// object under the cursor (diw, y2aw, caw, ...), or iW/aW when isWordChar is
// isBigWordChar. When the object runs out at the end of the buffer (d5aw on
// four words, daw on an empty last line), nothing changes, and as in Vim the
// cursor stays where the search ended.
func applyWordTextObject(editor Editor, buffer Buffer, op string, modifier rune, count int, isWordChar func(rune) bool) *EditorError {
	cursor := buffer.GetCursor()
	start, end, inclusive, ok := currentWord(buffer, cursor.Position, cursor.Position, false, count, modifier == 'a', isWordChar)
	if !ok {
		cursor.Position = Position{Row: end.Row, Col: min(end.Col, lastCol(buffer, end.Row, false))}
		buffer.SetCursor(cursor)
		return nil
	}
	if !inclusive {
		return applyOperatorToMotion(editor, buffer, op, start, end)
	}
	return applyOperatorToRange(editor, buffer, op, start, afterWordObject(buffer, end))
}

// afterWordObject returns the position after end, the inclusive end of a word
// object. When end is a line break, it isn't included: Vim's operators stop
// before it.
func afterWordObject(buffer Buffer, end Position) Position {
	if end.Col < buffer.LineRuneCount(end.Row) {
		end.Col++
	}
	return end
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
// findMatchingBracket implements the '%' motion: find the first bracket on the
// current line at or after pos and return the position of its matching pair.
func findMatchingBracket(buffer Buffer, pos Position) (Position, bool) {
	lineRunes := buffer.GetLineRunes(pos.Row)
	pairs := [][2]rune{{'(', ')'}, {'[', ']'}, {'{', '}'}}

	for col := pos.Col; col < len(lineRunes); col++ {
		for _, pair := range pairs {
			openChar, closeChar := pair[0], pair[1]
			if lineRunes[col] != openChar && lineRunes[col] != closeChar {
				continue
			}
			openPos, closePos, found := findBracketBounds(buffer, Position{Row: pos.Row, Col: col}, openChar, closeChar)
			if !found {
				return Position{}, false
			}
			if lineRunes[col] == openChar {
				return closePos, true
			}
			return openPos, true
		}
	}

	return Position{}, false
}

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
	openPos, closePos, ok := findBracketBounds(buffer, blockSearchStart(buffer, pos, openChar), openChar, closeChar)
	if !ok {
		return Position{}, Position{}, false
	}

	if modifier == 'a' {
		return openPos, closePos, true
	}
	return bracketInnerRange(buffer, openPos, closePos)
}

// blockSearchStart returns where to look for the block around pos. As Vim,
// for a { block a cursor in the indent first moves past it, so that on the
// indent before a { the block is the one it opens.
func blockSearchStart(buffer Buffer, pos Position, openChar rune) Position {
	if openChar == '{' {
		pos.Col = max(pos.Col, firstNonBlankCol(buffer.GetLineRunes(pos.Row)))
	}
	return pos
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

// applyOperatorToEmptyQuotes applies op to i" (or i', i`) when the quote
// pair found by a" (openCol, closeCol) has nothing between it, and does
// nothing otherwise. As Vim, the operator still runs, on the empty range at
// the closing quote.
func applyOperatorToEmptyQuotes(editor Editor, buffer Buffer, modifier rune, openCol, closeCol int, op string) *EditorError {
	if modifier != 'i' || closeCol != openCol+1 {
		return nil
	}
	empty := Position{Row: buffer.GetCursor().Position.Row, Col: closeCol}
	return applyOperatorToRange(editor, buffer, op, empty, empty)
}

// applyQuoteOp applies op (yank/delete/change) to the i/a quote text object for the given quote char.
func applyQuoteOp(editor Editor, buffer Buffer, modifier rune, quote rune, op string) *EditorError {
	cursor := buffer.GetCursor()
	startCol, endCol, found := quoteTextObjectRange(buffer, cursor.Position, modifier, quote)
	if !found {
		if openCol, closeCol, ok := quoteTextObjectRange(buffer, cursor.Position, 'a', quote); ok {
			return applyOperatorToEmptyQuotes(editor, buffer, modifier, openCol, closeCol, op)
		}
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
		if openCol, closeCol, ok := anyQuoteTextObjectRange(buffer, cursor.Position, 'a'); ok {
			return applyOperatorToEmptyQuotes(editor, buffer, modifier, openCol, closeCol, op)
		}
		return nil
	}
	return applyCharRangeOp(editor, buffer,
		Position{Row: cursor.Position.Row, Col: startCol},
		Position{Row: cursor.Position.Row, Col: endCol},
		op)
}

// applyBracketRange is the shared implementation for bracket text object operations.
// It applies op to the i or a block of the already-found openPos/closePos.
func applyBracketRange(editor Editor, buffer Buffer, modifier rune, openPos, closePos Position, op string) *EditorError {
	if modifier == 'a' {
		return applyOperatorToRange(editor, buffer, op, openPos, Position{Row: closePos.Row, Col: closePos.Col + 1})
	}
	// An empty block (nothing between the brackets, or only blanks before a
	// close bracket that starts its line) still runs the operator, on the
	// empty range just after the open bracket. A block that ends at the
	// start of a line is linewise when it starts in the indent.
	start, end := innerBlock(buffer, openPos, closePos)
	return applyOperatorToMotion(editor, buffer, op, start, end)
}

// applyBracketOp applies op to the i/a bracket text object for the given open/close chars.
func applyBracketOp(editor Editor, buffer Buffer, modifier rune, openChar, closeChar rune, op string) *EditorError {
	pos := blockSearchStart(buffer, buffer.GetCursor().Position, openChar)
	openPos, closePos, found := findBracketBounds(buffer, pos, openChar, closeChar)
	if !found {
		return nil
	}
	return applyBracketRange(editor, buffer, modifier, openPos, closePos, op)
}

// afterOpenBracket returns the position just after the open bracket at
// openPos, where the inner text object starts: the next character, or the
// start of the next line when the bracket ends its line. It reports false
// when the bracket ends the last line.
func afterOpenBracket(buffer Buffer, openPos Position) (Position, bool) {
	if openPos.Col+1 < buffer.LineRuneCount(openPos.Row) {
		return Position{Row: openPos.Row, Col: openPos.Col + 1}, true
	}
	if openPos.Row+1 >= buffer.LineCount() {
		return Position{}, false
	}
	return Position{Row: openPos.Row + 1, Col: 0}, true
}

// bracketInnerRange returns the inner block of the brackets at openPos and
// closePos (see innerBlock) as the inclusive range visual mode selects: a
// block that ends at the start of a line ends on the line break before it.
// It reports false for an empty block. It is a shared helper for visual
// mode and the case operators.
func bracketInnerRange(buffer Buffer, openPos, closePos Position) (innerStart, innerEnd Position, found bool) {
	start, end := innerBlock(buffer, openPos, closePos)
	if start == end {
		return Position{}, Position{}, false
	}
	if end.Col == 0 {
		return start, Position{Row: end.Row - 1, Col: buffer.LineRuneCount(end.Row - 1)}, true
	}
	return start, Position{Row: end.Row, Col: end.Col - 1}, true
}

// innerBlock returns the inner block of the brackets at openPos and
// closePos as [start, end), empty when end == start. As Vim's
// current_block, when only blanks come before the close bracket on its
// line, neither they nor the lines of only blanks above it are part of the
// block: it then ends at the start of the line after its last character
// (or after an empty line), so end.Col is 0. Otherwise end is closePos.
func innerBlock(buffer Buffer, openPos, closePos Position) (start, end Position) {
	// A close bracket follows the open one, so there is always a position after it.
	start, _ = afterOpenBracket(buffer, openPos)
	if closePos.Col > firstNonBlankCol(buffer.GetLineRunes(closePos.Row)) {
		return start, closePos
	}
	// The open bracket's line isn't blank, so this stops there at the latest.
	row := closePos.Row - 1
	for buffer.LineRuneCount(row) > 0 && onlyBlanksFrom(buffer, Position{Row: row}) {
		row--
	}
	return start, Position{Row: row + 1, Col: 0}
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
