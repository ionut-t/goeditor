package goeditor

import (
	"testing"

	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOneSignalListenerOutstanding checks that re-arming the signal listener
// while one is already waiting returns no command, so Update — which re-arms
// on every message — never leaves goroutines blocked on the channel.
func TestOneSignalListenerOutstanding(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))

	listen := m.Init()
	require.NotNil(t, listen)
	for range 3 {
		assert.Nil(t, m.listenForEditorUpdate(), "second listener armed while one waits")
	}

	// Once the waiting listener receives a signal, the next one may arm.
	m.GetEditor().DispatchSignal(core.CommandSignal{})
	assert.Equal(t, commandMsg{}, listen())
	assert.NotNil(t, m.listenForEditorUpdate())
}

// TestCopiesShareTheListener checks the guard survives Model being copied by
// value, as Bubble Tea does on every Update.
func TestCopiesShareTheListener(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	require.NotNil(t, m.Init())
	cp := m
	assert.Nil(t, cp.listenForEditorUpdate())
}
