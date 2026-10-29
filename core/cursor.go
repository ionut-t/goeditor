package core

import "math"

// maxCol is Preferred after $: j and k then go to the end of every line, as
// with Vim's curswant MAXCOL, until a command sets the column again.
const maxCol = math.MaxInt

// Cursor represents the current position for editing operations
type Cursor struct {
	Position  Position // Current position (row, column)
	Preferred int      // Column j/k go back to (Vim's curswant), as a rune index, not a screen column; math.MaxInt after $ (the end of every line)
}

// --- Cursor Movement ---

// clampCol ensures the column stays within the valid range for the given line
// Note: This clamps based on LOGICAL line length, which is correct.
func (c *Cursor) clampCol(buffer Buffer) {
	lineLen := buffer.LineRuneCount(c.Position.Row)
	if lineLen == 0 {
		c.Position.Col = 0
	} else if c.Position.Col > lineLen {
		// Allow col to be len(line) for insertions, but clamp for navigation if needed
		c.Position.Col = lineLen
	}
	if c.Position.Col < 0 {
		c.Position.Col = 0
	}
	// Never on a composing character: back to the start of its character.
	c.Position.Col = charStart(buffer.GetLineRunes(c.Position.Row), c.Position.Col)
}

// MoveLeft moves the cursor left by count characters, staying on the line. It
// moves as far as it can and returns ErrStartOfLine if count goes past column 0.
// At column 0 it doesn't move and leaves Preferred as it is, as Vim's h.
func (c *Cursor) MoveLeft(buffer Buffer, count int) error {
	var err error
	count = max(count, 0)
	if c.Position.Col <= 0 && count > 0 {
		return ErrStartOfLine
	}
	col, n := charsBack(buffer.GetLineRunes(c.Position.Row), c.Position.Col, count)
	if n < count {
		err = ErrStartOfLine
	}
	c.Position.Col = col
	c.clampCol(buffer)
	c.Preferred = c.Position.Col
	return err
}

// MoveRight moves the cursor right by count characters, up to the position
// after the last character. It moves as far as it can and returns ErrEndOfLine
// if count goes past that position. There it doesn't move and leaves Preferred
// as it is, as Vim's insert-mode <Right>.
func (c *Cursor) MoveRight(buffer Buffer, count int) error {
	var err error
	count = max(count, 0)
	if c.Position.Col >= buffer.LineRuneCount(c.Position.Row) && count > 0 {
		return ErrEndOfLine
	}
	col, n := charsForward(buffer.GetLineRunes(c.Position.Row), c.Position.Col, count)
	if n < count {
		err = ErrEndOfLine
	}
	c.Position.Col = col
	c.clampCol(buffer)
	c.Preferred = c.Position.Col
	return err
}

// MoveUp moves the cursor up by count lines, to the Preferred column or the
// last character of the line if it is shorter. Preferred is kept, so a later
// vertical move onto a longer line goes back to it, as Vim's curswant. A count
// past the first line stops there and returns ErrStartOfBuffer.
func (c *Cursor) MoveUp(buffer Buffer, count int) error {
	return c.moveUp(buffer, count, false)
}

// MoveDown moves the cursor down by count lines, to the Preferred column or
// the last character of the line if it is shorter. Preferred is kept, so a
// later vertical move onto a longer line goes back to it, as Vim's curswant. A
// count past the last line stops there and returns ErrEndOfBuffer.
func (c *Cursor) MoveDown(buffer Buffer, count int) error {
	return c.moveDown(buffer, count, false)
}

// moveUp is MoveUp. With pastEnd, as in insert and visual mode, a shorter line
// puts the cursor after its last character instead of on it.
func (c *Cursor) moveUp(buffer Buffer, count int, pastEnd bool) error {
	if c.Position.Row <= 0 {
		return ErrStartOfBuffer
	}
	var err error
	if count > c.Position.Row {
		count = c.Position.Row
		err = ErrStartOfBuffer
	}
	c.Position.Row -= max(count, 0)
	c.Position.Col = min(c.Preferred, lastCol(buffer, c.Position.Row, pastEnd))
	c.clampCol(buffer) // Preferred is exported, so it can be negative
	return err
}

