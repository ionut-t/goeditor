package goeditor

import (
	"testing"

	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYankFlashClear checks the timer that ends the yank highlight: it clears
// what a yank left selected, and leaves alone a visual mode entered since.
func TestYankFlashClear(t *testing.T) {
	newModel := func() Model {
		m := New(40, 10)
		m.Focus()
		m.SetContent("abcdef")
		return m
	}

	t.Run("clears the highlight a visual yank leaves", func(t *testing.T) {
		m := typeText("vly")(newModel())
		require.True(t, m.IsNormalMode())
		require.Equal(t, core.SelectionCharacter, m.GetEditor().GetState().YankSelection)

		m, _ = m.Update(clearYankMsg{})
		state := m.GetEditor().GetState()
		assert.Equal(t, core.Position{Row: -1, Col: -1}, state.VisualStart)
		assert.Equal(t, core.SelectionNone, state.YankSelection)
		assert.True(t, m.IsNormalMode())
	})

	t.Run("keeps a visual mode entered after the yank", func(t *testing.T) {
		m := typeText("vlyv")(newModel())
		require.True(t, m.IsVisualMode())

		m, _ = m.Update(clearYankMsg{})
		assert.True(t, m.IsVisualMode())
		assert.Equal(t, core.Position{Row: 0, Col: 0}, m.GetEditor().GetState().VisualStart)
	})
}
