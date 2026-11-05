package core

import "slices"

// wordWalker steps through the buffer as Vim's inc() and dec() do, for the
// word text objects: a character (with its composing characters) at a time.
// The line break of each line is a position of its own (column = line
// length), and reads as a blank, so a word ends at the end of its line and an
// empty line is one blank.
type wordWalker struct {
	buffer    Buffer
	pos       Position
	line      []rune
	charClass func(rune) int
}

// newWordWalker returns the walker by value, so that it stays on the stack.
func newWordWalker(buffer Buffer, pos Position, cls func(rune) int) wordWalker {
	w := wordWalker{buffer: buffer, charClass: cls}
	w.moveTo(pos)
	return w
}

func (w *wordWalker) moveTo(pos Position) {
	w.pos = pos
	w.line = w.buffer.GetLineRunes(pos.Row)
	w.pos.Col = charStart(w.line, min(pos.Col, len(w.line)))
}

// cls is Vim's cls(): the class of the character under the walker.
func (w *wordWalker) cls() int {
	if w.pos.Col >= len(w.line) {
		return blankClass
	}
	return w.charClass(w.line[w.pos.Col])
}

func (w *wordWalker) onEmptyLine() bool { return len(w.line) == 0 }

// inc moves one position forward: it returns 0 within the line, 2 onto the
// line break, 1 onto the next line and -1 at the end of the buffer.
func (w *wordWalker) inc() int {
	if w.pos.Col < len(w.line) {
		w.pos.Col = charEnd(w.line, w.pos.Col)
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
		w.pos.Col = charStart(w.line, w.pos.Col-1)
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
	w.pos.Col = charStart(w.line, w.pos.Col-1)
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
// when cls is bigWordClass), count times, with include for aw. A word
// is a run of one class of cls (blanks, punctuation, word characters, emoji
// or a script; see utfClass), so a WORD is a run of non-blanks, and a line
// break is a blank.
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
func currentWord(buffer Buffer, cursor, visualStart Position, visual bool, count int, include bool, cls func(rune) int) (start, end Position, inclusive, ok bool) {
	w := newWordWalker(buffer, cursor, cls)
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
		b := newWordWalker(buffer, start, cls)
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
// object under the cursor (diw, y2aw, caw, ...), or iW/aW when cls is
// bigWordClass. When the object runs out at the end of the buffer (d5aw on
// four words, daw on an empty last line), nothing changes, and as in Vim the
// cursor stays where the search ended.
func applyWordTextObject(editor Editor, buffer Buffer, op operator, modifier rune, count int, cls func(rune) int) *EditorError {
	cursor := buffer.GetCursor()
	start, end, inclusive, ok := currentWord(buffer, cursor.Position, cursor.Position, false, count, modifier == 'a', cls)
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
		return afterChar(buffer, end)
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

// quotePair finds the quote pair surrounding or nearest to pos on the same line.
// It collects all positions of quote, pairs them up ([0,1], [2,3], …), and returns the first
// pair whose closing quote is at or after pos.Col. This matches Vim's behaviour: if the cursor
// is inside a pair or to the left of a pair, that pair is selected.
func quotePair(buffer Buffer, pos Position, quote rune) (openCol, closeCol int, found bool) {
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
		return open, close, true
	}
	return 0, 0, false
}

// anyQuotePair returns the pair of iq/aq: of the quote pairs of any type
// (", ', `) around pos, quotes included, the count-th shortest, so 2iq goes
// out a level (x "a 'z' b" y from z: the " pair). Each type pairs its quotes
// from the start of the line. As mini.ai's q with search_method 'cover',
// which iq follows (Vim has no iq), a pair after pos doesn't count.
func anyQuotePair(buffer Buffer, pos Position, count int) (openCol, closeCol int, found bool) {
	line := buffer.GetLineRunes(pos.Row)
	var pairs [][2]int
	for _, q := range []rune{'"', '\'', '`'} {
		open := -1
		for c, r := range line {
			switch {
			case r != q:
			case open < 0:
				open = c
			default:
				if open <= pos.Col && pos.Col <= c {
					pairs = append(pairs, [2]int{open, c})
				}
				open = -1
			}
		}
	}
	if count < 1 {
		count = 1
	}
	if count > len(pairs) {
		return 0, 0, false
	}
	slices.SortStableFunc(pairs, func(a, b [2]int) int { return (a[1] - a[0]) - (b[1] - b[0]) })
	return pairs[count-1][0], pairs[count-1][1], true
}

// quoteObjectCols turns the quote pair at openCol, closeCol on row into the
// columns of the i or a object.
//
// modifier 'i' → the content between the quotes; not found when it is empty.
// With a count above 1, as Vim (:h i"), the quotes too but no white space.
// modifier 'a' → the quotes and, as Vim (:h a"), the white space after the
// closing quote or, when there is none, the white space before the opening one.
// Only spaces and tabs count, as in Vim. a ignores the count.
func quoteObjectCols(buffer Buffer, row, openCol, closeCol int, modifier rune, count int) (startCol, endCol int, found bool) {
	if modifier == 'i' {
		if count > 1 {
			return openCol, closeCol, true
		}
		if closeCol-openCol <= 1 {
			return 0, 0, false // empty quotes: nothing inside
		}
		return openCol + 1, closeCol - 1, true
	}
	lineRunes := buffer.GetLineRunes(row)
	startCol, endCol = openCol, closeCol
	for endCol+1 < len(lineRunes) && isWhiteSpace(lineRunes[endCol+1]) {
		endCol++
	}
	if endCol == closeCol {
		for startCol > 0 && isWhiteSpace(lineRunes[startCol-1]) {
			startCol--
		}
	}
	return startCol, endCol, true
}

// quoteTextObjectRange returns the columns of the i or a object for quote
// around or after pos (see quotePair and quoteObjectCols).
func quoteTextObjectRange(buffer Buffer, pos Position, modifier rune, quote rune, count int) (startCol, endCol int, found bool) {
	openCol, closeCol, ok := quotePair(buffer, pos, quote)
	if !ok {
		return 0, 0, false
	}
	return quoteObjectCols(buffer, pos.Row, openCol, closeCol, modifier, count)
}

// anyQuoteTextObjectRange is quoteTextObjectRange for the pair of any quote
// character that anyQuotePair picks; the count picks the pair, so 2iq
// doesn't take the quotes, as 2i" does.
func anyQuoteTextObjectRange(buffer Buffer, pos Position, modifier rune, count int) (startCol, endCol int, found bool) {
	openCol, closeCol, ok := anyQuotePair(buffer, pos, count)
	if !ok {
		return 0, 0, false
	}
	return quoteObjectCols(buffer, pos.Row, openCol, closeCol, modifier, 1)
}

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

// findBracketBounds locates the innermost bracket pair (openChar/closeChar) enclosing pos,
// handling nesting and multi-line spans. Returns the positions of the open and close bracket chars.
//
// If the cursor sits on openChar it is used directly. If it sits on closeChar, the backward scan
// starts from col-1 with depth 0 so the matching open is found correctly. Otherwise a backward
// depth-counting scan finds the enclosing open, and a forward scan finds the close.
func findBracketBounds(buffer Buffer, pos Position, openChar, closeChar rune) (openPos, closePos Position, found bool) {
	op, ok := findEnclosingOpen(buffer, pos, openChar, closeChar)
	if !ok {
		return Position{}, Position{}, false
	}
	cp, ok := findCloseBracket(buffer, op, openChar, closeChar)
	if !ok {
		return Position{}, Position{}, false
	}
	return op, cp, true
}

// findEnclosingOpen returns the open bracket of the innermost pair around
// pos (pos itself when it is on openChar). On closeChar it returns the
// bracket that one closes. The open bracket doesn't need to be closed.
func findEnclosingOpen(buffer Buffer, pos Position, openChar, closeChar rune) (Position, bool) {
	lineRunes := buffer.GetLineRunes(pos.Row)
	col := pos.Col
	if col < len(lineRunes) && lineRunes[col] == openChar {
		return pos, true
	}

	// If cursor is ON the close bracket, scan backward from just before it (depth stays 0
	// so we find its direct match, not an enclosing one).
	if col < len(lineRunes) && lineRunes[col] == closeChar {
		return findOpenBefore(buffer, pos, openChar, closeChar)
	}
	return scanOpenBackward(buffer, pos, openChar, closeChar)
}

// findOpenBefore returns the open bracket of the innermost pair around pos
// with the character at pos not counted, as Vim's findmatch: on an open
// bracket it is the pair around that one, on a close bracket the pair it
// closes. The open bracket doesn't need to be closed.
func findOpenBefore(buffer Buffer, pos Position, openChar, closeChar rune) (Position, bool) {
	pos.Col--
	return scanOpenBackward(buffer, pos, openChar, closeChar)
}

// scanOpenBackward scans back from pos (included; a column before the line
// or past its end is clamped) for an open bracket that no close bracket
// between them cancels.
func scanOpenBackward(buffer Buffer, pos Position, openChar, closeChar rune) (Position, bool) {
	depth := 0
	for r := pos.Row; r >= 0; r-- {
		line := buffer.GetLineRunes(r)
		endC := len(line) - 1
		if r == pos.Row {
			endC = pos.Col
		}
		// Clamp to avoid OOB on empty lines or past the line end.
		if endC >= len(line) {
			endC = len(line) - 1
		}
		for c := endC; c >= 0; c-- {
			switch line[c] {
			case closeChar:
				depth++
			case openChar:
				if depth == 0 {
					return Position{r, c}, true
				}
				depth--
			}
		}
	}
	return Position{}, false
}

// findNextOpen returns the first open bracket after pos that no close
// bracket between them cancels, as Vim's forward search for a block when
// none encloses the cursor: in "a ) (b)" from column 0 the ")" uses up the
// "(", so there is none.
func findNextOpen(buffer Buffer, pos Position, openChar, closeChar rune) (Position, bool) {
	depth := 0
	for r := pos.Row; r < buffer.LineCount(); r++ {
		line := buffer.GetLineRunes(r)
		startC := 0
		if r == pos.Row {
			startC = pos.Col + 1
		}
		for c := startC; c < len(line); c++ {
			switch line[c] {
			case closeChar:
				depth++
			case openChar:
				if depth == 0 {
					return Position{r, c}, true
				}
				depth--
			}
		}
	}
	return Position{}, false
}

// findCloseBracket scans forward from just past the open bracket at op for
// its matching close bracket.
func findCloseBracket(buffer Buffer, op Position, openChar, closeChar rune) (Position, bool) {
	depth := 0
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
					return Position{r, c}, true
				}
				depth--
			}
		}
	}
	return Position{}, false
}

