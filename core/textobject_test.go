package core

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// quoteTextObjectRange
// ---------------------------------------------------------------------------

func TestQuoteTextObjectRange(t *testing.T) {
	buf := func(line string) Buffer {
		e := newTestEditor(line)
		return e.GetBuffer()
	}

	t.Run("cursor inside double quotes – inner", func(t *testing.T) {
		b := buf(`say "hello" world`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'i', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 5, s)
		assert.Equal(t, 9, e)
	})

	t.Run("cursor inside double quotes – around takes the space after", func(t *testing.T) {
		b := buf(`say "hello" world`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'a', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 4, s)
		assert.Equal(t, 11, e)
	})

	t.Run("cursor on opening quote – inner", func(t *testing.T) {
		b := buf(`"hello"`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 5, e)
	})

	t.Run("cursor on closing quote – around", func(t *testing.T) {
		b := buf(`"hello"`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'a', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 0, s)
		assert.Equal(t, 6, e)
	})

	t.Run("cursor before first quote pair – uses that pair", func(t *testing.T) {
		b := buf(`x "foo" y`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 3, s)
		assert.Equal(t, 5, e)
	})

	t.Run("empty quotes – inner returns not found", func(t *testing.T) {
		b := buf(`""`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"', 1)
		assert.False(t, ok)
	})

	t.Run("empty quotes – around returns the two quote chars", func(t *testing.T) {
		b := buf(`""`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'a', '"', 1)
		assert.True(t, ok)
		assert.Equal(t, 0, s)
		assert.Equal(t, 1, e)
	})

	t.Run("no quotes – not found", func(t *testing.T) {
		b := buf(`hello`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 2}, 'i', '"', 1)
		assert.False(t, ok)
	})

	t.Run("single quote variant", func(t *testing.T) {
		// No apostrophes: two single-quote chars form one clean pair.
		b := buf(`hello 'world' end`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 8}, 'i', '\'', 1)
		assert.True(t, ok)
		assert.Equal(t, 7, s)
		assert.Equal(t, 11, e)
	})

	t.Run("backtick variant", func(t *testing.T) {
		b := buf("`cmd`")
		s, e, ok := quoteTextObjectRange(b, Position{0, 2}, 'i', '`', 1)
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 3, e)
	})

	t.Run("cursor past all pairs – not found", func(t *testing.T) {
		b := buf(`"a" x`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 4}, 'i', '"', 1)
		assert.False(t, ok)
	})
}

// TestAroundQuoteWhiteSpace checks a", a', a` and aq against Vim 9.2, which
// takes the white space after the closing quote or, when there is none, the
// white space before the opening one (:h a"). Only spaces and tabs count.
func TestAroundQuoteWhiteSpace(t *testing.T) {
	tests := []struct {
		content string
		col     int
		keys    string
		want    string
		cursor  int
		reg     string // checked for yanks only
		mode    Mode
	}{
		{`ab "y" z`, 0, `da"`, `ab z`, 3, "", NormalMode},
		{`ab "y" z`, 4, `da"`, `ab z`, 3, "", NormalMode},
		{`ab "y" z`, 5, `ya"`, `ab "y" z`, 3, `"y" `, NormalMode},
		{`ab "y" z`, 4, `ca"`, `ab z`, 3, "", InsertMode},
		{`ab "y" z`, 3, `va"d`, `ab z`, 3, "", NormalMode},
		{`ab "y"z`, 4, `da"`, `abz`, 2, "", NormalMode},
		{`ab "y"z`, 0, `ya"`, `ab "y"z`, 2, ` "y"`, NormalMode},
		{`ab "y"z`, 5, `va"d`, `abz`, 2, "", NormalMode},
		{`ab"y"z`, 3, `da"`, `abz`, 2, "", NormalMode},
		{`ab "y"`, 4, `da"`, `ab`, 1, "", NormalMode},
		{`ab "y"`, 4, `ya"`, `ab "y"`, 2, ` "y"`, NormalMode},
		{`ab "y"`, 0, `ca"`, `ab`, 2, "", InsertMode},
		{`ab"y"`, 3, `ca"`, `ab`, 2, "", InsertMode},
		{`ab"y"`, 3, `ca"X`, `abX`, 3, "", InsertMode},
		{`ab"y"`, 3, `da"`, `ab`, 1, "", NormalMode},
		{`ab "y"   `, 4, `da"`, `ab `, 2, "", NormalMode},
		{`ab  "y"  z`, 5, `da"`, `ab  z`, 4, "", NormalMode},
		{"ab\t\"y\"\tz", 4, `da"`, "ab\tz", 3, "", NormalMode},
		{"ab\t\"y\"z", 4, `ya"`, "ab\t\"y\"z", 2, "\t\"y\"", NormalMode},
		{`  "y"`, 3, `da"`, ``, 0, "", NormalMode},
		{`"y" z`, 1, `da"`, `z`, 0, "", NormalMode},
		{`"y"`, 1, `ya"`, `"y"`, 0, `"y"`, NormalMode},
		{"ab\u00a0\"y\"\u00a0z", 4, `da"`, "ab\u00a0\u00a0z", 3, "", NormalMode},
		{"ab\u3000\"y\"z", 4, `ya"`, "ab\u3000\"y\"z", 3, `"y"`, NormalMode},
		{`ab "y"z`, 4, `g~a"`, `ab "Y"z`, 2, "", NormalMode},
		{`ab 'y' z`, 4, `da'`, `ab z`, 3, "", NormalMode},
		{`ab 'y'z`, 0, `va'd`, `abz`, 2, "", NormalMode},
		{"ab `y`z", 4, "ya`", "ab `y`z", 2, " `y`", NormalMode},
		// Vim has no aq; it follows a" for the quote it picks.
		{`ab 'y' z`, 4, `daq`, `ab z`, 3, "", NormalMode},
		{"ab `y`z", 4, `caq`, `abz`, 2, "", InsertMode},
		{`ab "y"`, 4, `vaqd`, `ab`, 1, "", NormalMode}, // iq/aq only take a pair around the cursor
	}
	for _, tt := range tests {
		t.Run(tt.keys+" at "+strconv.Itoa(tt.col)+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			cursor := e.GetBuffer().GetCursor()
			cursor.Position.Col = tt.col
			e.GetBuffer().SetCursor(cursor)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, Position{0, tt.cursor}, cursorPos(e))
			assert.Equal(t, tt.mode, e.GetMode().Name())
			if strings.HasPrefix(tt.keys, "y") {
				assert.Equal(t, tt.reg, cb.content)
			}
		})
	}
}