// moveDown is MoveDown. With pastEnd, as in insert and visual mode, a shorter
// line puts the cursor after its last character instead of on it.
func (c *Cursor) moveDown(buffer Buffer, count int, pastEnd bool) error {
	lastRow := buffer.LineCount() - 1
	if c.Position.Row >= lastRow {
		return ErrEndOfBuffer
	}
	var err error
	if count > lastRow-c.Position.Row {
		count = lastRow - c.Position.Row
		err = ErrEndOfBuffer
	}
	c.Position.Row += max(count, 0)
	c.Position.Col = min(c.Preferred, lastCol(buffer, c.Position.Row, pastEnd))
	c.clampCol(buffer) // Preferred is exported, so it can be negative
	return err
}

// MoveDownToFirstNonBlank moves the cursor down count lines to the first
// non-blank, as Vim's <CR> and +. A count past the end stops on the last line;
// on the last line it returns ErrEndOfBuffer without moving.
func (c *Cursor) MoveDownToFirstNonBlank(buffer Buffer, count int) error {
	startRow := c.Position.Row
	err := c.MoveDown(buffer, count)
	if c.Position.Row != startRow {
		c.MoveToFirstNonBlank(buffer)
	}
	return err
}

// MoveLeftOrUp moves the cursor left by count characters, going to the last
// character of the line above from the start of a line, as Vim's <BS> with the
// default 'whichwrap'. Each line break counts as one character. It moves as far
// as it can and returns ErrStartOfBuffer if it reaches the start of the buffer.
func (c *Cursor) MoveLeftOrUp(buffer Buffer, count int) error {
	return c.moveLeftOrUp(buffer, count, false)
}

// MoveRightOrDown moves the cursor right by count characters, going to the
// start of the line below from the last character of a line, as Vim's <Space>
// with the default 'whichwrap'. Each line break counts as one character. It
// moves as far as it can and returns ErrEndOfBuffer if it reaches the last
// character of the buffer.
func (c *Cursor) MoveRightOrDown(buffer Buffer, count int) error {
	return c.moveRightOrDown(buffer, count, false)
}

// moveLeftOrUp is MoveLeftOrUp. With pastEnd, as in insert and visual mode,
// the cursor can sit after the last character, so from the start of a line it
// goes after the last character of the line above, as Vim's insert-mode <Left>
// with '[' in 'whichwrap'. If it can't move at all, Preferred is left as it
// is, as Vim's h and <BS>.
func (c *Cursor) moveLeftOrUp(buffer Buffer, count int, pastEnd bool) error {
	if count > 0 && c.Position.Row <= 0 && c.Position.Col <= 0 {
		return ErrStartOfBuffer
	}
	var err error
	for range count {
		if c.Position.Col > 0 {
			c.Position.Col, _ = charsBack(buffer.GetLineRunes(c.Position.Row), c.Position.Col, 1)
		} else if c.Position.Row > 0 {
			c.Position.Row--
			c.Position.Col = lastCol(buffer, c.Position.Row, pastEnd)
		} else {
			err = ErrStartOfBuffer
			break
		}
	}
	c.Preferred = c.Position.Col
	return err
}

// moveRightOrDown is MoveRightOrDown. With pastEnd, as in insert and visual
// mode, the cursor can sit after the last character, and goes from there to
// the start of the line below, as Vim's insert-mode <Right> with ']' in
// 'whichwrap'. If it can't move at all, Preferred is left as it is, as Vim's l
// and <Space>.
func (c *Cursor) moveRightOrDown(buffer Buffer, count int, pastEnd bool) error {
	if count > 0 && c.Position.Row >= buffer.LineCount()-1 && c.Position.Col >= lastCol(buffer, c.Position.Row, pastEnd) {
		return ErrEndOfBuffer
	}
	var err error
	for range count {
		if c.Position.Col < lastCol(buffer, c.Position.Row, pastEnd) {
			c.Position.Col, _ = charsForward(buffer.GetLineRunes(c.Position.Row), c.Position.Col, 1)
		} else if c.Position.Row < buffer.LineCount()-1 {
			c.Position.Row++
			c.Position.Col = 0
		} else {
			err = ErrEndOfBuffer
			break
		}
	}
	c.Preferred = c.Position.Col
	return err
}

// lastCol returns the last column the cursor can take on row: the start of
// the last character, or the position after it with pastEnd.
func lastCol(buffer Buffer, row int, pastEnd bool) int {
	if pastEnd {
		return buffer.LineRuneCount(row)
	}
	line := buffer.GetLineRunes(row)
	return charStart(line, max(len(line)-1, 0))
}