// findBlock returns the bracket pair of a block text object (i(, a[, …)
// at pos: the innermost pair around it or, as Vim when no open bracket
// encloses pos, the first pair that opens after it, even on a later line.
// ahead reports the second case. An enclosing open bracket that is never
// closed is no block, and Vim doesn't search forward then. count is the
// level (see outerLevels): 2i( is the block around the innermost one.
func findBlock(buffer Buffer, pos Position, openChar, closeChar rune, count int) (openPos, closePos Position, ahead, found bool) {
	openPos, closePos, ahead, found = searchBlock(buffer, pos, openChar, closeChar, findEnclosingOpen, true)
	openPos, closePos, found = outerLevels(openPos, closePos, ahead, found, count, bracketStep(buffer, openChar, closeChar))
	return openPos, closePos, ahead, found
}

// blockStep finds the block one level on from the block that opens at
// openPos: the one around it or, when ahead (the block was found by the
// forward search), the next one the forward search finds from it.
type blockStep func(openPos Position, ahead bool) (nextOpen, nextClose Position, found bool)

// outerLevels goes count-1 levels on from the block openPos-closePos with
// step, as Vim's current_block, which repeats its search count times: the
// search for the block around the cursor goes out a level each time, the
// forward search when none encloses the cursor finds the next open bracket
// after the last one (x(a(b)c)d d2i( from column 0 is b). With too few
// levels there is no block. A count below 2 keeps the block.
func outerLevels(openPos, closePos Position, ahead, found bool, count int, step blockStep) (Position, Position, bool) {
	for i := 1; i < count && found; i++ {
		openPos, closePos, found = step(openPos, ahead)
	}
	if !found {
		return Position{}, Position{}, false
	}
	return openPos, closePos, true
}

