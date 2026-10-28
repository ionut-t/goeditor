package core

import (
	"strings"
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

	// Vim 9.2: the last word stops at the end of its line, an empty line is
	// deleted linewise, and so is a delete over several lines from the indent
	// to the end of a line.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "last word of a line", content: "abc def\nab", keys: seq("wdw"), want: Position{0, 3}, text: "abc \nab"},
		{name: "last word of a line then .", content: "abc def\nab", keys: seq("wdw."), want: Position{0, 2}, text: "abc\nab"},
		{name: "last word of a line then jj", content: "abc def\nab\nabcdefghij", keys: seq("wdwjj"), want: Position{2, 3}},
		{name: "last word of a line then j.", content: "abc def\nab", keys: seq("wdwj."), want: Position{1, 0}, text: "abc \na"},
		{name: "2dw crosses the line", content: "abc def\nab cd", keys: seq("w2dw"), want: Position{0, 4}, text: "abc cd"},
		{name: "d2w crosses the line", content: "abc def\nab cd", keys: seq("wd2w"), want: Position{0, 4}, text: "abc cd"},
		{name: "3dw ends on a later line's last word", content: "abc def\nab cd\nxy", keys: seq("w3dw"), want: Position{0, 3}, text: "abc \nxy"},
		{name: "d3w ends on a later line's last word", content: "abc def\nab cd\nxy", keys: seq("wd3w"), want: Position{0, 3}, text: "abc \nxy"},
		{name: "3dw then .", content: "abc def\nab cd\nxy z\nq r\ns", keys: seq("w3dwj."), want: Position{1, 2}, text: "abc \nxy \ns"},
		{name: "3dw then . on the last line", content: "a b c d\ne\nf g h i", keys: seq("3dwG."), want: Position{2, 0}, text: "d\ne\ni"},
		{name: "count past the end of the buffer", content: "abc def\nab", keys: seq("w5dw"), want: Position{0, 3}, text: "abc "},
		{name: "next line empty", content: "abc def\n\nx", keys: seq("wdw"), want: Position{0, 3}, text: "abc \n\nx"},
		{name: "2dw onto an empty line", content: "abc def\n\nx", keys: seq("w2dw"), want: Position{0, 3}, text: "abc \nx"},
		{name: "trailing blanks", content: "abc def  \nab", keys: seq("wdw"), want: Position{0, 3}, text: "abc \nab"},
		{name: "on trailing blanks", content: "abc   \nx", keys: seq("3ldw"), want: Position{0, 2}, text: "abc\nx"},
		{name: "a line's only word", content: "abc\ndef", keys: seq("dw"), want: Position{0, 0}, text: "\ndef"},
		{name: "a line's only word then .", content: "abc\ndef", keys: seq("dw."), want: Position{0, 0}, text: "def"},
		{name: "last line", content: "ab\nabc def", keys: seq("jwdw"), want: Position{1, 3}, text: "ab\nabc "},
		{name: "only word of the last line", content: "ab\nabc", keys: seq("jdw"), want: Position{1, 0}, text: "ab\n"},
		{name: "from inside the word", content: "abc def\nab", keys: seq("wldw"), want: Position{0, 4}, text: "abc d\nab"},
		{name: "from inside the word then .", content: "abc def\nab def", keys: seq("wldwjl."), want: Position{1, 4}, text: "abc d\nab de"},
		{name: "last character", content: "abc def\nab", keys: seq("$dw"), want: Position{0, 5}, text: "abc de\nab"},
		{name: "empty line is deleted linewise", content: "abc\n\ndef", keys: seq("jdw"), want: Position{1, 0}, text: "abc\ndef"},
		{name: "empty line then .", content: "abc\n\n\ndef", keys: seq("jdw."), want: Position{1, 0}, text: "abc\ndef"},
		{name: "empty line above an indented line", content: "abc\n\n  def", keys: seq("jdw"), want: Position{1, 2}, text: "abc\n  def"},
		{name: "empty line above an empty last line", content: "abc\n\n\n", keys: seq("jdw"), want: Position{1, 0}, text: "abc\n"},
		{name: "empty last line does nothing", content: "abc\n\n", keys: seq("jdw"), want: Position{1, 0}, text: "abc\n"},
		{name: "2dw over two empty lines", content: "abc\n\n\ndef", keys: seq("j2dw"), want: Position{1, 0}, text: "abc\ndef"},
		{name: "2dw from an empty line", content: "abc\n\ndef ghi", keys: seq("j2dw"), want: Position{1, 0}, text: "abc\nghi"},
		{name: "2dw from the line start to a line end is linewise", content: "abc\ndef\nghi", keys: seq("2dw"), want: Position{0, 0}, text: "ghi"},
		{name: "2dw from the indent is linewise", content: "  abc\ndef\nghi", keys: seq("w2dw"), want: Position{0, 0}, text: "ghi"},
		{name: "2dw on the indent stays on the line", content: "  abc\ndef\nghi", keys: seq("2dw"), want: Position{0, 0}, text: "\ndef\nghi"},
		{name: "2dw to trailing blanks is linewise", content: "abc\ndef  \nghi", keys: seq("2dw"), want: Position{0, 0}, text: "ghi"},
		{name: "2dw to a later word is not linewise", content: "abc\ndef ghi", keys: seq("2dw"), want: Position{0, 0}, text: "ghi"},
		{name: "blank line", content: "   \nx", keys: seq("dw"), want: Position{0, 0}, text: "\nx"},
		{name: "2dw from a blank line", content: "   \nabc\nx", keys: seq("2dw"), want: Position{0, 0}, text: "x"},
		{name: "2dw over a blank line", content: "abc\n  \nx y", keys: seq("2dw"), want: Position{0, 0}, text: "y"},
		{name: "w2dw over a blank line", content: "abc def\n  \nx", keys: seq("w2dw"), want: Position{0, 3}, text: "abc "},
		{name: "w2dw from the indent over a blank line", content: "  abc\n  \nx", keys: seq("w2dw"), want: Position{0, 0}, empty: true},
		{name: "before a blank last line", content: "abc\n   ", keys: seq("dw"), want: Position{0, 0}, text: "\n   "},
		{name: "2dw onto a blank last line", content: "abc\n   ", keys: seq("2dw"), want: Position{0, 0}, empty: true},
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

	// Vim 9.2: an exclusive motion ending in column 0 ends at the end of the
	// line above, and is linewise if it starts in the indent.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "from column 0 is linewise", content: "abc\ndef", keys: seq("jdb"), want: Position{0, 0}, text: "def"},
		{name: "from column 0 after text", content: "x abc\ndef", keys: seq("jdb"), want: Position{0, 1}, text: "x \ndef"},
		{name: "over a blank line", content: "abc\n   \ndef", keys: seq("Gdb"), want: Position{0, 0}, text: "def"},
		{name: "from the indent", content: "abc\n  def", keys: seq("jwdb"), want: Position{0, 0}, text: "def"},
		{name: "dB from column 0", content: "abc\ndef", keys: seq("jdB"), want: Position{0, 0}, text: "def"},
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

	// Vim 9.2: past the last word e goes to the end of the buffer; a delete
	// from the indent to the end of a later line is linewise.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "past the last word", content: "foobar\n   ", keys: seq("$de"), want: Position{0, 4}, text: "fooba"},
		{name: "onto an empty last line", content: "foobar\n\n", keys: seq("$de"), want: Position{0, 4}, text: "fooba"},
		{name: "count past the end", content: "ab cd\n  ", keys: seq("l3de"), want: Position{0, 0}, text: "a"},
		{name: "2de from the indent to a line end is linewise", content: "  a\nbc", keys: seq("w2de"), want: Position{0, 0}, empty: true},
		{name: "d2E from the indent to a line end is linewise", content: "  a\nbc", keys: seq("wd2E"), want: Position{0, 0}, empty: true},
	})
}

