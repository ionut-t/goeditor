package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMoveLeft tests 'h' — move left one character.
// At column 0 it wraps to the end of the previous line (MoveLeftOrUp).
func TestMoveLeft(t *testing.T) {
	t.Run("moves left within line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'h')
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("at col 0 wraps to end of previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'h') // j → row 1 col 0; h wraps to row 0 end
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("at start of buffer stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'h')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 3h", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '3', 'h') // $ → col 4; 3h → col 1
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})
}

// TestMoveRight tests 'l' — move right one character.
// At end of line it wraps to start of the next line (MoveRightOrDown).
func TestMoveRight(t *testing.T) {
	t.Run("moves right within line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l')
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("at end of line wraps to start of next line", func(t *testing.T) {
		// "a" has lineLen=1: first 'l' moves to col=1 (=lineLen), second wraps down.
		e := newTestEditor("a\nb")
		keys(e, 'l', 'l')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at end of last line stays at lineLen", func(t *testing.T) {
		// 'l' can advance to col=lineLen (one past last char), but no further on the last line.
		e := newTestEditor("hello")
		keys(e, '$', 'l', 'l') // $→col4; l→col5; l→stays col5 (ErrEndOfBuffer)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("count: 3l", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', 'l')
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})
}

// TestMoveDown tests 'j' — move down one line.
func TestMoveDown(t *testing.T) {
	t.Run("moves down one line", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'j')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at last line stays put", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'j')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("preserves column when moving down", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		setWidth(e, 80)
		keys(e, '$', 'j') // $→col4 (Preferred=4); j→col4 on "world"
		assert.Equal(t, Position{1, 4}, cursorPos(e))
	})

	t.Run("clamps column to shorter line", func(t *testing.T) {
		e := newTestEditor("hello\nhi")
		setWidth(e, 80)
		// $ → col 4, Preferred=4; j → target col 4, lineLen("hi")=2, clamps to lineLen=2
		keys(e, '$', 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("count: 2j", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '2', 'j')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})
}

// TestMoveUp tests 'k' — move up one line.
func TestMoveUp(t *testing.T) {
	t.Run("moves up one line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at first line stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2k", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j', '2', 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveToLineStart tests '0' — move to column 0.
func TestMoveToLineStart(t *testing.T) {
	t.Run("moves to col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '0')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already at col 0 stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '0')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveToLineEnd tests '$' — move to last character of line.
func TestMoveToLineEnd(t *testing.T) {
	t.Run("moves to last char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("empty line stays at col 0", func(t *testing.T) {
		e := newTestEditor("")
		keys(e, '$')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already at end stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '$')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
}

// TestMoveToFirstNonBlank tests '^' — move to first non-whitespace character.
func TestMoveToFirstNonBlank(t *testing.T) {
	t.Run("skips leading spaces", func(t *testing.T) {
		e := newTestEditor("   hello")
		keys(e, '^')
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("no leading spaces goes to col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '^')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("all spaces goes to col 0", func(t *testing.T) {
		e := newTestEditor("   ")
		keys(e, '^')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveToBufferStart tests 'g' — move to first line.
func TestMoveToBufferStart(t *testing.T) {
	t.Run("moves to row 0 col 0", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j', 'g')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already on first line stays at row 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'g')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveToBufferEnd tests 'G' — move to last line (first non-blank).
func TestMoveToBufferEnd(t *testing.T) {
	t.Run("moves to last line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'G')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("last line with leading spaces lands on first non-blank", func(t *testing.T) {
		e := newTestEditor("one\n   indented")
		keys(e, 'G')
		assert.Equal(t, Position{1, 3}, cursorPos(e))
	})

	t.Run("single line buffer stays at row 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'G')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveWordForward tests 'w' — move to start of next word.
func TestMoveWordForward(t *testing.T) {
	t.Run("moves to start of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("from mid-word jumps to next word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("count: 2w", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'w')
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("wraps to next line", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'w')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestMoveWordBackward tests 'b' — move to start of current or previous word.
func TestMoveWordBackward(t *testing.T) {
	t.Run("from mid-word jumps to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'l', 'l', 'b') // col 8; b → col 6
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("from start of word jumps to previous word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'b') // col 6; b → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2b", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'w', '2', 'b') // col 8; 2b → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveWordToEnd tests 'e' — move to end of current or next word.
func TestMoveWordToEnd(t *testing.T) {
	t.Run("from start of word moves to end of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("from end of word jumps to end of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'e', 'e')
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})

	t.Run("count: 2e", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'e')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestMoveParagraphForward tests '}' — move to the next blank line (paragraph boundary).
// Like Vim: from a non-blank line, lands on the next blank line (or last line if none).
// From a blank line, skips the blank gap first, then lands on the following blank line.
func TestMoveParagraphForward(t *testing.T) {
	t.Run("lands on the blank line between paragraphs", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, '}') // from row 0: skip "one","two", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from within paragraph lands on blank separator", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'j', '}') // row 1 → skip "two", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from blank line skips blank gap and lands on next blank", func(t *testing.T) {
		e := newTestEditor("one\n\n\nthree\nfour\n\nsix")
		keys(e, 'j', '}') // row 1 (blank): skip blanks (1,2), skip non-blank (3,4), land on row 5 (blank)
		assert.Equal(t, Position{5, 0}, cursorPos(e))
	})

	t.Run("at last line stays put", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, 'G', '}') // G → last line (row 3); } is a no-op
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})

	t.Run("no blank line: lands on last line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '}') // skip non-blank until last line; no blank found
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("two } jumps land on successive blank lines", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, '}', '}') // row 0 → row 1 (blank) → row 3 (blank)
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})

	t.Run("count: 2} jumps two paragraph boundaries", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, '2', '}') // row 0 → row 1 (blank) → row 3 (blank)
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})
}

// TestMoveParagraphBackward tests '{' — move to the previous blank line (paragraph boundary).
// Like Vim: from a non-blank line, lands on the previous blank line (or row 0 if none).
// From a blank line, skips the blank gap first, then lands on the preceding blank line.
func TestMoveParagraphBackward(t *testing.T) {
	t.Run("lands on the blank line between paragraphs", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'G', '{') // from row 4: skip "four","three", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from start of paragraph lands on blank separator", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'j', 'j', 'j', '{') // row 3 → skip "three", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from blank line skips blank gap and lands on previous blank", func(t *testing.T) {
		e := newTestEditor("one\n\nthree\nfour\n\nsix")
		keys(e, 'G', '{', '{') // row 5 (blank) → skip blank, skip "four","three", land row 1 (blank)
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("no blank before first paragraph: lands on row 0", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, 'j', '{') // row 1 → skip "two","one", hit row 0 (start of buffer)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at first line stays put", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, '{') // row 0; { is a no-op
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("two { jumps land on successive blank lines", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, 'G', '{', '{') // row 4 → row 3 (blank) → row 1 (blank)
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}