// bracketStep is the blockStep for one bracket type.
func bracketStep(buffer Buffer, openChar, closeChar rune) blockStep {
	return func(openPos Position, ahead bool) (Position, Position, bool) {
		if ahead {
			return nextBlockAfter(buffer, openPos, openChar, closeChar)
		}
		openPos, closePos, _, found := findBlockBefore(buffer, openPos, openChar, closeChar, false)
		return openPos, closePos, found
	}
}

// anyBracketStep is the blockStep for ib/ab: the nearest block of any type
// around the open bracket or, when ahead, the first block of any type the
// forward search finds after it.
func anyBracketStep(buffer Buffer) blockStep {
	return func(openPos Position, ahead bool) (Position, Position, bool) {
		nextOpen, nextClose, _, _, found := nearestBlock(func(openChar, closeChar rune) (Position, Position, bool, bool) {
			if ahead {
				o, c, ok := nextBlockAfter(buffer, openPos, openChar, closeChar)
				return o, c, true, ok
			}
			return findBlockBefore(buffer, openPos, openChar, closeChar, false)
		})
		return nextOpen, nextClose, found
	}
}

// nextBlockAfter returns the block whose open bracket the forward search
// (findNextOpen) finds after pos.
func nextBlockAfter(buffer Buffer, pos Position, openChar, closeChar rune) (openPos, closePos Position, found bool) {
	openPos, found = findNextOpen(buffer, pos, openChar, closeChar)
	if !found {
		return Position{}, Position{}, false
	}
	closePos, found = findCloseBracket(buffer, openPos, openChar, closeChar)
	return openPos, closePos, found
}

