package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestYankLine tests 'yy' — yank current line (line-wise, includes trailing newline).
func TestYankLine(t *testing.T) {
	t.Run("single line", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y')
		assert.Equal(t, "hello\n", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("second of two lines", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y')
		assert.Equal(t, "second\n", cb.content)
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("count: 2yy", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, '2', 'y', 'y')
		assert.Equal(t, "one\ntwo\n", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestYankWord tests 'yw' — yank to start of next word (exclusive).
func TestYankWord(t *testing.T) {
	t.Run("full first word including trailing space", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'w')
		assert.Equal(t, "hello ", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("mid-word yanks to start of next word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'l', 'l', 'y', 'w')
		// from col 2, MoveWordForward lands on col 6; exclusive → yank "llo "
		assert.Equal(t, "llo ", cb.content)
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("count: 2yw", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one two three")
		keys(e, '2', 'y', 'w')
		assert.Equal(t, "one two ", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestYankWordBackward tests 'yb' — yank to start of previous word (exclusive).
func TestYankWordBackward(t *testing.T) {
	t.Run("from end of word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		// 'e' moves to col 4 (end of "hello")
		keys(e, 'e', 'y', 'b')
		// MoveWordBackward goes to col 0; exclusive → yank "hell"
		assert.Equal(t, "hell", cb.content)
		// yb moves cursor to the start of the yanked range (Vim behaviour)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid second word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		// w→col6, l→col7, l→col8
		keys(e, 'w', 'l', 'l', 'y', 'b')
		// MoveWordBackward from col 8 → col 6; exclusive → yank "wo"
		assert.Equal(t, "wo", cb.content)
		// yb moves cursor to the start of the yanked range (Vim behaviour)
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestYankToWordEnd tests 'ye' — yank to end of word (inclusive).
func TestYankToWordEnd(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e')
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("mid-word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'l', 'y', 'e')
		// MoveWordToEnd from col 1 → col 4; inclusive → yank "ello"
		assert.Equal(t, "ello", cb.content)
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("count: 2ye", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one two three")
		keys(e, '2', 'y', 'e')
		// MoveWordToEnd count=2 from col 0 → col 6; inclusive → yank "one two"
		assert.Equal(t, "one two", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestYankToEndOfLine tests 'y$' — yank to end of line.
func TestYankToEndOfLine(t *testing.T) {
	t.Run("from start", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		e.HandleKey(KeyEvent{Rune: 'y'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "hello world", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-line", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'w')
		e.HandleKey(KeyEvent{Rune: 'y'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "world", cb.content)
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestYankInsideWord tests 'yiw' — yank inside word.
func TestYankInsideWord(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'i', 'w')
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-word", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'l', 'l', 'y', 'i', 'w')
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestYankAroundWord tests 'yaw' — yank around word (includes surrounding space).
func TestYankAroundWord(t *testing.T) {
	t.Run("first word includes trailing space", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'a', 'w')
		assert.Equal(t, "hello ", cb.content)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("middle word includes trailing space", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one two three")
		keys(e, 'w', 'y', 'a', 'w')
		assert.Equal(t, "two ", cb.content)
	})
}
