package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeleteChar tests 'x' — delete character under cursor.
func TestDeleteChar(t *testing.T) {
	t.Run("middle of word", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("last char on line clears it", func(t *testing.T) {
		e := newTestEditor("a")
		keys(e, 'x')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteLine tests 'dd' — delete current line.
func TestDeleteLine(t *testing.T) {
	t.Run("single line becomes empty", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("first of two lines", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'd', 'd')
		assert.Equal(t, "second", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("last line is removed, cursor moves to previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'd', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2dd", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '2', 'd', 'd')
		assert.Equal(t, "three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteWord tests 'dw' — delete to start of next word (exclusive motion).
func TestDeleteWord(t *testing.T) {
	t.Run("full first word including trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'w')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("mid-word deletes to start of next word including space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'd', 'w')
		// from col 2, MoveWordForward lands on col 6 ('w'); deletes "llo " → "heworld"
		assert.Equal(t, "heworld", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("count: 2dw", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'd', 'w')
		assert.Equal(t, "three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteWordBackward tests 'db' — delete to start of previous word.
func TestDeleteWordBackward(t *testing.T) {
	t.Run("from end of word deletes back to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		// 'e' moves to end of first word (col 4)
		keys(e, 'e', 'd', 'b')
		assert.Equal(t, "o world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid second word", func(t *testing.T) {
		e := newTestEditor("hello world")
		// w→col6, l→col7, l→col8; db deletes back to start of "world" (col6)
		keys(e, 'w', 'l', 'l', 'd', 'b')
		assert.Equal(t, "hello rld", content(e))
	})
}

// TestDeleteToWordEnd tests 'de' — delete to end of word (inclusive).
func TestDeleteToWordEnd(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'e')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'd', 'e')
		assert.Equal(t, "h world", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("count: 2de", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'd', 'e')
		assert.Equal(t, " three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteToEndOfLine tests 'd$' — delete to end of line.
func TestDeleteToEndOfLine(t *testing.T) {
	t.Run("from start clears line", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.HandleKey(KeyEvent{Rune: 'd'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-line", func(t *testing.T) {
		e := newTestEditor("hello world")
		// 'w' moves to col 6 ('w' of "world")
		keys(e, 'w')
		e.HandleKey(KeyEvent{Rune: 'd'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "hello ", content(e))
		// Cursor stays at the column where deletion started
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestDeleteToEndOfBuffer tests 'dG' — delete to end of buffer.
func TestDeleteToEndOfBuffer(t *testing.T) {
	t.Run("from first line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		e.HandleKey(KeyEvent{Rune: 'd'})
		e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from second line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j')
		e.HandleKey(KeyEvent{Rune: 'd'})
		e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "one", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteInsideWord tests 'diw' — delete inside word (text object).
func TestDeleteInsideWord(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'i', 'w')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'd', 'i', 'w')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteAroundWord tests 'daw' — delete around word (includes surrounding space).
func TestDeleteAroundWord(t *testing.T) {
	t.Run("first word eats trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'a', 'w')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("middle word eats leading space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'd', 'a', 'w')
		assert.Equal(t, "one three", content(e))
	})
}

// TestDeleteCharBefore tests 'X' — delete character before cursor.
func TestDeleteCharBefore(t *testing.T) {
	t.Run("deletes character to the left of cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'X') // cursor at col 2; X deletes col 1 ('e')
		assert.Equal(t, "hllo", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("at col 0 does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'X')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("X at end of line deletes last char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'X') // col 4; X deletes col 3 ('l')
		assert.Equal(t, "helo", content(e))
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})
}

// TestDeleteToEndOfLineShortcut tests 'D' — shortcut for d$.
func TestDeleteToEndOfLineShortcut(t *testing.T) {
	t.Run("from start of line clears it", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'D')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-line deletes to end", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'D') // cursor at col 6; D deletes "world"
		assert.Equal(t, "hello ", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("on empty line does nothing", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'D') // move to blank line; D is a no-op
		assert.Equal(t, "hello\n\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestUndoDeleteLine verifies that undo restores both content and cursor position.
func TestUndoDeleteLine(t *testing.T) {
	t.Run("undo dd on last line restores cursor to deleted row", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'd', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))

		keys(e, 'u')
		assert.Equal(t, "first\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("undo dd on first line restores cursor to that row", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'd', 'd')
		assert.Equal(t, "second", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))

		keys(e, 'u')
		assert.Equal(t, "first\nsecond", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}