// findBlockBefore is findBlock with the character at pos not counted (see
// findOpenBefore), as Vim's search for a block from a selection over more
// than one character. It searches forward only when forward is true.
func findBlockBefore(buffer Buffer, pos Position, openChar, closeChar rune, forward bool) (openPos, closePos Position, ahead, found bool) {
	return searchBlock(buffer, pos, openChar, closeChar, findOpenBefore, forward)
}

// searchBlock finds the block at pos whose open bracket enclosing finds
// or, when there is none and forward is true, the first pair after pos.
func searchBlock(buffer Buffer, pos Position, openChar, closeChar rune,
	enclosing func(Buffer, Position, rune, rune) (Position, bool), forward bool,
) (openPos, closePos Position, ahead, found bool) {
	openPos, found = enclosing(buffer, pos, openChar, closeChar)
	if !found && forward {
		openPos, found = findNextOpen(buffer, pos, openChar, closeChar)
		ahead = true
	}
	if !found {
		return Position{}, Position{}, false, false
	}
	closePos, found = findCloseBracket(buffer, openPos, openChar, closeChar)
	if !found {
		return Position{}, Position{}, false, false
	}
	return openPos, closePos, ahead, true
}

// bracketTextObjectRange wraps findBlock and returns the text object range.
// It is used by visual mode, which only needs the inclusive [startPos, endPos] pair.
//
// modifier 'i' → content between the brackets (exclusive of the bracket chars).
// modifier 'a' → the bracket chars themselves.
func bracketTextObjectRange(buffer Buffer, pos Position, modifier rune, openChar, closeChar rune, count int) (startPos, endPos Position, found bool) {
	openPos, closePos, _, ok := findBlock(buffer, blockSearchStart(buffer, pos, openChar), openChar, closeChar, count)
	if !ok {
		return Position{}, Position{}, false
	}

	if modifier == 'a' {
		return openPos, closePos, true
	}
	return bracketInnerRange(buffer, openPos, closePos)
}

// blockFinder finds the block count levels out (see outerLevels) for a
// selection over more than one character from pos, the character at pos not
// counted (see findBlockBefore).
type blockFinder func(pos Position, forward bool, count int) (openPos, closePos Position, found bool)

// bracketFinder is the blockFinder for one bracket type.
func bracketFinder(buffer Buffer, openChar, closeChar rune) blockFinder {
	return func(pos Position, forward bool, count int) (Position, Position, bool) {
		openPos, closePos, ahead, found := findBlockBefore(buffer, pos, openChar, closeChar, forward)
		return outerLevels(openPos, closePos, ahead, found, count, bracketStep(buffer, openChar, closeChar))
	}
}

// anyBracketFinder is the blockFinder for ib/ab: the nearest block of any
// type (see nearestBlock).
func anyBracketFinder(buffer Buffer) blockFinder {
	return func(pos Position, forward bool, count int) (Position, Position, bool) {
		openPos, closePos, _, _, found := nearestBlock(func(openChar, closeChar rune) (Position, Position, bool, bool) {
			return findBlockBefore(buffer, pos, openChar, closeChar, forward)
		})
		// The forward search finds blocks after pos, the others are around it.
		ahead := positionBefore(pos, openPos)
		return outerLevels(openPos, closePos, ahead, found, count, anyBracketStep(buffer))
	}
}

