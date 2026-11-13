package core

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultSearch executes a forward, non-wrapping search with no case options.
func defaultSearch(e Editor, pattern string) {
	e.ExecuteSearch(pattern, SearchOptions{})
}

// wrapSearch executes a forward, wrapping search.
func wrapSearch(e Editor, pattern string) {
	e.ExecuteSearch(pattern, SearchOptions{Wrap: true})
}

// backwardSearch executes a backward, wrapping search.
func backwardSearch(e Editor, pattern string) {
	e.ExecuteSearch(pattern, SearchOptions{Backwards: true, Wrap: true})
}

// TestExecuteSearch tests basic forward pattern search behaviour.
func TestExecuteSearch(t *testing.T) {
	t.Run("finds first match after cursor on same line", func(t *testing.T) {
		e := newTestEditor("hello world hello")
		defaultSearch(e, "world")
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("finds match on a later line", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		defaultSearch(e, "bar")
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("pattern not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "xyz")
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.Empty(t, e.SearchResults())
	})

	t.Run("search results contain the matched position", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "world")
		assert.Equal(t, []Position{{0, 6}}, e.SearchResults())
	})

	t.Run("empty pattern produces no results", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "")
		assert.Empty(t, e.SearchResults())
	})

	t.Run("search starts after current cursor position", func(t *testing.T) {
		// "hello world hello": two occurrences of "hello" at col 0 and col 12.
		// Cursor starts at col 0, so the first match after the cursor is col 12.
		e := newTestEditor("hello world hello")
		defaultSearch(e, "hello")
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})

	t.Run("multiline: finds match on first line after wrapping line boundary", func(t *testing.T) {
		e := newTestEditor("foo bar\nbaz qux")
		defaultSearch(e, "baz")
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestSearchWrap tests wrap-around search behaviour.
func TestSearchWrap(t *testing.T) {
	t.Run("wraps to col 0 of line 0 when no match found after cursor", func(t *testing.T) {
		// "hello" only at col 0. Cursor at col 0; no match after it; wraps and finds col 0.
		e := newTestEditor("hello world")
		wrapSearch(e, "hello")
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.NotEmpty(t, e.SearchResults())
	})

	t.Run("without wrap, no match past cursor returns empty results", func(t *testing.T) {
		e := newTestEditor("hello world")
		// Cursor at col 0; "hello" is at col 0; no match after it; no wrap.
		defaultSearch(e, "hello")
		assert.Empty(t, e.SearchResults())
	})

	t.Run("failed ?pattern at the top scans the buffer once", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		finds := countFinds(e)
		backwardSearch(e, "xyz")
		assert.Empty(t, e.SearchResults())
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.Equal(t, 1, *finds)
	})

	t.Run("failed /pattern scans the buffer once", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		finds := countFinds(e)
		wrapSearch(e, "xyz")
		assert.Empty(t, e.SearchResults())
		assert.Equal(t, 1, *finds)
	})
}

// findCounter counts calls to Find on the wrapped buffer.
type findCounter struct {
	Buffer
	calls int
}

func (f *findCounter) Find(pattern string, start Position, options SearchOptions) (Position, bool) {
	f.calls++
	return f.Buffer.Find(pattern, start, options)
}

// countFinds makes e's buffer count Find calls and returns the counter.
func countFinds(e Editor) *int {
	rb := e.(*editor).buffer
	fc := &findCounter{Buffer: rb.Buffer}
	rb.Buffer = fc
	return &fc.calls
}

