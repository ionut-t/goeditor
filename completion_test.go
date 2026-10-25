package goeditor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
)

// TestLateCompletionResponse checks that a completion response only opens the
// menu if the cursor and its line are still what the request was built from.
// The RequestID check alone lets through a response that arrives after the
// user typed but before the next request went out.
func TestLateCompletionResponse(t *testing.T) {
	ctrlSpace := func(m Model) Model {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Mod: tea.ModCtrl})
		return m
	}

	tests := []struct {
		name     string
		after    modelAction
		wantMenu bool
	}{
		{"nothing changed", func(m Model) Model { return m }, true},
		{"typed a character", typeText("x"), false},
		{"typed then deleted it", func(m Model) Model {
			return pressSpecial(tea.KeyBackspace)(typeText("x")(m))
		}, true},
		{"moved the cursor", pressSpecial(tea.KeyLeft), false},
		{"left insert mode", escKey, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(40, 10)
			m.Focus()
			m.SetContent("fo")
			m = typeText("A")(m)
			m = ctrlSpace(m)
			ctx := m.completionContext

			m = tt.after(m)
			m, _ = m.Update(CompletionResponseMsg{
				Completions: []core.Completion{{Text: "foo", Label: "foo"}},
				Context:     ctx,
			})

			assert.Equal(t, tt.wantMenu, m.completionMenuVisible)
		})
	}
}
