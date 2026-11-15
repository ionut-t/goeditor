package goeditor

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startListener runs listen in a goroutine, as Bubble Tea does, and waits
// until it is blocked on the signal channel.
func startListener(t *testing.T, m Model, listen tea.Cmd) <-chan tea.Msg {
	t.Helper()
	require.NotNil(t, listen)
	got := make(chan tea.Msg, 1)
	go func() { got <- listen() }()
	require.Eventually(t, m.listening.Load, time.Second, time.Millisecond,
		"listener never started waiting")
	return got
}

func receive(t *testing.T, got <-chan tea.Msg) tea.Msg {
	t.Helper()
	select {
	case msg := <-got:
		return msg
	case <-time.After(time.Second):
		t.Fatal("listener got no signal")
		return nil
	}
}

// TestOneSignalListenerOutstanding checks that a listener run while another
// is waiting returns at once, so Update, which re-arms on every message,
// never leaves goroutines blocked on the channel.
func TestOneSignalListenerOutstanding(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	got := startListener(t, m, m.Init())

	for range 3 {
		assert.Nil(t, m.listenForEditorUpdate()(), "second listener waited")
	}

	// Once the waiting listener receives a signal, the next one may wait.
	m.GetEditor().DispatchSignal(core.CommandSignal{})
	assert.Equal(t, commandMsg{}, receive(t, got))
	got = startListener(t, m, m.listenForEditorUpdate())
	m.GetEditor().DispatchSignal(core.CommandSignal{})
	assert.Equal(t, commandMsg{}, receive(t, got))
}

// TestCopiesShareTheListener checks the guard survives Model being copied by
// value, as Bubble Tea does on every Update.
func TestCopiesShareTheListener(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	got := startListener(t, m, m.Init())
	cp := m
	assert.Nil(t, cp.listenForEditorUpdate()())

	m.GetEditor().DispatchSignal(core.CommandSignal{})
	assert.Equal(t, commandMsg{}, receive(t, got))
}

// TestDroppedListenerKeepsSignals checks that a host dropping the command
// Update returns (m.editor, _ = m.editor.Update(msg)) doesn't stop later
// listeners from delivering signals.
func TestDroppedListenerKeepsSignals(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	_ = m.Init()
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})

	m.GetEditor().DispatchSignal(core.CommandSignal{})
	assert.Equal(t, commandMsg{}, m.listenForEditorUpdate()())
}
