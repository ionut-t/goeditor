package core

import (
	"unicode"
)

// Cursor represents the current position for editing operations
type Cursor struct {
	Position  Position // Current position (row, column)
	Preferred int      // Column j/k go back to (Vim's curswant), as a rune index, not a screen column
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
}

// MoveLeft moves the cursor left by count characters, staying on the line. It
// moves as far as it can and returns ErrStartOfLine if count goes past column 0.
func (c *Cursor) MoveLeft(buffer Buffer, count int) error {
	var err error
	count = max(count, 0)
	if count > c.Position.Col {
		count = max(c.Position.Col, 0)
		err = ErrStartOfLine
	}
	c.Position.Col -= count
	c.clampCol(buffer)
	c.Preferred = c.Position.Col
	return err
}

// MoveRight moves the cursor right by count characters, up to the position
// after the last character. It moves as far as it can and returns ErrEndOfLine
// if count goes past that position.
func (c *Cursor) MoveRight(buffer Buffer, count int) error {
	var err error
	count = max(count, 0)
	if room := buffer.LineRuneCount(c.Position.Row) - c.Position.Col; count > room {
		count = max(room, 0)
		err = ErrEndOfLine
	}
	c.Position.Col += count
	c.clampCol(buffer)
	c.Preferred = c.Position.Col
	return err
}

// MoveUp moves the cursor up by count lines, to the Preferred column or the
// end of the line if it is shorter. Preferred is kept, so a later vertical
// move onto a longer line goes back to it, as Vim's curswant. A count past
// the first line stops there and returns ErrStartOfBuffer.
func (c *Cursor) MoveUp(buffer Buffer, count int) error {
	if c.Position.Row <= 0 {
		return ErrStartOfBuffer
	}
	var err error
	if count > c.Position.Row {
		count = c.Position.Row
		err = ErrStartOfBuffer
	}
	c.Position.Row -= max(count, 0)
	c.Position.Col = min(c.Preferred, buffer.LineRuneCount(c.Position.Row))
	c.clampCol(buffer) // Preferred is exported, so it can be negative
	return err
}

