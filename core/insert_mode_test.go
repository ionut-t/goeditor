package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Entering insert mode ---

// TestInsertBefore tests 'i' — enter insert mode before the cursor.
func TestInsertBefore(t *testing.T) {
	t.Run("enters insert mode at cursor position", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-word inserts before cursor char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'i')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		keys(e, 'X')
		assert.Equal(t, "heXllo", content(e))
	})
}

// TestInsertAtFirstNonBlank tests 'I' — enter insert mode at first non-blank.
func TestInsertAtFirstNonBlank(t *testing.T) {
	t.Run("jumps to first non-blank then enters insert mode", func(t *testing.T) {
		e := newTestEditor("   hello")
		keys(e, 'I')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("no indent: inserts at col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'I')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestAppendAfter tests 'a' — enter insert mode after the cursor.
func TestAppendAfter(t *testing.T) {
	t.Run("cursor advances one right before inserting", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'a')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("typing after 'a' appends correctly", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'a')
		keys(e, '!')
		assert.Equal(t, "h!ello", content(e))
	})

	t.Run("'a' at end of line positions cursor past last char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'a')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 5}, cursorPos(e)) // col = lineLen (after last char)
	})
}

// TestAppendAtLineEnd tests 'A' — enter insert mode at end of line.
func TestAppendAtLineEnd(t *testing.T) {
	t.Run("moves to after last char then enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A')
		assertInsertMode(t, e)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("typing after 'A' appends to end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A')
		keys(e, '!')
		assert.Equal(t, "hello!", content(e))
	})
}

// TestOpenLineBelow tests 'o' — open new line below current, enter insert mode.
func TestOpenLineBelow(t *testing.T) {
	t.Run("inserts blank line below current line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'o')
		assertInsertMode(t, e)
		assert.Equal(t, "hello\n", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("between two lines inserts in the middle", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'o')
		assertInsertMode(t, e)
		assert.Equal(t, "first\n\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("typing after 'o' writes to new line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'o')
		keys(e, 'n', 'e', 'w')
		assert.Equal(t, "first\nnew\nsecond", content(e))
	})
}

// TestOpenLineAbove tests 'O' — open new line above current, enter insert mode.
func TestOpenLineAbove(t *testing.T) {
	t.Run("inserts blank line above current line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'O')
		assertInsertMode(t, e)
		assert.Equal(t, "\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("between two lines inserts above cursor line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'O')
		assertInsertMode(t, e)
		assert.Equal(t, "first\n\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("typing after 'O' writes to new line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'O')
		keys(e, 'n', 'e', 'w')
		assert.Equal(t, "first\nnew\nsecond", content(e))
	})
}

// --- Typing in insert mode ---

// TestInsertTyping tests character insertion while in insert mode.
func TestInsertTyping(t *testing.T) {
	t.Run("typing inserts at cursor and advances it", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i', 'X')
		assert.Equal(t, "Xhello", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("multiple characters are inserted in order", func(t *testing.T) {
		e := newTestEditor("z")
		keys(e, 'i', 'a', 'b', 'c')
		assert.Equal(t, "abcz", content(e))
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("inserting mid-line pushes rest of line right", func(t *testing.T) {
		e := newTestEditor("world")
		keys(e, 'i', 'h', 'e', 'l', 'l', 'o', ' ')
		assert.Equal(t, "hello world", content(e))
	})
}

// TestInsertBackspace tests Backspace in insert mode.
func TestInsertBackspace(t *testing.T) {
	t.Run("deletes character before cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A') // col=5
		backspace(e)
		assert.Equal(t, "hell", content(e))
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("at col 0 merges with previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'i') // move to row 1, insert mode at col 0
		backspace(e)
		assert.Equal(t, "firstsecond", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e)) // cursor at join point
	})

	t.Run("at start of buffer does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i')
		backspace(e)
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestInsertEnter tests Enter in insert mode.
func TestInsertEnter(t *testing.T) {
	t.Run("splits line at cursor", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'i') // cursor at col 6 (start of "world"), insert mode
		enter(e)
		assert.Equal(t, "hello \nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at start of line inserts blank line above", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i')
		enter(e)
		assert.Equal(t, "\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at end of line adds empty line below", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A')
		enter(e)
		assert.Equal(t, "hello\n", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestInsertTab tests Tab in insert mode.
func TestInsertTab(t *testing.T) {
	t.Run("inserts a tab character at cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i')
		tab(e)
		assert.Equal(t, "\thello", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})
}

// TestInsertEscape tests that Escape returns to normal mode.
func TestInsertEscape(t *testing.T) {
	t.Run("escape exits insert mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'i')
		assertInsertMode(t, e)
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("content is preserved after escape", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A')
		keys(e, ' ', 'w', 'o', 'r', 'l', 'd')
		escape(e)
		assert.Equal(t, "hello world", content(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("undo after insert+escape restores original content", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'A')
		keys(e, '!')
		escape(e)
		assert.Equal(t, "hello!", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})
}
