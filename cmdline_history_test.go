package goeditor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
)

// TestCommandLineHistoryThroughModel checks the history API a host saves and
// restores the history with, and that the Bubble Tea keys walk it.
func TestCommandLineHistoryThroughModel(t *testing.T) {
	t.Run("restored entries are walked with the arrow keys", func(t *testing.T) {
		m := newBottomRowsModel(40, 10, nil)
		m.SetCommandLineHistory(core.SearchHistory, []string{"line 7", "line 3"})
		m.SetCommandLineHistory(core.CommandHistory, []string{"set rnu", "set nornu"})

		m = drainSignals(typeRune('/')(m))
		m = pressSpecial(tea.KeyUp)(m)
		m = pressSpecial(tea.KeyUp)(m)
		assert.Equal(t, "/line 7", m.editor.GetState().CommandLine)
		m = drainSignals(pressSpecial(tea.KeyEnter)(m))
		assert.Equal(t, core.Position{Row: 6, Col: 0}, m.GetCursorPosition())

		m = drainSignals(typeText(":s")(m))
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
		assert.Equal(t, ":set nornu", m.editor.GetState().CommandLine, "Shift+Up takes every entry")
		m = pressSpecial(tea.KeyDown)(m)
		assert.Equal(t, ":s", m.editor.GetState().CommandLine)
	})

	t.Run("typed lines are saved", func(t *testing.T) {
		m := newBottomRowsModel(40, 10, nil)
		m = drainSignals(typeText(":set rnu")(m))
		m = drainSignals(pressSpecial(tea.KeyEnter)(m))
		m = drainSignals(typeText("/line 9")(m))
		m = drainSignals(pressSpecial(tea.KeyEnter)(m))
		assert.Equal(t, []string{"set rnu"}, m.CommandLineHistory(core.CommandHistory))
		assert.Equal(t, []string{"line 9"}, m.CommandLineHistory(core.SearchHistory))
	})

	t.Run("size", func(t *testing.T) {
		m := newBottomRowsModel(40, 10, nil)
		m.SetCommandLineHistorySize(1)
		m.SetCommandLineHistory(core.SearchHistory, []string{"a", "b"})
		assert.Equal(t, []string{"b"}, m.CommandLineHistory(core.SearchHistory))
	})
}