// TestDeleteToEndOfLine tests 'd$' — delete to end of line.
func TestDeleteToEndOfLine(t *testing.T) {
	t.Run("from start clears line", func(t *testing.T) {
		e := newTestEditor("hello world")
		_ = e.HandleKey(KeyEvent{Rune: 'd'})
		_ = e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from mid-line", func(t *testing.T) {
		e := newTestEditor("hello world")
		// 'w' moves to col 6 ('w' of "world")
		keys(e, 'w')
		_ = e.HandleKey(KeyEvent{Rune: 'd'})
		_ = e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "hello ", content(e))
		// Cursor goes back onto the last character left (Vim 9.2)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})
}

// TestDeleteToEndOfBuffer tests 'dG' — delete to end of buffer.
func TestDeleteToEndOfBuffer(t *testing.T) {
	t.Run("from first line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		_ = e.HandleKey(KeyEvent{Rune: 'd'})
		_ = e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("from second line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j')
		_ = e.HandleKey(KeyEvent{Rune: 'd'})
		_ = e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "one", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: d3G deletes from cursor line down to line 3", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\nfour\nfive")
		keys(e, 'j', 'd', '3', 'G')
		assert.Equal(t, "one\nfour\nfive", content(e))
		assert.Equal(t, 1, cursorPos(e).Row)
	})

	t.Run("count: d2G deletes from cursor line up to line 2", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\nfour\nfive")
		keys(e, 'j', 'j', 'j', 'd', '2', 'G')
		assert.Equal(t, "one\nfive", content(e))
		assert.Equal(t, 1, cursorPos(e).Row)
	})

	t.Run("count before operator: 3dG", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\nfour\nfive")
		keys(e, '3', 'd', 'G')
		assert.Equal(t, "four\nfive", content(e))
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

// TestDeleteInsideParagraph tests 'dip' — delete inside paragraph (contiguous non-blank lines).
func TestDeleteInsideParagraph(t *testing.T) {
	t.Run("single paragraph — removes all lines leaving the blank separator", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		// cursor on row 0; paragraph is rows 0-1
		keys(e, 'd', 'i', 'p')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("cursor mid-paragraph still deletes the whole block", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\n\nfoo")
		keys(e, 'j', 'd', 'i', 'p') // cursor on row 1
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("last paragraph absorbs preceding newline", func(t *testing.T) {
		e := newTestEditor("foo\n\nhello\nworld")
		keys(e, 'j', 'j', 'd', 'i', 'p') // cursor on row 2
		assert.Equal(t, "foo\n", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("only content in buffer clears to empty", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'd', 'i', 'p')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("on blank line deletes the blank line", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'd', 'i', 'p') // cursor on blank row 1
		assert.Equal(t, "hello\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("on multiple consecutive blank lines deletes the whole blank block", func(t *testing.T) {
		e := newTestEditor("hello\n\n\nworld")
		keys(e, 'j', 'd', 'i', 'p') // cursor on blank row 1
		assert.Equal(t, "hello\nworld", content(e))
	})
}

// TestDeleteAroundParagraph tests 'dap' — delete around paragraph (block + surrounding blanks).
func TestDeleteAroundParagraph(t *testing.T) {
	t.Run("includes trailing blank lines", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\n\nfoo")
		// paragraph rows 0-1, trailing blanks rows 2-3
		keys(e, 'd', 'a', 'p')
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("no trailing blanks: absorbs leading blank lines instead", func(t *testing.T) {
		e := newTestEditor("foo\n\n\nhello\nworld")
		// paragraph rows 3-4, no trailing blanks; leading blanks rows 1-2
		// dap deletes rows 1-4; only "foo" remains → cursor at row 0
		keys(e, 'j', 'j', 'j', 'd', 'a', 'p') // cursor on row 3
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("no surrounding blanks: same as dip", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'd', 'a', 'p')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("on blank line between paragraphs: deletes blank + paragraph below", func(t *testing.T) {
		e := newTestEditor("foo\n\nhello\nworld")
		keys(e, 'j', 'd', 'a', 'p') // cursor on blank row 1; deletes "\nhello\nworld", leaving "foo"
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("on blank line at end: deletes blank + paragraph above", func(t *testing.T) {
		// "intro" at row 0, blank at row 1, "hello" at row 2, blank at row 3.
		// dap on row 3 absorbs the blank + the paragraph above ("hello"), leaving "intro".
		e := newTestEditor("intro\n\nhello\n")
		keys(e, 'j', 'j', 'j', 'd', 'a', 'p') // cursor on blank row 3
		assert.Equal(t, "intro", content(e))
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
		assert.Equal(t, Position{0, 5}, cursorPos(e)) // onto the last character left (Vim 9.2)
	})

	t.Run("on empty line does nothing", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'D') // move to blank line; D is a no-op
		assert.Equal(t, "hello\n\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestDeleteToBufferStart tests 'dgg' — delete from the first line up to and including the cursor line.
func TestDeleteToBufferStart(t *testing.T) {
	t.Run("dgg deletes from first line to cursor line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j') // cursor on row 2
		keys(e, 'd', 'g', 'g')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("dgg on first line deletes only the first line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'd', 'g', 'g')
		assert.Equal(t, "two\nthree", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("Escape after dg cancels without deleting", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j')
		_ = e.HandleKey(KeyEvent{Rune: 'd'})
		_ = e.HandleKey(KeyEvent{Rune: 'g'})
		escape(e)
		assert.Equal(t, "one\ntwo\nthree", content(e))
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

// TestDeleteInsideQuotes tests di"/da", di'/da', di`/da` — delete inside/around quotes.
func TestDeleteInsideQuotes(t *testing.T) {
	t.Run(`di" deletes content between double quotes`, func(t *testing.T) {
		e := newTestEditor(`say "hello" world`)
		keys(e, 'w', 'w') // cursor on 'h' inside the quotes (col 5)
		keys(e, 'd', 'i', '"')
		assert.Equal(t, `say "" world`, content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run(`da" deletes including the quote chars`, func(t *testing.T) {
		e := newTestEditor(`say "hello" world`)
		keys(e, 'w', 'w')
		keys(e, 'd', 'a', '"')
		assert.Equal(t, `say  world`, content(e))
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run(`di' deletes content between single quotes`, func(t *testing.T) {
		e := newTestEditor(`it 'works' now`)
		keys(e, 'w', 'l')
		keys(e, 'd', 'i', '\'')
		assert.Equal(t, `it '' now`, content(e))
	})

	t.Run("di` deletes content between backticks", func(t *testing.T) {
		e := newTestEditor("`cmd arg`")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', '`')
		assert.Equal(t, "``", content(e))
	})

	t.Run("cursor before quotes selects next pair", func(t *testing.T) {
		e := newTestEditor(`x "foo" y`)
		keys(e, 'd', 'i', '"')
		assert.Equal(t, `x "" y`, content(e))
	})
}

// TestDeleteInsideAnyQuote tests diq/daq — delete inside/around any quote type.
func TestDeleteInsideAnyQuote(t *testing.T) {
	t.Run("diq on double-quoted string", func(t *testing.T) {
		e := newTestEditor(`"hello"`)
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', 'q')
		assert.Equal(t, `""`, content(e))
	})

	t.Run("diq picks innermost quote", func(t *testing.T) {
		// outer " at 0,14; inner ' at 5,9; cursor inside 'bar' at col 7
		e := newTestEditor(`"foo 'bar' baz"`)
		for range 7 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', 'q')
		assert.Equal(t, `"foo '' baz"`, content(e))
	})

	t.Run("daq removes the quote chars", func(t *testing.T) {
		e := newTestEditor(`'hi'`)
		keys(e, 'l')
		keys(e, 'd', 'a', 'q')
		assert.Equal(t, ``, content(e))
	})
}

// TestDeleteInsideBrackets tests di(/da(, dib/dab and related bracket aliases.
func TestDeleteInsideBrackets(t *testing.T) {
	t.Run("di( deletes content inside parens", func(t *testing.T) {
		e := newTestEditor("foo(bar, baz)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "foo()", content(e))
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("dib inside parens", func(t *testing.T) {
		e := newTestEditor("foo(bar)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', 'b')
		assert.Equal(t, "foo()", content(e))
	})

	t.Run("dib inside square brackets", func(t *testing.T) {
		e := newTestEditor("[1, 2]")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', 'b')
		assert.Equal(t, "[]", content(e))
	})

	t.Run("dib inside curly braces", func(t *testing.T) {
		e := newTestEditor("{key: val}")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', 'b')
		assert.Equal(t, "{}", content(e))
	})

	t.Run("dib picks innermost when bracket types are nested", func(t *testing.T) {
		// cursor inside [], which is inside ()
		e := newTestEditor("([1, 2])")
		for range 3 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', 'b')
		assert.Equal(t, "([])", content(e))
	})

	t.Run("da( deletes including the parens", func(t *testing.T) {
		e := newTestEditor("foo(bar)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'd', 'a', '(')
		assert.Equal(t, "foo", content(e))
	})

	t.Run("dab inside parens", func(t *testing.T) {
		e := newTestEditor("foo(bar)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'd', 'a', 'b')
		assert.Equal(t, "foo", content(e))
	})

	t.Run("dab inside square brackets", func(t *testing.T) {
		e := newTestEditor("x[1]y")
		keys(e, 'l', 'l')
		keys(e, 'd', 'a', 'b')
		assert.Equal(t, "xy", content(e))
	})

	t.Run("dab inside curly braces", func(t *testing.T) {
		e := newTestEditor("x{a}y")
		keys(e, 'l', 'l')
		keys(e, 'd', 'a', 'b')
		assert.Equal(t, "xy", content(e))
	})

	t.Run("cursor on closing paren", func(t *testing.T) {
		e := newTestEditor("(hello)")
		keys(e, '$')
		keys(e, 'd', 'i', ')')
		assert.Equal(t, "()", content(e))
	})

	t.Run("nested parens – deletes innermost from inside inner", func(t *testing.T) {
		e := newTestEditor("(a(b)c)")
		for range 3 {
			keys(e, 'l')
		}
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "(a()c)", content(e))
	})

	t.Run("di{ and diB delete inside curly braces", func(t *testing.T) {
		e := newTestEditor("{hello}")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', '{')
		assert.Equal(t, "{}", content(e))
	})

	t.Run("diB is alias for di{", func(t *testing.T) {
		e := newTestEditor("{world}")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', 'B')
		assert.Equal(t, "{}", content(e))
	})

	t.Run("da{ deletes including the braces", func(t *testing.T) {
		e := newTestEditor("{ok}")
		keys(e, 'l')
		keys(e, 'd', 'a', '{')
		assert.Equal(t, "", content(e))
	})

	t.Run("di[ deletes content inside square brackets", func(t *testing.T) {
		e := newTestEditor("[1, 2, 3]")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', '[')
		assert.Equal(t, "[]", content(e))
	})

	t.Run("da] deletes including the square brackets", func(t *testing.T) {
		e := newTestEditor("[abc]")
		keys(e, 'l')
		keys(e, 'd', 'a', ']')
		assert.Equal(t, "", content(e))
	})

	t.Run("di< deletes content inside angle brackets", func(t *testing.T) {
		e := newTestEditor("<div>")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', '<')
		assert.Equal(t, "<>", content(e))
	})

	t.Run("da> deletes including the angle brackets", func(t *testing.T) {
		e := newTestEditor("<T>")
		keys(e, 'l')
		keys(e, 'd', 'a', '>')
		assert.Equal(t, "", content(e))
	})

	t.Run("di( across multiple lines keeps bracket lines", func(t *testing.T) {
		e := newTestEditor("(\n  foo\n  bar\n)")
		keys(e, 'j', 'l', 'l')
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "(\n)", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("da( across multiple lines removes brackets too", func(t *testing.T) {
		e := newTestEditor("(\n  foo\n)")
		keys(e, 'j', 'l', 'l')
		keys(e, 'd', 'a', '(')
		assert.Equal(t, "", content(e))
	})

	t.Run("no brackets – does nothing", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("no quotes – does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'i', '"')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("unmatched bracket – does nothing", func(t *testing.T) {
		e := newTestEditor("(hello")
		keys(e, 'l', 'l')
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "(hello", content(e))
	})

	t.Run("cursor outside all brackets – does nothing", func(t *testing.T) {
		e := newTestEditor("(x) y")
		keys(e, '$')
		keys(e, 'd', 'i', '(')
		assert.Equal(t, "(x) y", content(e))
	})
}

// TestDeleteCharsLeft tests 'dh' — delete count characters left of the cursor.
func TestDeleteCharsLeft(t *testing.T) {
	t.Run("single char left", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'd', 'h')
		assert.Equal(t, "hllo", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("count: 2dh", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'l', '2', 'd', 'h')
		assert.Equal(t, "hlo", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("at start of line is a no-op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'h')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestDeleteCharsRight tests 'dl' — delete count characters under/after the cursor.
func TestDeleteCharsRight(t *testing.T) {
	t.Run("single char under cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'l')
		assert.Equal(t, "ello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count clamps at end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'l', '9', 'd', 'l')
		assert.Equal(t, "hel", content(e))
	})

	t.Run("count after operator: d2l", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', '2', 'l')
		assert.Equal(t, "llo", content(e))
	})
}

// TestDeleteToLineStart tests 'd0' — delete back to the start of the line.
func TestDeleteToLineStart(t *testing.T) {
	t.Run("from end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'd', '0')
		assert.Equal(t, "o", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at start of line is a no-op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', '0')
		assert.Equal(t, "hello", content(e))
	})
}

// TestDeleteToFirstNonBlank tests 'd^' — delete back to the first non-blank character.
func TestDeleteToFirstNonBlank(t *testing.T) {
	t.Run("from end of line keeps leading whitespace", func(t *testing.T) {
		e := newTestEditor("  hello")
		keys(e, '$', 'd', '^')
		assert.Equal(t, "  o", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

// TestDeleteLinesDown tests 'dj' — delete current line and count lines below (line-wise).
func TestDeleteLinesDown(t *testing.T) {
	t.Run("current and next line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'd', 'j')
		assert.Equal(t, "three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2dj", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\nfour")
		keys(e, '2', 'd', 'j')
		assert.Equal(t, "four", content(e))
	})

	t.Run("on last line is a no-op", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'j', 'd', 'j')
		assert.Equal(t, "one\ntwo", content(e))
	})
}

// TestDeleteLinesUp tests 'dk' — delete current line and count lines above (line-wise).
func TestDeleteLinesUp(t *testing.T) {
	t.Run("current and previous line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'd', 'k')
		assert.Equal(t, "three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("on first line is a no-op", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'd', 'k')
		assert.Equal(t, "one\ntwo", content(e))
	})
}

// TestDeleteParagraphMotion tests 'd}' and 'd{' — delete to next/previous paragraph boundary.
func TestDeleteParagraphMotion(t *testing.T) {
	t.Run("d} deletes to next blank line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, 'd', '}')
		assert.Equal(t, "\nthree", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("d{ deletes back to previous blank line", func(t *testing.T) {
		e := newTestEditor("one\n\ntwo")
		keys(e, 'G', 'd', '{')
		assert.Equal(t, "one\ntwo", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	// Vim 9.2: an end in column 0 ends at the end of the line above, and the
	// delete is linewise if it starts in the indent.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "d} from the line start is linewise", content: "abc\ndef\n\nx", keys: seq("d}"), want: Position{0, 0}, text: "\nx"},
		{name: "d} from mid-line", content: "abc\ndef\n\nx", keys: seq("ld}"), want: Position{0, 0}, text: "a\n\nx"},
		{name: "d} from the indent", content: "  abc\ndef\n\nx", keys: seq("wd}"), want: Position{0, 0}, text: "\nx"},
		{name: "d{ from column 0", content: "x\n\nabc\ndef", keys: seq("Gd{"), want: Position{1, 0}, text: "x\ndef"},
		{name: "d{ from the line end", content: "x\n\nabc\ndef", keys: seq("G$d{"), want: Position{1, 0}, text: "x\nf"},
	})
}

// TestDeleteToEndOfPrevWord tests 'dge' — delete back to end of previous word (inclusive).
func TestDeleteToEndOfPrevWord(t *testing.T) {
	t.Run("from start of second word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'd', 'g', 'e')
		assert.Equal(t, "hellorld", content(e))
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	// Vim 9.2: a delete from the indent to the end of a later line is linewise.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "dge from a line end to the indent", content: "  a\nbc", keys: seq("j$dge"), want: Position{0, 0}, empty: true},
		{name: "dgE from a line end to the indent", content: "  a\nbc", keys: seq("j$dgE"), want: Position{0, 0}, empty: true},
		// From or to an empty line; ending on one adds nothing past its start.
		{name: "dge from an empty line", content: "abc\n\nx", keys: seq("jdge"), want: Position{0, 1}, text: "ab\nx"},
		{name: "dgE from an empty line", content: "abc\n\nx", keys: seq("jdgE"), want: Position{0, 1}, text: "ab\nx"},
		{name: "yge from an empty line", content: "abc\n\nx", keys: seq("jyge"), want: Position{0, 2}, text: "abc\n\nx", yanked: "c\n"},
		{name: "cge from an empty line", content: "abc\n\nx", keys: seq("jcge"), want: Position{0, 2}, text: "ab\nx"},
		{name: "d2ge over an empty line", content: "abc\n\n\nx", keys: seq("2jd2ge"), want: Position{0, 1}, text: "ab\nx"},
		{name: "dge from an empty line to an empty line", content: "abc\n\n\nx", keys: seq("2jdge"), want: Position{1, 0}, text: "abc\nx"},
		{name: "dge to an empty line", content: "abc\n\nx", keys: seq("Gdge"), want: Position{0, 0}, text: "abc"},
		{name: "dge from an empty line to the indent", content: "  a\n\nx", keys: seq("jdge"), want: Position{0, 0}, text: "x"},
	})
}

// TestOperatorCountAfterOperator tests that counts typed after the operator apply
// to the motion (e.g. d2w) and that escape cancels a pending operator.
func TestOperatorCountAfterOperator(t *testing.T) {
	t.Run("d2w deletes two words", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'd', '2', 'w')
		assert.Equal(t, "three", content(e))
	})

	t.Run("2dfx deletes through second occurrence", func(t *testing.T) {
		e := newTestEditor("axbxc")
		keys(e, '2', 'd', 'f', 'x')
		assert.Equal(t, "c", content(e))
	})

	t.Run("escape cancels pending operator", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd')
		escape(e)
		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})
}

// TestOperatorCountComposition tests that a count before the operator and one
// after it multiply, as in Vim ([count]op[count]motion).
func TestOperatorCountComposition(t *testing.T) {
	t.Run("2d3w deletes six words", func(t *testing.T) {
		e := newTestEditor("a b c d e f g h")
		keys(e, '2', 'd', '3', 'w')
		assert.Equal(t, "g h", content(e))
	})

	t.Run("multi-digit counts on both sides: 12d2w deletes 24 words", func(t *testing.T) {
		words := make([]string, 26)
		for i := range words {
			words[i] = string(rune('a' + i))
		}
		e := newTestEditor(strings.Join(words, " "))
		keys(e, '1', '2', 'd', '2', 'w')
		assert.Equal(t, "y z", content(e))
	})

	t.Run("2dd and d2d both delete two lines", func(t *testing.T) {
		for _, seq := range [][]rune{{'2', 'd', 'd'}, {'d', '2', 'd'}} {
			e := newTestEditor("a\nb\nc\nd")
			keys(e, seq...)
			assert.Equal(t, "c\nd", content(e), string(seq))
		}
	})

	t.Run("2d2d deletes four lines", func(t *testing.T) {
		e := newTestEditor("a\nb\nc\nd\ne")
		keys(e, '2', 'd', '2', 'd')
		assert.Equal(t, "e", content(e))
	})

	t.Run("2y3w yanks six words", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("a b c d e f g h")
		keys(e, '2', 'y', '3', 'w')
		assert.Equal(t, "a b c d e f ", cb.content)
	})

	t.Run("2d2fx deletes through the fourth x", func(t *testing.T) {
		e := newTestEditor("1x2x3x4x5x")
		keys(e, '2', 'd', '2', 'f', 'x')
		assert.Equal(t, "5x", content(e))
	})

	t.Run("2d2G deletes through line 4", func(t *testing.T) {
		e := newTestEditor("a\nb\nc\nd\ne")
		keys(e, '2', 'd', '2', 'G')
		assert.Equal(t, "e", content(e))
	})

	t.Run("count before an escaped operator is dropped", func(t *testing.T) {
		e := newTestEditor("a b c d e")
		keys(e, '3', 'd')
		escape(e)
		keys(e, 'd', 'w')
		assert.Equal(t, "b c d e", content(e))
	})

	t.Run("2d2aw deletes four words", func(t *testing.T) {
		e := newTestEditor("a b c d e f g h")
		keys(e, '2', 'd', '2', 'a', 'w')
		assert.Equal(t, "e f g h", content(e))
	})

	t.Run("no count after a text object modifier: the digit cancels, then w moves", func(t *testing.T) {
		for _, seq := range []string{"2di2w", "2gUi2w"} {
			e := newTestEditor("a b c d e f g h")
			keys(e, []rune(seq)...)
			assert.Equal(t, "a b c d e f g h", content(e), seq)
			assert.Equal(t, Position{0, 2}, cursorPos(e), seq)
		}
	})

	t.Run("count before an escaped case operator is dropped", func(t *testing.T) {
		e := newTestEditor("a b c d e f g h")
		keys(e, '2', 'g', 'U')
		escape(e)
		keys(e, 'd', 'w')
		assert.Equal(t, "b c d e f g h", content(e))
	})

	t.Run("2gU3w uppercases six words", func(t *testing.T) {
		e := newTestEditor("a b c d e f g h")
		keys(e, '2', 'g', 'U', '3', 'w')
		assert.Equal(t, "A B C D E F g h", content(e))
	})

	t.Run("2gUU uppercases two lines", func(t *testing.T) {
		e := newTestEditor("a\nb\nc")
		keys(e, '2', 'g', 'U', 'U')
		assert.Equal(t, "A\nB\nc", content(e))
	})
	t.Run("command line shows both counts as they are typed", func(t *testing.T) {
		for _, tc := range []struct {
			keys  string
			shown []string // command line after each key
		}{
			{"2d3w", []string{"2", "2d", "2d3", ""}},
			{"12y2w", []string{"1", "12", "12y", "12y2", ""}},
			{"2gU3w", []string{"2", "2g", "2gU", "2gU3", ""}},
			{"gUiw", []string{"g", "gU", "gUi", ""}},
			{"daw", []string{"d", "da", ""}},
			{"2d2aw", []string{"2", "2d", "2d2", "2d2a", ""}},
		} {
			e := newTestEditor("a b c d e f g h")
			for i, r := range tc.keys {
				keys(e, r)
				assert.Equal(t, tc.shown[i], e.GetState().CommandLine, "%s after %q", tc.keys, tc.keys[:i+1])
			}
		}
	})
}

// TestDeleteWORDMotions tests 'dW', 'dE', 'dB' — WORD variants of the word operators.
func TestDeleteWORDMotions(t *testing.T) {
	t.Run("dW deletes WORD including punctuation", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'd', 'W')
		assert.Equal(t, "baz", content(e))
	})

	t.Run("dE deletes to end of WORD inclusive", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'd', 'E')
		assert.Equal(t, " baz", content(e))
	})

	t.Run("dB deletes WORD backward", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'W', 'd', 'B')
		assert.Equal(t, "baz", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("dgE deletes back to end of previous WORD inclusive", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'W', 'd', 'g', 'E')
		assert.Equal(t, "foo-baaz", content(e))
	})

	// Vim 9.2: dW stops at the end of the line like dw.
	seq := keySeq
	runDollarCases(t, []dollarCase{
		{name: "dW on the last WORD of a line", content: "abc d.f\nab", keys: seq("wdW"), want: Position{0, 3}, text: "abc \nab"},
		{name: "dW then .", content: "abc d.f\nab d.f", keys: seq("wdWj."), want: Position{1, 2}, text: "abc \nab "},
		{name: "dw stops at punctuation", content: "abc d.f\nab", keys: seq("wdw"), want: Position{0, 4}, text: "abc .f\nab"},
		{name: "2dW crosses the line", content: "abc d.f\nab c.d", keys: seq("w2dW"), want: Position{0, 4}, text: "abc c.d"},
		{name: "dW on an empty line", content: "abc\n\ndef", keys: seq("jdW"), want: Position{1, 0}, text: "abc\ndef"},
	})
}

// TestDeleteMatchingBracket tests 'd%' — delete through the matching bracket.
func TestDeleteMatchingBracket(t *testing.T) {
	t.Run("from open bracket", func(t *testing.T) {
		e := newTestEditor("(foo) bar")
		keys(e, 'd', '%')
		assert.Equal(t, " bar", content(e))
	})

	// Vim 9.2: a delete from the indent to the end of a later line is linewise.
	runDollarCases(t, []dollarCase{
		{name: "from the indent to a line end", content: "  (abc\ndef)", keys: keySeq("wd%"), want: Position{0, 0}, empty: true},
		{name: "from the indent with text after", content: "  (abc\ndef) x", keys: keySeq("wd%"), want: Position{0, 2}, text: "   x"},
	})

	t.Run("from before the bracket deletes from cursor through match", func(t *testing.T) {
		e := newTestEditor("a (b) c")
		keys(e, 'd', '%')
		assert.Equal(t, " c", content(e))
	})

	t.Run("no bracket is a no-op", func(t *testing.T) {
		e := newTestEditor("plain")
		keys(e, 'd', '%')
		assert.Equal(t, "plain", content(e))
	})
}

// TestDeleteRepeatCharSearch tests 'd;' and 'd,' — repeat char search as operator motion.
func TestDeleteRepeatCharSearch(t *testing.T) {
	t.Run("d; repeats last f search", func(t *testing.T) {
		e := newTestEditor("a,b,c,d")
		keys(e, 'd', 'f', ',') // delete "a," → "b,c,d"
		assert.Equal(t, "b,c,d", content(e))
		keys(e, 'd', ';') // repeat → delete "b," → "c,d"
		assert.Equal(t, "c,d", content(e))
	})

	t.Run("d, repeats in opposite direction", func(t *testing.T) {
		e := newTestEditor("a,b,c")
		keys(e, 'f', ',', ';') // cursor on second comma (col 3)
		assert.Equal(t, Position{0, 3}, cursorPos(e))
		// Reverses to F, — deletes back to the first comma; F is exclusive, so
		// the cursor char stays (Vim 9.2: "a,c", cursor (0,1)).
		keys(e, 'd', ',')
		assert.Equal(t, "a,c", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("d; without previous search is a no-op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', ';')
		assert.Equal(t, "hello", content(e))
	})
}

// TestDeleteTextObjectCount tests counts applied to word text objects (d2aw, d3iw).
func TestDeleteTextObjectCount(t *testing.T) {
	t.Run("d2aw deletes two words with trailing space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'd', '2', 'a', 'w')
		assert.Equal(t, "three", content(e))
	})

	t.Run("2daw deletes two words with trailing space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'd', 'a', 'w')
		assert.Equal(t, "three", content(e))
	})

	t.Run("d3iw deletes word, space, word", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'd', '3', 'i', 'w')
		assert.Equal(t, " three", content(e))
	})
}

// TestJoinLines tests 'J' — join lines with a single space.
func TestJoinLines(t *testing.T) {
	t.Run("joins two lines with a space", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'J')
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e)) // cursor on the space
	})

	t.Run("strips leading whitespace of joined line", func(t *testing.T) {
		e := newTestEditor("foo\n   bar")
		keys(e, 'J')
		assert.Equal(t, "foo bar", content(e))
	})

	t.Run("no extra space when line ends in whitespace", func(t *testing.T) {
		e := newTestEditor("foo \nbar")
		keys(e, 'J')
		assert.Equal(t, "foo bar", content(e))
	})

	t.Run("no space before )", func(t *testing.T) {
		e := newTestEditor("foo(\n)")
		keys(e, 'J')
		assert.Equal(t, "foo()", content(e))
	})

	t.Run("count: 3J joins three lines", func(t *testing.T) {
		e := newTestEditor("a\nb\nc")
		keys(e, '3', 'J')
		assert.Equal(t, "a b c", content(e))
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("empty first line", func(t *testing.T) {
		e := newTestEditor("\nfoo")
		keys(e, 'J')
		assert.Equal(t, "foo", content(e))
	})

	t.Run("on last line is a no-op", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'j', 'J')
		assert.Equal(t, "one\ntwo", content(e))
	})
}

// TestDeleteCharBeforeCount checks [count]X against Vim 9.2: it deletes the
// count characters before the cursor (X is dh), fewer when the line start is
// nearer, and leaves the cursor where the deleted text began. At column 0 it
// fails, and . still repeats it with its count. X doesn't write the clipboard
// in goeditor yet, so the rows don't check it.
func TestDeleteCharBeforeCount(t *testing.T) {
	const (
		hello = "hello world\nabcdefghijklmnop"
		multi = "héllo wörld"
		tab   = "a\tb c"
		two   = "ab\ncd"
	)
	tests := []struct {
		content string
		keys    string
		text    string
		want    Position // cursor; Preferred is its column
	}{
		{hello, "$X", "hello word\nabcdefghijklmnop", Position{0, 9}},
		{hello, "$3X", "hello wd\nabcdefghijklmnop", Position{0, 7}},
		{hello, "w3X", "helworld\nabcdefghijklmnop", Position{0, 3}},
		{hello, "ll5X", "llo world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "ll2X", "llo world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "X", hello, Position{0, 0}},
		{hello, "5X", hello, Position{0, 0}},
		{hello, "l9X", "ello world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$10X", "d\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$11X", "d\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3X.", "helld\nabcdefghijklmnop", Position{0, 4}},
		{hello, "$3X..", "hd\nabcdefghijklmnop", Position{0, 1}},
		{hello, "w3X.", "world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3X2.", "hellod\nabcdefghijklmnop", Position{0, 5}},
		{hello, "w2X3.", "hworld\nabcdefghijklmnop", Position{0, 1}},
		{hello, "ll5X.", "llo world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3X0.", "hello wd\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3X0X.", "hello wd\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3X0X$.", "hello d\nabcdefghijklmnop", Position{0, 6}},
		{hello, "5X$.", "hellod\nabcdefghijklmnop", Position{0, 5}},
		{hello, "$3Xu", hello, Position{0, 7}},
		{hello, "w3Xu", hello, Position{0, 3}},
		{hello, "ll5Xu", hello, Position{0, 0}},
		{hello, "$3Xu.", "hellorld\nabcdefghijklmnop", Position{0, 4}},
		{hello, "w3X$.u", "helworld\nabcdefghijklmnop", Position{0, 4}},
		{hello, "$3Xu<C-r>", "hello wd\nabcdefghijklmnop", Position{0, 7}},
		{hello, "w3Xu<C-r>", "helworld\nabcdefghijklmnop", Position{0, 3}},
		{hello, "ll5Xu<C-r>", "llo world\nabcdefghijklmnop", Position{0, 0}},
		{hello, "$3Xj", "hello wd\nabcdefghijklmnop", Position{1, 7}},
		{hello, "w3Xj", "helworld\nabcdefghijklmnop", Position{1, 3}},
		{hello, "ll5Xj", "llo world\nabcdefghijklmnop", Position{1, 0}},
		{hello, "$Xj", "hello word\nabcdefghijklmnop", Position{1, 9}},
		{hello, "Xj", hello, Position{1, 0}},
		{hello, "$jX", "hello world\nabcdefghijklmnp", Position{1, 14}},
		{multi, "$3X", "héllo wd", Position{0, 7}},
		{multi, "w3X", "hélwörld", Position{0, 3}},
		{multi, "ll5X", "llo wörld", Position{0, 0}},
		{tab, "$X", "a\tbc", Position{0, 3}},
		{tab, "$3X", "ac", Position{0, 1}},
		{tab, "f 2X", "a c", Position{0, 1}},
		{"x", "X", "x", Position{0, 0}},
		{"x", "3X", "x", Position{0, 0}},
		{two, "j3X", two, Position{1, 0}},
		{two, "l3Xj", "b\ncd", Position{1, 0}},
		{two, "jl5X", "ab\nd", Position{1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.content+" "+tt.keys, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e), "text")
			assert.Equal(t, tt.want, cursorPos(e), "cursor")
			assert.Equal(t, tt.want.Col, e.GetBuffer().GetCursor().Preferred, "Preferred")
		})
	}
}
