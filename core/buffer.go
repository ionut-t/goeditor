package core

import (
	"bytes"
	"fmt"
	"strings"
)

// Buffer represents the text content being edited (Using Runes)
type Buffer interface {
	// Content access
	GetLines() []string              // Get lines as strings (for saving/display)
	GetLineRunes(lineNum int) []rune // Get specific line as runes (for editing)
	LineRuneCount(lineNum int) int   // Get rune count for a line
	GetSavedContent() string         // Get saved buffer content as a string
	GetCurrentContent() string       // Get entire buffer content as a string
	LineCount() int                  // Get number of lines

	// Modification
	InsertRunesAt(row, col int, runes []rune) error     // Insert runes (handles newlines)
	DeleteRunesAt(row, col int, count int) *EditorError // Delete runes (handles newlines)
	// ReplaceRunesAt(row, col int, count int, runes []rune) error // Replace (can be Delete + Insert)

	// Cursor
	GetCursor() Cursor
	SetCursor(Cursor)

	Find(pattern string, start Position, options SearchOptions) (Position, bool) // Find next/prev
	// Replace(pattern string, replacement string, options SearchOptions) int // Implement later if needed

	IsModified() bool          // Check if buffer has been modified
	SaveContent()              // Save content
	SetContent(content []byte) // Set content (from file or other source)
	IsEmpty() bool             // Check if buffer is empty
}

// SearchOptions represents options for search operations
type SearchOptions struct {
	CaseSensitive bool
	Backwards     bool
	Wrap          bool // Whether to wrap around the buffer
}

// textBuffer implementation using runes for better unicode handling
type textBuffer struct {
	lines        [][]rune // Store lines as slices of runes
	cursor       Cursor
	savedContent string
}

// NewBuffer creates a new empty buffer
func NewBuffer() Buffer {
	return &textBuffer{
		lines:  [][]rune{{}}, // Start with one empty line
		cursor: Cursor{Position: Position{0, 0}, Preferred: 0},
	}
}

func NewBufferFromBytes(content []byte) Buffer {
	b := textBuffer{
		lines:  [][]rune{{}}, // Start with one empty line
		cursor: Cursor{Position: Position{0, 0}, Preferred: 0},
	}

	b.SetContent(content)
	b.SaveContent()
	return &b
}

func (b *textBuffer) IsEmpty() bool {
	return len(b.lines) == 1 && len(b.lines[0]) == 0
}

func (b *textBuffer) SetContent(content []byte) {
	// Convert bytes to runes
	runes := bytes.Runes(content)
	linesRune := make([][]rune, 0)
	var currentLine []rune

	for _, r := range runes {
		if r == '\n' {
			linesRune = append(linesRune, currentLine)
			currentLine = []rune{} // Start a new line
		} else {
			currentLine = append(currentLine, r)
		}
	}

	if len(currentLine) > 0 {
		linesRune = append(linesRune, currentLine) // Add the last line if not empty
	}

	b.lines = linesRune
}

func (b *textBuffer) GetLines() []string {
	linesStr := make([]string, len(b.lines))
	for i, r := range b.lines {
		linesStr[i] = string(r)
	}
	return linesStr
}

func (b *textBuffer) GetLineRunes(lineNum int) []rune {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return nil // Or an empty slice? Return nil to indicate error clearly.
	}
	return b.lines[lineNum]
}

func (b *textBuffer) LineRuneCount(lineNum int) int {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return 0
	}
	return len(b.lines[lineNum])
}

func (b *textBuffer) IsModified() bool {
	return b.savedContent != b.GetCurrentContent()
}

func (b *textBuffer) SaveContent() {
	b.savedContent = b.GetCurrentContent()
}

// GetCurrentContent returns the entire buffer content as a string
func (b *textBuffer) GetCurrentContent() string {
	// More efficient way to join rune slices later if needed
	linesStr := make([]string, len(b.lines))
	for i, r := range b.lines {
		linesStr[i] = string(r)
	}
	return strings.Join(linesStr, "\n")
}

// GetSavedContent returns the saved content as a string
func (b *textBuffer) GetSavedContent() string {
	return b.savedContent
}

func (b *textBuffer) LineCount() int {
	return len(b.lines)
}

func (b *textBuffer) GetCursor() Cursor {
	return b.cursor
}

// SetCursor sets the cursor position, validating and clamping it.
func (b *textBuffer) SetCursor(cursor Cursor) {
	// Clamp Row
	if cursor.Position.Row < 0 {
		cursor.Position.Row = 0
	} else if cursor.Position.Row >= len(b.lines) {
		cursor.Position.Row = max(len(b.lines)-1, 0)
	}

	// Clamp Column
	lineLen := b.LineRuneCount(cursor.Position.Row)
	if cursor.Position.Col < 0 {
		cursor.Position.Col = 0
	} else if cursor.Position.Col > lineLen {
		// Allow cursor to be one position *past* the end of the line
		cursor.Position.Col = lineLen
	}

	b.cursor = cursor
}

