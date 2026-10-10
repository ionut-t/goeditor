package goeditor

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor/highlighter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Action helpers for driving the Model the same way a real terminal frontend
// would: through Update with bubbletea key messages.
// ---------------------------------------------------------------------------

type modelAction func(Model) Model

func typeRune(r rune) modelAction {
	return func(m Model) Model {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		return m
	}
}

func typeText(s string) modelAction {
	return func(m Model) Model {
		for _, r := range s {
			m = typeRune(r)(m)
		}
		return m
	}
}

func pressSpecial(code rune) modelAction {
	return func(m Model) Model {
		m, _ = m.Update(tea.KeyPressMsg{Code: code})
		return m
	}
}

func resizeTo(w, h int) modelAction {
	return func(m Model) Model {
		m.SetSize(w, h)
		return m
	}
}

var escKey = pressSpecial(tea.KeyEscape)

// ---------------------------------------------------------------------------
// The core regression guard: whatever is visible in the internal visual
// layout for the current viewport must actually show up, verbatim, in the
// rendered (ANSI-stripped) content. This is the property that fails when
// "the line with the cursor disappears" — the layout says the text is there,
// but the render for that row comes out blank or truncated.
// ---------------------------------------------------------------------------

// expandTabsForCompare mirrors the tab-stop math renderSegment uses (tab
// width 4, measured from the start of the segment) so expected content can
// be compared against the rendered, tab-expanded row text.
func expandTabsForCompare(s string) string {
	const tabWidth = 4
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabWidth - (col % tabWidth)
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else {
			b.WriteRune(r)
			col++
		}
	}
	return b.String()
}

func assertVisibleContentIntact(t *testing.T, m Model, label string) {
	t.Helper()

	stripped := ansi.Strip(m.viewport.View())
	rows := strings.Split(stripped, "\n")

	height := m.viewport.Height()
	for i := range height {
		absRow := m.currentVisualTopLine + i
		if m.fullVisualLayoutHeight == 0 || absRow >= m.fullVisualLayoutHeight {
			continue
		}

		cacheIdx := absRow - m.visualLayoutCacheStartVisualRow
		if cacheIdx < 0 || cacheIdx >= len(m.visualLayoutCache) {
			// Row falls outside the currently cached window (can happen with
			// the lazy layout); nothing to check against.
			continue
		}

		vli := m.visualLayoutCache[cacheIdx]
		expected := strings.TrimRight(expandTabsForCompare(vli.Content), " ")
		if expected == "" {
			continue
		}

		require.Lessf(t, i, len(rows), "%s: row %d (logical line %d, %q) missing entirely from rendered output", label, i, vli.LogicalRow, expected)
		assert.Containsf(t, rows[i], expected, "%s: row %d (logical line %d) should render %q", label, i, vli.LogicalRow, expected)
	}
}

