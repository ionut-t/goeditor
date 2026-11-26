package highlighter

import (
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_KnownLanguageTokenisesAsThatLanguage(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"func main() {}"}
	h.Tokenise(lines, 0, 1)

	tokens := h.GetTokensForLine(0, lines)
	require.NotNil(t, tokens)

	foundKeyword := false
	for _, tok := range tokens {
		if tok.Value == "func" && tok.Type.InCategory(chroma.Keyword) {
			foundKeyword = true
		}
	}
	assert.True(t, foundKeyword, "expected 'func' to be tokenised as a keyword, got %+v", tokens)
}

func TestNew_UnknownLanguageFallsBackWithoutPanic(t *testing.T) {
	require.NotPanics(t, func() {
		h := New("not-a-real-language-xyz", "monokai")
		lines := []string{"some plain text"}
		h.Tokenise(lines, 0, 1)
		tokens := h.GetTokensForLine(0, lines)
		assert.NotNil(t, tokens)
	})
}

// TestTokenisedLinesReconstructOriginalText is the highlighter-side analogue
// of the rendering-layer "content must never disappear" property: every rune
// of every tokenised line must show up, in order, in the concatenation of
// that line's token values, for every line in the requested range.
func TestTokenisedLinesReconstructOriginalText(t *testing.T) {
	source := `package main

import "fmt"

func main() {
	// a comment with unicode: héllo wörld 👍
	x := 1
	y := 2
	fmt.Println(x + y)
}
`
	lines := strings.Split(strings.TrimRight(source, "\n"), "\n")

	h := New("go", "monokai")
	h.Tokenise(lines, 0, len(lines))

	for i, line := range lines {
		tokens := h.GetTokensForLine(i, lines)
		require.NotNilf(t, tokens, "line %d: expected tokens to be cached", i)

		var reconstructed strings.Builder
		for _, tok := range tokens {
			reconstructed.WriteString(tok.Value)
		}
		assert.Equalf(t, line, reconstructed.String(), "line %d: token values should reconstruct the original line verbatim", i)
	}
}

func TestTokeniseOutOfRangeIsNoOp(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"a", "b", "c"}

	require.NotPanics(t, func() {
		h.Tokenise(lines, -1, 2)
		h.Tokenise(lines, 0, 10)
		h.Tokenise(lines, 2, 1)
		h.Tokenise(lines, 3, 3)
	})

	// None of the malformed calls should have populated the cache.
	assert.Nil(t, h.GetTokensForLine(0, lines))
}

func TestInvalidateCacheClearsTokensKeepsStyles(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1", "y := 2"}
	h.Tokenise(lines, 0, len(lines))
	require.NotNil(t, h.GetTokensForLine(0, lines))

	// Populate the style cache too.
	tokens := h.GetTokensForLine(0, lines)
	require.NotEmpty(t, tokens)
	style := h.GetStyleForToken(tokens[0].Type)

	h.InvalidateCache()

	assert.Nil(t, h.GetTokensForLine(0, lines))
	assert.Nil(t, h.GetTokensForLine(1, lines))
	// Styles depend only on the theme, so a content invalidation keeps them.
	assert.Contains(t, h.styleCache, tokens[0].Type)
	assert.Equal(t, style.String(), h.GetStyleForToken(tokens[0].Type).String())
}

// TestTokeniseSkipsReTokenisingAlreadyCachedLines proves the incremental
// caching optimisation: calling Tokenise again for a range that is already
// fully cached must not re-tokenise, even if the underlying `lines` slice
// changed in the meantime (a caller is expected to InvalidateCache first).
func TestTokeniseSkipsReTokenisingAlreadyCachedLines(t *testing.T) {
	h := New("go", "monokai")
	original := []string{"x := 1", "y := 2"}
	h.Tokenise(original, 0, len(original))

	before := h.GetTokensForLine(0, original)
	require.NotNil(t, before)

	mutated := []string{"totally different content here", "y := 2"}
	h.Tokenise(mutated, 0, len(mutated))

	after := h.GetTokensForLine(0, mutated)
	require.NotNil(t, after)

	var reconstructed strings.Builder
	for _, tok := range after {
		reconstructed.WriteString(tok.Value)
	}
	assert.Equal(t, original[0], reconstructed.String(),
		"Tokenise should have skipped re-tokenising an already-cached range, so stale (original) tokens should remain until InvalidateCache is called")
}