// moveRightToLastChar moves the cursor right by count characters, stopping on
// the last character of the line, or with pastEnd, as in visual mode, on the
// line break, as Vim's l. It moves as far as it can and returns ErrEndOfLine
// if it reaches that position. There it doesn't move and leaves Preferred as
// it is, as Vim's l, so after $ j and k still go to the end of every line.
func (c *Cursor) moveRightToLastChar(buffer Buffer, count int, pastEnd bool) error {
	var err error
	last := lastCol(buffer, c.Position.Row, pastEnd)
	if c.Position.Col >= last && count > 0 {
		return ErrEndOfLine
	}
	col, n := charsForward(buffer.GetLineRunes(c.Position.Row), c.Position.Col, count)
	if n < count || col > last {
		col = last
		err = ErrEndOfLine
	}
	c.Position.Col = col
	c.Preferred = c.Position.Col
	return err
}

// MoveToLineStart moves the cursor to the start of the current line (col 0)
func (c *Cursor) MoveToLineStart() {
	c.Position.Col = 0
	c.Preferred = 0
}

// MoveToLineEnd moves the cursor to the last character of the current line,
// as Vim's $. Preferred becomes math.MaxInt, so j and k go on to the end of
// every line.
func (c *Cursor) MoveToLineEnd(buffer Buffer) {
	_ = c.moveToLineEnd(buffer, 1, false) // a count of 1 never fails
}

// moveToLineEnd is [count]$: count-1 lines down, then to the last character,
// or with pastEnd, as in visual mode, onto the line break. A count past the
// last line stops there; on the last line a count above 1 returns
// ErrEndOfBuffer without moving. Either way j and k now go to the end of every
// line, as Vim sets curswant before it moves.
func (c *Cursor) moveToLineEnd(buffer Buffer, count int, pastEnd bool) error {
	c.Preferred = maxCol
	if count > 1 {
		lastRow := buffer.LineCount() - 1
		if c.Position.Row >= lastRow {
			return ErrEndOfBuffer
		}
		c.Position.Row = min(c.Position.Row+count-1, lastRow)
	}
	c.Position.Col = lastCol(buffer, c.Position.Row, pastEnd)
	return nil
}

// MoveToAfterLineEnd moves the cursor *after* the last character of the current line
func (c *Cursor) MoveToAfterLineEnd(buffer Buffer) {
	lineLen := buffer.LineRuneCount(c.Position.Row)
	c.Position.Col = lineLen // Position *after* last char
	// Preferred is the column *after* the last character
	c.Preferred = c.Position.Col
}

// MoveToFirstNonBlank moves the cursor to the first non-blank character, as
// Vim's ^. Blanks are spaces and tabs. On a line of only blanks it stops on
// the last one.
func (c *Cursor) MoveToFirstNonBlank(buffer Buffer) {
	line := buffer.GetLineRunes(c.Position.Row)
	col := firstNonBlankCol(line)
	if col == len(line) && col > 0 { // only blanks: stop on the last one
		col--
	}
	// A composing character on a blank is part of the blank's character:
	// stand on that.
	c.Position.Col = charStart(line, col)
	c.Preferred = c.Position.Col
}

// moveToInsertStart moves the cursor to where Vim's I inserts: the first
// non-blank, or after the blanks on a line of only blanks.
func (c *Cursor) moveToInsertStart(buffer Buffer) {
	c.Position.Col = firstNonBlankCol(buffer.GetLineRunes(c.Position.Row))
	c.Preferred = c.Position.Col
}

// firstNonBlankCol returns the column of the first non-blank in line, or
// len(line) if there is none. A composing character on a blank counts as a
// non-blank, as in Vim's skipwhite, so I inserts between the blank and its
// composing character.
func firstNonBlankCol(line []rune) int {
	for i, r := range line {
		if !isWhiteSpace(r) {
			return i
		}
	}
	return len(line)
}

// MoveToBufferStart moves the cursor to the start of the buffer
func (c *Cursor) MoveToBufferStart() {
	c.Position.Row = 0
	c.Position.Col = 0
	c.Preferred = 0
}

// MoveToBufferEnd moves the cursor to the first non-blank of the last line
func (c *Cursor) MoveToBufferEnd(buffer Buffer) {
	c.moveToLine(buffer, 0)
}

