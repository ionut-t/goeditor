package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReplaceChar tests 'r{char}' — replace character under cursor without entering insert mode.
func TestReplaceChar(t *testing.T) {
	t.Run("replaces char under cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r', 'X')
		assert.Equal(t, "Xello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("replaces char at mid-line position", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'r', 'Z') // cursor at col 2; replace 'l' with 'Z'
		assert.Equal(t, "heZlo", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("Escape after r cancels replace, stays in normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r')
		escape(e) // cancel replace
		assert.Equal(t, "hello", content(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("on empty line does nothing", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'r', 'X') // move to blank line; r is a no-op
		assert.Equal(t, "hello\n\nworld", content(e))
	})

	t.Run("count replaces that many chars, cursor on the last", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', 'r', 'X')
		assert.Equal(t, "XXXlo", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("count past the line end replaces nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', '4', 'r', 'X')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("count too large to fit replaces nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		typeKeys(t, e, "l99999999999999999999rX")
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("undo restores original char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r', 'X')
		assert.Equal(t, "Xello", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})
}