func TestTokeniseCachesOnlyTheRequestedRange(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1", "y := 2", "z := 3"}
	h.Tokenise(lines, 0, 2)

	require.NotNil(t, h.GetTokensForLine(0, lines))
	require.NotNil(t, h.GetTokensForLine(1, lines))
	// An entry for line 2, even an empty one, would make a later Tokenise
	// treat it as already done.
	assert.Nil(t, h.GetTokensForLine(2, lines))
}

func TestTokeniseCachesAnEmptyLastLine(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1", ""}
	h.Tokenise(lines, 0, 2)

	// The joined range already ends in a newline, so the empty line's entry
	// comes from that newline and must not be cut off with the range end.
	tokens := h.GetTokensForLine(1, lines)
	assert.NotNil(t, tokens)
	assert.Empty(t, tokens)
}

func TestTokeniseKeepsTokensForTheLineAfterTheRange(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1", "y := 2", "z := 3", "w := 4"}
	h.Tokenise(lines, 2, 4)
	h.Tokenise(lines, 0, 2)

	var b strings.Builder
	for _, tok := range h.GetTokensForLine(2, lines) {
		b.WriteString(tok.Value)
	}
	assert.Equal(t, lines[2], b.String())
}

func TestGetTokensForLineUnknownLineReturnsNil(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1"}
	assert.Nil(t, h.GetTokensForLine(5, lines))
}

func TestGetStyleForTokenIsCachedAndConsistent(t *testing.T) {
	h := New("go", "monokai")

	s1 := h.GetStyleForToken(chroma.Keyword)
	s2 := h.GetStyleForToken(chroma.Keyword)
	assert.Equal(t, s1.String(), s2.String())

	s3 := h.GetStyleForToken(chroma.Comment)
	// Different token categories in a real theme should not collide on the
	// exact same rendered style (a weak but useful sanity check that styles
	// are actually being differentiated, not just defaulted).
	assert.NotEqual(t, s1.String(), s3.String())
}

