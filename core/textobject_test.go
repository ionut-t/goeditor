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
// quotePair: which quotes off a quote, and escapes
// ---------------------------------------------------------------------------

// TestQuotePair checks the pair Vim's current_quote picks for i" (open-bugs
// 54): on a quote, quotes pair from the start of the line; off a quote, the
// nearest quote before the cursor opens the string and the next one closes
// it; with none before, the first two after. A quote after a backslash
// (Vim's default 'quoteescape') doesn't close a string. All results checked
// against Vim 9.2.
func TestQuotePair(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		col         int
		quote       rune
		open, close int
		found       bool
	}{
		{"between two strings", `a"z"b x"y"`, 4, '"', 3, 7, true},
		{"between two strings, single quote", `a'z'b x'y'`, 4, '\'', 3, 7, true},
		{"between two strings, backtick", "a`z`b x`y`", 4, '`', 3, 7, true},
		{"inside the first string", `a"z"b x"y"`, 2, '"', 1, 3, true},
		{"on the opening quote", `a"z"b x"y"`, 1, '"', 1, 3, true},
		{"on a closing quote: paired from the line start", `a"z"b x"y"`, 3, '"', 1, 3, true},
		{"on the second string's opening quote", `a"z"b x"y"`, 7, '"', 7, 9, true},
		{"before the first quote", `ab "x" cd`, 0, '"', 3, 5, true},
		{"after the last quote", `"x" ab`, 5, '"', 0, 0, false},
		{"odd quotes, between the second and third", `a"b"c"d`, 4, '"', 3, 5, true},
		{"odd quotes, after the last", `a"b"c"d`, 6, '"', 0, 0, false},
		{"odd quotes, on the unpaired last quote", `a"b"c"d`, 5, '"', 0, 0, false},
		{"no quotes", `no quotes`, 3, '"', 0, 0, false},
		{"escaped quote doesn't close", `"a\"b" c`, 1, '"', 0, 5, true},
		{"escaped quote skipped searching back", `"a\"b" c`, 4, '"', 0, 5, true},
		{"on an escaped quote: paired from the line start", `"a\"b" c`, 3, '"', 0, 5, true},
		{"escaped quote between two strings", `x "a" \" b "c"`, 8, '"', 4, 11, true},
		{"escaped quote opens searching forward", `x \"a" b`, 0, '"', 3, 5, true},
		{"escaped quote before the cursor opens from the line start", `\"ab`, 2, '"', 0, 0, false},
		{"escaped backslash: the quote closes", `"a\\"b" c`, 1, '"', 0, 4, true},
		{"backslash at the line end", `x "a\`, 3, '"', 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferFromBytes([]byte(tt.line))
			open, close, found := quotePair(b, Position{0, tt.col}, tt.quote)
			assert.Equal(t, tt.found, found)
			if tt.found {
				assert.Equal(t, [2]int{tt.open, tt.close}, [2]int{open, close})
			}
		})
	}
}

// TestQuoteObjectBetweenStrings runs the operators, visual mode, counts and
// . on quote objects with the cursor off a quote (open-bugs 54). Results
// from Vim 9.2.
func TestQuoteObjectBetweenStrings(t *testing.T) {
	tests := []struct {
		content, keys string
		want          string
		cursor        Position
		clipboard     string // "" when the keys don't yank
	}{
		{`a"z"b x"y"`, `4ldi"`, `a"z""y"`, Position{0, 4}, ""},
		{`a"z"b x"y"`, `4lda"`, `a"zy"`, Position{0, 3}, ""},
		{`a"z"b x"y"`, `4lyi"`, `a"z"b x"y"`, Position{0, 4}, "b x"},
		{`a"z"b x"y"`, `4lya"`, `a"z"b x"y"`, Position{0, 3}, `"b x"`},
		{`a"z"b x"y"`, `4lci"X<Esc>`, `a"z"X"y"`, Position{0, 4}, ""},
		{`a"z"b x"y"`, `4lca"X<Esc>`, `a"zXy"`, Position{0, 3}, ""},
		{`a"z"b x"y"`, `4lgUi"`, `a"z"B X"y"`, Position{0, 4}, ""},
		{`a"z"b x"y"`, `4lgUa"`, `a"z"B X"y"`, Position{0, 3}, ""},
		{`a"z"b x"y"`, `4l2di"`, `a"zy"`, Position{0, 3}, ""},
		{`a"z"b x"y"`, `4l2yi"`, `a"z"b x"y"`, Position{0, 3}, `"b x"`},
		{`a'z'b x'y'`, `4ldi'`, `a'z''y'`, Position{0, 4}, ""},
		{"a`z`b x`y`", "4lda`", "a`zy`", Position{0, 3}, ""},
		{`a"b"c"d`, `4ldi"`, `a"b""d`, Position{0, 4}, ""},
		{`a"b"c"d"e`, `4lda"`, `a"bd"e`, Position{0, 3}, ""},
		{`a"b"c"d`, `6ldi"`, `a"b"c"d`, Position{0, 6}, ""},
		{`a"" b ""c`, `4ldi"`, `a""""c`, Position{0, 3}, ""},
		// a" takes the white space after the closing quote, else before the
		// opening one, of the string found between the two.
		{`a "z"  b  "y" c`, `7lda"`, `a "zy" c`, Position{0, 4}, ""},
		{`"z" b " y"`, `4lda"`, `"zy"`, Position{0, 2}, ""},
		// Escaped quotes.
		{`"a\"b" c`, `4ldi"`, `"" c`, Position{0, 1}, ""},
		{`x "a\" b" "c"`, `6ldi"`, `x "" "c"`, Position{0, 3}, ""},
		{`x "a" \" b "c"`, `8ldi"`, `x "a""c"`, Position{0, 5}, ""},
		// . finds the pair again at the new cursor.
		{`a"z"b x"y"c"w"d`, `4ldi"3l.`, `a"z""y""w"d`, Position{0, 7}, ""},
		{`x "a" y "b" z "c" w`, `6l2di"4l.`, `x "abc" w`, Position{0, 5}, ""},
		{`a"z"b x"y"c"w"d e`, `4lci"Q<Esc>4l.`, `a"z"Q"y"Q"w"d e`, Position{0, 8}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, tt.cursor.Col, e.GetBuffer().GetCursor().Preferred, "preferred column")
			assert.Equal(t, NormalMode, e.GetMode().Name())
			if tt.clipboard != "" {
				assert.Equal(t, tt.clipboard, cb.content)
			}
		})
	}
}

