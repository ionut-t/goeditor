package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPending(t *testing.T) {
	tests := []struct {
		name    string
		keys    string
		pending bool
	}{
		{"nothing typed", "", false},
		{"count", "3", true},
		{"operator", "d", true},
		{"operator with count", "2d", true},
		{"operator and text object modifier", "di", true},
		{"g prefix", "g", true},
		{"z prefix", "z", true},
		{"char search", "f", true},
		{"replace char", "r", true},
		{"completed dd", "dd", false},
		{"completed motion", "3j", false},
		{"completed char search", "fl", false},
		{"cancelled operator", "d<Esc>", false},
		{"visual mode", "v", false},
		{"visual text object modifier", "vi", true},
		{"visual g prefix", "Vg", true},
		{"visual count", "v2", true},
		{"insert mode", "i", false},
		{"command line", ":", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor("hello\nworld\nagain\nmore")
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.pending, e.CommandPending())
		})
	}
}

func TestCommandPendingWhileMappingHeld(t *testing.T) {
	e := newTestEditor("hello world")
	require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
	require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

	keys(e, 'Q')
	assert.True(t, e.CommandPending(), "Q is held: QQ could still match")

	fireMapTimeout(t, e)
	assert.False(t, e.CommandPending(), "the timeout ran Q")
}
