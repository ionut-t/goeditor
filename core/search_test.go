package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestSearchInlineModifiers tests the \c, \C, \<, and \> inline modifiers.
func TestSearchInlineModifiers(t *testing.T) {
	t.Run(`\c forces case-insensitive search`, func(t *testing.T) {
		e := newTestEditor("Hello World")
		e.ExecuteSearch(`hello\c`, SearchOptions{})
		assert.Equal(t, Position{0, 0}, cursorPos(e))
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

	t.Run("* on non-word character does nothing", func(t *testing.T) {
		e := newTestEditor("foo bar")
		keys(e, 'w') // move to "bar" at col 4
		keys(e, 'e') // move to end of "bar" → col 6
		keys(e, 'l') // move right to space? Actually 'l' from end of word stays. Let's go to space via '$' then 'F' ' '
		// simpler: position on the space between words
		e2 := newTestEditor("foo bar")
		keys(e2, 'f', ' ') // jump to space at col 3
		initialPos := cursorPos(e2)
		e2.SearchWordUnderCursor(false)
		assert.Equal(t, initialPos, cursorPos(e2))
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