// --- Buffer Modification (using Runes, more robust newline handling) ---

// InsertRunesAt inserts runes at the specified position. Handles newlines correctly.
func (b *textBuffer) InsertRunesAt(row, col int, runes []rune) error {
	if row < 0 || row >= len(b.lines) {
		return fmt.Errorf("InsertRunesAt: %w: row %d out of bounds [0, %d)", ErrInvalidPosition, row, len(b.lines))
	}

	line := b.lines[row]
	if col < 0 || col > len(line) { // Allow insertion at len(line)
		return fmt.Errorf("InsertRunesAt: %w: col %d out of bounds [0, %d]", ErrInvalidPosition, col, len(line))
	}

	// Check for newlines within the runes to insert
	textToInsert := string(runes) // Convert once for splitting
	if strings.Contains(textToInsert, "\n") {
		parts := strings.Split(textToInsert, "\n")

		// Runes before the insertion point
		head := line[:col]
		// Runes after the insertion point
		tail := make([]rune, len(line)-col)
		copy(tail, line[col:]) // Make a copy

		// Modify the current line (first part of insertion)
		b.lines[row] = append(head, []rune(parts[0])...)

		// Lines to insert between current and next original line
		newLines := make([][]rune, len(parts)-1)
		for i := 1; i < len(parts); i++ {
			newLines[i-1] = []rune(parts[i])
		}

		// The last part of the inserted text gets prepended to the tail
		newLines[len(newLines)-1] = append(newLines[len(newLines)-1], tail...)

		// --- Re-think Slice Insertion ---
		originalAfter := make([][]rune, len(b.lines)-(row+1))
		if row < len(b.lines)-1 {
			copy(originalAfter, b.lines[row+1:])
		}

		// Slice up to insertion point (exclusive of inserted lines)
		finalLines := b.lines[:row+1] // Includes the modified first line
		// Append the new intermediate lines
		finalLines = append(finalLines, newLines...)
		// Append the original lines that came after
		finalLines = append(finalLines, originalAfter...)

		b.lines = finalLines

	} else {
		// Simple insertion within the line (no newlines)
		newLine := make([]rune, 0, len(line)+len(runes))
		newLine = append(newLine, line[:col]...)
		newLine = append(newLine, runes...)
		newLine = append(newLine, line[col:]...)
		b.lines[row] = newLine
	}

	return nil
}

// DeleteRunesAt deletes count runes starting at the specified position. Handles crossing lines.
func (b *textBuffer) DeleteRunesAt(row, col int, count int) *EditorError {
	if count <= 0 {
		return nil
	} // Nothing to delete

	if row < 0 || row >= len(b.lines) {
		return &EditorError{
			id:  ErrInvalidPositionId,
			err: fmt.Errorf("%s: row %d out of bounds [0, %d)", ErrInvalidPosition, row, len(b.lines)),
		}
	}

	line := b.lines[row]
	lineLen := len(line)

	if col < 0 || col > lineLen { // Allow deleting *from* len(line) if merging lines
		return &EditorError{
			id:  ErrInvalidPositionId,
			err: fmt.Errorf("%s: col %d out of bounds [0, %d]", ErrInvalidPosition, col, lineLen),
		}
	}

	// Deletion entirely within the current line
	if col+count <= lineLen {
		newLine := make([]rune, 0, lineLen-count)
		newLine = append(newLine, line[:col]...)
		newLine = append(newLine, line[col+count:]...)
		b.lines[row] = newLine
		return nil
	}

	// Deletion crosses into subsequent lines
	runesToDeleteOnThisLine := lineLen - col
	remainingToDelete := count - runesToDeleteOnThisLine

	// Delete to the end of the current line
	b.lines[row] = line[:col]

	// Now, delete the newline character and potentially merge/delete lines
	linesToDelete := 0
	colOnLastDeletedLine := 0 // Column where deletion *stops* on the last affected line

	currentRow := row + 1
	for remainingToDelete > 0 && currentRow < len(b.lines) {
		linesToDelete++
		currentLineRunes := b.lines[currentRow]
		currentLineLen := len(currentLineRunes)

		// Deleting the newline + content of this line
		if remainingToDelete >= currentLineLen+1 { // +1 for the newline
			remainingToDelete -= (currentLineLen + 1)
			currentRow++
		} else {
			// Deletion ends within this line
			colOnLastDeletedLine = remainingToDelete - 1 // -1 because we consumed the newline
			remainingToDelete = 0
			break // Stop iterating through lines
		}
	}

	// If deletion consumed lines, merge and remove them
	if linesToDelete > 0 {
		lastAffectedRow := row + linesToDelete
		if lastAffectedRow < len(b.lines) {
			// Merge end of start line with remaining part of last affected line
			remainingPartOfLastLine := b.lines[lastAffectedRow][colOnLastDeletedLine:]
			b.lines[row] = append(b.lines[row], remainingPartOfLastLine...)

			// Remove the intermediate lines
			copy(b.lines[row+1:], b.lines[lastAffectedRow+1:])
			newLen := len(b.lines) - linesToDelete
			b.lines = b.lines[:newLen]
		} else {
			// Deletion went to or past the end of the buffer
			// Just need to remove the lines
			if row+1 < len(b.lines) { // Check if there are lines to remove
				newLen := row + 1
				b.lines = b.lines[:newLen]
			}
		}
	}

	// Ensure buffer always has at least one (potentially empty) line
	if len(b.lines) == 0 {
		b.lines = [][]rune{{}}
		b.cursor = Cursor{Position{0, 0}, 0} // Reset cursor if buffer was emptied
	}

	return nil
}

