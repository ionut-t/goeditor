package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testClipboard struct{ content string }

func (c *testClipboard) Write(text string) error { c.content = text; return nil }
func (c *testClipboard) Read() (string, error)   { return c.content, nil }

// visualEditCase is keys typed on content, and the text, cursor and preferred
// column they leave in normal mode (J, gJ and r tests).
type visualEditCase struct {
	content, keys string
	want          string
	cursor        Position
	preferred     int
}

func runVisualEditCases(t *testing.T, tests []visualEditCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, tt.preferred, e.GetBuffer().GetCursor().Preferred, "preferred column")
			assert.Equal(t, NormalMode, e.GetMode().Name())
		})
	}
}

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
		_ = e.HandleKey(KeyEvent{Rune: r})
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

// ctrlKey sends Ctrl+<r>, e.g. ctrlKey(e, 'r') for redo.
func ctrlKey(e Editor, r rune) { _ = e.HandleKey(Ctrl(r)) }

func escape(e Editor)    { _ = e.HandleKey(KeyEvent{Key: KeyEscape}) }
func backspace(e Editor) { _ = e.HandleKey(KeyEvent{Key: KeyBackspace}) }
func enter(e Editor)     { _ = e.HandleKey(KeyEvent{Key: KeyEnter}) }
func tab(e Editor)       { _ = e.HandleKey(KeyEvent{Key: KeyTab}) }

// countedObjectCase is keys typed on content and what they leave: the text,
// the mode ("n", "i", "v" or "V"), the cursor and, in visual modes, the
// anchor. reg is checked unless it is anyReg. The clipboard starts as "REG".
type countedObjectCase struct {
	content, keys string
	text          string
	mode          string
	cursor        Position
	anchor        Position // visual modes only
	reg           string
}

func runCountedObjectCases(t *testing.T, tests []countedObjectCase) {
	t.Helper()
	modes := map[Mode]string{NormalMode: "n", InsertMode: "i", VisualMode: "v", VisualLineMode: "V"}
	for _, tt := range tests {
		t.Run(tt.content+" "+tt.keys, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e), "text")
			mode := modes[e.GetMode().Name()]
			assert.Equal(t, tt.mode, mode, "mode")
			assert.Equal(t, tt.cursor, cursorPos(e), "cursor")
			assert.Equal(t, tt.cursor.Col, e.GetBuffer().GetCursor().Preferred, "preferred column")
			if mode == "v" || mode == "V" {
				assert.Equal(t, tt.anchor, e.GetState().VisualStart, "anchor")
			}
			if tt.reg != anyReg {
				assert.Equal(t, tt.reg, cb.content, "register")
			}
		})
	}
}
