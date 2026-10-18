package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalModePendingState checks what the mode reports part way through a
// command: the mapping layer and undo boundaries read these.
func TestNormalModePendingState(t *testing.T) {
	tests := []struct {
		name            string
		keys            string
		operatorPending bool
		awaitingLiteral bool
		idle            bool
	}{
		{name: "nothing typed", keys: "", idle: true},
		{name: "count alone", keys: "2", idle: true},
		{name: "completed command", keys: "dw", idle: true},
		// The count before an operator is set aside until the motion; it
		// must not outlive the command.
		{name: "completed command with counts", keys: "2d3w", idle: true},
		{name: "completed operator gg with count", keys: "2dgg", idle: true},
		{name: "completed operator ge with count", keys: "2dge", idle: true},
		{name: "completed char search with count", keys: "2dft", idle: true},
		{name: "completed text object with count", keys: "2diw", idle: true},
		{name: "completed case operator with count", keys: "2gUw", idle: true},
		{name: "completed case operator gg with count", keys: "2gUgg", idle: true},
		{name: "completed case text object with count", keys: "2gUiw", idle: true},
		{name: "operator", keys: "d", operatorPending: true},
		{name: "operator with counts", keys: "2d3", operatorPending: true},
		{name: "operator and g", keys: "dg", operatorPending: true},
		{name: "operator and text object modifier", keys: "di", operatorPending: true},
		{name: "operator and char search", keys: "df", operatorPending: true, awaitingLiteral: true},
		{name: "case operator", keys: "gU", operatorPending: true},
		{name: "case operator and g", keys: "gUg", operatorPending: true},
		{name: "case operator and modifier", keys: "gUi", operatorPending: true},
		{name: "g prefix", keys: "g"},
		{name: "z prefix", keys: "z"},
		{name: "replace", keys: "r", awaitingLiteral: true},
		{name: "char search", keys: "t", awaitingLiteral: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor("one two three")
			keys(e, []rune(tt.keys)...)

			m, ok := e.GetMode().(*normalMode)
			require.True(t, ok)
			assert.Equal(t, tt.operatorPending, m.OperatorPending(), "OperatorPending")
			assert.Equal(t, tt.awaitingLiteral, m.AwaitingLiteral(), "AwaitingLiteral")
			assert.Equal(t, tt.idle, m.Idle(), "Idle")
		})
	}
}

func TestCaseOperatorGEscapeCancels(t *testing.T) {
	e := newTestEditor("one two\nthree")
	keys(e, 'j', '2', 'g', 'U', 'g')
	escape(e)

	m, ok := e.GetMode().(*normalMode)
	require.True(t, ok)
	assert.True(t, m.Idle())
	assert.Nil(t, e.GetState().PendingCount)

	keys(e, 'g', 'g')
	assert.Equal(t, "one two\nthree", content(e))
	assert.Equal(t, Position{0, 0}, cursorPos(e))
}