// Find searches forward or backward for the next occurrence of pattern.
// Returns the position and true if found, or false otherwise.
// TODO: Implement Wrap option.
func (b *textBuffer) Find(pattern string, start Position, options SearchOptions) (Position, bool) {
	if pattern == "" {
		return Position{}, false
	}

	searchRunes := []rune(pattern)
	if !options.CaseSensitive {
		searchRunes = []rune(strings.ToLower(pattern))
	}
	searchLen := len(searchRunes)

	numLines := b.LineCount()
	currentLine := start.Row
	currentCol := start.Col

	if options.Backwards {
		// Adjust starting column for backward search
		if currentCol > 0 {
			currentCol-- // Start searching *before* the cursor
		} else if currentLine > 0 {
			currentLine--
			currentCol = b.LineRuneCount(currentLine) // Start at end of previous line
		} else {
			return Position{}, false // Already at start of buffer
		}

		for r := currentLine; r >= 0; r-- {
			lineRunes := b.GetLineRunes(r)
			lineContent := lineRunes
			if !options.CaseSensitive {
				lineContent = []rune(strings.ToLower(string(lineRunes)))
			}

			startSearchCol := currentCol
			if r != start.Row { // If we moved to a previous line, search whole line
				startSearchCol = len(lineContent) - searchLen
			} else {
				// Search from currentCol backwards
				startSearchCol = currentCol - searchLen + 1
			}
			if startSearchCol < 0 {
				startSearchCol = 0
			}

			// Iterate backwards within the line checking for matches
			foundIdx := -1
			tempLineStr := string(lineContent) // Temp string for Index
			tempSearchStr := string(searchRunes)
			searchStartOffset := 0

			for col := startSearchCol; col >= 0; col-- {
				idx := strings.LastIndex(tempLineStr[searchStartOffset:col+searchLen], tempSearchStr) // More complex search needed here
				// FIXME: LastIndex logic needs refinement for backwards scan starting mid-line.
				// Let's use simple forward scan logic adapted for backwards iteration for now.
				// This requires iterating runes manually backwards.

				println("Checking backwards from col", col, "to", col+searchLen, "in line", r, idx)

				// Manual backwards check (simpler to reason about)
				match := true
				if col+searchLen <= len(lineContent) { // Check bounds
					for k := range searchLen {
						if lineContent[col+k] != searchRunes[k] {
							match = false
							break
						}
					}
					if match {
						foundIdx = col
						break // Found the last match starting at or before startSearchCol
					}
				}
			}

			if foundIdx != -1 {
				return Position{Row: r, Col: foundIdx}, true
			}

			// Reset column for next line up
			if r > 0 {
				currentCol = b.LineRuneCount(r - 1)
			}
		}

	} else { // Forward search
		// Adjust starting column for forward search
		currentCol++ // Start search *after* the cursor

		for r := currentLine; r < numLines; r++ {
			lineRunes := b.GetLineRunes(r)
			lineContent := lineRunes
			if !options.CaseSensitive {
				lineContent = []rune(strings.ToLower(string(lineRunes)))
			}

			startSearchCol := 0
			if r == start.Row {
				startSearchCol = currentCol
			}

			// Use strings.Index on the relevant part of the line
			if startSearchCol < len(lineContent) {
				lineSuffix := string(lineContent[startSearchCol:])
				searchStr := string(searchRunes)
				idx := strings.Index(lineSuffix, searchStr)
				if idx != -1 {
					return Position{Row: r, Col: startSearchCol + idx}, true
				}
			}

			// Reset column for next line down
			currentCol = 0 // Start search from beginning of subsequent lines
		}
	}

	// TODO: Implement wrapping if options.Wrap is true

	return Position{}, false // Not found
}