// TestHighlighterCacheAccessIsRace-free exercises Tokenise, GetTokensForLine,
// GetStyleForToken and InvalidateCache concurrently. Run with
// `go test -race` to catch data races on the internal caches guarded by
// cacheMutex/styleCacheMutex.
func TestHighlighterCacheAccessIsRaceFree(t *testing.T) {
	h := New("go", "monokai")
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "x := 1 // filler line to have something to tokenise"
	}

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := range 50 {
				start := (seed + i) % 190
				end := start + 10
				h.Tokenise(lines, start, end)
				for l := start; l < end; l++ {
					tokens := h.GetTokensForLine(l, lines)
					for _, tok := range tokens {
						h.GetStyleForToken(tok.Type)
					}
				}
				if i%11 == 0 {
					h.InvalidateCache()
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestGetTokenPositionsAreRuneBased(t *testing.T) {
	tokens := []chroma.Token{
		{Type: chroma.Comment, Value: "// héllo"},
		{Type: chroma.Text, Value: " "},
		{Type: chroma.Name, Value: "wörld"},
	}

	positions := GetTokenPositions(tokens)
	require.Len(t, positions, 3)

	assert.Equal(t, 0, positions[0].StartCol)
	assert.Equal(t, len([]rune("// héllo")), positions[0].EndCol)

	assert.Equal(t, positions[0].EndCol, positions[1].StartCol)
	assert.Equal(t, positions[1].EndCol, positions[2].StartCol)

	lastEnd := positions[len(positions)-1].EndCol
	totalRunes := 0
	for _, tok := range tokens {
		totalRunes += len([]rune(tok.Value))
	}
	assert.Equal(t, totalRunes, lastEnd)
}

func TestFindTokenAtPosition(t *testing.T) {
	tokens := []chroma.Token{
		{Type: chroma.Keyword, Value: "func"},
		{Type: chroma.Text, Value: " "},
		{Type: chroma.Name, Value: "main"},
	}
	positions := GetTokenPositions(tokens)

	tok, ok := FindTokenAtPosition(positions, 0)
	require.True(t, ok)
	assert.Equal(t, chroma.Keyword, tok.Type)

	tok, ok = FindTokenAtPosition(positions, 3)
	require.True(t, ok)
	assert.Equal(t, chroma.Keyword, tok.Type, "col 3 is still inside 'func' (0-3)")

	tok, ok = FindTokenAtPosition(positions, 4)
	require.True(t, ok)
	assert.Equal(t, chroma.Text, tok.Type, "col 4 is the space, the boundary is exclusive on the previous token")

	tok, ok = FindTokenAtPosition(positions, 5)
	require.True(t, ok)
	assert.Equal(t, chroma.Name, tok.Type)

	_, ok = FindTokenAtPosition(positions, 9)
	assert.False(t, ok, "col at/after the end of the last token should not match")

	_, ok = FindTokenAtPosition(nil, 0)
	assert.False(t, ok)
}

func TestTokeniseEmptyLinesGetEmptyNonNilCache(t *testing.T) {
	h := New("go", "monokai")
	lines := []string{"x := 1", "", "y := 2"}
	h.Tokenise(lines, 0, len(lines))

	blank := h.GetTokensForLine(1, lines)
	assert.NotNil(t, blank, "a blank line should still get a cache entry so the renderer doesn't treat it as invalidated")
	assert.Empty(t, blank)
}

func TestSyncRange(t *testing.T) {
	lines := strings.Split(strings.Join([]string{
		"# Title",   // 0
		"```go",     // 1
		"x := 1",    // 2
		"y := 2",    // 3
		"```",       // 4
		"prose",     // 5
		"```",       // 6 opens a block without a language
		"text",      // 7
		"```",       // 8
		"```go x",   // 9 not an opening line: a space after the name
		"after",     // 10
		"```python", // 11 no closing fence below: no block
		"tail",      // 12
	}, "\n"), "\n")

	tests := []struct {
		name               string
		start, end         int
		wantStart, wantEnd int
	}{
		{"outside any block", 5, 6, 5, 6},
		{"start inside a block", 3, 6, 1, 6},
		{"start on the closing fence", 4, 6, 1, 6},
		{"start on the opening fence", 1, 6, 1, 6},
		{"end inside a block", 0, 3, 0, 5},
		{"end on the closing fence", 0, 4, 0, 5},
		{"both inside one block", 2, 3, 1, 5},
		{"block without a language", 7, 8, 6, 9},
		{"not a fence", 10, 11, 10, 11},
		{"unclosed fence", 12, 13, 12, 13},
	}
	h := New("markdown", "monokai")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := h.syncRange(lines, tt.start, tt.end)
			assert.Equal(t, [2]int{tt.wantStart, tt.wantEnd}, [2]int{start, end})
		})
	}

	t.Run("other languages are left alone", func(t *testing.T) {
		start, end := New("go", "monokai").syncRange(lines, 3, 6)
		assert.Equal(t, [2]int{3, 6}, [2]int{start, end})
	})
}

func TestTokeniseLexesMarkdownFromTheOpeningFence(t *testing.T) {
	lines := []string{"```go", "x := `a", "b`", "```", "## After"}
	whole := New("markdown", "monokai")
	whole.Tokenise(lines, 0, len(lines))

	h := New("markdown", "monokai")
	h.Tokenise(lines, 2, len(lines)) // starts inside the block
	for i := range lines {
		assert.Equalf(t, whole.GetTokensForLine(i, lines), h.GetTokensForLine(i, lines), "line %d", i)
	}
}

