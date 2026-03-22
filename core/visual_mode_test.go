package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Visual mode (character-wise) ---

// TestVisualModeEnterExit tests entering and exiting character-wise visual mode.
func TestVisualModeEnterExit(t *testing.T) {
	t.Run("v enters visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		assert.True(t, e.IsVisualMode())
	})

	t.Run("Escape exits visual mode to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v while in visual mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'v')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V switches from visual to visual-line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'V')
		assert.True(t, e.IsVisualLineMode())
	})
}

// TestVisualModeDelete tests 'd'/'x' in character-wise visual mode.
func TestVisualModeDelete(t *testing.T) {
	t.Run("d deletes single selected char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'd')
		assert.Equal(t, "ello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v+l+l then d deletes 3 chars", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'd')
		assert.Equal(t, "lo world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+$ then d deletes to end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', '$', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+w then d deletes up to and including first char of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w', 'd')
		// w is a plain motion in visual mode: cursor lands on 'w' of "world" (col 6),
		// so the inclusive selection covers "hello w" (cols 0–6).
		assert.Equal(t, "orld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeDeleteMultiLine tests multi-line deletion in character-wise visual mode.
func TestVisualModeDeleteMultiLine(t *testing.T) {
	t.Run("v+j then d deletes from cursor to same col on next line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'v', 'j', 'd')
		// selection: {0,0}→{1,0}; deletes "first" from row 0 and 's' from row 1, then merges
		assert.Equal(t, "econd", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeYank tests 'y' in character-wise visual mode.
func TestVisualModeYank(t *testing.T) {
	t.Run("y yanks selected text to clipboard", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'y') // select "hello"
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, "hello world", content(e)) // content unchanged
	})

	t.Run("y with $ selects to end of line", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		keys(e, 'v', '$', 'y')
		assert.Equal(t, "hello", cb.content)
	})
}

// TestVisualModeChange tests 'c' in character-wise visual mode.
func TestVisualModeChange(t *testing.T) {
	t.Run("c deletes selection and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c') // select "hello"
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c replaces selection", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c')
		keys(e, 'h', 'i')
		assert.Equal(t, "hi world", content(e))
	})
}

// TestVisualModeTextObjects tests text object selections in character-wise visual mode.
func TestVisualModeTextObjects(t *testing.T) {
	// viw / vaw

	t.Run("viw selects inside word from start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("viw selects inside word from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw selects around word including trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'a', 'w', 'd')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw on middle word includes leading space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'v', 'a', 'w', 'd')
		assert.Equal(t, "one three", content(e))
	})

	// vip / vap

	t.Run("vip switches to visual line mode", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("vip then d deletes paragraph", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vap then d deletes paragraph and surrounding blanks", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\n\nfoo")
		keys(e, 'v', 'a', 'p', 'd')
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip from mid-paragraph selects whole paragraph", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\n\nfoo")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip on blank line selects just the blank line", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "hello\nworld", content(e))
	})
}

// --- Visual line mode ---

// TestVisualLineModeEnterExit tests entering and exiting visual line mode.
func TestVisualLineModeEnterExit(t *testing.T) {
	t.Run("V enters visual line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("Escape exits to normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V while in visual-line mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'V')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v switches from visual-line to character-wise visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'v')
		assert.True(t, e.IsVisualMode())
	})
}

// TestVisualLineModeDelete tests 'd' in visual line mode.
func TestVisualLineModeDelete(t *testing.T) {
	t.Run("d deletes current line (single line buffer)", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("d deletes current line (multi-line buffer)", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'd')
		assert.Equal(t, "second\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j then d deletes two lines", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'd')
		assert.Equal(t, "third", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j+j then d deletes three lines", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'V', 'j', 'j', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("d from last line moves cursor to previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'V', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualLineModeYank tests 'y' in visual line mode.
func TestVisualLineModeYank(t *testing.T) {
	t.Run("y yanks current line with trailing newline", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'V', 'y')
		assert.Equal(t, "hello\n", cb.content)
		assert.Equal(t, "hello\nworld", content(e)) // unchanged
	})

	t.Run("V+j then y yanks two lines", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, 'V', 'j', 'y')
		assert.Equal(t, "one\ntwo\n", cb.content)
		assert.Equal(t, "one\ntwo\nthree", content(e)) // unchanged
	})
}

// TestVisualLineModeChange tests 'c' in visual line mode.
func TestVisualLineModeChange(t *testing.T) {
	t.Run("c deletes selected lines and enters insert mode", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		assert.Equal(t, "third", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c inserts before remaining content", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		// Cursor is at (0,0) of "third"; typing inserts before it on the same line.
		keys(e, 'n', 'e', 'w')
		assert.Equal(t, "newthird", content(e))
	})
}

// TestVisualModeMovementSequences tests that navigation keys work after shared-motion keys.
func TestVisualModeMovementSequences(t *testing.T) {
	t.Run("v+j+l: j then l both move cursor", func(t *testing.T) {
		e := newTestEditor("abcd\nefgh")
		keys(e, 'v', 'j') // enter visual, move down
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		keys(e, 'l') // move right
		assert.Equal(t, Position{1, 1}, cursorPos(e))
	})
	t.Run("v+w: word forward moves cursor to start of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
	t.Run("v+w+w: second w advances to start of second next word", func(t *testing.T) {
		e := newTestEditor("hello world foo")
		keys(e, 'v', 'w', 'w')
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})
	t.Run("v+e: word end moves to last char of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("v+b: word backward from middle moves to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'v', 'b') // cursor at col 3, then v+b
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
	t.Run("V+j: j in visual line mode extends selection", func(t *testing.T) {
		e := newTestEditor("line1\nline2\nline3")
		keys(e, 'V', 'j', 'd') // visual line, extend down, delete 2 lines
		assert.Equal(t, "line3", content(e))
	})
	t.Run("V+w: w moves cursor forward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world\nfoo")
		keys(e, 'V', 'w') // enter visual line, move forward one word
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assert.True(t, e.IsVisualLineMode())
	})
	t.Run("V+e: e moves cursor to word end in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'V', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("V+b: b moves cursor word backward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'V', 'b') // cursor at col 8 (mid "world"), then V+b → start of "world"
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}