// TestVisualQuoteObjectBetweenStrings: v and V from one character select the
// quote object off a quote as i" does; V switches to v.
func TestVisualQuoteObjectBetweenStrings(t *testing.T) {
	tests := []struct {
		content, keys  string
		anchor, cursor Position
	}{
		{`a"z"b x"y"`, `4lvi"`, Position{0, 4}, Position{0, 6}},
		{`a"z"b x"y"`, `4lva"`, Position{0, 3}, Position{0, 7}},
		{`a"z"b x"y"`, `4lVi"`, Position{0, 4}, Position{0, 6}},
		{`a"z"b x"y"`, `4lVa"`, Position{0, 3}, Position{0, 7}},
		{`a"z"b x"y"`, `4lv2i"`, Position{0, 3}, Position{0, 7}},
		{`x "a" y "b" z`, `6lva"`, Position{0, 4}, Position{0, 8}},
		{`a"b"c"d"e`, `4lva"`, Position{0, 3}, Position{0, 5}},
		{`"z" b " y"`, `4lva"`, Position{0, 2}, Position{0, 7}},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, VisualMode, e.GetMode().Name())
			assert.Equal(t, tt.anchor, e.GetState().VisualStart)
			assert.Equal(t, tt.cursor, cursorPos(e))
		})
	}
}

