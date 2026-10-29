package core

import "unicode"

// A character, as Vim's cursor and character commands see it, is a rune and
// the composing characters after it (utfc_ptr2len in mbyte.c). Columns stay
// rune indexes, but the cursor only stands on the first rune of a character,
// and commands that act on characters take its composing characters with it.
// This isn't a grapheme cluster: Vim doesn't join emoji ZWJ sequences, flags,
// Hangul jamo or spacing marks, and it does join a mark to a tab.

// joinsPrevious reports whether r is part of the character that prev is in,
// as Vim's utf_composinglike: r is a nonspacing or enclosing mark (Vim's
// utf_iscomposing), or an alef after a lam, which Vim joins with the default
// 'arabicshape'. A mark at the start of a line is a character of its own.
func joinsPrevious(prev, r rune) bool {
	if r < 0x300 {
		return false
	}
	if unicode.In(r, unicode.Mn, unicode.Me) {
		return true
	}
	return prev == 0x644 && (r == 0x622 || r == 0x623 || r == 0x625 || r == 0x627)
}

// charStart returns the column of the first rune of the character col is in.
// A col past the end of line is returned as it is.
func charStart(line []rune, col int) int {
	if col >= len(line) {
		return col
	}
	for col > 0 && joinsPrevious(line[col-1], line[col]) {
		col--
	}
	return max(col, 0)
}

// charEnd returns the column after the character at col: past its
// composing characters.
func charEnd(line []rune, col int) int {
	if col >= len(line) {
		return col + 1
	}
	for col++; col < len(line) && joinsPrevious(line[col-1], line[col]); col++ {
	}
	return col
}

// charsForward returns the column count characters after col, and how many
// it went, which is less than count at the end of line.
func charsForward(line []rune, col, count int) (int, int) {
	n := 0
	for ; n < count && col < len(line); n++ {
		col = charEnd(line, col)
	}
	return col, n
}

// charsBack returns the column count characters before col, and how many it
// went, which is less than count at the start of line.
func charsBack(line []rune, col, count int) (int, int) {
	n := 0
	for ; n < count && col > 0; n++ {
		col = charStart(line, col-1)
	}
	return col, n
}

// classAt is cls of the character col is in: its first rune decides, so a
// mark on a space is a blank and a mark on a letter is part of its word.
func classAt(line []rune, col int, cls func(rune) int) int {
	return cls(line[charStart(line, col)])
}

// blankAt reports whether the character col is in is a blank (isBlank).
func blankAt(line []rune, col int) bool {
	return isBlank(line[charStart(line, col)])
}

// afterChar returns the exclusive end of a range whose last character is at
// pos: the column after that character and its composing characters.
func afterChar(buffer Buffer, pos Position) Position {
	return Position{Row: pos.Row, Col: charEnd(buffer.GetLineRunes(pos.Row), pos.Col)}
}

// lastRuneOf returns pos moved to the last rune of its character, for code
// that takes an inclusive end (a visual selection's) rune by rune. A pos on
// a line break is returned as it is.
func lastRuneOf(buffer Buffer, pos Position) Position {
	return Position{Row: pos.Row, Col: afterChar(buffer, pos).Col - 1}
}

// charCount returns how many characters start in line[from:to].
func charCount(line []rune, from, to int) int {
	n := 0
	for col := from; col < min(to, len(line)); col = charEnd(line, col) {
		n++
	}
	return n
}
