package core

import (
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
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'i', '"')
		assert.True(t, ok)
		assert.Equal(t, 5, s)
		assert.Equal(t, 9, e)
	})

	t.Run("cursor inside double quotes – around", func(t *testing.T) {
		b := buf(`say "hello" world`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'a', '"')
		assert.True(t, ok)
		assert.Equal(t, 4, s)
		assert.Equal(t, 10, e)
	})

	t.Run("cursor on opening quote – inner", func(t *testing.T) {
		b := buf(`"hello"`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"')
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 5, e)
	})

	t.Run("cursor on closing quote – around", func(t *testing.T) {
		b := buf(`"hello"`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 6}, 'a', '"')
		assert.True(t, ok)
		assert.Equal(t, 0, s)
		assert.Equal(t, 6, e)
	})

	t.Run("cursor before first quote pair – uses that pair", func(t *testing.T) {
		b := buf(`x "foo" y`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"')
		assert.True(t, ok)
		assert.Equal(t, 3, s)
		assert.Equal(t, 5, e)
	})

	t.Run("empty quotes – inner returns not found", func(t *testing.T) {
		b := buf(`""`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 0}, 'i', '"')
		assert.False(t, ok)
	})

	t.Run("empty quotes – around returns the two quote chars", func(t *testing.T) {
		b := buf(`""`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 0}, 'a', '"')
		assert.True(t, ok)
		assert.Equal(t, 0, s)
		assert.Equal(t, 1, e)
	})

	t.Run("no quotes – not found", func(t *testing.T) {
		b := buf(`hello`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 2}, 'i', '"')
		assert.False(t, ok)
	})

	t.Run("single quote variant", func(t *testing.T) {
		// No apostrophes: two single-quote chars form one clean pair.
		b := buf(`hello 'world' end`)
		s, e, ok := quoteTextObjectRange(b, Position{0, 8}, 'i', '\'')
		assert.True(t, ok)
		assert.Equal(t, 7, s)
		assert.Equal(t, 11, e)
	})

	t.Run("backtick variant", func(t *testing.T) {
		b := buf("`cmd`")
		s, e, ok := quoteTextObjectRange(b, Position{0, 2}, 'i', '`')
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 3, e)
	})

	t.Run("cursor past all pairs – not found", func(t *testing.T) {
		b := buf(`"a" x`)
		_, _, ok := quoteTextObjectRange(b, Position{0, 4}, 'i', '"')
		assert.False(t, ok)
	})
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
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 3}, 'i')
		assert.True(t, ok)
		assert.Equal(t, 1, s)
		assert.Equal(t, 5, e)
	})

	t.Run("picks innermost when nested quotes on same line", func(t *testing.T) {
		// outer: " at 0 and 12; inner: ' at 5 and 9
		b := buf(`"foo 'bar' baz"`)
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 7}, 'i')
		assert.True(t, ok)
		// should pick the ' pair (span 4) rather than the " pair (span 14)
		assert.Equal(t, 6, s)
		assert.Equal(t, 8, e)
	})

	t.Run("no quotes – not found", func(t *testing.T) {
		b := buf(`hello`)
		_, _, ok := anyQuoteTextObjectRange(b, Position{0, 2}, 'i')
		assert.False(t, ok)
	})

	t.Run("around modifier includes quote chars", func(t *testing.T) {
		b := buf(`'hi'`)
		s, e, ok := anyQuoteTextObjectRange(b, Position{0, 1}, 'a')
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
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'i', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 5}, e)
	})

	t.Run("cursor inside parens – around", func(t *testing.T) {
		b := buf("(hello)")
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'a', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 6}, e)
	})

	t.Run("cursor on open paren – inner", func(t *testing.T) {
		b := buf("(foo)")
		s, e, ok := bracketTextObjectRange(b, Position{0, 0}, 'i', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("cursor on close paren – around", func(t *testing.T) {
		b := buf("(foo)")
		// cursor at col 4 (the ')'); findBracketBounds scans backward from col 3 to find '('
		s, e, ok := bracketTextObjectRange(b, Position{0, 4}, 'a', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 4}, e)
	})

	t.Run("nested parens – inner selects innermost from inside", func(t *testing.T) {
		b := buf("(a(b)c)")
		// cursor on 'b' at col 3
		s, e, ok := bracketTextObjectRange(b, Position{0, 3}, 'i', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 3}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("nested parens – outer selected when cursor outside inner", func(t *testing.T) {
		b := buf("(a(b)c)")
		// cursor on 'a' at col 1
		s, e, ok := bracketTextObjectRange(b, Position{0, 1}, 'i', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 5}, e)
	})

	t.Run("empty parens – inner not found", func(t *testing.T) {
		b := buf("()")
		_, _, ok := bracketTextObjectRange(b, Position{0, 0}, 'i', '(', ')')
		assert.False(t, ok)
	})

	t.Run("empty parens – around returns the two chars", func(t *testing.T) {
		b := buf("()")
		s, e, ok := bracketTextObjectRange(b, Position{0, 0}, 'a', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{0, 1}, e)
	})

	t.Run("square brackets", func(t *testing.T) {
		b := buf("[1, 2, 3]")
		s, e, ok := bracketTextObjectRange(b, Position{0, 4}, 'i', '[', ']')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 7}, e)
	})

	t.Run("curly braces", func(t *testing.T) {
		b := buf("{key: val}")
		s, e, ok := bracketTextObjectRange(b, Position{0, 5}, 'i', '{', '}')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 8}, e)
	})

	t.Run("angle brackets", func(t *testing.T) {
		b := buf("<div>")
		s, e, ok := bracketTextObjectRange(b, Position{0, 2}, 'i', '<', '>')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 1}, s)
		assert.Equal(t, Position{0, 3}, e)
	})

	t.Run("multi-line – inner", func(t *testing.T) {
		b := buf("(", "  foo", "  bar", ")")
		// cursor on "foo" line
		s, e, ok := bracketTextObjectRange(b, Position{1, 2}, 'i', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{1, 0}, s) // first char of line after '('
		assert.Equal(t, Position{2, 4}, e) // last char of line before ')'
	})

	t.Run("multi-line – around", func(t *testing.T) {
		b := buf("(", "  foo", ")")
		s, e, ok := bracketTextObjectRange(b, Position{1, 2}, 'a', '(', ')')
		assert.True(t, ok)
		assert.Equal(t, Position{0, 0}, s)
		assert.Equal(t, Position{2, 0}, e)
	})

	t.Run("no enclosing bracket – not found", func(t *testing.T) {
		b := buf("hello")
		_, _, ok := bracketTextObjectRange(b, Position{0, 2}, 'i', '(', ')')
		assert.False(t, ok)
	})
}
