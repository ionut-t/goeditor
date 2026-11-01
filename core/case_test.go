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

	// Vim 9.2: w skips lines of only blanks; e goes to the end of the buffer.
	runDollarCases(t, []dollarCase{
		{name: "gU2w over a blank line", content: "abc\n  \nx y", keys: keySeq("gU2w"), want: Position{0, 0}, text: "ABC\n  \nX y"},
		{name: "gU2e past the last word", content: "foobar\n   x", keys: keySeq("$gU2e"), want: Position{0, 5}, text: "foobaR\n   X"},
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

// TestCaseOnEmptyBlock checks gU, gu and g~ on an inner block with nothing
// in it. As d, y and c, and as Vim 9.2, the text stays as it is and the
// cursor, and the column j and k go to, move to just after the open
// bracket: on the close bracket, or at the start of the next line when the
// open bracket ends its line.
func TestCaseOnEmptyBlock(t *testing.T) {
	const (
		pair     = "x () y\nabcdefghij"
		ahead    = "f() {\n  x\n}\nabcdefghij"
		split    = "x (\n)\nabcdefghij"
		blank    = "x (\n    \n  )\nabcdefghij"
		indented = "x (\n  )\nabcdefghij"
	)

	runCurswantCases(t, []curswantCase{
		{pair, "llgUi(", NormalMode, Position{0, 3}, 3},
		{pair, "llgui(", NormalMode, Position{0, 3}, 3},
		{pair, "llg~i(", NormalMode, Position{0, 3}, 3},
		{pair, "gUi(", NormalMode, Position{0, 3}, 3},
		{pair, "gUi(j", NormalMode, Position{1, 3}, 3},
		{pair, "gUi(jk", NormalMode, Position{0, 3}, 3},
		{pair, "lllgUi(", NormalMode, Position{0, 3}, 3},
		{pair, "gUi)", NormalMode, Position{0, 3}, 3},
		{pair, "gUib", NormalMode, Position{0, 3}, 3},
		{"x [] y\nabcdefghij", "gUi[", NormalMode, Position{0, 3}, 3},
		{"x [] y\nabcdefghij", "gUi]", NormalMode, Position{0, 3}, 3},
		{"x [] y\nabcdefghij", "gUib", NormalMode, Position{0, 3}, 3},
		{"x {} y\nabcdefghij", "gUi{", NormalMode, Position{0, 3}, 3},
		{"x {} y\nabcdefghij", "gUi}", NormalMode, Position{0, 3}, 3},
		{"x {} y\nabcdefghij", "gUiB", NormalMode, Position{0, 3}, 3},
		{"x <> y\nabcdefghij", "gUi<lt>", NormalMode, Position{0, 3}, 3},
		{"x <> y\nabcdefghij", "gUi>", NormalMode, Position{0, 3}, 3},
		{ahead, "gUi(", NormalMode, Position{0, 2}, 2},
		{ahead, "gUi(j", NormalMode, Position{1, 2}, 2},
		{split, "llgUi(", NormalMode, Position{1, 0}, 0},
		{split, "llgUi(j", NormalMode, Position{2, 0}, 0},
		{split, "jgUi(", NormalMode, Position{1, 0}, 0},
		{blank, "jllgUi(", NormalMode, Position{1, 0}, 0},
		{blank, "jllgUi(j", NormalMode, Position{2, 0}, 0},
		{indented, "jlgUi(", NormalMode, Position{1, 0}, 0},
	})

	// The text doesn't change, but the command is the change '.' repeats.
	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "gUi( on an empty pair", content: "x () y", keys: "gUi(", text: "x () y", want: Position{0, 3}, reg: emptyRangeReg},
		{name: "gUi( is repeated", content: "x () y\nab (c) d", keys: "AQ<Esc>0gUi(jfc.", text: "x () yQ\nab (C) d", want: Position{1, 4}, reg: emptyRangeReg},
		{name: "g~ib is repeated", content: "x () y\nab (c) d", keys: "AQ<Esc>0g~ibjfc.", text: "x () yQ\nab (C) d", want: Position{1, 4}, reg: emptyRangeReg},
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

// ---------------------------------------------------------------------------
// gU/gu/g~ with h, l, j, k, f, F, t, T, ; and ,
// ---------------------------------------------------------------------------

// caseMotionCase is a key sequence and what Vim 9.2 leaves after it: the
// text, the cursor and the column j and k go to (-1 for MAXCOL).
type caseMotionCase struct {
	content string
	keys    string
	want    string
	pos     Position
	col     int
}

// TestCaseOperatorMotions checks the case operators with the character, line
// and character search motions against Vim 9.2 (open-bugs 36, 43 and 17).
// Vim got each step as its own feedkeys, so a motion that failed didn't
// flush the keys after it. Not here: h and 0 in column 0, where Vim runs the
// operator on an empty range and its op_tilde then changes the character
// under the cursor on line 1 and the whole line below that.
func TestCaseOperatorMotions(t *testing.T) {
	tests := []caseMotionCase{
		// h and l (36): the cursor goes to the start; l takes the last character at the end of the line, and a count goes only as far as the line.
		{"aBcDeF\naBcDeF\naBcDeF", "gUl", "ABcDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "gU2l", "ABcDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "2gU3l", "ABCDEF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "gU9l", "ABCDEF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "g~l", "ABcDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "gul", "aBcDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "$g~l", "aBcDef\naBcDeF\naBcDeF", Position{0, 5}, 5},
		{"aBcDeF\naBcDeF\naBcDeF", "$g~2l", "aBcDef\naBcDeF\naBcDeF", Position{0, 5}, 5},
		{"aBcDeF\naBcDeF\naBcDeF", "3lgUh", "aBCDeF\naBcDeF\naBcDeF", Position{0, 2}, 2},
		{"aBcDeF\naBcDeF\naBcDeF", "3lgU2h", "aBCDeF\naBcDeF\naBcDeF", Position{0, 1}, 1},
		{"aBcDeF\naBcDeF\naBcDeF", "3l2gUh", "aBCDeF\naBcDeF\naBcDeF", Position{0, 1}, 1},
		{"aBcDeF\naBcDeF\naBcDeF", "3l2gU2h", "ABCDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "3lgU9h", "ABCDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "3lg~9h", "AbCDeF\naBcDeF\naBcDeF", Position{0, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "$gUh", "aBcDEF\naBcDeF\naBcDeF", Position{0, 4}, 4},
		{"aBc\n\naBc", "jgUl", "aBc\n\naBc", Position{1, 0}, 0},
		{"aBc\n\naBc", "jgUlj.", "aBc\n\nABc", Position{2, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "gUlj.", "ABcDeF\nABcDeF\naBcDeF", Position{1, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "gU2lj3.", "ABcDeF\nABCDeF\naBcDeF", Position{1, 0}, 0},
		{"aBcDeF\naBcDeF\naBcDeF", "3lgU2hj.", "aBCDeF\nABcDeF\naBcDeF", Position{1, 0}, 0},
		// h and l over composing characters: the marks go with their letter.
		{"ae\u0301 b\nae\u0301 b", "lgUl", "aE\u0301 b\nae\u0301 b", Position{0, 1}, 1},
		{"ae\u0301 b\nae\u0301 b", "gU2l", "AE\u0301 b\nae\u0301 b", Position{0, 0}, 0},
		{"ae\u0301 b\nae\u0301 b", "llgUh", "aE\u0301 b\nae\u0301 b", Position{0, 1}, 1},
		{"ae\u0301 b\nae\u0301 b", "$gU3h", "AE\u0301 b\nae\u0301 b", Position{0, 0}, 0},
		{"ae\u0301 b\nae\u0301 b", "$gUl", "ae\u0301 B\nae\u0301 b", Position{0, 4}, 4},
		{"xe\u0301\u0301yz\nxe\u0301\u0301yz", "lgUl", "xE\u0301\u0301yz\nxe\u0301\u0301yz", Position{0, 1}, 1},
		{"xe\u0301\u0301yz\nxe\u0301\u0301yz", "llgUh", "xE\u0301\u0301yz\nxe\u0301\u0301yz", Position{0, 1}, 1},
		{"xe\u0301\u0301yz\nxe\u0301\u0301yz", "$gU2h", "xE\u0301\u0301Yz\nxe\u0301\u0301yz", Position{0, 1}, 1},
		{"xe\u0301\u0301yz\nxe\u0301\u0301yz", "gU2l", "XE\u0301\u0301yz\nxe\u0301\u0301yz", Position{0, 0}, 0},
		{"xe\u0301\u0301yz\nxe\u0301\u0301yz", "$gU2hj.", "xE\u0301\u0301Yz\nXe\u0301\u0301yz", Position{1, 0}, 0},
		// f, F, t and T (43): inclusive forwards, exclusive backwards; a search that finds nothing changes nothing and keeps the column.
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfc", "ABCDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gU2fc", "ABCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "2gUfc", "ABCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "g~fc", "AbCDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gufc", "abcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfx", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gU3fc", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfH", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUtc", "ABcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gU2tc", "ABCDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUtB", "ABcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$gUFa", "ABCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$g~Fa", "AbCdCF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$gUTa", "aBCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 1}, 1},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$gU2Fc", "aBCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 2}, 2},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$g~Tc", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 5}, 5},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "$gUFx", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 5}, -1},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "lgUFa", "ABcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "lg~Ta", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 1}, 1},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fc0gUFa", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfcj", "ABCDcF\naBcDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfcj.", "ABCDcF\nABCDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfcj2.", "ABCDcF\nABCDCF\naBcDeFgHiJ", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfcj0;.", "ABCDcF\naBCDCF\naBcDeFgHiJ", Position{1, 2}, 2},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gU2tcj.", "ABCDcF\nABCDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfxj.", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfx;", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUtc;", "ABcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 1}, 1},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gUfcgUfxj.", "ABCDcF\nABCDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		// f, F, t and T over composing characters: T can start on a mark, the cursor goes to its letter.
		{"ae\u0301 b\nae\u0301 b", "gUfb", "AE\u0301 B\nae\u0301 b", Position{0, 0}, 0},
		{"ae\u0301 b\nae\u0301 b", "gUtb", "AE\u0301 b\nae\u0301 b", Position{0, 0}, 0},
		{"ae\u0301 b\nae\u0301 b", "$gUFe", "aE\u0301 b\nae\u0301 b", Position{0, 1}, 1},
		{"ae\u0301 b\nae\u0301 b", "$gUTa", "aE\u0301 b\nae\u0301 b", Position{0, 1}, 1},
		{"ae\u0301 b\nae\u0301 b", "$g~Te", "ae\u0301 b\nae\u0301 b", Position{0, 1}, 1},
		// ; and , (43) repeat the last search, , the other way, ; past an adjacent t target; with none, nothing happens.
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fc0gU;", "ABCDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fc0gU2;", "ABCDCF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "2fc0gU;", "ABCDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fc$gU,", "aBcDCF\naBcDcF\naBcDeFgHiJ", Position{0, 4}, 4},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fc$gU,j.", "aBcDCF\naBCDcF\naBcDeFgHiJ", Position{1, 2}, 2},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "tc0gU;", "ABcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "tc0g~,", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "gU;", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "g~,", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "Fx0gU;", "aBcDcF\naBcDcF\naBcDeFgHiJ", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF\naBcDeFgHiJ", "fcgU;j0.", "aBCDCF\nABCDcF\naBcDeFgHiJ", Position{1, 0}, 0},
		// j and k (17) are linewise: the cursor stays for j and goes where k goes for k; on the last or first line nothing happens and the column stays.
		{"one Two\nthree Four\nfive Six\nSeven", "gUj", "ONE TWO\nTHREE FOUR\nfive Six\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "guj", "one two\nthree four\nfive Six\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "g~j", "ONE tWO\nTHREE fOUR\nfive Six\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "gU2j", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "2gUj", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "2gU2j", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSEVEN", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "gU9j", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSEVEN", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "GgUk", "one Two\nthree Four\nFIVE SIX\nSEVEN", Position{2, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "Gg~2k", "one Two\nTHREE fOUR\nFIVE sIX\nsEVEN", Position{1, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "GgU9k", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSEVEN", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "gUk", "one Two\nthree Four\nfive Six\nSeven", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "GgUj", "one Two\nthree Four\nfive Six\nSeven", Position{3, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "wgUj", "ONE TWO\nTHREE FOUR\nfive Six\nSeven", Position{0, 4}, 4},
		{"one Two\nthree Four\nfive Six\nSeven", "jwgUk", "ONE TWO\nTHREE FOUR\nfive Six\nSeven", Position{0, 6}, 6},
		{"one Two\nthree Four\nfive Six\nSeven", "$gUj", "ONE TWO\nTHREE FOUR\nfive Six\nSeven", Position{0, 6}, 6},
		{"one Two\nthree Four\nfive Six\nSeven", "jj$gUk", "one Two\nTHREE FOUR\nFIVE SIX\nSeven", Position{1, 9}, 9},
		{"one Two\nthree Four\nfive Six\nSeven", "jj$gUkj", "one Two\nTHREE FOUR\nFIVE SIX\nSeven", Position{2, 7}, 9},
		{"one Two\nthree Four\nfive Six\nSeven", "jjwgUk", "one Two\nTHREE FOUR\nFIVE SIX\nSeven", Position{1, 5}, 5},
		{"one Two\nthree Four\nfive Six\nSeven", "Gwg~k", "one Two\nthree Four\nFIVE sIX\nsEVEN", Position{2, 4}, 4},
		{"aBc\n\naBc", "jgUj", "aBc\n\nABC", Position{1, 0}, 0},
		{"aBc\n\naBc", "jgUk", "ABC\n\naBc", Position{0, 0}, 0},
		{"one Two\nthree Four", "$gUkj", "one Two\nthree Four", Position{1, 9}, -1},
		{"one Two\nthree Four", "j$gUjk", "one Two\nthree Four", Position{0, 6}, -1},
		{"one Two\nthree Four\nfive Six\nSeven", "gUjjj.", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSEVEN", Position{2, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "gU2jj2.", "ONE TWO\nTHREE FOUR\nFIVE SIX\nSEVEN", Position{1, 0}, 0},
		{"one Two\nthree Four\nfive Six\nSeven", "GgUkgg.", "one Two\nthree Four\nFIVE SIX\nSEVEN", Position{0, 0}, 0},
		// '.' repeats an empty range, but not a search or j/k that failed.
		{"aBcDcF\naBcDcF", "iQ<Esc>$g~Tcj$.", "QaBcDcF\naBcDcF", Position{1, 5}, 5},
		{"aBcDcF\naBcDcF", "iQ<Esc>$g~Txj$.", "QaBcDcF\naBcDcQF", Position{1, 5}, 5},
		{"aBcDcF\naBcDcF", "iQ<Esc>g~fxj.", "QaBcDcF\nQaBcDcF", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF", "iQ<Esc>g~;j.", "QaBcDcF\nQaBcDcF", Position{1, 0}, 0},
		{"aBcDcF\naBcDcF", "iQ<Esc>jg~jk.", "QQaBcDcF\naBcDcF", Position{0, 0}, 0},
		{"aBcDcF\naBcDcF", "iQ<Esc>g~kj.", "QaBcDcF\nQaBcDcF", Position{1, 0}, 0},
		{"aBc\n\n\naBc", "iQ<Esc>jgUjjj.", "QaBc\n\n\naBc", Position{3, 0}, 0},
		// u puts the cursor where the change started.
		{"one Two\nthree Four\nfive Six", "wgUjwu", "one Two\nthree Four\nfive Six", Position{0, 4}, 4},
		{"one Two\nthree Four\nfive Six", "jjwgUkwu", "one Two\nthree Four\nfive Six", Position{1, 5}, 5},
		{"one Two\nthree Four\nfive Six", "wlgUl0u", "one Two\nthree Four\nfive Six", Position{0, 5}, 5},
		{"one Two\nthree Four\nfive Six", "$gUFo0u", "one Two\nthree Four\nfive Six", Position{0, 0}, 0},
		{"one Two\nthree Four\nfive Six", "$gU3h0u", "one Two\nthree Four\nfive Six", Position{0, 3}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, tt.pos, cursorPos(e))
			col := e.GetBuffer().GetCursor().Preferred
			if col == maxCol {
				col = -1
			}
			assert.Equal(t, tt.col, col, "column for j and k")
		})
	}
}
