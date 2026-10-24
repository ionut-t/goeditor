package goeditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGxSendsOpenMsg checks that gx reaches the host as an OpenMsg.
func TestGxSendsOpenMsg(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	m.Focus()
	m.SetContent("see https://example.com/a.")
	m = typeText("w")(m)
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		<-m.GetEditor().GetUpdateSignalChan()
	}

	m = typeText("gx")(m)

	var msg any
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		if msg = m.listenForEditorUpdate()(); msg != (commandMsg{}) {
			break
		}
	}
	assert.Equal(t, OpenMsg{Target: "https://example.com/a"}, msg)
}

func TestOpenWithSystemEmptyTarget(t *testing.T) {
	assert.ErrorIs(t, OpenWithSystem(""), ErrNothingToOpen)
}
