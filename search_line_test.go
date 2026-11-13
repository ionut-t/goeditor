package goeditor

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drainSignals delivers the editor's pending signals to Update, the way Bubble
// Tea would once their listener command has run.
func drainSignals(m Model) Model {
	for len(m.editor.GetUpdateSignalChan()) > 0 {
		if msg := m.listenForEditorUpdate()(); msg != nil {
			m, _ = m.Update(msg)
		}
	}
	return m
}

// TestSearchKeysBeforeSignals checks that a search runs with every key typed,
// whether the keys arrive before or after the editor's signals are delivered
// (open bug 60: keys typed before the search input was focused were dropped).
func TestSearchKeysBeforeSignals(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(Model) Model
	}{
		{"keys before signals", func(m Model) Model {
			m = typeText("/line 7")(m)
			m = pressSpecial(tea.KeyEnter)(m)
			return drainSignals(m)
		}},
		{"signals between keys", func(m Model) Model {
			m = drainSignals(typeRune('/')(m))
			m = drainSignals(typeText("line 7")(m))
			m = pressSpecial(tea.KeyEnter)(m)
			return drainSignals(m)
		}},
		{"mapping", func(m Model) Model {
			require.NoError(t, m.Map(core.MapNormal, "s", "/line 7<CR>", true))
			m = typeRune('s')(m)
			return drainSignals(m)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.run(newBottomRowsModel(40, 10, nil))
			assert.False(t, m.editor.IsSearchMode())
			assert.Equal(t, "line 7", m.editor.GetState().SearchQuery.Pattern)
			assert.Equal(t, core.Position{Row: 6, Col: 0}, m.GetCursorPosition())
		})
	}
}

// sgrFg is the SGR parameters that set c as the foreground.
func sgrFg(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8)
}

// TestSearchLineStyles checks that the search line is drawn with the theme's
// SearchInput styles: the prompt, the text, and the cursor cell in reverse
// video.
func TestSearchLineStyles(t *testing.T) {
	for _, prompt := range []rune{'/', '?'} {
		t.Run(string(prompt), func(t *testing.T) {
			m := newBottomRowsModel(40, 10, nil)
			m = typeText(string(prompt) + "lin")(m)
			m = pressSpecial(tea.KeyLeft)(m)

			row, active := m.getCommandLine(m.editor.GetState())
			assert.True(t, active)
			assert.Equal(t, 40, ansi.StringWidth(row))
			assert.Equal(t, string(prompt)+"lin", strings.TrimRight(ansi.Strip(row), " "))
			assert.Equal(t, "n", cursorCell(row))

			segs := strings.Split(row, "\x1b[m")
			require.GreaterOrEqual(t, len(segs), 3)
			assert.Equal(t, string(prompt), ansi.Strip(segs[0]))
			assert.Contains(t, segs[0], sgrFg(m.theme.SearchInputPromptStyle.GetForeground()), "prompt style")
			assert.Equal(t, "li", ansi.Strip(segs[1]))
			assert.Contains(t, segs[1], sgrFg(m.theme.SearchInputTextStyle.GetForeground()), "text style")

			// Reverse video swaps the cursor style's colours, so its
			// foreground becomes the cell's background.
			cursorFg := m.theme.SearchInputCursorStyle.GetForeground()
			r, g, b, _ := cursorFg.RGBA()
			assert.Contains(t, segs[2], "\x1b[7;")
			assert.Contains(t, segs[2], fmt.Sprintf("%d;%d;%d", r>>8, g>>8, b>>8), "cursor style")
		})
	}
}

// TestSearchEchoOnCommandLine checks that the command line row keeps the
// search after <CR>, gives way to pending keys and comes back after them.
func TestSearchEchoOnCommandLine(t *testing.T) {
	m := newBottomRowsModel(40, 10, nil)
	row := func() string {
		r, _ := m.getCommandLine(m.editor.GetState())
		return strings.TrimRight(ansi.Strip(r), " ")
	}

	m = typeText("/line 7")(m)
	m = pressSpecial(tea.KeyEnter)(m)
	assert.Equal(t, "/line 7", row())

	m = typeText("j2")(m)
	assert.Equal(t, "2", row(), "pending count over the search")
	m = pressSpecial(tea.KeyEscape)(m)
	assert.Equal(t, "/line 7", row())

	m = typeRune('i')(m)
	assert.Equal(t, "", row(), "a mode change clears it")
}