// MoveDown moves the cursor down by count lines, to the Preferred column or
// the end of the line if it is shorter. Preferred is kept, so a later vertical
// move onto a longer line goes back to it, as Vim's curswant. A count past
// the last line stops there and returns ErrEndOfBuffer.
func (c *Cursor) MoveDown(buffer Buffer, count int) error {
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
	c.Position.Col = min(c.Preferred, buffer.LineRuneCount(c.Position.Row))
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

// moveLeftOrUp is MoveLeftOrUp. With pastEnd, as in insert mode, the cursor
// can sit after the last character, so from the start of a line it goes after
// the last character of the line above, as Vim's insert-mode <Left> with '['
// in 'whichwrap'.
func (c *Cursor) moveLeftOrUp(buffer Buffer, count int, pastEnd bool) error {
	var err error
	for range count {
		if c.Position.Col > 0 {
			c.Position.Col--
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

// moveRightOrDown is MoveRightOrDown. With pastEnd, as in insert mode, the
// cursor can sit after the last character, and goes from there to the start
// of the line below, as Vim's insert-mode <Right> with ']' in 'whichwrap'.
func (c *Cursor) moveRightOrDown(buffer Buffer, count int, pastEnd bool) error {
	var err error
	for range count {
		if c.Position.Col < lastCol(buffer, c.Position.Row, pastEnd) {
			c.Position.Col++
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

// lastCol returns the last column the cursor can take on row: the last
// character, or the position after it with pastEnd.
func lastCol(buffer Buffer, row int, pastEnd bool) int {
	if pastEnd {
		return buffer.LineRuneCount(row)
	}
	return max(buffer.LineRuneCount(row)-1, 0)
}

// moveRightToLastChar moves the cursor right by count characters, stopping on
// the last character of the line, as Vim's l in normal and visual mode. It
// moves as far as it can and returns ErrEndOfLine if it reaches that character.
func (c *Cursor) moveRightToLastChar(buffer Buffer, count int) error {
	var err error
	last := lastCol(buffer, c.Position.Row, false)
	if c.Position.Col+count > last {
		count = max(last-c.Position.Col, 0)
		err = ErrEndOfLine
	}
	c.Position.Col += count
	c.Preferred = c.Position.Col
	return err
}

// MoveToLineStart moves the cursor to the start of the current line (col 0)
func (c *Cursor) MoveToLineStart() {
	c.Position.Col = 0
	c.Preferred = 0
}

// MoveToLineEnd moves the cursor to the *last character* of the current line
func (c *Cursor) MoveToLineEnd(buffer Buffer) {
	lineLen := buffer.LineRuneCount(c.Position.Row)
	if lineLen > 0 {
		c.Position.Col = lineLen - 1 // Position is on the last char
	} else {
		c.Position.Col = 0 // Empty line, stay at col 0
	}
	// Preferred is the column of the last character
	c.Preferred = c.Position.Col
}

// MoveToAfterLineEnd moves the cursor *after* the last character of the current line
func (c *Cursor) MoveToAfterLineEnd(buffer Buffer) {
	lineLen := buffer.LineRuneCount(c.Position.Row)
	c.Position.Col = lineLen // Position *after* last char
	// Preferred is the column *after* the last character
	c.Preferred = c.Position.Col
}

// MoveToFirstNonBlank moves the cursor to the first non-whitespace character
func (c *Cursor) MoveToFirstNonBlank(buffer Buffer) {
	line := buffer.GetLineRunes(c.Position.Row)
	firstNonBlank := 0
	found := false
	for i, r := range line {
		if !unicode.IsSpace(r) {
			firstNonBlank = i
			found = true
			break
		}
	}
	// If loop finishes and not found, all are spaces (or empty), move to col 0
	if !found {
		firstNonBlank = 0
	}

	c.Position.Col = firstNonBlank
	c.Preferred = c.Position.Col
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
func isWhiteSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isBigWordChar treats any non-whitespace rune as a word character, giving the
// WORD semantics of the W/E/B/gE motions.
func isBigWordChar(r rune) bool {
	return !isWhiteSpace(r)
}

// MoveWordForward moves the cursor forward by count words (Vim 'w' behavior)
func (c *Cursor) MoveWordForward(buffer Buffer, count int, isWordChar func(rune) bool) error {
	for range count {
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
				// Find first non-blank on the new line
				newLineRunes := buffer.GetLineRunes(c.Position.Row)
				for startCol, r := range newLineRunes {
					if !isWhiteSpace(r) {
						c.Position.Col = startCol
						break
					}
				}
				moved = true // Moved to a new line
				continue     // Restart logic on the new line if needed
			}

			// Current position
			pos := c.Position.Col
			currentChar := lineRunes[pos]

			// State 1: On a word character -> move to end of word, then skip whitespace
			if isWordChar(currentChar) {
				// Move to end of word
				for pos < lineLen && isWordChar(lineRunes[pos]) {
					pos++
				}
				// Skip whitespace
				for pos < lineLen && isWhiteSpace(lineRunes[pos]) {
					pos++
				}
			} else if isWhiteSpace(currentChar) {
				// State 2: On whitespace -> skip whitespace
				for pos < lineLen && isWhiteSpace(lineRunes[pos]) {
					pos++
				}
			} else {
				// State 3: On punctuation -> move past current punctuation block, then skip whitespace
				for pos < lineLen && !isWordChar(lineRunes[pos]) && !isWhiteSpace(lineRunes[pos]) {
					pos++
				}
				// Skip whitespace
				for pos < lineLen && isWhiteSpace(lineRunes[pos]) {
					pos++
				}
			}

			// If we moved past the end of the line during scan
			if pos >= lineLen {
				// Don't move yet, let the next iteration handle moving to the next line
				c.Position.Col = lineLen // Stay at the end for now
			} else {
				c.Position.Col = pos
				moved = true // Moved within the line
			}
		} // End inner loop for line spanning
	} // End outer loop for count
	c.Preferred = c.Position.Col // Update preferred column
	return nil
}

// moveWordToEndOnce advances the cursor to the end of the next word once.
// Returns true if the end of the buffer was reached without finding a word end.
func (c *Cursor) moveWordToEndOnce(buffer Buffer, isWordChar func(rune) bool) bool {
	pos := c.Position.Col + 1
	for {
		lineRunes := buffer.GetLineRunes(c.Position.Row)
		lineLen := len(lineRunes)

		if pos >= lineLen {
			if c.Position.Row >= buffer.LineCount()-1 {
				return true // end of buffer
			}
			c.Position.Row++
			pos = 0
			continue
		}

		for pos < lineLen && isWhiteSpace(lineRunes[pos]) {
			pos++
		}
		if pos >= lineLen {
			continue
		}

		if isWordChar(lineRunes[pos]) {
			for pos < lineLen && isWordChar(lineRunes[pos]) {
				pos++
			}
		} else {
			for pos < lineLen && !isWordChar(lineRunes[pos]) && !isWhiteSpace(lineRunes[pos]) {
				pos++
			}
		}

		c.Position.Col = pos - 1
		return false
	}
}

// MoveWordToEnd moves the cursor to the end of the word count times (Vim 'e' behavior).
func (c *Cursor) MoveWordToEnd(buffer Buffer, count int, isWordChar func(rune) bool) error {
	for i := range count {
		if atEnd := c.moveWordToEndOnce(buffer, isWordChar); atEnd {
			if i == 0 {
				return ErrEndOfBuffer
			}
			break
		}
	}
	c.Preferred = c.Position.Col
	return nil
}

// MoveWordBackward moves the cursor backward by count words (Vim 'b' behavior)
func (c *Cursor) MoveWordBackward(buffer Buffer, count int, isWordChar func(rune) bool) error {
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
					c.Position.Col--
				}
				// Don't set moved = true yet, need to find the word start on this (prev) line
			} else {
				// Move back one character to start the scan from within the current line
				c.Position.Col--
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

			// Current character type after moving back (or onto prev line)
			currentChar := lineRunes[pos]

			// State 1: Landed on whitespace -> skip back over whitespace
			if isWhiteSpace(currentChar) {
				for pos >= 0 && isWhiteSpace(lineRunes[pos]) {
					pos--
				}
			}

			// If we skipped whitespace and are not at start of line
			if pos >= 0 {
				currentChar = lineRunes[pos] // Re-evaluate current char
				// State 2: Now on word char -> skip back to beginning of this word
				if isWordChar(currentChar) {
					for pos >= 0 && isWordChar(lineRunes[pos]) {
						pos--
					}
				} else {
					// State 3: Now on punctuation -> skip back to beginning of this punctuation block
					for pos >= 0 && !isWordChar(lineRunes[pos]) && !isWhiteSpace(lineRunes[pos]) {
						pos--
					}
				}
				// Move one forward to land on the start of the word/punctuation found
				pos++
			} else {
				// If we skipped whitespace and hit beginning of line
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
func (c *Cursor) moveWordToEndBackwardOnce(buffer Buffer, isWordChar func(rune) bool) bool {
	row, col := c.Position.Row, c.Position.Col
	lineRunes := buffer.GetLineRunes(row)

	// Exit the current token going backward so we don't stop inside the word we are on.
	if col < len(lineRunes) && !isWhiteSpace(lineRunes[col]) {
		if isWordChar(lineRunes[col]) {
			for col >= 0 && isWordChar(lineRunes[col]) {
				col--
			}
		} else {
			for col >= 0 && !isWordChar(lineRunes[col]) && !isWhiteSpace(lineRunes[col]) {
				col--
			}
		}
	}

	// Skip whitespace backward, crossing lines, until we land on a non-whitespace character.
	for {
		for col >= 0 && col < len(lineRunes) && isWhiteSpace(lineRunes[col]) {
			col--
		}

		if col >= 0 {
			c.Position.Row = row
			c.Position.Col = col
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
func (c *Cursor) MoveWordToEndBackward(buffer Buffer, count int, isWordChar func(rune) bool) error {
	for i := range count {
		if atStart := c.moveWordToEndBackwardOnce(buffer, isWordChar); atStart {
			if i == 0 {
				return ErrStartOfBuffer
			}
			break
		}
	}
	c.Preferred = c.Position.Col
	return nil
}

func (c *Cursor) MoveBlockBackward(buffer Buffer, count int) error {
	if c.Position.Row <= 0 {
		return ErrStartOfBuffer
	}

	for range count {
		if c.Position.Row <= 0 {
			break
		}

		// If on a blank line, skip past it first (backward)
		for c.Position.Row > 0 {
			line := buffer.GetLineRunes(c.Position.Row)
			if len(line) != 0 {
				break
			}
			c.Position.Row--
		}

		// Skip non-blank lines backward until we hit a blank line or the start of buffer
		for c.Position.Row > 0 {
			line := buffer.GetLineRunes(c.Position.Row)
			if len(line) == 0 {
				break
			}
			c.Position.Row--
		}
	}

	c.Position.Col = 0

	return nil
}

func (c *Cursor) MoveBlockForward(buffer Buffer, count int) error {
	if c.Position.Row >= buffer.LineCount()-1 {
		return ErrEndOfBuffer
	}

	for range count {
		if c.Position.Row >= buffer.LineCount()-1 {
			break
		}

		// If on a blank line, skip past it first
		for c.Position.Row < buffer.LineCount()-1 {
			line := buffer.GetLineRunes(c.Position.Row)
			if len(line) != 0 {
				break
			}
			c.Position.Row++
		}

		// Skip non-blank lines forward until we hit a blank line or the end of buffer
		for c.Position.Row < buffer.LineCount()-1 {
			line := buffer.GetLineRunes(c.Position.Row)
			if len(line) == 0 {
				break
			}
			c.Position.Row++
		}
	}

	c.Position.Col = 0

	return nil
}

// Deprecated: ScrollUp only moves the cursor and leaves the window where it
// is. CTRL-U is Editor.ScrollHalfPage(false, count), which scrolls both.
// availableWidth is unused; it is kept so existing callers still compile.
func (c *Cursor) ScrollUp(buffer Buffer, viewportHeight, availableWidth int) error {
	return c.MoveUp(buffer, max(viewportHeight/2, 1))
}

// Deprecated: ScrollDown only moves the cursor and leaves the window where it
// is. CTRL-D is Editor.ScrollHalfPage(true, count), which scrolls both.
// availableWidth is unused; it is kept so existing callers still compile.
func (c *Cursor) ScrollDown(buffer Buffer, viewportHeight, availableWidth int) error {
	return c.MoveDown(buffer, max(viewportHeight/2, 1))
}
