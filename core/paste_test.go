package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPasteCharacterWise tests 'p' after a character-wise yank.
// Content is inserted AFTER the cursor char — matching Vim's 'p' behaviour.
// The cursor lands on the last character of the pasted text.
func TestPasteCharacterWise(t *testing.T) {
	t.Run("yw then p inserts after cursor char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'w') // yank "hello " to clipboard; cursor stays at col 0 ('h')
		assert.Equal(t, "hello ", cb.content)
		keys(e, 'p') // insert "hello " after 'h' → "hhello ello world"
		assert.Equal(t, "hhello ello world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("ye then p inserts word after cursor char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello" (no trailing space); cursor at col 0
		assert.Equal(t, "hello", cb.content)
		keys(e, 'p') // insert "hello" after 'h' → "hhelloello world"
		assert.Equal(t, "hhelloello world", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("p mid-line inserts after cursor char", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'w')      // move to col 6 ('w' of "world")
		keys(e, 'p')      // insert "hello" after 'w' → "hello whelloorld"
		assert.Equal(t, "hello whelloorld", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})
}

// TestPasteCharacterWiseOnEmptyLine tests 'p' and 'P' on an empty line.
// On an empty line there is no character to insert after, so the text is inserted at column 0.
func TestPasteCharacterWiseOnEmptyLine(t *testing.T) {
	t.Run("p on empty line inserts text at col 0", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\n\nworld")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'j')      // move to empty line (row 1)
		keys(e, 'p')      // paste on empty line
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 5}, cursorPos(e))
	})

	t.Run("P on empty line inserts text at col 0", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\n\nworld")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'j')      // move to empty line (row 1)
		keys(e, 'P')      // paste before on empty line
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 5}, cursorPos(e))
	})
}

// TestPasteLinewise tests 'p' after a line-wise yank ('yy').
// Linewise paste inserts the yanked line below the current line and moves the cursor
// to the start of the newly inserted line — matching Vim's behaviour.
func TestPasteLinewise(t *testing.T) {
	t.Run("yy then p on single line duplicates it below", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy then p on first of two lines inserts copy between them", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "first\nfirst\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy on second line then p duplicates it below", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y', 'p')
		assert.Equal(t, "first\nsecond\nsecond", content(e))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("cursor column is irrelevant: paste always goes below current line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, '$', 'y', 'y', 'p') // cursor at end of line 0; should still paste below row 0
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("repeated p pastes on successive lines", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'p', 'p') // yy then p, p
		assert.Equal(t, "hello\nhello\nhello", content(e))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("count: 3p pastes three copies below", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', '3', 'p')
		assert.Equal(t, "hello\nhello\nhello\nhello", content(e))
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})
}

// TestPasteCharacterWiseBefore tests 'P' after a character-wise yank.
// Content is inserted at the cursor position (same column) — cursor moves right by the pasted length.
func TestPasteCharacterWiseBefore(t *testing.T) {
	t.Run("ye then P inserts word at cursor position", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello" (no trailing space)
		assert.Equal(t, "hello", cb.content)
		keys(e, 'P')
		assert.Equal(t, "hellohello world", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("P mid-line inserts at cursor column", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'w')      // move to col 6 ("world")
		keys(e, 'P')      // paste "hello" at col 6
		assert.Equal(t, "hello helloworld", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})
}

// TestPasteLinewiseBefore tests 'P' after a line-wise yank ('yy').
// Linewise P inserts the yanked line above the current line; cursor stays at the newly inserted line.
func TestPasteLinewiseBefore(t *testing.T) {
	t.Run("yy then P on single line inserts copy above", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'P')
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("yy then P on second line inserts copy above it", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y', 'P') // yank "second", paste above row 1
		assert.Equal(t, "first\nsecond\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy then P on first of two lines inserts copy at top", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'y', 'y', 'P') // yank "first", paste above row 0
		assert.Equal(t, "first\nfirst\nsecond", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("repeated P pastes accumulate above original line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'P', 'P') // yy then P, P
		assert.Equal(t, "hello\nhello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 3P pastes three copies above", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', '3', 'P')
		assert.Equal(t, "hello\nhello\nhello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestPasteOverSelection tests 'p' in visual character mode (v).
// The selected text is deleted and the clipboard is pasted in its place.
func TestPasteOverSelection(t *testing.T) {
	t.Run("p over character-wise selection replaces it", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'v', 'e')
		keys(e, 'p')
		assert.Equal(t, "bye world", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("p over mid-line selection replaces selected word", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'w', 'v', 'e') // move to "world", select it
		keys(e, 'p')
		assert.Equal(t, "hello bye", content(e))
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("p over single char at end of line", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, '$', 'v', 'p') // select last char 'd', paste
		assert.Equal(t, "hello worlbye", content(e))
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})
}

// TestPasteBeforeOverSelection tests 'P' in visual modes — identical behaviour to 'p'.
func TestPasteBeforeOverSelection(t *testing.T) {
	t.Run("P over character-wise selection replaces it", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'v', 'e', 'P')
		assert.Equal(t, "bye world", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("VP replaces line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'j', 'y', 'y') // yank "world\n" from row 1
		keys(e, 'k', 'V', 'P') // select row 0, paste
		assert.Equal(t, "world\nworld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestPasteLinewiseOverSelection tests 'p' in visual line mode (V).
// The selected lines are deleted and the clipboard is pasted in their place.
func TestPasteLinewiseOverSelection(t *testing.T) {
	t.Run("Vp replaces last line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'y', 'y')      // yank "hello\n" from row 0
		keys(e, 'j', 'V', 'p') // select row 1, paste
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("Vp replaces first line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'j', 'y', 'y') // yank "world\n" from row 1
		keys(e, 'k', 'V', 'p') // select row 0, paste
		assert.Equal(t, "world\nworld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e)) // cursor on the pasted line
	})

	t.Run("Vp puts pasted content at the selected row, not below it", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond\nthird")
		keys(e, 'j', 'j', 'y', 'y') // yank "third\n" from row 2
		keys(e, 'k', 'k', 'V', 'p') // select row 0, paste
		assert.Equal(t, "third\nsecond\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("Vjp replaces multi-line selection with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, 'y', 'y')           // yank "one\n" from row 0
		keys(e, 'j', 'V', 'j', 'p') // select rows 1+2, paste
		assert.Equal(t, "one\none", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}
