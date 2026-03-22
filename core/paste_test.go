package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPasteCharacterWise tests 'p' after a character-wise yank.
// Content is inserted at the cursor position; cursor moves right by the pasted length.
func TestPasteCharacterWise(t *testing.T) {
	t.Run("yw then p inserts at cursor", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'w') // yank "hello " to clipboard
		assert.Equal(t, "hello ", cb.content)
		keys(e, 'p')
		assert.Equal(t, "hello hello world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("ye then p inserts word at cursor", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello" (no trailing space)
		assert.Equal(t, "hello", cb.content)
		keys(e, 'p')
		assert.Equal(t, "hellohello world", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("p mid-line inserts at cursor column", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'w')      // move to col 6 ("world")
		keys(e, 'p')      // paste "hello" at col 6
		assert.Equal(t, "hello helloworld", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})
}

// TestPasteLinewise tests 'p' after a line-wise yank ('yy').
// The yanked text (with trailing newline) is inserted at the cursor position, creating a new line.
// Note: unlike Vim's 'p' which pastes below the current line, this editor inserts at the cursor column.
// TODO: make this more Vim-like by always pasting on a new line below the current line, regardless of cursor column.
func TestPasteLinewise(t *testing.T) {
	t.Run("yy then p on single line duplicates it", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "hello\nhello", content(e))
		// Cursor moves right by len("hello\n")=6; stops at end of line 0.
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("yy then p on first of two lines duplicates first", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "first\nfirst\nsecond", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("yy on second line then p duplicates second line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y', 'p')
		assert.Equal(t, "first\nsecond\nsecond", content(e))
		// Cursor moves right by len("second\n")=7; stops at end of line 1.
		assert.Equal(t, Position{1, 6}, cursorPos(e))
	})
}