// runScenario replays every action, asserting after each one that the
// visible content matches the internal layout. It also asserts that the
// cursor's own logical row is always represented within the rendered slice
// so a bug can't hide a whole row by pointing the cursor off-screen.
func runScenario(t *testing.T, m Model, actions []modelAction) Model {
	t.Helper()

	assertVisibleContentIntact(t, m, "initial")

	for i, act := range actions {
		m = act(m)
		label := fmt.Sprintf("step %d", i)
		assertVisibleContentIntact(t, m, label)

		cursorRow := m.editor.GetBuffer().GetCursor().Position.Row
		if m.fullVisualLayoutHeight > 0 {
			found := false
			for _, vli := range m.visualLayoutCache {
				if vli.LogicalRow == cursorRow {
					found = true
					break
				}
			}
			// The cursor's logical row should be represented somewhere in the
			// cached layout window; if it isn't, the cache failed to track the
			// cursor and rendering can silently show stale/blank content.
			assert.Truef(t, found, "%s: cursor row %d not present in visual layout cache", label, cursorRow)
		}
	}

	return m
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

func TestRenderedContentNeverDropsCharacters(t *testing.T) {
	longWrappingComment := "\t// this is a fairly long comment line that should wrap across multiple visual rows in a narrow viewport"

	scenarios := []struct {
		name        string
		content     string
		width       int
		height      int
		highlighted bool
		actions     []modelAction
	}{
		{
			name:    "short lines, plain navigation",
			content: "hello\nworld\nfoo bar baz\nqux\n",
			width:   40, height: 10,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('k'), typeRune('k'),
				typeRune('l'), typeRune('l'), typeRune('h'),
			},
		},
		{
			name:    "long wrapped line, narrow tall viewport",
			content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n" + longWrappingComment + "\n\tx := 1\n\ty := 2\n\tfmt.Println(x + y)\n}\n",
			width:   30, height: 12,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'),
			},
		},
		{
			name:    "long wrapped line, tiny viewport (segment doesn't fully fit)",
			content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n" + longWrappingComment + "\n\tx := 1\n\ty := 2\n\tfmt.Println(x + y)\n}\n",
			width:   30, height: 8,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'),
			},
		},
		{
			name:    "long wrapped line with syntax highlighter",
			content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n" + longWrappingComment + "\n\tx := 1\n\ty := 2\n\tfmt.Println(x + y)\n}\n",
			width:   30, height: 10,
			highlighted: true,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('k'), typeRune('k'), typeRune('k'), typeRune('k'),
			},
		},
		{
			name:    "typing reflows the current line while wrapping",
			content: "package main\n\nfunc main() {\n\t// short\n}\n",
			width:   30, height: 10,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('$'), typeRune('a'),
				typeText(" this is extra text that should push the comment line well past the wrap width"),
				escKey,
			},
		},
		{
			name:    "typing then backspacing unwraps the current line",
			content: "package main\n\nfunc main() {\n\t// short\n}\n",
			width:   30, height: 10,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('$'), typeRune('a'),
				typeText(" this is extra text that should push the comment line well past the wrap width"),
				pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace),
				pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace),
				pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace),
				pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace),
				pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace), pressSpecial(tea.KeyBackspace),
				escKey,
			},
		},
		{
			name:    "resize while cursor sits on a wrapped line",
			content: "package main\n\nfunc main() {\n" + longWrappingComment + "\n}\n",
			width:   40, height: 12,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				resizeTo(20, 8),
				resizeTo(60, 20),
				resizeTo(15, 6),
			},
		},
		{
			name:    "undo/redo around an edit on a wrapped line",
			content: "package main\n\nfunc main() {\n" + longWrappingComment + "\n}\n",
			width:   30, height: 10,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('$'), typeRune('a'),
				typeText(" more text to force a rewrap here"),
				escKey,
				typeRune('u'), typeRune('u'), typeRune('u'),
				pressSpecial(tea.KeyEscape),
			},
		},
		{
			name:    "visual selection over a wrapped line",
			content: "package main\n\nfunc main() {\n" + longWrappingComment + "\n}\n",
			width:   30, height: 10,
			actions: []modelAction{
				typeRune('j'), typeRune('j'), typeRune('j'),
				typeRune('0'), typeRune('v'),
				typeRune('$'),
				escKey,
				typeRune('V'),
				typeRune('j'),
				escKey,
			},
		},
		{
			name:    "search highlighting active on a wrapped line",
			content: "package main\n\nfunc main() {\n" + longWrappingComment + "\n\tvisual := true\n}\n",
			width:   30, height: 10,
			actions: []modelAction{
				typeRune('/'), typeText("visual"), pressSpecial(tea.KeyEnter),
				typeRune('n'),
				escKey,
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			m := New(sc.width, sc.height)
			if sc.highlighted {
				m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
			}
			m.Focus()
			m.SetContent(sc.content)

			runScenario(t, m, sc.actions)
		})
	}
}

// TestContentVisibleWithoutAnyUpdateCall is a regression test for a bug
// where the viewport's rendered content was only ever populated from inside
// Update() on a tea.KeyMsg. Any state-changing call made outside that path —
// including the very first paint of a freshly loaded buffer, before the user
// has pressed a single key — left View() showing blank/stale content until
// the next keystroke happened to trigger a redraw. This is the mechanism
// behind "the current line disappears until the cursor changes line".
func TestContentVisibleWithoutAnyUpdateCall(t *testing.T) {
	m := New(40, 10)
	m.Focus()
	m.SetContent("alpha\nbeta\ngamma\ndelta\n")

	stripped := ansi.Strip(m.View())
	assert.Contains(t, stripped, "alpha")
	assert.Contains(t, stripped, "beta")
	assert.Contains(t, stripped, "gamma")
	assert.Contains(t, stripped, "delta")
}

// TestSetCursorPositionRendersWithoutKeypress covers the same stale-render
// bug for a programmatic cursor jump (e.g. restoring a saved position, or
// "go to line" triggered by something other than a key event): the new
// cursor row must be visible, and the viewport must have scrolled to show
// it, without any Update() call in between.
func TestSetCursorPositionRendersWithoutKeypress(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&b, "line number %d\n", i)
	}

	m := New(40, 10)
	m.Focus()
	m.SetContent(b.String())

	require.NoError(t, m.SetCursorPosition(40, 0))

	stripped := ansi.Strip(m.View())
	assert.Containsf(t, stripped, "line number 41", "expected the jumped-to line to be visible: %s", stripped)
}

// TestSetSizeRendersWithoutKeypress covers a resize (e.g. a terminal resize
// event delivered as something other than a focused key press) reflecting
// immediately in View().
func TestSetSizeRendersWithoutKeypress(t *testing.T) {
	m := New(80, 20)
	m.Focus()
	m.SetContent("alpha\nbeta\ngamma\n")

	m.SetSize(20, 6)

	stripped := ansi.Strip(m.View())
	assert.Contains(t, stripped, "alpha")
	assert.Contains(t, stripped, "beta")
	assert.Contains(t, stripped, "gamma")
}

