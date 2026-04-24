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
		assert.Equal(t, Position{0, 6}, cursorPos(e))
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
		e.HandleKey(KeyEvent{Rune: 'd'})
		e.HandleKey(KeyEvent{Rune: 'g'})
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