// selectionBlockRange returns the selection a block object (modifier i or
// a) gives from a selection over more than one character, start to end, as
// Vim's current_block in Visual mode:
//   - the block is searched from the start of the selection, the character
//     there not counted: on an open bracket it is the pair around that one,
//     and with none around it the first pair after it; count levels out
//     (see outerLevels), as v2i( from one character;
//   - "a" takes the pair found, even when it is smaller than the selection;
//   - "i" that doesn't make the selection bigger at either end (and is more
//     than one character) goes out a level: the block around the character
//     before the start, that character not counted, with no forward search
//     and no count (as Vim; with a count above 1 the block always grows,
//     since its level 2 opens before the selection).
//     When that character is a close bracket, its pair is the block
//     ("(a)b c(de)f" v3l…7l i( selects "a"), as Vim.
//
// An empty block, or no block, isn't found and the selection is kept.
func selectionBlockRange(buffer Buffer, start, end Position, modifier rune, count int, find blockFinder) (newStart, newEnd Position, found bool) {
	openPos, closePos, found := find(start, true, count)
	for found {
		if modifier == 'a' {
			return openPos, closePos, true
		}
		newStart, newEnd, found = bracketInnerRange(buffer, openPos, closePos)
		if !found {
			break
		}
		// Vim compares the end before taking in the line break.
		cmpEnd := newEnd
		if cmpEnd.Col > 0 && cmpEnd.Col == buffer.LineRuneCount(cmpEnd.Row) {
			cmpEnd.Col--
		}
		if positionBefore(newStart, start) || positionBefore(end, cmpEnd) || newStart == cmpEnd {
			return newStart, newEnd, true
		}
		openPos, closePos, found = find(charBefore(buffer, start), false, 1)
	}
	return Position{}, Position{}, false
}

