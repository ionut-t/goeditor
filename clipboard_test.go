package goeditor

import (
	"testing"

	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithClipboardNil checks that WithClipboard(nil) disables the clipboard:
// yank and paste return an error instead of panicking.
func TestWithClipboardNil(t *testing.T) {
	newModel := func() Model {
		m := New(40, 10, WithClipboard(nil))
		m.Focus()
		m.SetContent("hello\nworld")
		return m
	}

	for _, keys := range []string{"yy", "p", "P", "vy", "$vp", "Vp"} {
		t.Run(keys, func(t *testing.T) {
			m := newModel()
			e := m.GetEditor()
			var err *core.EditorError
			for _, r := range keys {
				err = e.HandleKey(core.KeyEvent{Rune: r})
			}
			require.NotNil(t, err)
		})
	}

	t.Run("through the model", func(t *testing.T) {
		assert.NotPanics(t, func() { typeText("yyp$vpVp")(newModel()) })
	})
}
