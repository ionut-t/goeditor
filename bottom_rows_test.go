package goeditor

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBottomRowsModel(width, height int, setup func(*Model)) Model {
	var b strings.Builder
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}

	m := New(width, height)
	m.Focus()
	m.SetContent(b.String())
	if setup != nil {
		setup(&m)
	}
	return m
}

func viewLines(m Model) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

// TestBottomRowsMatchWhatIsRendered checks that every combination of status
// line and command line settings renders exactly the requested height, with
// the viewport filling whatever the bottom rows don't use.
func TestBottomRowsMatchWhatIsRendered(t *testing.T) {
	const width, height = 40, 10

	tests := []struct {
		name  string
		setup func(*Model)
		rows  int
	}{
		{"default", nil, 2},
		{"status line hidden", func(m *Model) { m.HideStatusLine(true) }, 1},
		{"vim mode disabled", func(m *Model) { m.DisableVimMode(true) }, 0},
		{"command and search disabled", func(m *Model) {
			m.DisableCommandMode(true)
			m.DisableSearchMode(true)
		}, 1},
		{"only command disabled keeps the row for search", func(m *Model) { m.DisableCommandMode(true) }, 2},
		{"everything hidden", func(m *Model) {
			m.HideStatusLine(true)
			m.DisableCommandMode(true)
			m.DisableSearchMode(true)
		}, 0},
		{"command line height 0", func(m *Model) { m.SetCommandLineHeight(0) }, 1},
		{"command line height 0 without status line", func(m *Model) {
			m.SetCommandLineHeight(0)
			m.HideStatusLine(true)
		}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newBottomRowsModel(width, height, tt.setup)

			assert.Equal(t, height-tt.rows, m.viewport.Height())
			assert.Equal(t, height-tt.rows, m.editor.GetState().ViewportHeight)

			lines := viewLines(m)
			require.Len(t, lines, height)
			assert.Containsf(t, lines[height-tt.rows-1], fmt.Sprintf("line %d", height-tt.rows),
				"last viewport row should hold content: %q", lines)
		})
	}
}

// TestSettingsChangeReleasesRowsImmediately covers toggling the bottom rows
// after the model is sized: the viewport has to grow or shrink straight away.
func TestSettingsChangeReleasesRowsImmediately(t *testing.T) {
	m := newBottomRowsModel(40, 10, nil)
	require.Equal(t, 8, m.viewport.Height())

	m.HideStatusLine(true)
	assert.Equal(t, 9, m.viewport.Height())
	assert.Len(t, viewLines(m), 10)

	m.HideStatusLine(false)
	assert.Equal(t, 8, m.viewport.Height())
	assert.Len(t, viewLines(m), 10)
}

func TestCommandLineSharesStatusRow(t *testing.T) {
	m := newBottomRowsModel(40, 10, func(m *Model) { m.SetCommandLineHeight(0) })

	last := func(m Model) string {
		lines := viewLines(m)
		require.Len(t, lines, 10)
		return lines[9]
	}

	assert.Contains(t, last(m), "NORMAL")

	// A pending count stays out of the way of the status line.
	m = typeRune('3')(m)
	assert.Contains(t, last(m), "NORMAL")
	m = escKey(m)

	m = typeText(":wq")(m)
	assert.Contains(t, last(m), ":wq")
	assert.NotContains(t, last(m), "COMMAND")

	m = escKey(m)
	assert.Contains(t, last(m), "NORMAL")

	m = typeRune('/')(m)
	assert.True(t, strings.HasPrefix(last(m), "/"), "search prompt should take the row: %q", last(m))
	assert.NotContains(t, last(m), "NORMAL")

	m = escKey(m)
	assert.Contains(t, last(m), "NORMAL")
}

func TestStatusRowShowsMessagesWithoutCommandLine(t *testing.T) {
	m := newBottomRowsModel(40, 10, func(m *Model) {
		m.DisableCommandMode(true)
		m.DisableSearchMode(true)
	})

	m.message = "saved"
	lines := viewLines(m)
	require.Len(t, lines, 10)
	assert.Contains(t, lines[9], "saved")

	m.message = ""
	assert.Contains(t, viewLines(m)[9], "NORMAL")
}

// TestSearchRowFillsWidthWithCommandLineBackground checks that the search
// prompt row looks like the command-mode row: padded to the full width, and
// with the command line background behind the prompt and the query too, not
// just up to the first SGR reset.
func TestSearchRowFillsWidthWithCommandLineBackground(t *testing.T) {
	m := newBottomRowsModel(40, 10, nil)
	m = typeRune('/')(m)
	require.True(t, m.editor.IsSearchMode())
	// Focus and keys reach the input through the editor's update channel,
	// which these synchronous helpers don't drain.
	m.searchInput.Focus()
	m.searchInput.SetValue("li")

	row, active := m.getCommandLine(m.editor.GetState())
	assert.True(t, active)
	assert.Equal(t, 40, ansi.StringWidth(row), "search row should span the full width: %q", row)

	bg := m.theme.CommandLineStyle.GetBackground()
	r, g, b, _ := bg.RGBA()
	bgSGR := fmt.Sprintf("48;2;%d;%d;%d", r>>8, g>>8, b>>8)
	// Each segment ends in a reset, so each one with visible text must set
	// the background again. The cursor is reverse video and needs none.
	for _, seg := range strings.Split(row, "\x1b[m") {
		visible := ansi.Strip(seg)
		if visible == "" || strings.Contains(seg, "\x1b[7") {
			continue
		}
		assert.Contains(t, seg, bgSGR, "segment %q has no command line background", visible)
	}
}

// TestSearchRowPadsWithSearchTextBackground checks that a theme giving the
// search text a background of its own gets it across the whole row, not just
// behind the query.
func TestSearchRowPadsWithSearchTextBackground(t *testing.T) {
	m := newBottomRowsModel(40, 10, nil)
	theme := DefaultTheme(true)
	theme.SearchInputTextStyle = theme.SearchInputTextStyle.Background(lipgloss.Color("#123456"))
	m.WithTheme(theme)

	m = typeRune('/')(m)
	require.True(t, m.editor.IsSearchMode())
	m.searchInput.Focus()
	m.searchInput.SetValue("li")

	row, _ := m.getCommandLine(m.editor.GetState())
	assert.Equal(t, 40, ansi.StringWidth(row))

	segs := strings.Split(row, "\x1b[m")
	var padding string
	for i := len(segs) - 1; i >= 0; i-- {
		if ansi.Strip(segs[i]) != "" {
			padding = segs[i]
			break
		}
	}
	require.Empty(t, strings.TrimSpace(ansi.Strip(padding)), "last segment should be padding: %q", padding)
	assert.Contains(t, padding, "48;2;18;52;86", "padding should use the search text background: %q", padding)
	assert.NotContains(t, padding, "48;2;30;30;46")
}