// TestVisualAroundQuoteWhiteSpace checks the selection of va" and vaq.
func TestVisualAroundQuoteWhiteSpace(t *testing.T) {
	tests := []struct {
		content    string
		col        int
		keys       string
		start, end int
	}{
		{`ab "y" z`, 0, `va"`, 3, 6},
		{`ab "y" z`, 5, `va"`, 3, 6},
		{`ab "y"z`, 4, `va"`, 2, 5},
		{`ab"y"z`, 4, `va"`, 2, 4},
		{`ab "y"   `, 4, `va"`, 3, 8},
		{"ab\t\"y\"z", 4, `va"`, 2, 5},
		{"ab\u00a0\"y\"\u00a0z", 4, `va"`, 3, 5},
		{`  "y"`, 0, `va"`, 0, 4},
		{`ab 'y'z`, 4, `vaq`, 2, 5},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" at "+strconv.Itoa(tt.col)+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			cursor := e.GetBuffer().GetCursor()
			cursor.Position.Col = tt.col
			e.GetBuffer().SetCursor(cursor)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, Position{0, tt.start}, e.GetState().VisualStart)
			assert.Equal(t, Position{0, tt.end}, cursorPos(e))
		})
	}
}

// ---------------------------------------------------------------------------
// anyQuoteTextObjectRange
// ---------------------------------------------------------------------------

func TestAnyQuoteTextObjectRange(t *testing.T) {
	buf := func(line string) Buffer {
		e := newTestEditor(line)
		return e.GetBuffer()
	}

	t.Run("picks double quotes when inside them", func(t *testing.T) {
		b := buf(`"hello"`)
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 3}, 'i', 1)
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 5, e)
	})

	t.Run("picks innermost when nested quotes on same line", func(t *testing.T) {
		// outer: " at 0 and 12; inner: ' at 5 and 9
		b := buf(`"foo 'bar' baz"`)
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 7}, 'i', 1)
		assert.True(t, ok)
		// should pick the ' pair (span 4) rather than the " pair (span 14)
		assert.Equal(t, 6, s)
		assert.Equal(t, 8, e)
	})

	t.Run("no quotes – not found", func(t *testing.T) {
		b := buf(`hello`)
		_, _, ok := anyQuoteTextObjectRange(b, Position{0, 2}, 'i', 1)
		assert.False(t, ok)
	})

	t.Run("around modifier includes quote chars", func(t *testing.T) {
		b := buf(`'hi'`)
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 1}, 'a', 1)
		assert.True(t, ok)
		assert.Equal(t, 0, s)
		assert.Equal(t, 3, e)
	})
}

// ---------------------------------------------------------------------------
// bracketTextObjectRange
// ---------------------------------------------------------------------------