// charBefore returns the position of the character before pos, as Vim's
// decl: the last character of the line above from column 0 (its column 0
// when it is empty), and pos itself at the start of the buffer.
func charBefore(buffer Buffer, pos Position) Position {
	switch {
	case pos.Col > 0:
		return Position{Row: pos.Row, Col: pos.Col - 1}
	case pos.Row == 0:
		return pos
	}
	return Position{Row: pos.Row - 1, Col: max(buffer.LineRuneCount(pos.Row-1)-1, 0)}
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
func applyCharRangeOp(editor Editor, buffer Buffer, startPos, endPos Position, op operator) *EditorError {
	switch op {
	case opYank:
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

	case opDelete, opChange:
		// deleteRange end is exclusive; convert inclusive endPos.
		end := Position{endPos.Row, endPos.Col + 1}
		lineLen := buffer.LineRuneCount(endPos.Row)
		if end.Col > lineLen {
			end = Position{endPos.Row, lineLen}
		}

		if err := deleteRange(buffer, startPos, end); err != nil {
			return err
		}

		cursor := buffer.GetCursor()
		cursor.Position = startPos
		lineCount := buffer.LineCount()
		if cursor.Position.Row >= lineCount {
			cursor.Position.Row = lineCount - 1
		}
		// Insert mode can sit after the last character; Normal mode can't.
		lineLen = buffer.LineRuneCount(cursor.Position.Row)
		if op == opDelete && cursor.Position.Col >= lineLen && lineLen > 0 {
			cursor.Position.Col = lineLen - 1
		}
		buffer.SetCursor(cursor)

		if op == opChange {
			editor.SetInsertMode()
		}
	}
	return nil
}

// applyOperatorToEmptyQuotes applies op to i" (or i', i`) when the quote
// pair (openCol, closeCol) has nothing between it, and does
// nothing otherwise. As Vim, the operator still runs, on the empty range at
// the closing quote.
func applyOperatorToEmptyQuotes(editor Editor, buffer Buffer, modifier rune, openCol, closeCol int, op operator) *EditorError {
	if modifier != 'i' || closeCol != openCol+1 {
		return nil
	}
	empty := Position{Row: buffer.GetCursor().Position.Row, Col: closeCol}
	return applyOperatorToRange(editor, buffer, op, empty, empty)
}

// applyQuoteOp applies op (yank/delete/change) to the i/a quote text object
// for the given quote char; with a count above 1, i takes the quotes (d2i").
func applyQuoteOp(editor Editor, buffer Buffer, modifier rune, quote rune, op operator, count int) *EditorError {
	cursor := buffer.GetCursor()
	startCol, endCol, found := quoteTextObjectRange(buffer, cursor.Position, modifier, quote, count)
	if !found {
		if openCol, closeCol, ok := quotePair(buffer, cursor.Position, quote); ok {
			return applyOperatorToEmptyQuotes(editor, buffer, modifier, openCol, closeCol, op)
		}
		return nil
	}
	return applyCharRangeOp(editor, buffer,
		Position{Row: cursor.Position.Row, Col: startCol},
		Position{Row: cursor.Position.Row, Col: endCol},
		op)
}

// applyAnyQuoteOp applies op to the quote pair of any type (", ', `) around
// the cursor, count levels out (see anyQuotePair).
func applyAnyQuoteOp(editor Editor, buffer Buffer, modifier rune, op operator, count int) *EditorError {
	cursor := buffer.GetCursor()
	startCol, endCol, found := anyQuoteTextObjectRange(buffer, cursor.Position, modifier, count)
	if !found {
		if openCol, closeCol, ok := anyQuotePair(buffer, cursor.Position, count); ok {
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
func applyBracketRange(editor Editor, buffer Buffer, modifier rune, openPos, closePos Position, op operator) *EditorError {
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

// applyBracketOp applies op to the i/a bracket text object for the given
// open/close chars, count levels out (d2i(); too few levels change nothing.
func applyBracketOp(editor Editor, buffer Buffer, modifier rune, openChar, closeChar rune, op operator, count int) *EditorError {
	pos := blockSearchStart(buffer, buffer.GetCursor().Position, openChar)
	openPos, closePos, _, found := findBlock(buffer, pos, openChar, closeChar, count)
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

// findNearestBracketBounds finds the block of any standard bracket type — (), [], {} — at pos
// (see findBlock): the innermost pair that encloses pos, i.e. the one whose opening bracket is
// closest to the cursor when scanning backward. With none, it is the pair that opens first after
// pos. count levels out, each level the nearest block of any type (see
// anyBracketStep): x(a[b(c)d]e)f d2ib from c is b(c)d. Returns the open/close
// positions.
func findNearestBracketBounds(buffer Buffer, pos Position, count int) (openPos, closePos Position, found bool) {
	openPos, closePos, _, _, found = nearestBlock(func(openChar, closeChar rune) (Position, Position, bool, bool) {
		return findBlock(buffer, pos, openChar, closeChar, 1)
	})
	// The forward search finds blocks after pos, the others are around it.
	ahead := positionBefore(pos, openPos)
	return outerLevels(openPos, closePos, ahead, found, count, anyBracketStep(buffer))
}

// nearestBlock picks, from the block find gives for each standard bracket
// type, the innermost enclosing one, else the first one ahead.
func nearestBlock(find func(openChar, closeChar rune) (openPos, closePos Position, ahead, found bool)) (openPos, closePos Position, openChar, closeChar rune, found bool) {
	type candidate struct {
		openPos  Position
		closePos Position
		open     rune
		close    rune
	}

	// The innermost enclosing pair, else the first pair after pos.
	var enclosing, next *candidate
	for _, pair := range [][2]rune{{'(', ')'}, {'[', ']'}, {'{', '}'}} {
		op, cp, ahead, ok := find(pair[0], pair[1])
		if !ok {
			continue
		}
		c := &candidate{op, cp, pair[0], pair[1]}
		switch {
		case !ahead && (enclosing == nil || positionBefore(enclosing.openPos, op)):
			enclosing = c
		case ahead && (next == nil || positionBefore(op, next.openPos)):
			next = c
		}
	}

	best := enclosing
	if best == nil {
		best = next
	}
	if best == nil {
		return Position{}, Position{}, 0, 0, false
	}
	return best.openPos, best.closePos, best.open, best.close, true
}

// positionBefore reports whether a comes before b in the buffer.
func positionBefore(a, b Position) bool {
	return a.Row < b.Row || (a.Row == b.Row && a.Col < b.Col)
}

// applyAnyBracketOp applies op to the nearest bracket pair of any type (()/[]/{}/)
// at the cursor, count levels out (see findNearestBracketBounds). Used by the 'b' text object alias.
func applyAnyBracketOp(editor Editor, buffer Buffer, modifier rune, op operator, count int) *EditorError {
	cursor := buffer.GetCursor()
	openPos, closePos, found := findNearestBracketBounds(buffer, cursor.Position, count)
	if !found {
		return nil
	}
	return applyBracketRange(editor, buffer, modifier, openPos, closePos, op)
}