// gotoLineRow returns the 0-indexed row a [count]G motion targets: line count
// (1-indexed, clamped to the buffer), or the last line when count is 0.
func gotoLineRow(buffer Buffer, count int) int {
	lastLine := max(buffer.LineCount()-1, 0)
	if count <= 0 {
		return lastLine
	}
	return min(count-1, lastLine)
}

// moveToLine moves the cursor to the first non-blank of the line a [count]G
// motion targets (the last line when count is 0).
func (c *Cursor) moveToLine(buffer Buffer, count int) {
	c.Position.Row = gotoLineRow(buffer, count)
	c.MoveToFirstNonBlank(buffer)
}

// --- Word Movement (Using Unicode and Runes) ---
// These generally work on logical positions, but update the preferred column at the end.

// isWhiteSpace is Vim's VIM_ISWHITE: space or tab. The first non-blank, J and
// indent use it, so they keep a no-break space as text.
func isWhiteSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isBlank reports whether r is in class 0 of Vim's utf_class (mbyte.c): the
// blanks that word motions, word objects and <cWORD> skip. Besides space and
// tab these are the no-break space, the Ogham space mark, the spaces from
// U+2000 to U+200B, the line and paragraph separators, the narrow no-break
// space, the medium mathematical space and the ideographic space, but not
// U+000B or U+0085.
func isBlank(r rune) bool {
	switch r {
	case ' ', '\t', 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200b
}

// MoveWordForward moves the cursor forward by count words (Vim 'w' behavior)
func (c *Cursor) MoveWordForward(buffer Buffer, count int, cls func(rune) int) error {
	return c.moveWordForward(buffer, count, cls, false)
}

// moveWordForward is w (W with bigWordClass); a word is a run of one class of
// cls. It stops on empty lines but not on lines of only
// blanks, and at the end of the buffer it leaves the cursor after the last
// character. With stopAtEOL, as w after an operator, the last of the count
// words stops at the end of its line instead of going on to the next one,
// and from an empty line it stops at the start of the next line, as Vim's
// fwd_word with eol.
func (c *Cursor) moveWordForward(buffer Buffer, count int, cls func(rune) int, stopAtEOL bool) error {
	for i := range count {
		eol := stopAtEOL && i == count-1
		moved := false // Flag to track if we actually moved

		// Loop in case a 'w' spans multiple lines
		for !moved {
			lineRunes := buffer.GetLineRunes(c.Position.Row)
			lineLen := len(lineRunes)

			// If at end of line, or empty line, move to next line's start
			if c.Position.Col >= lineLen || lineLen == 0 {
				if c.Position.Row >= buffer.LineCount()-1 {
					return ErrEndOfBuffer // Cannot move past last line
				}
				c.Position.Row++
				c.Position.Col = 0
				if eol {
					break
				}
				// Skip the blanks at the start of the new line; a line of only
				// blanks is skipped like the blanks at the end of a line.
				newLineRunes := buffer.GetLineRunes(c.Position.Row)
				for c.Position.Col < len(newLineRunes) && blankAt(newLineRunes, c.Position.Col) {
					c.Position.Col++
				}
				moved = len(newLineRunes) == 0 || c.Position.Col < len(newLineRunes)
				continue
			}

			// Move past the word under the cursor, then over the blanks after it.
			pos := c.Position.Col
			if class := classAt(lineRunes, pos, cls); class != blankClass {
				for pos < lineLen && classAt(lineRunes, pos, cls) == class {
					pos++
				}
			}
			for pos < lineLen && blankAt(lineRunes, pos) {
				pos++
			}

			// If we moved past the end of the line during scan
			if pos >= lineLen {
				// Don't move yet, let the next iteration handle moving to the next line
				c.Position.Col = lineLen // Stay at the end for now
				moved = eol
			} else {
				c.Position.Col = pos
				moved = true // Moved within the line
			}
		} // End inner loop for line spanning
	} // End outer loop for count
	c.Preferred = c.Position.Col // Update preferred column
	return nil
}

// stayOnLastChar moves the cursor back onto the last character after w or W
// ran into the end of the buffer (err), which leaves it after that character.
// Normal mode can't stand there; Vim's w stops on the last character.
func (c *Cursor) stayOnLastChar(buffer Buffer, err error) {
	if err == ErrEndOfBuffer {
		c.Position.Col = lastCol(buffer, c.Position.Row, false)
		c.Preferred = c.Position.Col
	}
}

// moveWordToEndOnce advances the cursor to the end of the next word once.
// Returns true if the end of the buffer was reached without finding a word
// end; the cursor is then on the last character of the buffer, as Vim's e.
func (c *Cursor) moveWordToEndOnce(buffer Buffer, cls func(rune) int) bool {
	pos := charEnd(buffer.GetLineRunes(c.Position.Row), c.Position.Col)
	for {
		lineRunes := buffer.GetLineRunes(c.Position.Row)
		lineLen := len(lineRunes)

		if pos >= lineLen {
			if c.Position.Row >= buffer.LineCount()-1 {
				c.Position.Col = lastCol(buffer, c.Position.Row, false)
				return true // end of buffer
			}
			c.Position.Row++
			pos = 0
			continue
		}

		for pos < lineLen && blankAt(lineRunes, pos) {
			pos++
		}
		if pos >= lineLen {
			continue
		}

		class := classAt(lineRunes, pos, cls)
		for pos < lineLen && classAt(lineRunes, pos, cls) == class {
			pos++
		}

		c.Position.Col = charStart(lineRunes, pos-1)
		return false
	}
}

// MoveWordToEnd moves the cursor to the end of the word count times (Vim 'e'
// behavior). Past the last word it goes to the last character of the buffer,
// and returns ErrEndOfBuffer if there was no word end at all to go to.
func (c *Cursor) MoveWordToEnd(buffer Buffer, count int, cls func(rune) int) error {
	var err error
	for i := range count {
		if atEnd := c.moveWordToEndOnce(buffer, cls); atEnd {
			if i == 0 {
				err = ErrEndOfBuffer
			}
			break
		}
	}
	c.Preferred = c.Position.Col
	return err
}

// MoveWordBackward moves the cursor backward by count words (Vim 'b' behavior)
func (c *Cursor) MoveWordBackward(buffer Buffer, count int, cls func(rune) int) error {
	for range count {
		moved := false // Flag to track if we actually moved

		// Loop in case a 'b' spans multiple lines
		for !moved {
			// If at start of line, move to previous line's end
			if c.Position.Col <= 0 {
				if c.Position.Row <= 0 {
					return ErrStartOfBuffer // Cannot move before first line
				}
				c.Position.Row--
				c.Position.Col = buffer.LineRuneCount(c.Position.Row) // Go to end of prev line
				// If the prev line is empty, Col becomes 0, loop continues correctly
				if c.Position.Col > 0 {
					// Start scan from last character of previous line
					c.Position.Col = lastCol(buffer, c.Position.Row, false)
				}
				// Don't set moved = true yet, need to find the word start on this (prev) line
			} else {
				// Move back one character to start the scan from within the current line
				c.Position.Col, _ = charsBack(buffer.GetLineRunes(c.Position.Row), c.Position.Col, 1)
			}

			lineRunes := buffer.GetLineRunes(c.Position.Row)
			lineLen := len(lineRunes)
			pos := c.Position.Col // The character we are currently checking

			// Handle empty line edge case after moving lines or if starting on empty line
			if lineLen == 0 {
				c.Position.Col = 0
				moved = true // Moved to start of empty line
				continue
			}

			// State 1: Landed on whitespace -> skip back over whitespace
			if blankAt(lineRunes, pos) {
				for pos >= 0 && blankAt(lineRunes, pos) {
					pos--
				}
			}

			// If we skipped whitespace and are not at start of line
			if pos >= 0 {
				// Skip back to the start of the word before the cursor.
				class := classAt(lineRunes, pos, cls)
				for pos >= 0 && classAt(lineRunes, pos, cls) == class {
					pos--
				}
				// Move one forward to land on the start of the word/punctuation found
				pos++
			} else if c.Position.Row > 0 {
				// Only blanks before the cursor: they aren't a word, so carry
				// on from the end of the previous line (Vim's b doesn't stop
				// on indentation or on a line of blanks).
				c.Position.Col = 0
				continue
			} else {
				// Blanks at the start of the buffer: stop at its start.
				pos = 0
			}

			c.Position.Col = pos
			moved = true // Moved within the line or arrived at start

		} // End inner loop for line spanning
	} // End outer loop for count

	c.Preferred = c.Position.Col // Update preferred column
	return nil
}

// moveWordToEndBackwardOnce moves the cursor to the end of the previous word once.
// Returns true if the start of the buffer was reached without finding a previous word end.
func (c *Cursor) moveWordToEndBackwardOnce(buffer Buffer, cls func(rune) int) bool {
	row, col := c.Position.Row, c.Position.Col
	lineRunes := buffer.GetLineRunes(row)

	// Exit the current token going backward so we don't stop inside the word we are on.
	if col < len(lineRunes) && !blankAt(lineRunes, col) {
		class := classAt(lineRunes, col, cls)
		for col >= 0 && classAt(lineRunes, col, cls) == class {
			col--
		}
	}

	// Skip whitespace backward, crossing lines, until we land on a non-whitespace
	// character. Past the line end (visual mode) counts as whitespace.
	startRow := row
	col = min(col, len(lineRunes)-1)
	for {
		for col >= 0 && blankAt(lineRunes, col) {
			col--
		}

		if col >= 0 {
			c.Position.Row, c.Position.Col = row, charStart(lineRunes, col)
			return false
		}

		// Vim counts an empty line as a word for ge, except the one it starts on.
		if len(lineRunes) == 0 && row != startRow {
			c.Position.Row, c.Position.Col = row, 0
			return false
		}

		if row <= 0 {
			c.Position.Row = 0
			c.Position.Col = 0
			return true // start of buffer
		}
		row--
		lineRunes = buffer.GetLineRunes(row)
		col = len(lineRunes) - 1
	}
}

// MoveWordToEndBackward moves the cursor to the end of the previous word count times (Vim 'ge' behaviour).
func (c *Cursor) MoveWordToEndBackward(buffer Buffer, count int, cls func(rune) int) error {
	for i := range count {
		if atStart := c.moveWordToEndBackwardOnce(buffer, cls); atStart {
			if i == 0 {
				return ErrStartOfBuffer
			}
			break
		}
	}
	c.Preferred = c.Position.Col
	return nil
}

// MoveBlockBackward moves the cursor back count paragraphs (Vim's {). It
// returns ErrStartOfBuffer, without moving, if the start of the buffer comes
// before the count is used up. Either way it sets Preferred, as Vim.
func (c *Cursor) MoveBlockBackward(buffer Buffer, count int) error {
	return c.moveParagraph(buffer, count, false)
}

// MoveBlockForward moves the cursor forward count paragraphs (Vim's }). It
// returns ErrEndOfBuffer, without moving, if the end of the buffer comes
// before the count is used up. Either way it sets Preferred, as Vim.
func (c *Cursor) MoveBlockForward(buffer Buffer, count int) error {
	return c.moveParagraph(buffer, count, true)
}

func (c *Cursor) moveParagraph(buffer Buffer, count int, forward bool) error {
	pos, _, ok := findParagraph(buffer, c.Position.Row, count, forward)
	if ok {
		c.Position = pos
	}
	// Vim sets curswant even when the motion fails. Visual mode can leave the
	// cursor on the line break, which isn't a column j and k go back to.
	c.Preferred = min(c.Position.Col, lastCol(buffer, c.Position.Row, false))
	switch {
	case ok:
		return nil
	case forward:
		return ErrEndOfBuffer
	default:
		return ErrStartOfBuffer
	}
}

// findParagraph is Vim's findpar for { and } from row: count times, it moves
// past at least one non-empty line and stops on the next empty one. If it
// reaches the edge of the buffer with the count used up, it stops there: on
// column 0 of the first line, or on the last character of the last line,
// where the motion is inclusive. With count left over it fails (ok false).
func findParagraph(buffer Buffer, row, count int, forward bool) (pos Position, inclusive, ok bool) {
	dir := -1
	if forward {
		dir = 1
	}
	lastRow := buffer.LineCount() - 1
	for count = max(count, 1); count > 0; {
		count--
		movedOverText := false
		for first := true; ; first = false {
			empty := buffer.LineRuneCount(row) == 0
			if !empty {
				movedOverText = true
			}
			if !first && movedOverText && empty {
				break
			}
			if next := row + dir; next >= 0 && next <= lastRow {
				row = next
				continue
			}
			if count > 0 {
				return Position{}, false, false
			}
			break
		}
	}
	if forward && row == lastRow && buffer.LineRuneCount(row) > 0 {
		return Position{Row: row, Col: lastCol(buffer, row, false)}, true, true
	}
	return Position{Row: row}, false, true
}
