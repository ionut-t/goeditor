package goeditor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/ionut-t/goeditor/highlighter"

	"github.com/stretchr/testify/require"
)

// alternatingGoLines returns lines that alternate between a comment and code,
// so tokens cached for one line are wrong for its neighbours.
func alternatingGoLines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if i%2 == 0 {
			fmt.Fprintf(&b, "// comment %d\n", i)
		} else {
			fmt.Fprintf(&b, "x := %d\n", i)
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func newHighlightedModel(content string) Model {
	m := New(60, 22)
	m.Focus()
	m.SetContent(content)
	m.SetLanguage("go", "monokai")
	return m
}

// cacheBottomOfFile scrolls through the end of the file and back to the top.
// Before the fix this left tokens cached for lines 250-299, so a later jump
// there by a setter (which tokenises nothing if its range is fully cached)
// drew those stale tokens.
func cacheBottomOfFile(m Model) Model {
	m = typeText("G")(m)
	for range 3 {
		m = pressCtrl('u')(m)
	}
	return typeText("gg")(m)
}

// requireCachedTokensMatchBuffer checks that the screen matches a freshly
// loaded model showing the same lines, and that every cached line's tokens
// spell out the current buffer line.
func requireCachedTokensMatchBuffer(t *testing.T, m Model, cursorRow int) {
	t.Helper()
	lines := m.editor.GetBuffer().GetLines()

	fresh := newHighlightedModel(strings.Join(lines, "\n"))
	require.NoError(t, fresh.SetCursorPosition(cursorRow, 0))
	require.Equal(t, fresh.viewport.View(), m.viewport.View(), "screen should match a freshly loaded model")

	for i, line := range lines {
		tokens := m.highlighter.GetTokensForLine(i, nil)
		if tokens == nil {
			continue
		}
		var b strings.Builder
		for _, tok := range tokens {
			b.WriteString(tok.Value)
		}
		require.Equalf(t, line, b.String(), "cached tokens for line %d", i)
	}
}

func TestOffscreenTokensRefreshAfterLineShift(t *testing.T) {
	m := cacheBottomOfFile(newHighlightedModel(alternatingGoLines(300)))

	m = typeText("dd")(m)
	require.Len(t, m.editor.GetBuffer().GetLines(), 299, "dd should delete a line")

	require.NoError(t, m.SetCursorPosition(280, 0))
	requireCachedTokensMatchBuffer(t, m, 280)
}

func TestOffscreenTokensRefreshAfterMultiLineEdit(t *testing.T) {
	m := cacheBottomOfFile(newHighlightedModel(alternatingGoLines(300)))

	// Upper-casing every line keeps the line count but changes lines far
	// from the cursor, which ends up back on line 0.
	m = typeText("VGU")(m)
	require.Equal(t, "X := 1", m.editor.GetBuffer().GetLines()[0], "VGU should upper-case the file")
	require.Equal(t, 0, m.editor.GetBuffer().GetCursor().Position.Row)

	require.NoError(t, m.SetCursorPosition(280, 0))
	requireCachedTokensMatchBuffer(t, m, 280)
}

func TestOffscreenTokensRefreshAfterSetBytes(t *testing.T) {
	m := cacheBottomOfFile(newHighlightedModel(alternatingGoLines(300)))

	// Same line count, every line different.
	m.SetContent("\n" + alternatingGoLines(299))
	require.Len(t, m.editor.GetBuffer().GetLines(), 300)

	require.NoError(t, m.SetCursorPosition(280, 0))
	requireCachedTokensMatchBuffer(t, m, 280)
}

// markdownWithCodeBlocks returns n lines of markdown: sections whose body is
// a fenced Go block of blockLines lines.
func markdownWithCodeBlocks(n, blockLines int) string {
	var b strings.Builder
	for i := 0; i < n; {
		fmt.Fprintf(&b, "## Section %d\n\nSome *prose* here.\n\n```go\n", i)
		i += 5
		for j := 0; j < blockLines && i < n; j++ {
			fmt.Fprintf(&b, "\tx := foo(%d) // note\n", j)
			i++
		}
		b.WriteString("```\n\n")
		i += 2
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// firstToken returns the address of the first cached token of line, which
// changes when the line is lexed again.
func firstToken(t *testing.T, m Model, line int) *chroma.Token {
	t.Helper()
	tokens := m.highlighter.GetTokensForLine(line, nil)
	require.NotEmptyf(t, tokens, "line %d should be tokenised", line)
	return &tokens[0]
}

func TestKeysThatChangeNoTextKeepTokens(t *testing.T) {
	m := newHighlightedModel(alternatingGoLines(300))
	m = typeText("jj")(m) // render, cursor away from the top
	tok := firstToken(t, m, 0)

	for _, keys := range []string{"j", "k", "w", "b", "v", "\x1b", "yy", "i"} {
		if keys == "\x1b" {
			m = escKey(m)
		} else {
			m = typeText(keys)(m)
		}
		require.Samef(t, tok, firstToken(t, m, 0), "%q should not re-lex", keys)
	}
	m = escKey(m)
	m = resizeTo(70, 22)(m)
	require.Same(t, tok, firstToken(t, m, 0), "a resize should not re-lex")

	m = typeText("x")(m)
	require.NotSame(t, tok, firstToken(t, m, 0), "an edit should re-lex")
}

// BenchmarkKeypressMarkdown measures a j/k keypress over a long markdown
// document with the default context lines.
func BenchmarkKeypressMarkdown(b *testing.B) {
	m := New(100, 52)
	m.Focus()
	m.SetContent(markdownWithCodeBlocks(2000, 40))
	m.SetLanguage("markdown", "monokai")
	m = typeText("1000G")(m)
	keys := []string{"j", "k"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m = typeText(keys[i%2])(m)
	}
}

// requireTokensMatchWholeFileLex checks that the cached tokens of lines
// [from, to) have the types a lex of the whole file gives them.
func requireTokensMatchWholeFileLex(t *testing.T, m Model, from, to int) {
	t.Helper()
	lines := m.editor.GetBuffer().GetLines()
	whole := highlighter.New("markdown", "monokai")
	whole.Tokenise(lines, 0, len(lines))
	for i := from; i < to; i++ {
		require.Equalf(t, whole.GetTokensForLine(i, nil), m.highlighter.GetTokensForLine(i, nil), "tokens of line %d %q", i, lines[i])
	}
}

func TestMarkdownTopInsideALongCodeBlock(t *testing.T) {
	// Sections at lines 0 and 307; the first block runs from line 4 to 305,
	// longer than the screen.
	content := markdownWithCodeBlocks(600, 300)
	require.Equal(t, "## Section 307", strings.Split(content, "\n")[307])

	m := New(60, 22)
	m.Focus()
	m.SetContent(content)
	m.SetLanguage("markdown", "monokai")
	require.NoError(t, m.SetCursorPosition(308, 0))
	top := m.editor.GetState().TopLine
	require.Less(t, top, 305, "the top should be inside the block")

	// Lexed from inside the block, the closing fence read as an opening one:
	// the heading after it and the next block came out as plain text.
	requireTokensMatchWholeFileLex(t, m, top, 309)
	heading := m.highlighter.GetTokensForLine(307, nil)
	require.Equal(t, chroma.GenericSubheading, heading[0].Type)
}

func TestMarkdownCodeBlockCutAtTheBottom(t *testing.T) {
	// The block from line 4 to 305 ends far below the screen; lexed without
	// its closing fence it was plain text.
	m := New(60, 22)
	m.Focus()
	m.SetContent(markdownWithCodeBlocks(600, 300))
	m.SetLanguage("markdown", "monokai")
	requireTokensMatchWholeFileLex(t, m, 0, 20)
}

func TestMarkdownHasNoDefaultContextLines(t *testing.T) {
	m := New(60, 22)
	m.SetLanguage("markdown", "monokai")
	require.Zero(t, m.extraHighlightedContextLines, "the fence sync replaces the old default of 100")
}