// TestQuoteObjectFromSelection checks i"/a" (and ' and `) on a selection over
// more than one character, as the Visual part of Vim 9.2's current_quote (see
// selectionQuoteRange). Rows come from Vim; columns are 0-based.
func TestQuoteObjectFromSelection(t *testing.T) {
	const (
		zz               = `a "zz" b`
		between          = `a"x" "y"b`
		long             = `x "ab cd" y "ef" z`
		escaped          = `a "b\"c" d`
		escapedBackslash = `a "b\\" c "d"`
		empty            = `"a" "" "b"`
	)
	tests := []struct {
		content, keys  string
		anchor, cursor Position
		mode           Mode
	}{
		// Repeated on the selection it made: i" takes the quotes.
		{zz, `3lvi"i"`, Position{0, 2}, Position{0, 5}, VisualMode},
		{zz, `3lva"a"`, Position{0, 2}, Position{0, 6}, VisualMode},
		{zz, `3lvi"a"`, Position{0, 2}, Position{0, 6}, VisualMode},
		{zz, `3lva"i"`, Position{0, 2}, Position{0, 6}, VisualMode},
		{zz, `3lvl2i"`, Position{0, 2}, Position{0, 5}, VisualMode},
		{zz, `3lvli"`, Position{0, 2}, Position{0, 5}, VisualMode},
		// Forwards: the anchor stays unless the selection was just inside or
		// started off a quote; on a quote the cursor takes the next string.
		{zz, `vlli"`, Position{0, 0}, Position{0, 4}, VisualMode},
		{zz, `vllla"`, Position{0, 0}, Position{0, 6}, VisualMode},
		{zz, `2lvli"`, Position{0, 2}, Position{0, 4}, VisualMode},
		{zz, `4lvlli"`, Position{0, 4}, Position{0, 6}, VisualMode},
		{zz, `5lvli"`, Position{0, 5}, Position{0, 6}, VisualMode},
		// Backwards: the cursor goes to the start, the anchor to the end when
		// the selection holds no quote and isn't next to one.
		{zz, `5lvhi"`, Position{0, 5}, Position{0, 3}, VisualMode},
		{zz, `3lvhi"`, Position{0, 3}, Position{0, 2}, VisualMode},
		{zz, `4lvhhi"`, Position{0, 4}, Position{0, 2}, VisualMode},
		{zz, `7lvhi"`, Position{0, 4}, Position{0, 3}, VisualMode},
		{zz, `3lvlloi"`, Position{0, 5}, Position{0, 3}, VisualMode},
		// Cursor past the end of the line: no string after it, kept.
		{zz, `7lvli"`, Position{0, 7}, Position{0, 8}, VisualMode},
		// Between two strings.
		{between, `lvlli"`, Position{0, 1}, Position{0, 6}, VisualMode},
		{between, `3lvlli"`, Position{0, 3}, Position{0, 6}, VisualMode},
		{between, `4lvli"`, Position{0, 4}, Position{0, 6}, VisualMode},
		{between, `7lvhi"`, Position{0, 7}, Position{0, 6}, VisualMode},
		{between, `5lvhhi"`, Position{0, 5}, Position{0, 2}, VisualMode},
		{between, `3lvhhi"`, Position{0, 3}, Position{0, 1}, VisualMode},
		{between, `lvlla"`, Position{0, 1}, Position{0, 7}, VisualMode},
		// Inside, forwards and backwards from between the strings, counts.
		{long, `4lvli"`, Position{0, 3}, Position{0, 7}, VisualMode},
		{long, `10lvli"`, Position{0, 13}, Position{0, 14}, VisualMode},
		{long, `10lvhi"`, Position{0, 7}, Position{0, 3}, VisualMode},
		{long, `4lvlloi"`, Position{0, 7}, Position{0, 3}, VisualMode},
		{long, `6lvhhi"`, Position{0, 7}, Position{0, 3}, VisualMode},
		{long, `3lvhhi"`, Position{0, 3}, Position{0, 1}, VisualMode},
		{long, `10lvla"`, Position{0, 12}, Position{0, 16}, VisualMode},
		{long, `10lvl2i"`, Position{0, 12}, Position{0, 15}, VisualMode},
		// Escaped quotes: \" doesn't close a string, but the checks for
		// "just inside" and "holds a quote" look at the characters only.
		{escaped, `3lvli"`, Position{0, 2}, Position{0, 7}, VisualMode},
		{escaped, `5lvli"`, Position{0, 5}, Position{0, 6}, VisualMode},
		{escaped, `3lvi"i"`, Position{0, 3}, Position{0, 7}, VisualMode},
		{escaped, `6lvhi"`, Position{0, 6}, Position{0, 3}, VisualMode},
		{escaped, `3lvi"a"`, Position{0, 3}, Position{0, 8}, VisualMode},
		{escapedBackslash, `3lvli"`, Position{0, 3}, Position{0, 5}, VisualMode},
		{escapedBackslash, `3lvi"i"`, Position{0, 2}, Position{0, 6}, VisualMode},
		{escapedBackslash, `7lvli"`, Position{0, 7}, Position{0, 11}, VisualMode},
		// Empty quotes.
		{empty, `5lvli"`, Position{0, 5}, Position{0, 9}, VisualMode},
		{empty, `3lvli"`, Position{0, 3}, Position{0, 7}, VisualMode},
		{empty, `5lvhi"`, Position{0, 5}, Position{0, 1}, VisualMode},
		{empty, `4lvli"`, Position{0, 4}, Position{0, 8}, VisualMode},
		// The other quotes, and no string at all (kept).
		{`a 'zz' b`, `3lvi'i'`, Position{0, 2}, Position{0, 5}, VisualMode},
		{"a `zz` b", "3lvi`i`", Position{0, 2}, Position{0, 5}, VisualMode},
		{`a 'zz' b`, `3lvl2a'`, Position{0, 2}, Position{0, 6}, VisualMode},
		{`abc def`, `lvli"`, Position{0, 1}, Position{0, 2}, VisualMode},
		// From V on one line: found switches to v, not found stays V.
		{zz, `Vlli"`, Position{0, 0}, Position{0, 4}, VisualMode},
		{zz, `3lVlli"`, Position{0, 3}, Position{0, 5}, VisualLineMode},
		{zz, `3lVhi"`, Position{0, 3}, Position{0, 2}, VisualLineMode},
		{zz, `2lVli"`, Position{0, 2}, Position{0, 4}, VisualMode},
		{zz, `3lVi"i"`, Position{0, 2}, Position{0, 5}, VisualMode},
		{zz, `3lVla"`, Position{0, 2}, Position{0, 6}, VisualMode},
		// Over more than one line: never found, kept.
		{"a \"zz\" b\nc \"yy\" d", `vji"`, Position{0, 0}, Position{1, 0}, VisualMode},
		{"a \"zz\" b\nc \"yy\" d", `Vji"`, Position{0, 0}, Position{1, 0}, VisualLineMode},
		{"a \"zz\" b\nc \"yy\" d", `3ljvki"`, Position{1, 3}, Position{0, 3}, VisualMode},
		{"a \"zz\" b\nc \"yy\" d", `3lvjoa"`, Position{1, 3}, Position{0, 3}, VisualMode},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.content, content(e))
			assert.Equal(t, tt.mode, e.GetMode().Name())
			assert.Equal(t, tt.anchor, e.GetState().VisualStart, "anchor")
			assert.Equal(t, tt.cursor, cursorPos(e), "cursor")
		})
	}
}

