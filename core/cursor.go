package core

import "unicode"

// Cursor represents the current position for editing operations
type Cursor struct {
	Position  Position // Current position (row, column)
	Preferred int      // Preferred column for vertical movement (sticky column)
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

// MoveLeft moves the cursor left by count characters, aware of visual wrapping.
// availableWidth is the width used for rendering text (excluding line numbers).
func (c *Cursor) MoveLeft(buffer Buffer, count int, availableWidth int) error {
	if availableWidth <= 0 { // Avoid division by zero or nonsensical behavior
		availableWidth = 1 // Fallback to minimal width
	}
	for range count {
		if c.Position.Col <= 0 {
			// Already at the logical start of the line
			return ErrStartOfLine
		}
		c.Position.Col--
	}
	c.clampCol(buffer) // Clamp just in case (shouldn't be needed after bounds check)

	// Update preferred column based on visual position if wrapping is active
	// Preferred column should reflect the visual column on the screen.
	c.Preferred = c.Position.Col % availableWidth

	return nil
}

// MoveRight moves the cursor right by count characters, aware of visual wrapping.
// availableWidth is the width used for rendering text (excluding line numbers).
func (c *Cursor) MoveRight(buffer Buffer, count int, availableWidth int) error {
	if availableWidth <= 0 { // Avoid division by zero or nonsensical behavior
		availableWidth = 1 // Fallback to minimal width
	}
	lineLen := buffer.LineRuneCount(c.Position.Row)
	for range count {
		// Allow moving *to* the position *after* the last logical char
		if c.Position.Col >= lineLen {
			return ErrEndOfLine
		}
		c.Position.Col++
	}
	c.clampCol(buffer) // Clamp just in case (e.g., if lineLen was 0)

	// Update preferred column based on visual position if wrapping is active
	// Preferred column should reflect the visual column on the screen.
	c.Preferred = c.Position.Col % availableWidth

	return nil
}

// MoveUp moves the cursor up by count lines
func (c *Cursor) MoveUp(buffer Buffer, count int, availableWidth int) error {
	if c.Position.Row <= 0 {
		return ErrStartOfBuffer
	}
	if availableWidth <= 0 { // Ensure positive width
		availableWidth = 1
	}

	// Store visual preferred column before moving
	preferredVisualCol := c.Preferred

	for range count {
		if c.Position.Row <= 0 {
			return ErrStartOfBuffer // Stop if we hit the top
		}
		c.Position.Row--
	}

	// Adjust column based on preferred *visual* column and new line length
	lineLen := buffer.LineRuneCount(c.Position.Row)
	if lineLen == 0 {
		c.Position.Col = 0
	} else {
		// Find the logical column that corresponds to the preferred visual column
		// on the new line. This involves figuring out which wrapped segment
		// the preferred visual column belongs to and calculating the offset.
		targetVisualRow := preferredVisualCol / availableWidth // Which wrapped line segment index
		targetCharInRow := preferredVisualCol % availableWidth // Index within that segment

		// Calculate the target logical column
		c.Position.Col = targetVisualRow*availableWidth + targetCharInRow

		// Clamp the calculated logical column to the actual line length
		if c.Position.Col >= lineLen {
			c.Position.Col = lineLen // Place cursor at the end of the line if preferred is beyond
			// Recalculate visual preferred based on clamped position
			c.Preferred = c.Position.Col % availableWidth
		} else {
			// Keep original preferred visual column if target was reachable
			c.Preferred = preferredVisualCol
		}

		if c.Position.Col < 0 { // Should not happen, but safety
			c.Position.Col = 0
			c.Preferred = 0
		}
	}

	// Final clamp on logical column (redundant but safe)
	c.clampCol(buffer)

	return nil
}

// MoveDown moves the cursor down by count lines
func (c *Cursor) MoveDown(buffer Buffer, count int, availableWidth int) error {
	if c.Position.Row >= buffer.LineCount()-1 {
		return ErrEndOfBuffer
	}
	if availableWidth <= 0 { // Ensure positive width
		availableWidth = 1
	}

	// Store visual preferred column before moving
	preferredVisualCol := c.Preferred

	for range count {
		if c.Position.Row >= buffer.LineCount()-1 {
			return ErrEndOfBuffer // Stop if we hit the bottom
		}
		c.Position.Row++
	}

	// Adjust column based on preferred *visual* column and new line length
	lineLen := buffer.LineRuneCount(c.Position.Row)
	if lineLen == 0 {
		c.Position.Col = 0
	} else {
		// Find the logical column that corresponds to the preferred visual column
		targetVisualRow := preferredVisualCol / availableWidth
		targetCharInRow := preferredVisualCol % availableWidth
		c.Position.Col = targetVisualRow*availableWidth + targetCharInRow

		// Clamp the calculated logical column to the actual line length
		if c.Position.Col >= lineLen {
			c.Position.Col = lineLen // Place cursor at the end of the line if preferred is beyond
			// Recalculate visual preferred based on clamped position
			c.Preferred = c.Position.Col % availableWidth
		} else {
			// Keep original preferred visual column if target was reachable
			c.Preferred = preferredVisualCol
		}

		if c.Position.Col < 0 { // Should not happen, but safety
			c.Position.Col = 0
			c.Preferred = 0
		}
	}

	// Final clamp on logical column (redundant but safe)
	c.clampCol(buffer)

	return nil
}

// MoveLeftOrUp moves the cursor left or up based on the current column and visual width
func (c *Cursor) MoveLeftOrUp(buffer Buffer, count, availableWidth int) error {
	if availableWidth <= 0 {
		availableWidth = 1
	}
	// Check if cursor is visually at the start of a wrapped line (but not logical start)
	if c.Position.Col > 0 && (c.Position.Col%availableWidth == 0) {
		// Visually at start of a wrapped segment, move left logically
		return c.MoveLeft(buffer, count, availableWidth)
	} else if c.Position.Col > 0 {
		// Not at visual start, simple move left
		return c.MoveLeft(buffer, count, availableWidth)
	} else {
		// At logical start (Col == 0), try moving up
		if err := c.MoveUp(buffer, count, availableWidth); err != nil {
			return err // Return error if already at buffer start
		}
		// If moved up successfully, move to end of the new line
		c.MoveToLineEnd(buffer, availableWidth) // Pass width to update Preferred correctly
		return nil
	}
}

// MoveRightOrDown moves the cursor right or down based on the current column and visual width
func (c *Cursor) MoveRightOrDown(buffer Buffer, count, availableWidth int) error {
	if availableWidth <= 0 {
		availableWidth = 1
	}
	lineLen := buffer.LineRuneCount(c.Position.Row)
	// Check if cursor is visually at the end of a wrapped line (but not logical end)
	if c.Position.Col < lineLen && ((c.Position.Col+1)%availableWidth == 0) {
		// Visually at end of a wrapped segment, move right logically
		return c.MoveRight(buffer, count, availableWidth)
	} else if c.Position.Col < lineLen {
		// Not at visual end, simple move right
		return c.MoveRight(buffer, count, availableWidth)
	} else {
		// At logical end (Col == lineLen), try moving down
		if err := c.MoveDown(buffer, count, availableWidth); err != nil {
			return err // Return error if already at buffer end
		}
		// If moved down successfully, move to start of the new line
		c.MoveToLineStart()
		return nil
	}
}

// MoveToLineStart moves the cursor to the start of the current line (col 0)
func (c *Cursor) MoveToLineStart() {
	c.Position.Col = 0
	c.Preferred = 0
}

// MoveToLineEnd moves the cursor to the *last character* of the current line
func (c *Cursor) MoveToLineEnd(buffer Buffer, availableWidth int) {
	if availableWidth <= 0 {
		availableWidth = 1
	}
	lineLen := buffer.LineRuneCount(c.Position.Row)
	if lineLen > 0 {
		c.Position.Col = lineLen - 1 // Position is on the last char
	} else {
		c.Position.Col = 0 // Empty line, stay at col 0
	}
	// Preferred should be the visual column of the last character
	c.Preferred = c.Position.Col % availableWidth
}

// MoveToAfterLineEnd moves the cursor *after* the last character of the current line
func (c *Cursor) MoveToAfterLineEnd(buffer Buffer, availableWidth int) {
	if availableWidth <= 0 {
		availableWidth = 1
	}
	lineLen := buffer.LineRuneCount(c.Position.Row)
	c.Position.Col = lineLen // Position *after* last char
	// Preferred should be the visual column *after* the last character
	c.Preferred = c.Position.Col % availableWidth
}

// MoveToFirstNonBlank moves the cursor to the first non-whitespace character
func (c *Cursor) MoveToFirstNonBlank(buffer Buffer, availableWidth int) {
	if availableWidth <= 0 {
		availableWidth = 1
	}
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
	c.Preferred = c.Position.Col % availableWidth
}

// MoveToBufferStart moves the cursor to the start of the buffer
func (c *Cursor) MoveToBufferStart() {
	c.Position.Row = 0
	c.Position.Col = 0
	c.Preferred = 0
}

// MoveToBufferEnd moves the cursor to the start of the last line
func (c *Cursor) MoveToBufferEnd(buffer Buffer, availableWidth int) error {
	lastLine := max(buffer.LineCount()-1, 0)
	c.Position.Row = lastLine
	// Option 1: Go to logical column 0
	// c.Position.Col = 0
	// c.Preferred = 0

	// Option 2: Go to first non-blank (like Vim's G)
	c.MoveToFirstNonBlank(buffer, availableWidth) // This updates Col and Preferred

	return nil
}

// --- Word Movement (Using Unicode and Runes) ---
// These generally work on logical positions, but update the preferred visual column at the end.
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
}

func isWhiteSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// MoveWordForward moves the cursor forward by count words (Vim 'w' behavior)
func (c *Cursor) MoveWordForward(buffer Buffer, count int, availableWidth int) error {
	if availableWidth <= 0 {
		availableWidth = 1
	}
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
	c.Preferred = c.Position.Col % availableWidth // Update preferred visual column
	return nil
}

// MoveWordToEnd moves the cursor to the end of the word count times (Vim 'e' behavior).
func (c *Cursor) MoveWordToEnd(buffer Buffer, count int, availableWidth int) error {
	if availableWidth <= 0 {
		availableWidth = 1
	}

	for i := range count {
		pos := c.Position.Col + 1

		// This loop handles moving across lines to find the next word end.
	searchLoop:
		for {
			lineRunes := buffer.GetLineRunes(c.Position.Row)
			lineLen := len(lineRunes)

			// If our starting position is beyond the current line, we need to move to the next.
			if pos >= lineLen {
				if c.Position.Row >= buffer.LineCount()-1 {
					// We are at the end of the buffer.
					if i == 0 {
						return ErrEndOfBuffer
					} // If we haven't moved at all, it's an error.
					goto endMove // Otherwise, we just stop here.
				}
				// Move to the start of the next line.
				c.Position.Row++
				pos = 0
				// The searchLoop will restart, processing the new line.
				continue
			}

			// 1. Skip any whitespace to find the start of the next word/punctuation.
			for pos < lineLen && isWhiteSpace(lineRunes[pos]) {
				pos++
			}

			// If skipping whitespace took us to the end of the line, restart the search
			// on the next line.
			if pos >= lineLen {
				continue
			}

			// 2. Now we are at the start of a word or punctuation. Find its end.
			if isWordChar(lineRunes[pos]) {
				for pos < lineLen && isWordChar(lineRunes[pos]) {
					pos++
				}
			} else { // Punctuation
				for pos < lineLen && !isWordChar(lineRunes[pos]) && !isWhiteSpace(lineRunes[pos]) {
					pos++
				}
			}

			// pos is now one char *past* the end. We want to be on the end.
			c.Position.Col = pos - 1
			// We've found the word end for this iteration of the count,
			// so we break out of the searchLoop.
			break searchLoop
		}
	}

endMove:
	c.Preferred = c.Position.Col % availableWidth
	return nil
}

// MoveWordBackward moves the cursor backward by count words (Vim 'b' behavior)
func (c *Cursor) MoveWordBackward(buffer Buffer, count int, availableWidth int) error {
	if availableWidth <= 0 {
		availableWidth = 1
	}
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

	c.Preferred = c.Position.Col % availableWidth // Update preferred visual column
	return nil
}