func TestSyncRangeFrontmatter(t *testing.T) {
	lines := []string{
		"---",      // 0 opens the frontmatter
		"title: x", // 1
		"---",      // 2 closes it
		"prose",    // 3
		"---",      // 4 a thematic break
		"more",     // 5
		"```",      // 6
		"code",     // 7
		"```",      // 8
		"---",      // 9 a thematic break right after a closing fence
		"tail",     // 10
	}
	tests := []struct {
		name               string
		start, end         int
		wantStart, wantEnd int
	}{
		{"inside the frontmatter", 1, 4, 0, 4},
		{"on its closing line", 2, 4, 0, 4},
		{"cutting it at the bottom", 0, 1, 0, 3},
		{"after it", 3, 5, 3, 5},
		{"on a thematic break", 4, 6, 4, 6},
		{"on a thematic break after a fence", 9, 11, 9, 11},
	}
	h := New("markdown", "monokai")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := h.syncRange(lines, tt.start, tt.end)
			assert.Equal(t, [2]int{tt.wantStart, tt.wantEnd}, [2]int{start, end})
		})
	}
}

// TestTokeniseMarkdownMatchesAWholeFileLex lexes every range of a document
// with frontmatter, code blocks and thematic breaks, and checks each line's
// tokens against a lex of the whole document.
func TestTokeniseMarkdownMatchesAWholeFileLex(t *testing.T) {
	docs := map[string][]string{
		"with frontmatter": {
			"---", "title: x", "tags: [a]", "---", "# Head", "---", "*em*", "```go", "x := `a", "---", "b`", "```", "---", "## Sub", "---", "end",
		},
		"without frontmatter": {
			"# Head", "---", "key: value", "---", "*em*", "```", "---", "```", "---", "---", "tail",
		},
		"unclosed frontmatter": {
			"---", "title: x", "# Head", "```go", "x := 1", "```",
		},
	}
	crlfDocs := make(map[string][]string, len(docs))
	for name, lines := range docs {
		crlf := make([]string, len(lines))
		for i, line := range lines {
			crlf[i] = line + "\r" // a file with CRLF line endings, which chroma lexes as LF
		}
		crlfDocs[name+" (CRLF)"] = crlf
	}
	maps.Copy(docs, crlfDocs)
	for name, lines := range docs {
		t.Run(name, func(t *testing.T) {
			whole := New("markdown", "monokai")
			whole.Tokenise(lines, 0, len(lines))
			for start := 0; start < len(lines); start++ {
				for end := start + 1; end <= len(lines); end++ {
					h := New("markdown", "monokai")
					h.Tokenise(lines, start, end)
					for i := start; i < end; i++ {
						require.Equalf(t, whole.GetTokensForLine(i, lines), h.GetTokensForLine(i, lines), "range [%d, %d), line %d %q", start, end, i, lines[i])
					}
				}
			}
		})
	}
}

func TestGetTokenPositionsForLine(t *testing.T) {
	lines := []string{"x := \"é\" // a", "", "y := 2"}
	h := New("go", "monokai")
	assert.Nil(t, h.GetTokenPositionsForLine(0), "nothing is tokenised yet")

	h.Tokenise(lines, 0, 2)
	for i := range 2 {
		assert.Equalf(t, GetTokenPositions(h.GetTokensForLine(i, lines)), h.GetTokenPositionsForLine(i), "line %d", i)
	}
	assert.NotNil(t, h.GetTokenPositionsForLine(1), "a blank line should get empty positions, not nil")
	assert.Nil(t, h.GetTokenPositionsForLine(2), "the line after the range isn't tokenised")

	h.InvalidateCache()
	assert.Nil(t, h.GetTokenPositionsForLine(0))
}

func TestTokeniseCachesASingleEmptyLine(t *testing.T) {
	// An empty buffer is one empty line: with no entry for it every redraw
	// found the range uncached and tokenised it again.
	h := New("go", "monokai")
	h.Tokenise([]string{""}, 0, 1)
	assert.NotNil(t, h.GetTokensForLine(0, nil))
	assert.NotNil(t, h.GetTokenPositionsForLine(0))
}
