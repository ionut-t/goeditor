package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// ~ — toggle case of character(s) under cursor
// ---------------------------------------------------------------------------

func TestToggleCaseChar(t *testing.T) {
	t.Run("lowercase → uppercase", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '~')
		assert.Equal(t, "Hello", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("uppercase → lowercase", func(t *testing.T) {
		e := newTestEditor("HELLO")
		keys(e, '~')
		assert.Equal(t, "hELLO", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("count: 3~ toggles three chars", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', '~')
		assert.Equal(t, "HELlo", content(e))
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("count clamped to end of line", func(t *testing.T) {
		e := newTestEditor("hi")
		keys(e, '9', '~')
		assert.Equal(t, "HI", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("non-letter is left unchanged", func(t *testing.T) {
		e := newTestEditor("a1b")
		keys(e, 'l', '~') // cursor on '1'
		assert.Equal(t, "a1b", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("at end of line is a no-op", func(t *testing.T) {
		e := newTestEditor("hi")
		keys(e, '$', '~') // cursor on last char 'i'
		assert.Equal(t, "hI", content(e))
	})

	t.Run("empty line is a no-op", func(t *testing.T) {
		e := newTestEditor("")
		keys(e, '~')
		assert.Equal(t, "", content(e))
	})

	t.Run("cursor does not advance past end of line", func(t *testing.T) {
		e := newTestEditor("a")
		keys(e, '~')
		assert.Equal(t, "A", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e)) // single char line: stays at 0
	})
}

// ---------------------------------------------------------------------------
// guu — lowercase current line
// ---------------------------------------------------------------------------

func TestLowercaseLine(t *testing.T) {
	t.Run("uppercased line becomes lowercase", func(t *testing.T) {
		e := newTestEditor("HELLO WORLD")
		keys(e, 'g', 'u', 'u')
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("mixed case line", func(t *testing.T) {
		e := newTestEditor("HeLLo")
		keys(e, 'g', 'u', 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("already lowercase is unchanged", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'g', 'u', 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("count 2guu lowercases two lines", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nbaz")
		keys(e, '2', 'g', 'u', 'u')
		assert.Equal(t, "foo\nbar\nbaz", content(e))
	})

	t.Run("empty line is a no-op", func(t *testing.T) {
		e := newTestEditor("A\n\nB")
		keys(e, 'j', 'g', 'u', 'u') // cursor on empty line
		assert.Equal(t, "A\n\nB", content(e))
	})
}

// ---------------------------------------------------------------------------
// gUU — uppercase current line
// ---------------------------------------------------------------------------

func TestUppercaseLine(t *testing.T) {
	t.Run("lowercase line becomes uppercase", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'U')
		assert.Equal(t, "HELLO WORLD", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count 2gUU uppercases two lines", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		keys(e, '2', 'g', 'U', 'U')
		assert.Equal(t, "FOO\nBAR\nbaz", content(e))
	})
}

// ---------------------------------------------------------------------------
// g~~ — toggle case of current line
// ---------------------------------------------------------------------------

func TestToggleCaseLine(t *testing.T) {
	t.Run("mixed case line is toggled", func(t *testing.T) {
		e := newTestEditor("Hello World")
		keys(e, 'g', '~', '~')
		assert.Equal(t, "hELLO wORLD", content(e))
	})

	t.Run("count 2g~~ toggles two lines", func(t *testing.T) {
		e := newTestEditor("Hello\nWorld")
		keys(e, '2', 'g', '~', '~')
		assert.Equal(t, "hELLO\nwORLD", content(e))
	})
}

// ---------------------------------------------------------------------------
// guw / gUw / g~w — case change to next word
// ---------------------------------------------------------------------------

func TestCaseWord(t *testing.T) {
	t.Run("guw lowercases word forward", func(t *testing.T) {
		e := newTestEditor("HELLO world")
		keys(e, 'g', 'u', 'w')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("gUw uppercases word forward", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'w')
		assert.Equal(t, "HELLO world", content(e))
	})

	t.Run("g~w toggles case of word forward", func(t *testing.T) {
		e := newTestEditor("Hello world")
		keys(e, 'g', '~', 'w')
		assert.Equal(t, "hELLO world", content(e))
	})

	t.Run("count 2gUw uppercases two words", func(t *testing.T) {
		e := newTestEditor("foo bar baz")
		keys(e, '2', 'g', 'U', 'w')
		assert.Equal(t, "FOO BAR baz", content(e))
	})
}

// ---------------------------------------------------------------------------
// gue / gUe / g~e — case change to word end
// ---------------------------------------------------------------------------

func TestCaseWordEnd(t *testing.T) {
	t.Run("gue lowercases to word end", func(t *testing.T) {
		e := newTestEditor("HELLO world")
		keys(e, 'g', 'u', 'e')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("gUe uppercases to word end", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'e')
		assert.Equal(t, "HELLO world", content(e))
	})

	t.Run("g~e toggles to word end mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'g', '~', 'e') // cursor on 'l' (col 2), end of "hello" is col 4
		assert.Equal(t, "heLLO world", content(e))
	})
}

// ---------------------------------------------------------------------------
// gub / gUb / g~b — case change to previous word start
// ---------------------------------------------------------------------------

func TestCaseWordBackward(t *testing.T) {
	t.Run("gub lowercases back to previous word start", func(t *testing.T) {
		// cursor at 'w' of "world" (col 6); 'b' goes back to 'H' (col 0).
		// Range [0, 6) = "HELLO " → lowercased.
		e := newTestEditor("HELLO world")
		keys(e, 'w', 'g', 'u', 'b')
		assert.Equal(t, "hello world", content(e))
		// cursor should land at start of backward range
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gUb uppercases back to previous word start", func(t *testing.T) {
		// cursor at 'w' of "world" (col 6); 'b' goes back to 'h' (col 0).
		// Range [0, 6) = "hello " → uppercased.
		e := newTestEditor("hello world")
		keys(e, 'w', 'g', 'U', 'b')
		assert.Equal(t, "HELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// ---------------------------------------------------------------------------
// gu$ / gU$ / g~$ — case change to end of line
// ---------------------------------------------------------------------------

func TestCaseToEndOfLine(t *testing.T) {
	t.Run("gu$ lowercases to end of line", func(t *testing.T) {
		e := newTestEditor("hello WORLD")
		keys(e, 'w') // cursor at 'W' (col 6)
		keys(e, 'g', 'u', '$')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("gU$ uppercases to end of line", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w') // cursor at 'w' of "world" (col 6)
		keys(e, 'g', 'U', '$')
		assert.Equal(t, "hello WORLD", content(e))
	})

	t.Run("g~$ toggles to end of line", func(t *testing.T) {
		e := newTestEditor("Hello World")
		keys(e, 'w') // cursor at 'W' (col 6)
		keys(e, 'g', '~', '$')
		assert.Equal(t, "Hello wORLD", content(e))
	})

	t.Run("applies to the last character on line", func(t *testing.T) {
		e := newTestEditor("hi")
		keys(e, '$') // cursor at 'i' (last char, col 1)
		keys(e, 'g', 'U', '$')
		assert.Equal(t, "hI", content(e))
	})
}

// ---------------------------------------------------------------------------
// guG / gUG / g~G — case change to end of buffer
// ---------------------------------------------------------------------------

func TestCaseToBufferEnd(t *testing.T) {
	t.Run("gUG uppercases from cursor to end of buffer", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		keys(e, 'j') // cursor on row 1
		keys(e, 'g', 'U', 'G')
		assert.Equal(t, "foo\nBAR\nBAZ", content(e))
	})

	t.Run("guG lowercases from cursor to end of buffer", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nBAZ")
		keys(e, 'j')
		keys(e, 'g', 'u', 'G')
		assert.Equal(t, "FOO\nbar\nbaz", content(e))
	})

	t.Run("g~G toggles from cursor to end of buffer", func(t *testing.T) {
		e := newTestEditor("Hello\nWorld")
		keys(e, 'g', '~', 'G')
		assert.Equal(t, "hELLO\nwORLD", content(e))
	})
}

// ---------------------------------------------------------------------------
// gU{i,a}w / gu{i,a}w / g~{i,a}w — case change inside/around word
// ---------------------------------------------------------------------------

func TestCaseInsideWord(t *testing.T) {
	t.Run("gUiw uppercases inside word from start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'i', 'w')
		assert.Equal(t, "HELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gUiw uppercases inside word from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'g', 'U', 'i', 'w')
		assert.Equal(t, "HELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("guiw lowercases inside word", func(t *testing.T) {
		e := newTestEditor("HELLO world")
		keys(e, 'g', 'u', 'i', 'w')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("g~iw toggles inside word", func(t *testing.T) {
		e := newTestEditor("Hello world")
		keys(e, 'l', 'g', '~', 'i', 'w') // mid-word
		assert.Equal(t, "hELLO world", content(e))
	})
}

func TestCaseAroundWord(t *testing.T) {
	t.Run("gUaw uppercases word + trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'a', 'w')
		// 'aw' on first word = "hello " (word + trailing space)
		assert.Equal(t, "HELLO world", content(e))
	})

	t.Run("guaw lowercases word + leading space (middle word)", func(t *testing.T) {
		e := newTestEditor("one TWO three")
		keys(e, 'w', 'g', 'u', 'a', 'w') // cursor on 'T'
		// 'aw' on middle word absorbs trailing space: "TWO " → "two "
		assert.Equal(t, "one two three", content(e))
	})

	t.Run("g~aw toggles word + space", func(t *testing.T) {
		e := newTestEditor("Hello world")
		keys(e, 'g', '~', 'a', 'w')
		assert.Equal(t, "hELLO world", content(e))
	})
}

// ---------------------------------------------------------------------------
// gU{i,a}" / etc. — case change inside/around quotes
// ---------------------------------------------------------------------------

func TestCaseInsideQuotes(t *testing.T) {
	t.Run(`gUi" uppercases inside double quotes`, func(t *testing.T) {
		e := newTestEditor(`"hello"`)
		keys(e, 'l', 'l', 'g', 'U', 'i', '"')
		assert.Equal(t, `"HELLO"`, content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run(`gui' lowercases inside single quotes`, func(t *testing.T) {
		e := newTestEditor(`'HELLO'`)
		keys(e, 'l', 'g', 'u', 'i', '\'')
		assert.Equal(t, `'hello'`, content(e))
	})

	t.Run(`gUa" uppercases around double quotes (incl. quote chars)`, func(t *testing.T) {
		e := newTestEditor(`"hello"`)
		keys(e, 'l', 'g', 'U', 'a', '"')
		assert.Equal(t, `"HELLO"`, content(e))
	})
}

// ---------------------------------------------------------------------------
// gU{i,a}( / etc. — case change inside/around brackets
// ---------------------------------------------------------------------------

func TestCaseInsideBrackets(t *testing.T) {
	t.Run("gUi( uppercases inside parens", func(t *testing.T) {
		e := newTestEditor("func(hello)")
		for range 5 {
			keys(e, 'l') // cursor on 'h'
		}
		keys(e, 'g', 'U', 'i', '(')
		assert.Equal(t, "func(HELLO)", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("gUa( uppercases around parens", func(t *testing.T) {
		e := newTestEditor("func(hello)")
		for range 5 {
			keys(e, 'l')
		}
		keys(e, 'g', 'U', 'a', '(')
		assert.Equal(t, "func(HELLO)", content(e))
	})

	t.Run("gUib uppercases inside nearest bracket", func(t *testing.T) {
		e := newTestEditor("[hello]")
		keys(e, 'l', 'l', 'g', 'U', 'i', 'b')
		assert.Equal(t, "[HELLO]", content(e))
	})
}

// ---------------------------------------------------------------------------
// Visual mode: u / U / ~ on charwise selection
// ---------------------------------------------------------------------------

func TestVisualCaseOp(t *testing.T) {
	t.Run("v + u lowercases selection", func(t *testing.T) {
		e := newTestEditor("HELLO world")
		keys(e, 'v', 'e', 'u') // select "HELLO", then lowercase
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v + U uppercases selection", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'e', 'U') // select "hello", then uppercase
		assert.Equal(t, "HELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v + ~ toggles case of selection", func(t *testing.T) {
		e := newTestEditor("Hello world")
		keys(e, 'v', 'e', '~') // select "Hello", then toggle
		assert.Equal(t, "hELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("returns to normal mode after op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'e', 'U')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("backward selection (cursor before start)", func(t *testing.T) {
		// startPos = col 6 ('w' of "world"), then 'b' in visual mode moves cursor to col 0.
		// Normalised selection: [0, 6] inclusive → uppercase "hello w" → "HELLO W".
		e := newTestEditor("hello world")
		keys(e, 'w')           // cursor at 'w' (col 6)
		keys(e, 'v', 'b', 'U') // enter visual, extend backward to col 0, uppercase
		assert.Equal(t, "HELLO World", content(e))
	})
}

// ---------------------------------------------------------------------------
// Visual line mode: u / U / ~ on line selection
// ---------------------------------------------------------------------------

func TestVisualLineCaseOp(t *testing.T) {
	t.Run("V + u lowercases selected lines", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nbaz")
		keys(e, 'V', 'j', 'u') // select lines 0-1, then lowercase
		assert.Equal(t, "foo\nbar\nbaz", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V + U uppercases selected lines", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		keys(e, 'V', 'j', 'U') // select lines 0-1, then uppercase
		assert.Equal(t, "FOO\nBAR\nbaz", content(e))
	})

	t.Run("V + ~ toggles case of selected lines", func(t *testing.T) {
		e := newTestEditor("Hello\nWorld")
		keys(e, 'V', 'j', '~')
		assert.Equal(t, "hELLO\nwORLD", content(e))
	})

	t.Run("returns to normal mode after op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'U')
		assert.True(t, e.IsNormalMode())
	})
}

// ---------------------------------------------------------------------------
// gu0 / gU0 / g~0 — case change to beginning of line
// ---------------------------------------------------------------------------

func TestCaseToLineStart(t *testing.T) {
	t.Run("gU0 uppercases from start of line to cursor", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w') // cursor at 'w' of "world" (col 6)
		keys(e, 'g', 'U', '0')
		assert.Equal(t, "HELLO world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gu0 lowercases from start of line to cursor", func(t *testing.T) {
		e := newTestEditor("HELLO world")
		keys(e, 'w') // cursor at 'w' (col 6)
		keys(e, 'g', 'u', '0')
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("g~0 toggles from start of line to cursor", func(t *testing.T) {
		e := newTestEditor("Hello World")
		keys(e, 'w') // cursor at 'W' (col 6)
		keys(e, 'g', '~', '0')
		assert.Equal(t, "hELLO World", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("cursor at col 0 is a no-op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'g', 'U', '0')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// ---------------------------------------------------------------------------
// gu^ / gU^ / g~^ — case change to first non-blank
// ---------------------------------------------------------------------------

func TestCaseToFirstNonBlank(t *testing.T) {
	t.Run("gU^ uppercases from first non-blank to cursor (cursor past it)", func(t *testing.T) {
		// ^ is exclusive: char under cursor is NOT included.
		e := newTestEditor("  hello")
		keys(e, '$') // cursor at 'o' (col 6)
		keys(e, 'g', 'U', '^')
		assert.Equal(t, "  HELLo", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("gu^ lowercases between cursor and first non-blank (cursor before it)", func(t *testing.T) {
		// cursor at col 0, first non-blank at col 2 → range [0,2) = spaces → no change, cursor moves
		e := newTestEditor("  HELLO")
		keys(e, 'g', 'u', '^')
		assert.Equal(t, "  HELLO", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("g~^ toggles from first non-blank to cursor", func(t *testing.T) {
		// ^ is exclusive: 'd' at col 10 (cursor) is NOT included.
		e := newTestEditor("Hello World")
		keys(e, '$') // cursor at 'd' (col 10)
		keys(e, 'g', '~', '^')
		assert.Equal(t, "hELLO wORLd", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("cursor at first non-blank is a no-op", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'g', 'U', '^') // cursor already at col 0 = first non-blank
		assert.Equal(t, "hello", content(e))
	})
}

// ---------------------------------------------------------------------------
// gugg / gUgg / g~gg — case change to beginning of buffer
// ---------------------------------------------------------------------------

func TestCaseToBufferStart(t *testing.T) {
	t.Run("gUgg uppercases from beginning of buffer to cursor line", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		keys(e, 'j') // cursor on row 1
		keys(e, 'g', 'U', 'g', 'g')
		assert.Equal(t, "FOO\nBAR\nbaz", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gugg lowercases from beginning of buffer to cursor line", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nBAZ")
		keys(e, 'j', 'j') // cursor on row 2
		keys(e, 'g', 'u', 'g', 'g')
		assert.Equal(t, "foo\nbar\nbaz", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("g~gg toggles from beginning of buffer to cursor line", func(t *testing.T) {
		e := newTestEditor("Hello\nWorld\nfoo")
		keys(e, 'j') // cursor on row 1
		keys(e, 'g', '~', 'g', 'g')
		assert.Equal(t, "hELLO\nwORLD\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gugg on first row lowercases that line", func(t *testing.T) {
		e := newTestEditor("HELLO\nworld")
		keys(e, 'g', 'u', 'g', 'g')
		assert.Equal(t, "hello\nworld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// ---------------------------------------------------------------------------
// gugu / gUgU / g~g~ — operator-repeat alias for guu/gUU/g~~
// ---------------------------------------------------------------------------

func TestCaseOpRepeat(t *testing.T) {
	t.Run("gugu lowercases current line (alias for guu)", func(t *testing.T) {
		e := newTestEditor("HELLO WORLD")
		keys(e, 'g', 'u', 'g', 'u')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("gUgU uppercases current line (alias for gUU)", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'g', 'U', 'g', 'U')
		assert.Equal(t, "HELLO WORLD", content(e))
	})

	t.Run("g~g~ toggles current line (alias for g~~)", func(t *testing.T) {
		e := newTestEditor("Hello World")
		keys(e, 'g', '~', 'g', '~')
		assert.Equal(t, "hELLO wORLD", content(e))
	})
}

// ---------------------------------------------------------------------------
// gu{n}G — case change to a specific line number
// ---------------------------------------------------------------------------

func TestCaseToTargetLine(t *testing.T) {
	t.Run("gU3G uppercases from cursor to line 3 (1-indexed)", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz\nqux")
		keys(e, 'g', 'U', '3', 'G')
		assert.Equal(t, "FOO\nBAR\nBAZ\nqux", content(e))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("gu2G lowercases from cursor line down to line 2", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nBAZ")
		keys(e, 'g', 'u', '2', 'G')
		assert.Equal(t, "foo\nbar\nBAZ", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("gU2G from row 2 up to line 2 is a single-line op", func(t *testing.T) {
		e := newTestEditor("foo\nhello\nbaz")
		keys(e, 'j', 'j') // cursor on row 2
		keys(e, 'g', 'U', '2', 'G')
		assert.Equal(t, "foo\nHELLO\nBAZ", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("guG (no count) still goes to end of buffer", func(t *testing.T) {
		e := newTestEditor("FOO\nBAR\nBAZ")
		keys(e, 'g', 'u', 'G')
		assert.Equal(t, "foo\nbar\nbaz", content(e))
	})
}
