package core

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Buffer represents the text content being edited (Using Runes).
// Implementations needn't move the cursor when they edit: the editor clamps
// it after edits made through Editor.GetBuffer().
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
	// FindAll(pattern string, options SearchOptions) []Position TODO: Implement later if needed
	// Replace(pattern string, replacement string, options SearchOptions) int // Implement later if needed)

	IsModified() bool          // Check if buffer has been modified
	SaveContent()              // Save content
	SetContent(content []byte) // Set content (from file or other source)
	IsEmpty() bool             // Check if buffer is empty
}

// SearchOptions represents options for search operations
type SearchOptions struct {
	IgnoreCase        bool            // Case insensitive search
	SmartCase         bool            // ...unless search contains uppercase
	Backwards         bool            // Whether to search backwards
	Wrap              bool            // Whether to wrap around the buffer
	WholeWord         bool            // Only match whole words — used by * and #
	WordBoundaryStart bool            // \< anchor: match must start at a word boundary
	WordBoundaryEnd   bool            // \> anchor: match must end at a word boundary
	IsWordChar        func(rune) bool // Classifier for boundary checks; nil falls back to letter|digit|_
}

// defaultWordCharClassifier is the fallback word-character classifier used when
// SearchOptions.IsWordChar is nil.
var defaultWordCharClassifier = func(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// checkWordBoundaries reports whether the match at [col, col+matchLen) on lineRunes
// satisfies the word-boundary constraints in opts.
// lineRunes must be the original (non-lowercased) line so boundary chars are classified correctly.
func checkWordBoundaries(lineRunes []rune, col, matchLen int, opts SearchOptions) bool {
	isWordChar := opts.IsWordChar
	if isWordChar == nil {
		isWordChar = defaultWordCharClassifier
	}
	if (opts.WholeWord || opts.WordBoundaryStart) && col > 0 && isWordChar(lineRunes[col-1]) {
		return false
	}
	if (opts.WholeWord || opts.WordBoundaryEnd) && col+matchLen < len(lineRunes) && isWordChar(lineRunes[col+matchLen]) {
		return false
	}
	return true
}

// isSmartCaseUpper reports whether r changes when lowercased. This is Vim's smartcase test,
// so it also covers titlecase (ǅ) and Other_Uppercase (Ⓐ), which unicode.IsUpper misses.
func isSmartCaseUpper(r rune) bool { return unicode.ToLower(r) != r }

// foldRune returns the smallest rune in r's simple case-fold orbit, so runes that
// fold together (ς σ Σ, ſ s S, K k and the Kelvin sign) map to the same rune.
// Like Vim, there's no multi-rune folding (ß doesn't match ss), and İ and ı fold
// only to themselves.
func foldRune(r rune) rune {
	if r < utf8.RuneSelf {
		// The smallest rune in an ASCII letter's orbit is its ASCII uppercase.
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	m := r
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		m = min(m, f)
	}
	return m
}

// matchRunesAt reports whether pattern occurs in line at rune column col.
// With ignoreCase, pattern must already be folded with foldRune; line runes are folded as they are compared.
func matchRunesAt(line, pattern []rune, col int, ignoreCase bool) bool {
	if col < 0 || col+len(pattern) > len(line) {
		return false
	}
	for i, p := range pattern {
		r := line[col+i]
		if r == p {
			continue // p is already folded, so an equal rune folds to it too
		}
		if ignoreCase {
			r = foldRune(r)
		}
		if r != p {
			return false
		}
	}
	return true
}

// needsBoundaryCheck reports whether any word-boundary option is active.
func needsBoundaryCheck(opts SearchOptions) bool {
	return opts.WholeWord || opts.WordBoundaryStart || opts.WordBoundaryEnd
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

	// Empty content is one empty line, as in NewBuffer
	if len(linesRune) == 0 {
		linesRune = [][]rune{{}}
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
			colOnLastDeletedLine = currentLineLen // the whole line is consumed
			currentRow++
		} else {
			// Deletion ends within this line
			colOnLastDeletedLine = remainingToDelete - 1 // -1 because we consumed the newline
			break                                        // Stop iterating through lines
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

// Find searches forward or backward for the next occurrence of pattern, starting
// after (or before) start. With options.Wrap, a search that reaches the end (or
// start) of the buffer continues from the other end, up to and including start.
// Returns the position and true if found, or false otherwise.
func (b *textBuffer) Find(pattern string, start Position, options SearchOptions) (Position, bool) {
	if pattern == "" {
		return Position{}, false
	}

	// Smartcase: with IgnoreCase on, a pattern containing an uppercase letter is case-sensitive.
	ignoreCase := options.IgnoreCase && (!options.SmartCase || !strings.ContainsFunc(pattern, isSmartCaseUpper))

	searchRunes := []rune(pattern)
	if ignoreCase {
		for i, r := range searchRunes {
			searchRunes[i] = foldRune(r)
		}
	}

	numLines := b.LineCount()

	// firstRunes holds every rune that can match the pattern's first rune: its whole fold
	// orbit with ignoreCase (at most 4 runes in Unicode). Comparing against it rejects most
	// columns without folding the line rune, which costs table lookups for non-ASCII text.
	var firstRunes [4]rune
	firstRunes[0] = searchRunes[0]
	numFirst := 1
	if ignoreCase {
		for f := unicode.SimpleFold(searchRunes[0]); f != searchRunes[0] && numFirst < len(firstRunes); f = unicode.SimpleFold(f) {
			firstRunes[numFirst] = f
			numFirst++
		}
	}

	// matchAt reports whether a match starts at rune column c and passes the boundary checks.
	matchAt := func(lineRunes []rune, c int) bool {
		if c < 0 || c >= len(lineRunes) {
			return false
		}
		// Compare with firstRunes[0] before calling slices.Contains: calling it for every
		// column made exact search about 35% slower.
		if r := lineRunes[c]; r != firstRunes[0] && !slices.Contains(firstRunes[1:numFirst], r) {
			return false
		}
		return matchRunesAt(lineRunes, searchRunes, c, ignoreCase) &&
			(!needsBoundaryCheck(options) || checkWordBoundaries(lineRunes, c, len(searchRunes), options))
	}

	// Scan from just after (or before) start to the end (or start) of the buffer. With Wrap,
	// continue from the other end back to start, inclusive, so a lone match at start is found.
	// Each position is checked at most once, even when nothing matches.
	if options.Backwards {
		for r := min(start.Row, numLines-1); r >= 0; r-- {
			lineRunes := b.GetLineRunes(r)
			from := len(lineRunes) - 1
			if r == start.Row {
				from = min(start.Col-1, from)
			}
			for c := from; c >= 0; c-- {
				if matchAt(lineRunes, c) {
					return Position{Row: r, Col: c}, true
				}
			}
		}
		if !options.Wrap {
			return Position{}, false
		}
		for r := numLines - 1; r >= max(start.Row, 0); r-- {
			lineRunes := b.GetLineRunes(r)
			to := 0
			if r == start.Row {
				to = max(start.Col, 0)
			}
			for c := len(lineRunes) - 1; c >= to; c-- {
				if matchAt(lineRunes, c) {
					return Position{Row: r, Col: c}, true
				}
			}
		}
		return Position{}, false
	}

	for r := max(start.Row, 0); r < numLines; r++ {
		lineRunes := b.GetLineRunes(r)
		from := 0
		if r == start.Row {
			from = max(start.Col+1, 0)
		}
		for c := from; c < len(lineRunes); c++ {
			if matchAt(lineRunes, c) {
				return Position{Row: r, Col: c}, true
			}
		}
	}
	if !options.Wrap {
		return Position{}, false
	}
	for r := 0; r <= min(start.Row, numLines-1); r++ {
		lineRunes := b.GetLineRunes(r)
		to := len(lineRunes) - 1
		if r == start.Row {
			to = min(start.Col, to)
		}
		for c := 0; c <= to; c++ {
			if matchAt(lineRunes, c) {
				return Position{Row: r, Col: c}, true
			}
		}
	}
	return Position{}, false
}
