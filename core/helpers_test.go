package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testClipboard struct{ content string }

func (c *testClipboard) Write(text string) error { c.content = text; return nil }
func (c *testClipboard) Read() (string, error)   { return c.content, nil }

func newTestEditor(content string) Editor {
	e := New(nil)
	e.SetContent([]byte(content))
	return e
}

func newTestEditorWithClipboard(content string) (Editor, *testClipboard) {
	cb := &testClipboard{}
	e := New(cb)
	e.SetContent([]byte(content))
	return e, cb
}

func keys(e Editor, runes ...rune) {
	for _, r := range runes {
		e.HandleKey(KeyEvent{Rune: r})
	}
}

func content(e Editor) string {
	return e.GetBuffer().GetCurrentContent()
}

func cursorPos(e Editor) Position {
	return e.GetBuffer().GetCursor().Position
}

func assertInsertMode(t *testing.T, e Editor) {
	t.Helper()
	assert.True(t, e.IsInsertMode(), "expected editor to be in insert mode")
}

// setWidth configures the editor's available text width, which is required for
// correct column-preservation behaviour when moving up/down.
func setWidth(e Editor, width int) {
	s := e.GetState()
	s.AvailableWidth = width
	e.SetState(s)
}
