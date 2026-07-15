package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChangeLine tests 'cc' — delete current line and enter insert mode.
func TestChangeLine(t *testing.T) {
	t.Run("single line becomes empty", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'c', 'c')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("first of two lines", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'c', 'c')
		assert.Equal(t, "second", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("last line is removed, cursor moves to previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'c', 'c')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("count: 2cc", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '2', 'c', 'c')
		assert.Equal(t, "three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeWord tests 'cw' — change to end of current word (same motion as 'ce').
func TestChangeWord(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'w')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'c', 'w')
		assert.Equal(t, "h world", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("count: 2cw", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'c', 'w')
		assert.Equal(t, " three", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeToWordEnd tests 'ce' — same motion as 'cw'.
func TestChangeToWordEnd(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'e')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'c', 'e')
		assert.Equal(t, "he world", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeWordBackward tests 'cb' — change to start of previous word.
func TestChangeWordBackward(t *testing.T) {
	t.Run("from end of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'e', 'c', 'b')
		assert.Equal(t, "o world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("from mid second word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'l', 'l', 'c', 'b')
		assert.Equal(t, "hello rld", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeToEndOfLineShortcut tests 'C' — shortcut for c$.
func TestChangeToEndOfLineShortcut(t *testing.T) {
	t.Run("from start of line clears it and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'C')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("from mid-line deletes to end and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'C') // cursor at col 6; C deletes "world"
		assert.Equal(t, "hello ", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after C replaces rest of line", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'C')
		keys(e, 'e', 'a', 'r', 't', 'h')
		assert.Equal(t, "hello earth", content(e))
	})
}

// TestChangeToEndOfLine tests 'c$' — change to end of line.
func TestChangeToEndOfLine(t *testing.T) {
	t.Run("from start clears line", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.HandleKey(KeyEvent{Rune: 'c'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("from mid-line", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w')
		e.HandleKey(KeyEvent{Rune: 'c'})
		e.HandleKey(KeyEvent{Rune: '$'})
		assert.Equal(t, "hello ", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeToEndOfBuffer tests 'cG' — change to end of buffer.
func TestChangeToEndOfBuffer(t *testing.T) {
	t.Run("from first line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		e.HandleKey(KeyEvent{Rune: 'c'})
		e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("from second line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j')
		e.HandleKey(KeyEvent{Rune: 'c'})
		e.HandleKey(KeyEvent{Rune: 'G'})
		assert.Equal(t, "one", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeInsideWord tests 'ciw' — change inside word.
func TestChangeInsideWord(t *testing.T) {
	t.Run("from start of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'i', 'w')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'c', 'i', 'w')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeAroundWord tests 'caw' — change around word (includes surrounding space).
func TestChangeAroundWord(t *testing.T) {
	t.Run("first word eats trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'a', 'w')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("middle word eats leading space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'c', 'a', 'w')
		assert.Equal(t, "one three", content(e))
		assertInsertMode(t, e)
	})
}

// TestChangeInsideParagraph tests 'cip' — replace paragraph with one empty line and enter insert mode.
func TestChangeInsideParagraph(t *testing.T) {
	t.Run("opens empty line in place of paragraph", func(t *testing.T) {
		// cip leaves one blank line where the paragraph was (like Vim's 'c' on a linewise selection).
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'c', 'i', 'p')
		assert.Equal(t, "\n\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("on blank line preserves blank line and enters insert mode", func(t *testing.T) {
		// cip on a blank line behaves like 'cc' on a blank line: no content change, just insert mode.
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'c', 'i', 'p') // cursor on blank row 1
		assert.Equal(t, "hello\n\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeAroundParagraph tests 'cap' — delete paragraph + surrounding blanks, then open one
// blank line at the original paragraph position (matching Vim's cap behaviour).
func TestChangeAroundParagraph(t *testing.T) {
	t.Run("opens blank line before next paragraph", func(t *testing.T) {
		// dap would leave "foo"; cap additionally opens a blank line so cursor is ready to type.
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'c', 'a', 'p')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("no following paragraph: cursor on empty line", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'c', 'a', 'p')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("preceding content: opens blank line below it", func(t *testing.T) {
		// 'a' absorbs the leading blank; the blank + paragraph are removed and a new
		// blank line is opened after "foo" for the replacement content.
		e := newTestEditor("foo\n\nhello\nworld")
		keys(e, 'j', 'j', 'c', 'a', 'p') // cursor on "hello"
		assert.Equal(t, "foo\n", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeToBufferStart tests 'cgg' — delete from the first line to the cursor line and enter insert mode.
func TestChangeToBufferStart(t *testing.T) {
	t.Run("cgg deletes to top and enters insert mode", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j') // cursor on row 2
		keys(e, 'c', 'g', 'g')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeInsideQuotes tests ci"/ca", ci'/ca', ci`/ca` — change inside/around quotes.
func TestChangeInsideQuotes(t *testing.T) {
	t.Run(`ci" clears content and enters insert mode`, func(t *testing.T) {
		e := newTestEditor(`"hello"`)
		keys(e, 'l', 'l')
		keys(e, 'c', 'i', '"')
		assert.Equal(t, `""`, content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run(`ca" removes quotes and enters insert mode`, func(t *testing.T) {
		e := newTestEditor(`"hi"`)
		keys(e, 'l')
		keys(e, 'c', 'a', '"')
		assert.Equal(t, ``, content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

// TestChangeInsideBrackets tests cib/cab, ciB/caB and caq — change inside/around brackets and quotes.
func TestChangeInsideBrackets(t *testing.T) {
	t.Run("cib clears parens content and enters insert mode", func(t *testing.T) {
		e := newTestEditor("func(arg)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'c', 'i', 'b')
		assert.Equal(t, "func()", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("cab removes parens and content, enters insert mode", func(t *testing.T) {
		e := newTestEditor("func(arg)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'c', 'a', 'b')
		assert.Equal(t, "func", content(e))
		assertInsertMode(t, e)
	})

	t.Run("caq clears any-quote content and enters insert mode", func(t *testing.T) {
		e := newTestEditor(`"hello"`)
		keys(e, 'l', 'l')
		keys(e, 'c', 'a', 'q')
		assert.Equal(t, "", content(e))
		assertInsertMode(t, e)
	})
}

// TestChangeOperatorMotions tests the change variants of the h/l/0/j/k/ge motions.
func TestChangeOperatorMotions(t *testing.T) {
	t.Run("cl changes char under cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'c', 'l')
		assert.Equal(t, "ello", content(e))
		assertInsertMode(t, e)
	})

	t.Run("c0 changes back to line start", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'c', '0')
		assert.Equal(t, "o", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("cj changes current and next line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'c', 'j')
		assert.Equal(t, "three", content(e))
		assertInsertMode(t, e)
	})
}

// TestSubstituteChar tests 's' — substitute character(s), like 'cl'.
func TestSubstituteChar(t *testing.T) {
	t.Run("deletes char under cursor and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 's')
		assert.Equal(t, "ello", content(e))
		assertInsertMode(t, e)
	})

	t.Run("count: 3s", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', 's')
		assert.Equal(t, "lo", content(e))
		assertInsertMode(t, e)
	})

	t.Run("on empty line still enters insert mode", func(t *testing.T) {
		e := newTestEditor("")
		keys(e, 's')
		assert.Equal(t, "", content(e))
		assertInsertMode(t, e)
	})
}

// TestSubstituteLine tests 'S' — substitute line(s), like 'cc'.
func TestSubstituteLine(t *testing.T) {
	t.Run("clears current line and enters insert mode", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'S')
		assert.Equal(t, "two", content(e))
		assertInsertMode(t, e)
	})

	t.Run("count: 2S", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '2', 'S')
		assert.Equal(t, "three", content(e))
		assertInsertMode(t, e)
	})
}

// TestChangeWORD tests 'cW' — change to end of WORD (like cE).
func TestChangeWORD(t *testing.T) {
	t.Run("changes WORD including punctuation", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'c', 'W')
		assert.Equal(t, " baz", content(e))
		assertInsertMode(t, e)
	})
}
