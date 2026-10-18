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
		// Cancelled commands (\x1b = Esc) must not leave anything behind either.
		{name: "operator escape with count", keys: "2d\x1b", idle: true},
		{name: "operator count escape", keys: "2d3\x1b", idle: true},
		{name: "operator invalid motion with count", keys: "2dq", idle: true},
		{name: "operator g invalid key with count", keys: "2dgx", idle: true},
		{name: "operator g escape with count", keys: "2dg\x1b", idle: true},
		{name: "operator modifier escape with count", keys: "2di\x1b", idle: true},
		{name: "operator invalid text object with count", keys: "2diz", idle: true},
		{name: "operator char search escape with count", keys: "2df\x1b", idle: true},
		{name: "case operator escape with count", keys: "2gU\x1b", idle: true},
		{name: "case operator invalid motion with count", keys: "2gUq", idle: true},
		{name: "case operator g invalid key with count", keys: "2gUgx", idle: true},
		{name: "case operator modifier escape with count", keys: "2gUi\x1b", idle: true},
		{name: "replace escape", keys: "r\x1b", idle: true},
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
			for _, r := range tt.keys {
				if r == '\x1b' {
					escape(e)
				} else {
					keys(e, r)
				}
			}

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

// newReadOnlyEditor returns an editor with editing disabled after a change to
// undo (u, U), a change to redo (Ctrl+R) and a yank to paste (p, P).
func newReadOnlyEditor() Editor {
	e, _ := newTestEditorWithClipboard("one two three four five")
	keys(e, 'x', 'x', 'u', 'y', 'w')
	e.DisableInsertMode(true)
	return e
}

// readOnlyKeys is every printable key and every Ctrl+letter.
func readOnlyKeys() []KeyEvent {
	var all []KeyEvent
	for r := rune(' ' + 1); r < 0x7f; r++ {
		all = append(all, KeyEvent{Rune: r})
	}
	for r := 'a'; r <= 'z'; r++ {
		all = append(all, Ctrl(r))
	}
	return all
}

// TestEditingDisabledDropsCount checks that a refused edit drops the count
// typed before it, so the next command doesn't pick it up (3x then w).
func TestEditingDisabledDropsCount(t *testing.T) {
	refused := []KeyEvent{Ctrl('r')}
	for _, r := range "IaAoOxXDr~sSJCdcpPuU" {
		refused = append(refused, KeyEvent{Rune: r})
	}
	for _, k := range refused {
		t.Run(k.String(), func(t *testing.T) {
			e := newReadOnlyEditor()
			before := content(e)

			keys(e, '3')
			_ = e.HandleKey(k)
			assert.Nil(t, e.GetState().PendingCount)
			assert.False(t, e.IsInsertMode())

			keys(e, 'w')
			assert.Equal(t, Position{Row: 0, Col: 3}, cursorPos(e))
			assert.Equal(t, before, content(e))
		})
	}
}

// TestEditingDisabledKeepsText sends every key with editing disabled, so a
// new editing command that isn't in editsText fails here.
func TestEditingDisabledKeepsText(t *testing.T) {
	for _, k := range readOnlyKeys() {
		t.Run(k.String(), func(t *testing.T) {
			e := newReadOnlyEditor()
			before := content(e)

			keys(e, '3')
			_ = e.HandleKey(k)
			keys(e, 'w') // completes f/t/r/g/z/d/c/y prefixes
			assert.Equal(t, before, content(e))
			assert.False(t, e.IsInsertMode())
		})
	}
}