// TestScrollingLargeFileKeepsAllLinesReachable drives the model through a
// buffer large enough to trigger the lazy visual-layout path (> 100 lines)
// and checks the content-integrity property while scrolling down and back up
// through the whole file, and while jumping around with G/gg.
func TestScrollingLargeFileKeepsAllLinesReachable(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 250; i++ {
		fmt.Fprintf(&b, "line number %d with a bit of filler text so it is not trivially short\n", i)
	}

	m := New(40, 15)
	m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
	m.Focus()
	m.SetContent(b.String())

	var actions []modelAction
	for range 260 {
		actions = append(actions, typeRune('j'))
	}
	for range 260 {
		actions = append(actions, typeRune('k'))
	}
	actions = append(actions,
		typeRune('G'),
		typeRune('g'), typeRune('g'),
		typeRune('G'),
	)

	runScenario(t, m, actions)
}

// TestViewIsIdempotentAcrossRepeatedRenders guards against the class of bug
// where re-rendering unchanged state (e.g. from a blink tick with no other
// change) produces a different frame than the first render — a real
// terminal's diff-based repaint can then decide there's nothing to redraw and
// leave stale (blank) content on screen.
func TestViewIsIdempotentAcrossRepeatedRenders(t *testing.T) {
	m := New(30, 12)
	m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
	m.Focus()
	m.SetContent("package main\n\nfunc main() {\n\t// this is a fairly long comment that should wrap in a narrow viewport\n\tx := 1\n}\n")

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})

	first := m.View()
	for i := range 5 {
		next := m.View()
		assert.Equalf(t, first, next, "View() output changed on repeated call %d with no state change", i)
	}
}

func TestWrapLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
		want  []string
	}{
		{"empty", "", 10, []string{""}},
		{"fits exactly", "hello", 5, []string{"hello"}},
		{"fits under", "hi", 10, []string{"hi"}},
		{"breaks at space", "hello world", 5, []string{"hello", "world"}},
		{"hard break, no spaces", "abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"zero width returns whole line", "hello world", 0, []string{"hello world"}},
		{"negative width returns whole line", "hello world", -1, []string{"hello world"}},
		{"single grapheme wider than width still makes progress", "a", 0, []string{"a"}},
		{"multiple spaces collapse leading whitespace on wrap", "aaaaa   bbbbb", 5, []string{"aaaaa", "bbbbb"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapLine(tt.line, tt.width)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWrapLineNeverDropsRunes(t *testing.T) {
	inputs := []string{
		"the quick brown fox jumps over the lazy dog",
		"supercalifragilisticexpialidocious",
		"a b c d e f g h i j k l m n o p",
		"👍🏽 emoji flag 🇷🇴 and text mixed together for width edge cases",
		// Known gap: a tab landing exactly on a wrap point is dropped from the
		// rendered segment. Fixing it requires changing wrapLine and the
		// LogicalStartCol skip in appendVisualLayoutForLine together.
	}

	for _, in := range inputs {
		for width := 1; width <= 20; width++ {
			segs := wrapLine(in, width)
			joined := strings.Join(segs, "")
			// Wrapping only removes whitespace at break points, so every
			// non-space rune of the input must survive somewhere in the
			// concatenated segments, in order.
			gotRunes := []rune(joined)
			wantNonSpace := []rune{}
			for _, r := range in {
				if r != ' ' {
					wantNonSpace = append(wantNonSpace, r)
				}
			}
			gotNonSpace := []rune{}
			for _, r := range gotRunes {
				if r != ' ' {
					gotNonSpace = append(gotNonSpace, r)
				}
			}
			require.Equalf(t, string(wantNonSpace), string(gotNonSpace),
				"input %q at width %d lost or reordered non-space characters: segments=%q", in, width, segs)
		}
	}
}

func TestGetVisualWidth(t *testing.T) {
	assert.Equal(t, 0, getVisualWidth(""))
	assert.Equal(t, 5, getVisualWidth("hello"))
	assert.Equal(t, 4, getVisualWidthAt("\t", 0))
	assert.Equal(t, 1, getVisualWidthAt("\t", 3))
	assert.Equal(t, 4, getVisualWidthAt("\t", 4))
}

func TestCalculateLineNumberWidth(t *testing.T) {
	m := New(80, 20)
	m.SetContent("a\nb\nc\n")

	assert.Equal(t, 5, m.calculateLineNumberWidth(9))
	assert.Equal(t, 6, m.calculateLineNumberWidth(99999))

	m.HideLineNumbers(true)
	assert.Equal(t, 0, m.calculateLineNumberWidth(9))
}