// TestFindWrap tests wrapping in Buffer.Find itself, which hosts can call directly.
func TestFindWrap(t *testing.T) {
	find := func(content, pattern string, start Position, opts SearchOptions) (Position, bool) {
		b := NewBuffer()
		b.SetContent([]byte(content))
		return b.Find(pattern, start, opts)
	}

	t.Run("forward wraps to the top", func(t *testing.T) {
		pos, found := find("foo\nbar\nbaz", "foo", Position{1, 1}, SearchOptions{Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{0, 0}, pos)
	})

	t.Run("forward wraps to an earlier column on the start line", func(t *testing.T) {
		pos, found := find("foo bar", "foo", Position{0, 4}, SearchOptions{Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{0, 0}, pos)
	})

	t.Run("forward without wrap stops at the bottom", func(t *testing.T) {
		_, found := find("foo\nbar\nbaz", "foo", Position{1, 1}, SearchOptions{})
		assert.False(t, found)
	})

	t.Run("forward wrap finds the match at the start position", func(t *testing.T) {
		pos, found := find("x foo y", "foo", Position{0, 2}, SearchOptions{Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{0, 2}, pos)
	})

	t.Run("backward wraps to the bottom", func(t *testing.T) {
		pos, found := find("foo\nbar\nbaz", "baz", Position{1, 1}, SearchOptions{Backwards: true, Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{2, 0}, pos)
	})

	t.Run("backward wraps to a later column on the start line", func(t *testing.T) {
		pos, found := find("foo bar", "bar", Position{0, 2}, SearchOptions{Backwards: true, Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{0, 4}, pos)
	})

	t.Run("backward without wrap stops at the top", func(t *testing.T) {
		_, found := find("foo\nbar\nbaz", "baz", Position{0, 0}, SearchOptions{Backwards: true})
		assert.False(t, found)
	})

	t.Run("backward wrap finds the match at the start position", func(t *testing.T) {
		pos, found := find("x foo y", "foo", Position{0, 2}, SearchOptions{Backwards: true, Wrap: true})
		assert.True(t, found)
		assert.Equal(t, Position{0, 2}, pos)
	})

	t.Run("start outside the buffer doesn't panic", func(t *testing.T) {
		for _, start := range []Position{{-5, 0}, {100, 0}, {0, -3}, {0, 1000}, {100, 100}} {
			for _, backwards := range []bool{false, true} {
				pos, found := find("foo\nbar", "bar", start, SearchOptions{Backwards: backwards, Wrap: true})
				assert.True(t, found, "start %v backwards %v", start, backwards)
				assert.Equal(t, Position{1, 0}, pos, "start %v backwards %v", start, backwards)
			}
		}
	})
}

// TestSearchCaseInsensitive tests IgnoreCase search option.
func TestSearchCaseInsensitive(t *testing.T) {
	t.Run("matches uppercase pattern with lowercase text", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.ExecuteSearch("HELLO", SearchOptions{IgnoreCase: true})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("matches mixed-case pattern", func(t *testing.T) {
		e := newTestEditor("Hello World")
		e.ExecuteSearch("hello", SearchOptions{IgnoreCase: true})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("case-sensitive search misses differently-cased match", func(t *testing.T) {
		e := newTestEditor("Hello World")
		defaultSearch(e, "hello")
		assert.Empty(t, e.SearchResults())
	})
}

// TestSearchSmartCase tests that SmartCase makes a pattern with an uppercase letter case-sensitive.
func TestSearchSmartCase(t *testing.T) {
	smartCase := SearchOptions{IgnoreCase: true, SmartCase: true}

	t.Run("uppercase pattern is case-sensitive", func(t *testing.T) {
		e := newTestEditor("x foo Foo")
		e.ExecuteSearch("Foo", smartCase)
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("uppercase pattern misses lowercase-only text", func(t *testing.T) {
		e := newTestEditor("x foo foo")
		e.ExecuteSearch("Foo", smartCase)
		assert.Empty(t, e.SearchResults())
	})

	t.Run("lowercase pattern stays case-insensitive", func(t *testing.T) {
		e := newTestEditor("x FOO foo")
		e.ExecuteSearch("foo", smartCase)
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("non-ASCII uppercase counts", func(t *testing.T) {
		e := newTestEditor("x élan Élan")
		e.ExecuteSearch("Élan", smartCase)
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("titlecase counts as uppercase", func(t *testing.T) {
		// ǅ (U+01C5) is titlecase, not IsUpper; Vim still treats it as uppercase for smartcase.
		e := newTestEditor("xǄ ǆ ǅ")
		e.ExecuteSearch("ǅ", smartCase)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("Other_Uppercase counts as uppercase", func(t *testing.T) {
		// Ⓐ (U+24B6) isn't IsUpper or IsTitle, but it has a lowercase form.
		e := newTestEditor("xⓐ Ⓐ")
		e.ExecuteSearch("Ⓐ", smartCase)
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("uppercase without a lowercase form doesn't count", func(t *testing.T) {
		// ϒ (U+03D2) is IsUpper but has no lowercase form, so Vim keeps ϒa case-insensitive.
		e := newTestEditor("xϒA ϒa")
		e.ExecuteSearch("ϒa", smartCase)
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("n and N stay case-sensitive", func(t *testing.T) {
		e := newTestEditor("Foo foo Foo foo")
		e.ExecuteSearch("Foo", SearchOptions{IgnoreCase: true, SmartCase: true, Wrap: true})
		assert.Equal(t, Position{0, 8}, cursorPos(e))
		e.NextSearchResult()
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		e.PreviousSearchResult()
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("backward search is case-sensitive", func(t *testing.T) {
		e := newTestEditor("Foo foo")
		e.GetBuffer().SetCursor(Cursor{Position: Position{Row: 0, Col: 6}})
		e.ExecuteSearch("Foo", SearchOptions{IgnoreCase: true, SmartCase: true, Backwards: true})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("without IgnoreCase, SmartCase has no effect", func(t *testing.T) {
		e := newTestEditor("x FOO foo")
		e.ExecuteSearch("foo", SearchOptions{SmartCase: true})
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run(`\c overrides smartcase`, func(t *testing.T) {
		e := newTestEditor("x foo Foo")
		e.ExecuteSearch(`Foo\c`, smartCase)
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

// TestSearchMultibyte tests that match columns are rune indices when the line contains multibyte text.
func TestSearchMultibyte(t *testing.T) {
	t.Run("forward match after multibyte text", func(t *testing.T) {
		e := newTestEditor("é foo")
		defaultSearch(e, "foo")
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("forward search starts after the cursor rune", func(t *testing.T) {
		// "ééxx": cursor on the first x (col 2) must find the second x (col 3).
		e := newTestEditor("ééxx")
		e.GetBuffer().SetCursor(Cursor{Position: Position{Row: 0, Col: 2}})
		defaultSearch(e, "x")
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("n and N after multibyte text", func(t *testing.T) {
		e := newTestEditor("é x é x")
		wrapSearch(e, "x")
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		e.NextSearchResult()
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		e.NextSearchResult()
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		e.PreviousSearchResult()
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("forward ignore-case match after multibyte text", func(t *testing.T) {
		e := newTestEditor("É FOO")
		e.ExecuteSearch("foo", SearchOptions{IgnoreCase: true})
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("backward match after multibyte text", func(t *testing.T) {
		e := newTestEditor("é foo x")
		e.GetBuffer().SetCursor(Cursor{Position: Position{Row: 0, Col: 6}})
		e.ExecuteSearch("foo", SearchOptions{Backwards: true})
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run(`\<x after word-char multibyte text does not panic`, func(t *testing.T) {
		// é is a word character, so x does not start a word.
		e := newTestEditor("éééx")
		assert.NotPanics(t, func() { defaultSearch(e, `\<x`) })
		assert.Empty(t, e.SearchResults())
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run(`\<x after multibyte text`, func(t *testing.T) {
		e := newTestEditor("ééé x")
		defaultSearch(e, `\<x`)
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("* after multibyte text", func(t *testing.T) {
		e := newTestEditor("éé foo foobar foo")
		e.GetBuffer().SetCursor(Cursor{Position: Position{Row: 0, Col: 3}})
		assert.NotPanics(t, func() { e.SearchWordUnderCursor(false) })
		assert.Equal(t, Position{0, 14}, cursorPos(e))
	})

	t.Run("ignore-case match after text whose lowercase is longer in bytes", func(t *testing.T) {
		// Ⱥ (2 bytes) lowercases to ⱥ (3 bytes); the column must not depend on either.
		e := newTestEditor("Ⱥ FOO")
		e.ExecuteSearch("foo", SearchOptions{IgnoreCase: true})
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("ignore-case pattern with multibyte uppercase", func(t *testing.T) {
		e := newTestEditor("x ÉLAN élan")
		e.ExecuteSearch("élan", SearchOptions{IgnoreCase: true})
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

// TestSearchCaseFolding checks that ignore case matches a rune's whole simple-fold
// orbit, as Vim does, rather than comparing lowercased runes.
func TestSearchCaseFolding(t *testing.T) {
	ignoreCase := SearchOptions{IgnoreCase: true}

	tests := []struct {
		name    string
		content string
		pattern string
		opts    SearchOptions
		want    Position
	}{
		// ToLower(ς) is ς, so lowercasing never matched it to σ or Σ.
		{"final sigma matches capital sigma", "xΣ", "ς", ignoreCase, Position{0, 1}},
		{"sigma matches final sigma", "xς", "σ", ignoreCase, Position{0, 1}},
		{"capital sigma matches final sigma", "xς", "Σ", ignoreCase, Position{0, 1}},
		// ToLower(ſ) is ſ.
		{"s matches long s", "xſ", "s", ignoreCase, Position{0, 1}},
		{"long s matches S", "xS", "ſ", ignoreCase, Position{0, 1}},
		// Go's ToLower(İ) is i, but İ is alone in its fold orbit, so Vim matches neither way.
		{"i does not match dotted capital I", "xİ i", "i", ignoreCase, Position{0, 3}},
		{`İ\c does not match i or I`, "xi I İ", `İ\c`, SearchOptions{}, Position{0, 5}},
		{"dotless i matches only itself", "xi I ı", "ı", ignoreCase, Position{0, 5}},
		// These already matched by lowercasing; folding must keep them.
		{"k matches Kelvin sign", "x\u212a", "k", ignoreCase, Position{0, 1}},
		{"Kelvin sign matches K", "xK", "\u212a", ignoreCase, Position{0, 1}},
		{"omega matches Ohm sign", "x\u2126", "ω", ignoreCase, Position{0, 1}},
		{"sharp s matches capital sharp s", "xẞ", "ß", ignoreCase, Position{0, 1}},
		{"titlecase dz matches uppercase", "xǄ", "ǅ", ignoreCase, Position{0, 1}},
		// Lowercase-only sigmas keep smartcase case-insensitive.
		{"final sigma with smartcase", "xΣ", "ς", SearchOptions{IgnoreCase: true, SmartCase: true}, Position{0, 1}},
		{"sigma with smartcase", "xς", "σ", SearchOptions{IgnoreCase: true, SmartCase: true}, Position{0, 1}},
		{`\c without ignore case folds too`, "xΣ", `ς\c`, SearchOptions{}, Position{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			e.ExecuteSearch(tt.pattern, tt.opts)
			assert.Equal(t, tt.want, cursorPos(e))
		})
	}

	t.Run("sharp s does not match ss", func(t *testing.T) {
		e := newTestEditor("x ss SS")
		e.ExecuteSearch("ß", ignoreCase)
		assert.Empty(t, e.SearchResults())
	})

	t.Run("backward search folds", func(t *testing.T) {
		e := newTestEditor("Σ x")
		e.GetBuffer().SetCursor(Cursor{Position: Position{Row: 0, Col: 2}})
		e.ExecuteSearch("ς", SearchOptions{IgnoreCase: true, Backwards: true})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestFoldRune checks foldRune against unicode.SimpleFold for every rune: it must
// return the smallest rune in the orbit (the ASCII fast path included), and no orbit
// may be larger than Find's firstRunes array, which would drop matches.
func TestFoldRune(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		want, size := r, 1
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			want = min(want, f)
			size++
		}
		if got := foldRune(r); got != want {
			t.Fatalf("foldRune(%U) = %U, want %U", r, got, want)
		}
		if size > 4 {
			t.Fatalf("fold orbit of %U has %d runes; grow firstRunes in Find", r, size)
		}
	}
}

// TestSearchInlineModifiers tests the \c, \C, \<, and \> inline modifiers.
func TestSearchInlineModifiers(t *testing.T) {
	t.Run(`\c forces case-insensitive search`, func(t *testing.T) {
		e := newTestEditor("Hello World")
		e.ExecuteSearch(`hello\c`, SearchOptions{})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run(`\c keeps a trailing c in the pattern`, func(t *testing.T) {
		// "music\c" must search for "music", not "musi" (col 4 is "MUSI" with no C).
		e := newTestEditor("abc MUSI MUSIC")
		e.ExecuteSearch(`music\c`, SearchOptions{})
		assert.Equal(t, "music", e.GetState().SearchQuery.Term)
		assert.Equal(t, Position{0, 9}, cursorPos(e))
	})

	t.Run(`\C keeps a trailing C in the pattern`, func(t *testing.T) {
		// "ABC\C" must search for "ABC", not "AB" (col 2 is "AB" with no C).
		e := newTestEditor("x AB ABC")
		e.ExecuteSearch(`ABC\C`, SearchOptions{})
		assert.Equal(t, "ABC", e.GetState().SearchQuery.Term)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run(`\C forces case-sensitive search and disables smartcase`, func(t *testing.T) {
		e := newTestEditor("x hello Hello")
		e.ExecuteSearch(`Hello\C`, SearchOptions{IgnoreCase: true, SmartCase: true})
		assert.Equal(t, Position{0, 8}, cursorPos(e))
		assert.False(t, e.GetState().SearchOptions.IgnoreCase)
		assert.False(t, e.GetState().SearchOptions.SmartCase)
	})

	t.Run(`\c forces case-insensitive search and disables smartcase`, func(t *testing.T) {
		e := newTestEditor("x hello Hello")
		e.ExecuteSearch(`Hello\c`, SearchOptions{SmartCase: true})
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assert.True(t, e.GetState().SearchOptions.IgnoreCase)
		assert.False(t, e.GetState().SearchOptions.SmartCase)
	})

	t.Run(`\<word\> matches whole word, skips partial`, func(t *testing.T) {
		// "foo foobar foo": whole-word "foo" at col 0 and col 11; "foobar" at col 4 is not a match.
		e := newTestEditor("foo foobar foo")
		defaultSearch(e, `\<foo\>`)
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})

	t.Run(`\< alone matches word-start boundary only`, func(t *testing.T) {
		// "foo foobar": \<foo matches "foo" at col 0 and "foobar" at col 4 (both start at word boundary).
		// Cursor at col 0, search after it → finds col 4 ("foobar" starts a word).
		e := newTestEditor("foo foobar")
		defaultSearch(e, `\<foo`)
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run(`\> alone matches word-end boundary only`, func(t *testing.T) {
		// "foobar foo": foo\> matches "foo" ending at a word boundary.
		// "foobar" contains "foo" but is followed by 'b' (word char) → not a match.
		// "foo" at col 7 is followed by end-of-line → matches.
		e := newTestEditor("foobar foo")
		defaultSearch(e, `foo\>`)
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run(`\<word\> combined with \c is case-insensitive whole-word`, func(t *testing.T) {
		e := newTestEditor("foo Foobar FOO")
		e.ExecuteSearch(`\<foo\>\c`, SearchOptions{Wrap: true})
		// "foo" at col 0 → search after col 0 → wraps → "FOO" at col 11
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})
}

// TestNextSearchResult tests the 'n' command — advance to the next match.
func TestNextSearchResult(t *testing.T) {
	t.Run("advances to the next occurrence", func(t *testing.T) {
		// "foo bar foo baz foo": occurrences of "foo" at col 0, 8, 16.
		e := newTestEditor("foo bar foo baz foo")
		wrapSearch(e, "foo") // lands on col 8 (first match after cursor at col 0)
		assert.Equal(t, Position{0, 8}, cursorPos(e))
		e.NextSearchResult()
		assert.Equal(t, Position{0, 16}, cursorPos(e))
	})

	t.Run("wraps around to col 0 of line 0", func(t *testing.T) {
		// "foo bar foo": "foo" at col 0 and col 8.
		// wrapSearch from col 0 finds col 8 (next after cursor).
		// NextSearchResult from col 8: not found past col 8, wraps to col 0.
		e := newTestEditor("foo bar foo")
		wrapSearch(e, "foo")
		assert.Equal(t, Position{0, 8}, cursorPos(e))
		e.NextSearchResult() // wraps to col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("no-op when there are no search results", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "xyz") // no match
		cursor := e.NextSearchResult()
		assert.Equal(t, Position{0, 0}, cursor.Position)
	})

	t.Run("advances across lines and wraps to line 0", func(t *testing.T) {
		// "foo\nbar\nfoo": "foo" at {0,0} and {2,0}.
		// wrapSearch from {0,0}: finds {2,0} (next after cursor).
		// NextSearchResult from {2,0}: not found past end, wraps to {0,0}.
		e := newTestEditor("foo\nbar\nfoo")
		wrapSearch(e, "foo")
		assert.Equal(t, Position{2, 0}, cursorPos(e))
		e.NextSearchResult()
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestPreviousSearchResult tests the 'N' command — go to the previous match.
func TestPreviousSearchResult(t *testing.T) {
	t.Run("moves to the previous occurrence", func(t *testing.T) {
		// "foo bar foo baz foo": col 0, 8, 16.
		e := newTestEditor("foo bar foo baz foo")
		wrapSearch(e, "foo") // → col 8
		e.NextSearchResult() // → col 16
		e.PreviousSearchResult()
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("wraps to the last match when at first match", func(t *testing.T) {
		e := newTestEditor("foo bar foo")
		wrapSearch(e, "foo") // → col 8 (first after cursor)
		e.PreviousSearchResult()
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("no-op when there are no search results", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "xyz")
		cursor := e.PreviousSearchResult()
		assert.Equal(t, Position{0, 0}, cursor.Position)
	})

	t.Run("moves backward across lines to line 0 col 0", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nfoo")
		wrapSearch(e, "foo")     // → {2,0}
		e.PreviousSearchResult() // backward from {2,0} → {0,0}
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestBackwardSearch tests '?pattern' — search backward through the buffer.
func TestBackwardSearch(t *testing.T) {
	t.Run("finds the match before the cursor", func(t *testing.T) {
		// "foo bar foo": cursor starts at col 0; position at end first, then search back.
		// Place cursor at col 8 by doing a forward search, then do a backward search.
		e := newTestEditor("foo bar foo")
		wrapSearch(e, "foo")     // → col 8
		backwardSearch(e, "foo") // backward from col 8 → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("finds the match on an earlier line", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nbaz")
		// Move cursor to last line then search backward.
		keys(e, 'G') // go to last line
		backwardSearch(e, "foo")
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("wraps to the end when no match before cursor", func(t *testing.T) {
		// "foo bar foo": backward search from col 0 finds nothing before it,
		// wraps to the end and finds col 8.
		e := newTestEditor("foo bar foo")
		backwardSearch(e, "foo") // cursor at col 0; nothing before it; wraps to col 8
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("pattern not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.ExecuteSearch("xyz", SearchOptions{Backwards: true})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.Empty(t, e.SearchResults())
	})
}

// TestNAfterBackwardSearch tests that 'n' repeats in the same direction as '?'.
func TestNAfterBackwardSearch(t *testing.T) {
	t.Run("n repeats backward after ?", func(t *testing.T) {
		// "foo bar foo baz foo": "foo" at col 0, 8, 16.
		// Start at end, search backward to col 8, then n continues backward to col 0.
		e := newTestEditor("foo bar foo baz foo")
		keys(e, '$')             // move to end of line (col 18)
		backwardSearch(e, "foo") // backward from col 18 → col 16
		assert.Equal(t, Position{0, 16}, cursorPos(e))
		e.NextSearchResult() // n: same direction (backward) → col 8
		assert.Equal(t, Position{0, 8}, cursorPos(e))
		e.NextSearchResult() // n: backward → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("n wraps to end when at the first match", func(t *testing.T) {
		// "foo bar foo": cursor at col 0 after wrapping backward search.
		// Another n wraps backward to the last "foo" at col 8.
		e := newTestEditor("foo bar foo")
		keys(e, '$')
		backwardSearch(e, "foo") // → col 8
		e.NextSearchResult()     // backward: not found before col 8, wraps to col 0
		// Wait - "foo" IS at col 0, and backward from col 8 should find it:
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		e.NextSearchResult() // backward from col 0: not found, wraps to col 8
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("N goes forward after ?", func(t *testing.T) {
		// After a backward search, N (opposite direction) should go forward.
		e := newTestEditor("foo bar foo")
		keys(e, '$')
		backwardSearch(e, "foo") // → col 8
		e.NextSearchResult()     // backward → col 0
		e.PreviousSearchResult() // N: opposite = forward → col 8
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})
}

// TestCancelSearch tests that cancelling clears all search state.
func TestCancelSearch(t *testing.T) {
	t.Run("clears results after cancel", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "world") // "world" at col 6, found after cursor at col 0
		assert.NotEmpty(t, e.SearchResults())
		e.CancelSearch()
		assert.Empty(t, e.SearchResults())
	})

	t.Run("resets index and options after cancel", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.ExecuteSearch("world", SearchOptions{IgnoreCase: true, Backwards: true, Wrap: true})
		e.CancelSearch()
		state := e.GetState()
		assert.Equal(t, SearchQuery{}, state.SearchQuery)
		assert.Equal(t, -1, state.SearchResultIndex)
		assert.Equal(t, SearchOptions{}, state.SearchOptions)
	})

	t.Run("next result is no-op after cancel", func(t *testing.T) {
		e := newTestEditor("hello world")
		defaultSearch(e, "world") // moves cursor to col 6
		e.CancelSearch()
		initialPos := cursorPos(e)
		e.NextSearchResult()
		assert.Equal(t, initialPos, cursorPos(e))
	})
}

// TestSearchWordUnderCursor tests '*' (forward) and '#' (backward) — whole-word search.
func TestSearchWordUnderCursor(t *testing.T) {
	t.Run("* finds next whole-word match forward", func(t *testing.T) {
		// "foo foobar foo": whole-word "foo" at col 0 and col 11; "foobar" at col 4 is NOT a match.
		e := newTestEditor("foo foobar foo")
		e.SearchWordUnderCursor(false) // cursor on "foo" at col 0 → next whole-word match → col 11
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})

	t.Run("* on 'foobar' wraps back to itself (only one occurrence)", func(t *testing.T) {
		e := newTestEditor("foobar foo")
		// cursor on "foobar" at col 0 — * searches for "foobar" (the word under cursor), not "foo"
		e.SearchWordUnderCursor(false)
		// "foobar" only appears once; wraps and stays at col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("# finds previous whole-word match backward", func(t *testing.T) {
		e := newTestEditor("foo foobar foo")
		// move cursor to last "foo" at col 11
		keys(e, '$')
		keys(e, 'b')                  // move back to start of last "foo"
		e.SearchWordUnderCursor(true) // backward from col 11 → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("* wraps around when no match after cursor", func(t *testing.T) {
		e := newTestEditor("foo bar foo")
		wrapSearch(e, "foo")           // move to col 8
		e.SearchWordUnderCursor(false) // * on "foo" at col 8 → wraps → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("* works across lines", func(t *testing.T) {
		e := newTestEditor("foo\nbar\nfoo")
		e.SearchWordUnderCursor(false) // * on "foo" at {0,0} → {2,0}
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("n after * continues as whole-word search", func(t *testing.T) {
		// "foo foobar foo bar foo": whole-word "foo" at col 0, 11, 19
		e := newTestEditor("foo foobar foo bar foo")
		e.SearchWordUnderCursor(false) // * from col 0 → col 11 (skipping "foobar" at col 4? no)
		// Wait: "foo foobar foo bar foo"
		//        0123456789012345678901
		// "foo" at col 0, col 11, col 19. "foobar" at col 4.
		// * from col 0 → next whole-word "foo" after col 0 → col 11
		assert.Equal(t, Position{0, 11}, cursorPos(e))
		e.NextSearchResult() // n → col 19
		assert.Equal(t, Position{0, 19}, cursorPos(e))
	})
}

// TestStarTarget checks what * and # search for when the cursor isn't on a
// keyword, as Vim 9.2's find_ident_under_cursor: the first keyword after the
// cursor on the line, else the non-blank text under or after it, searched
// from its start. The pattern is the one Vim keeps in @/.
func TestStarTarget(t *testing.T) {
	tests := []struct {
		content string
		keys    string
		cursor  Position
		pattern string
	}{
		{"foo 1 x 1", "3l*", Position{0, 8}, `\<1\>`},
		{"foo 1 x 1", "3l#", Position{0, 8}, `\<1\>`},
		{"a+b a b", "l*", Position{0, 6}, `\<b\>`},
		{"a+b a b", "l#", Position{0, 6}, `\<b\>`},
		{"(foo) foo", "*", Position{0, 6}, `\<foo\>`},
		{"  foo x foo", "*", Position{0, 8}, `\<foo\>`},
		{"ab ++ c++", "3l*", Position{0, 6}, `\<c\>`},
		{"ab .x.\n.x.", "3l*", Position{1, 1}, `\<x\>`},
		{"x.y x.y", "l*", Position{0, 6}, `\<y\>`},
		{"ab a.b a.b", "3l*", Position{0, 7}, `\<a\>`},
		{"foo bar", "3l*", Position{0, 4}, `\<bar\>`},
		// No keyword from the cursor on: the non-blank text, as it is.
		{"ab ++ ++", "3l*", Position{0, 6}, "++"},
		{"ab ++ ++", "3l#", Position{0, 6}, "++"},
		{"ab ++ ++", "3l*n", Position{0, 3}, "++"},
		{"ab ++\n++", "3l*", Position{1, 0}, "++"},
		{"ab ++\n++", "2l*", Position{1, 0}, "++"},
		{"ab +-", "4l*", Position{0, 3}, "+-"},
		{"ab (", "2l*", Position{0, 3}, "("},
		// A mid-word # starts from the word's start, so it skips the word.
		{"foo foo", "5l#", Position{0, 0}, `\<foo\>`},
	}
	for _, tt := range tests {
		t.Run(tt.content+" "+tt.keys, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, tt.pattern, e.GetState().SearchQuery.Pattern)
		})
	}
}

// TestStarNoString checks Vim's E348: * and # with only blanks from the
// cursor to the end of the line move nothing, keep the last search and
// report ErrNoStringUnderCursor.
func TestStarNoString(t *testing.T) {
	for _, tt := range []struct{ content, keys string }{
		{"foo  \nfoo", "4l*"},
		{"foo  \nfoo", "4l#"},
		{"\nfoo", "*"},
	} {
		t.Run(tt.content+" "+tt.keys, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys[:len(tt.keys)-1])
			start := cursorPos(e)
			typeKeys(t, e, tt.keys[len(tt.keys)-1:])
			assert.Equal(t, start, cursorPos(e))
			assert.Equal(t, "", e.GetState().SearchQuery.Pattern)
			assert.Equal(t, ErrNoStringUnderCursorId, nextErrorID(t, e))
		})
	}
}

// TestStarTargetPastLineEnd checks that a column at or past the end of the
// line finds nothing instead of indexing past it. The editor clamps its cursor
// to the line length, so only a direct caller can pass a larger column.
func TestStarTargetPastLineEnd(t *testing.T) {
	for _, col := range []int{2, 3, 9} {
		_, _, _, ok := starTarget([]rune("ab"), col, utfClass)
		assert.False(t, ok, "col %d", col)
	}
}

// nextErrorID returns the ID of the first ErrorSignal waiting on e's signal
// channel, failing the test when there is none.
func nextErrorID(t *testing.T, e Editor) ErrorId {
	t.Helper()
	for {
		select {
		case s := <-e.GetUpdateSignalChan():
			if err, ok := s.(ErrorSignal); ok {
				id, _ := err.Value()
				return id
			}
		default:
			t.Fatal("no ErrorSignal")
			return 0
		}
	}
}

// TestRepeatFailedResultsSignal checks the matches a failed n sends: none
// once an edit removed them all, and the current one while it is still there
// (n past the last match with Wrap off), as SearchResults.
func TestRepeatFailedResultsSignal(t *testing.T) {
	lastResults := func(e Editor) []Position {
		got := []Position{{-1, -1}} // no SearchResultsSignal
		for {
			select {
			case s := <-e.GetUpdateSignalChan():
				if r, ok := s.(SearchResultsSignal); ok {
					got = r.Value()
				}
			default:
				return got
			}
		}
	}

	e := newTestEditor("foo 1\nbar 2\nfoo 3\nbar 4")
	typeKeys(t, e, "/bar<CR>ddGdd")
	drainSignals(e)
	typeKeys(t, e, "ggn")
	assert.Empty(t, lastResults(e), "the stored match is now foo 3")

	e = newTestEditor("foo 1\nbar 2\nfoo 3\nbar 4")
	e.SetSearchOptions(SearchOptions{Wrap: false})
	typeKeys(t, e, "/bar<CR>n")
	drainSignals(e)
	typeKeys(t, e, "n")
	assert.Equal(t, Position{3, 0}, cursorPos(e))
	assert.Equal(t, []Position{{3, 0}}, lastResults(e))
}

// TestSearchNotFound checks the error a search, n or N sends when it finds
// no match, as Vim 9.2: E486 with Wrap on, E385/E384 with it off (going
// forward/backward), even when the pattern is nowhere in the buffer. Only
// the cases with a message send an error at all.
func TestSearchNotFound(t *testing.T) {
	tests := []struct {
		noWrap bool
		keys   string
		cursor Position
		err    string
	}{
		{false, "/zzz<CR>", Position{0, 0}, "pattern not found: zzz"},
		{false, "?zzz<CR>", Position{0, 0}, "pattern not found: zzz"},
		{false, "/bar<CR>ddGddggn", Position{0, 0}, "pattern not found: bar"},
		{false, "/bar<CR>ddGddggN", Position{0, 0}, "pattern not found: bar"},
		{false, "/bar<CR>ddGdd/<C-U><CR>", Position{1, 0}, "pattern not found: bar"},
		{true, "/bar<CR>nn", Position{3, 0}, "search hit BOTTOM without match for: bar"},
		{true, "G?bar<CR>n", Position{1, 0}, "search hit TOP without match for: bar"},
		{true, "/zzz<CR>", Position{0, 0}, "search hit BOTTOM without match for: zzz"},
		{true, "G/bar<CR>", Position{3, 0}, "search hit BOTTOM without match for: bar"},
		// Found: no error.
		{false, "/bar<CR>n", Position{3, 0}, ""},
		{false, "/bar<CR>cwxyz<Esc>ggn", Position{3, 0}, ""},
		{true, "/bar<CR>nN", Position{1, 0}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor("foo 1\nbar 2\nfoo 3\nbar 4")
			if tt.noWrap {
				e.SetSearchOptions(SearchOptions{Wrap: false})
			}
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.cursor, cursorPos(e))

			var errs []string
			for done := false; !done; {
				select {
				case s := <-e.GetUpdateSignalChan():
					if es, ok := s.(ErrorSignal); ok {
						_, err := es.Value()
						errs = append(errs, err.Error())
					}
				default:
					done = true
				}
			}
			if tt.err == "" {
				assert.Empty(t, errs)
			} else {
				assert.Equal(t, []string{tt.err}, errs)
			}
		})
	}
}

// TestRepeatWithoutLastSearch checks that n and N with no last search move
// nothing and report ErrNoPreviousPattern, as Vim 9.2's E35 (also in visual
// mode, and after abandoning the search line, which clears the last search).
func TestRepeatWithoutLastSearch(t *testing.T) {
	for _, keys := range []string{"n", "N", "jn", "vjN", "/bar<CR>gg/<Esc>n"} {
		t.Run(keys, func(t *testing.T) {
			e := newTestEditor("foo 1\nbar 2")
			typeKeys(t, e, keys[:len(keys)-1])
			before := cursorPos(e)
			drainSignals(e)
			typeKeys(t, e, keys[len(keys)-1:])
			assert.Equal(t, before, cursorPos(e))
			assert.Equal(t, ErrNoPreviousPatternId, nextErrorID(t, e))
		})
	}
}

// TestRepeatAfterFailedSearch checks that n and N search again after a search
// that found nothing, as Vim 9.2: an edit may have added a match since.
func TestRepeatAfterFailedSearch(t *testing.T) {
	tests := []struct {
		keys   string
		cursor Position
	}{
		{"/zzz<CR>n", Position{0, 0}},
		{"/zzz<CR>Ozzz<Esc>Gn", Position{0, 0}},
		{"/zzz<CR>Ozzz<Esc>GN", Position{0, 0}},
		{"?zzz<CR>Ozzz<Esc>jn", Position{0, 0}},
		{"/bar<CR>/<C-U>zzz<CR>Ozzz<Esc>Gn", Position{1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor("foo 1\nbar 2\nfoo 3\nbar 4")
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, "zzz", e.GetState().SearchQuery.Pattern)
		})
	}
}

// TestSearchResultsFollowEdits checks that SearchResults drops a match an edit
// changed (bug 58): the highlight is painted from these positions.
func TestSearchResultsFollowEdits(t *testing.T) {
	t.Run("changing the match drops it", func(t *testing.T) {
		e := newTestEditor("the sensors read\nsensors")
		wrapSearch(e, "sensors")
		require.Equal(t, []Position{{0, 4}}, e.SearchResults())
		typeKeys(t, e, "ciwdetectors<Esc>")
		assert.Empty(t, e.SearchResults())
	})

	t.Run("an edit before the match on its line drops it", func(t *testing.T) {
		e := newTestEditor("the sensors read")
		wrapSearch(e, "sensors")
		typeKeys(t, e, "0x")
		assert.Empty(t, e.SearchResults())
	})

	t.Run("lines put above move the line away from the match", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("abc\nthe sensors")
		wrapSearch(e, "sensors")
		require.Equal(t, []Position{{1, 4}}, e.SearchResults())
		typeKeys(t, e, "kyyP")
		require.Equal(t, "abc\nabc\nthe sensors", e.GetBuffer().GetCurrentContent())
		assert.Empty(t, e.SearchResults())
	})

	t.Run("deleting the match line past the end drops it", func(t *testing.T) {
		e := newTestEditor("abc\nthe sensors")
		wrapSearch(e, "sensors")
		typeKeys(t, e, "dd")
		assert.Empty(t, e.SearchResults())
	})

	t.Run("an edit that leaves the match keeps it", func(t *testing.T) {
		e := newTestEditor("the sensors read")
		wrapSearch(e, "sensors")
		typeKeys(t, e, "$x")
		assert.Equal(t, []Position{{0, 4}}, e.SearchResults())
	})

	t.Run("undo brings the match back", func(t *testing.T) {
		e := newTestEditor("the sensors read")
		wrapSearch(e, "sensors")
		typeKeys(t, e, "xu")
		assert.Equal(t, []Position{{0, 4}}, e.SearchResults())
	})

	t.Run("search options apply to the check", func(t *testing.T) {
		e := newTestEditor("the Sensors read")
		e.ExecuteSearch(`\<sensors\>\c`, SearchOptions{Wrap: true})
		require.Equal(t, []Position{{0, 4}}, e.SearchResults())
		typeKeys(t, e, "$x")
		assert.Equal(t, []Position{{0, 4}}, e.SearchResults())
		typeKeys(t, e, "0fsas<Esc>")
		assert.Empty(t, e.SearchResults(), "the word boundary no longer holds")
	})

	t.Run("n still searches after the match is dropped", func(t *testing.T) {
		e := newTestEditor("the sensors read\nsensors")
		wrapSearch(e, "sensors")
		typeKeys(t, e, "ciwdetectors<Esc>")
		require.Empty(t, e.SearchResults())
		typeKeys(t, e, "n")
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		assert.Equal(t, []Position{{1, 0}}, e.SearchResults())
	})
}
