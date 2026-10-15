package goeditor

import (
	"fmt"
	"strings"
	"testing"

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
		// tokeniseRange leaves an empty entry for the line after the range
		// it tokenised. It holds no tokens, so it can't draw stale colours.
		tokens := m.highlighter.GetTokensForLine(i, nil)
		if len(tokens) == 0 {
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