func TestBracketTextObjectRange(t *testing.T) {
	buf := func(lines ...string) Buffer {
		e := newTestEditor(strings.Join(lines, "\n"))
		return e.GetBuffer()
	}

	t.Run("cursor inside parens – inner", func(t *testing.T) {
		b := buf("(hello)")
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'i', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 5}, e)
	})

	t.Run("cursor inside parens – around", func(t *testing.T) {
		b := buf("(hello)")
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'a', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 6}, e)
	})

	t.Run("cursor on open paren – inner", func(t *testing.T) {
		b := buf("(foo)")
		s, e, ok := bracketTextObjectRange(b, Position{0, 0}, 'i', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("cursor on close paren – around", func(t *testing.T) {
		b := buf("(foo)")
		// cursor at col 4 (the ')'); findBracketBounds scans backward from col 3 to find '('
		s, e, ok := bracketTextObjectRange(b, Position{0, 4}, 'a', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 4}, e)
	})

	t.Run("nested parens – inner selects innermost from inside", func(t *testing.T) {
		b := buf("(a(b)c)")
		// cursor on 'b' at col 3
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'i', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 3}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("nested parens – outer selected when cursor outside inner", func(t *testing.T) {
		b := buf("(a(b)c)")
		// cursor on 'a' at col 1
		s, e, ok := bracketTextObjectRange(b, Position{0, 1}, 'i', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 5}, e)
	})

	t.Run("empty parens – inner not found", func(t *testing.T) {
		b := buf("()")
		_, _, ok := bracketTextObjectRange(b, Position{0, 0}, 'i', '(', ')', 1)
		assert.False(t, ok)
	})

	t.Run("empty parens – around returns the two chars", func(t *testing.T) {
		b := buf("()")
		s, e, ok := bracketTextObjectRange(b, Position{0, 0}, 'a', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 1}, e)
	})

	t.Run("square brackets", func(t *testing.T) {
		b := buf("[1, 2, 3]")
		s, e, ok := bracketTextObjectRange(b, Position{0, 4}, 'i', '[', ']', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 7}, e)
	})

	t.Run("curly braces", func(t *testing.T) {
		b := buf("{key: val}")
		s, e, ok := bracketTextObjectRange(b, Position{0, 5}, 'i', '{', '}', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 8}, e)
	})

	t.Run("angle brackets", func(t *testing.T) {
		b := buf("<div>")
		s, e, ok := bracketTextObjectRange(b, Position{0, 2}, 'i', '<', '>', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("multi-line – inner", func(t *testing.T) {
		b := buf("(", "  foo", "  bar", ")")
		// cursor on "foo" line
		s, e, ok := bracketTextObjectRange(b, Position{1, 2}, 'i', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{1, 0}, s) // first char of line after '('
		assert.Equal(t, Position{2, 5}, e) // line break before ')' (Vim's vi( selects it)
	})

	t.Run("multi-line – around", func(t *testing.T) {
		b := buf("(", "  foo", ")")
		s, e, ok := bracketTextObjectRange(b, Position{1, 2}, 'a', '(', ')', 1)
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{2, 0}, e)
	})

	t.Run("no enclosing bracket – not found", func(t *testing.T) {
		b := buf("hello")
		_, _, ok := bracketTextObjectRange(b, Position{0, 2}, 'i', '(', ')', 1)
		assert.False(t, ok)
	})
}

// TestCountedQuoteObjects checks a count on i" (and i', i`): above 1 it takes
// the quotes but no white space (:h i"); every count above 1 is the same. a"
// ignores the count; '.' repeats it. iq has its own rows
// (TestCountedAnyQuoteObject). Rows from Vim 9.2 (current_quote); the
// clipboard starts as "REG", and d and c rows don't check it.
func TestCountedQuoteObjects(t *testing.T) {
	const two = `a"z"b x"y"`
	const spaced = `a "z" b`
	const wide = `a "zz" b`
	runCountedObjectCases(t, []countedObjectCase{
		{two, `2ld2i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `2ld3i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `2ld9i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `2ly2i"`, two, "n", Position{0, 1}, Position{}, `"z"`},
		{two, `2lc2i"`, `ab x"y"`, "i", Position{0, 1}, Position{}, anyReg},
		{two, `ld2i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `3ld2i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `d2i"`, `ab x"y"`, "n", Position{0, 1}, Position{}, anyReg},
		{two, `$d2i"`, `a"z"b x`, "n", Position{0, 6}, Position{}, anyReg},
		{spaced, `3ld2i"`, `a  b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `ld2i"`, `a  b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `d2i"`, `a  b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `3l2di"`, `a  b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `3ly3i"`, spaced, "n", Position{0, 2}, Position{}, `"z"`},
		{spaced, `3ld2a"`, `a b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `d2a"`, `a b`, "n", Position{0, 2}, Position{}, anyReg},
		{spaced, `3lgU2i"`, `a "Z" b`, "n", Position{0, 2}, Position{}, "REG"},
		{`a""b`, `ld2i"`, `ab`, "n", Position{0, 1}, Position{}, anyReg},
		{`a""b`, `ld2a"`, `ab`, "n", Position{0, 1}, Position{}, anyReg},
		{`a'z'b`, `2ld2i'`, `ab`, "n", Position{0, 1}, Position{}, anyReg},
		{"a`z`b", "2ld2i`", "ab", "n", Position{0, 1}, Position{}, anyReg},
		// Visual: the quotes, selected forward, charwise from V.
		{wide, `3lv2i"`, wide, "v", Position{0, 5}, Position{0, 2}, "REG"},
		{wide, `3lv3i"`, wide, "v", Position{0, 5}, Position{0, 2}, "REG"},
		{wide, `3lv2a"`, wide, "v", Position{0, 6}, Position{0, 2}, "REG"},
		{wide, `3lvl2i"`, wide, "v", Position{0, 5}, Position{0, 2}, "REG"},
		{wide, `3lV2i"`, wide, "v", Position{0, 5}, Position{0, 2}, "REG"},
		{wide, `v2i"`, wide, "v", Position{0, 5}, Position{0, 2}, "REG"},
		{wide, `3lv2i"d`, `a  b`, "n", Position{0, 2}, Position{}, anyReg},
		{`a""b`, `lv2i"`, `a""b`, "v", Position{0, 2}, Position{0, 1}, "REG"},
		// '.' repeats the count.
		{"a\"z\"b\na\"z\"b", `2lc2i"Q<Esc>j02l.`, "aQb\naQb", "n", Position{1, 1}, Position{}, anyReg},
		{spaced + "\n" + spaced, `3ld2i"j03l.`, "a  b\na  b", "n", Position{1, 2}, Position{}, anyReg},
	})
}

// TestCountedAnyQuoteObject checks iq/aq, which Vim doesn't have. They
// follow mini.ai's q with search_method 'cover':
// the shortest pair of any quote type around the cursor, quotes included,
// none when no pair is around it, and a count goes out that many pairs. aq
// takes white space as a" does. Unlike mini.ai, a failed v2iq keeps the
// selection, as every other object that isn't found.
func TestCountedAnyQuoteObject(t *testing.T) {
	const nested = `x "a 'z' b" y` // z at column 7
	runCountedObjectCases(t, []countedObjectCase{
		{nested, "7ldiq", `x "a '' b" y`, "n", Position{0, 6}, Position{}, anyReg},
		{nested, "7ld2iq", `x "" y`, "n", Position{0, 3}, Position{}, anyReg},
		{nested, "7ld3iq", nested, "n", Position{0, 7}, Position{}, anyReg},
		{nested, "5ldiq", `x "a '' b" y`, "n", Position{0, 6}, Position{}, anyReg}, // on the '
		{nested, "4ldiq", `x "" y`, "n", Position{0, 3}, Position{}, anyReg},       // inside " only
		{nested, "4ld2iq", nested, "n", Position{0, 4}, Position{}, anyReg},
		{nested, "7ly2iq", nested, "n", Position{0, 3}, Position{}, `a 'z' b`},
		{nested, "7lc2iq", `x "" y`, "i", Position{0, 3}, Position{}, anyReg},
		{nested, "7ld2aq", "x y", "n", Position{0, 2}, Position{}, anyReg},
		{nested, "7lgU2iq", `x "A 'Z' B" y`, "n", Position{0, 3}, Position{}, "REG"},
		{nested, "7lv2iq", nested, "v", Position{0, 9}, Position{0, 3}, "REG"},
		{nested, "7lv3iq", nested, "v", Position{0, 7}, Position{0, 7}, "REG"},
		{nested + "\n" + nested, "7ld2iqj07l.", `x "" y` + "\n" + `x "" y`, "n", Position{1, 3}, Position{}, anyReg},
		{`a 'z' b`, "3ld2iq", `a 'z' b`, "n", Position{0, 3}, Position{}, anyReg},
		{`a 'z' b`, "3ld2aq", `a 'z' b`, "n", Position{0, 3}, Position{}, anyReg},
		{`a "z" b`, "diq", `a "z" b`, "n", Position{0, 0}, Position{}, anyReg}, // no pair ahead
		{`a"z"b x"y"`, "4ldiq", `a"z"b x"y"`, "n", Position{0, 4}, Position{}, anyReg},
		{`a 'zz' b`, "3lviq", `a 'zz' b`, "v", Position{0, 4}, Position{0, 3}, "REG"},
		{`a 'zz' b`, "3lv2iq", `a 'zz' b`, "v", Position{0, 3}, Position{0, 3}, "REG"},
	})
}