// TestOperatorAfterQuoteObjectFromSelection: d, y, c, U and . act on the
// selection a quote object made from a selection over more than one
// character, as in Vim 9.2.
func TestOperatorAfterQuoteObjectFromSelection(t *testing.T) {
	tests := []struct {
		content, keys, text string
		cursor              Position
		clipboard           string // checked when not empty (y)
	}{
		{`a "zz" b "yy" c`, `3lvi"i"d`, `a  b "yy" c`, Position{0, 2}, ""},
		{`a "zz" b "yy" c`, `3lvi"i"y`, `a "zz" b "yy" c`, Position{0, 2}, `"zz"`},
		{`a "zz" b "yy" c`, `3lvi"i"cX<Esc>`, `a X b "yy" c`, Position{0, 2}, ""},
		{`a "zz" b "yy" c`, `3lvi"i"U`, `a "ZZ" b "yy" c`, Position{0, 2}, ""},
		{`a "zz" b "yy" c`, `vlli"d`, `"yy" c`, Position{0, 0}, ""},
		{`a "zz" b "yy" c`, `3lvhi"y`, `a "zz" b "yy" c`, Position{0, 2}, `"z`},
		{`a "zz" b "yy" c`, `10lvhi"d`, `a "y" c`, Position{0, 3}, ""},
		{`a "zz" b "yy" c`, `10lvli"y`, `a "zz" b "yy" c`, Position{0, 9}, `"yy"`},
		{`a "zz" b "yy" c`, `3lvi"i"U7l.`, `a "ZZ" b "YY" c`, Position{0, 9}, ""},
		{`a "zz" b "yy" c`, `vlli"dw.`, `"`, Position{0, 0}, ""},
		{`a "zz" b "yy" c`, `3lvi"i"cX<Esc>6l.`, `a X b "yX`, Position{0, 8}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, NormalMode, e.GetMode().Name())
			assert.Equal(t, tt.cursor, cursorPos(e))
			if tt.clipboard != "" {
				assert.Equal(t, tt.clipboard, cb.content)
			}
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
