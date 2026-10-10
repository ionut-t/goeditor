package goeditor

import (
	"fmt"
	"strings"
	"testing"

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
