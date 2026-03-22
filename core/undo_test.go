package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUndoBasic tests 'u' — undo the last change.
func TestUndoBasic(t *testing.T) {
	t.Run("undo dd restores content", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		assert.Equal(t, "", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("undo dw restores word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'w')
		assert.Equal(t, "world", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("undo x restores character", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("undo at oldest change does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'u') // nothing to undo
		assert.Equal(t, "hello", content(e))
	})
}

// TestUndoCursorPosition tests that undo restores the cursor to its pre-change position.
func TestUndoCursorPosition(t *testing.T) {
	t.Run("undo dd from second line restores cursor to that row", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'd', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		keys(e, 'u')
		assert.Equal(t, "first\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("undo dw restores cursor to delete start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'd', 'w') // move to col 6, delete "world"
		assert.Equal(t, "hello ", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		keys(e, 'u')
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestUndoMultipleSteps tests undoing several changes in sequence.
func TestUndoMultipleSteps(t *testing.T) {
	t.Run("undo two dd operations step by step", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'd', 'd') // delete "one"
		assert.Equal(t, "two\nthree", content(e))
		keys(e, 'd', 'd') // delete "two"
		assert.Equal(t, "three", content(e))

		keys(e, 'u') // undo second dd
		assert.Equal(t, "two\nthree", content(e))

		keys(e, 'u') // undo first dd
		assert.Equal(t, "one\ntwo\nthree", content(e))
	})

	t.Run("undo count: 2dd then u twice restores fully", func(t *testing.T) {
		e := newTestEditor("a\nb\nc")
		keys(e, '2', 'd', 'd')
		assert.Equal(t, "c", content(e))
		keys(e, 'u')
		assert.Equal(t, "a\nb\nc", content(e))
	})
}

// TestRedoBasic tests 'U' — redo the last undone change.
func TestRedoBasic(t *testing.T) {
	t.Run("redo after undo reapplies dd", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		keys(e, 'U') // redo
		assert.Equal(t, "", content(e))
	})

	t.Run("redo at newest change does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		keys(e, 'U') // nothing to redo yet
		assert.Equal(t, "", content(e))
	})

	t.Run("redo restores cursor to post-change position", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'd', 'd') // delete "second", cursor goes to row 0
		keys(e, 'u')            // undo: "second" restored, cursor at row 1
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		keys(e, 'U') // redo: "second" deleted again, cursor at row 0
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestUndoTruncatesRedo verifies that making a new change after undo discards the redo history.
func TestUndoTruncatesRedo(t *testing.T) {
	t.Run("new edit after undo prevents redo", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'd', 'd') // delete "one" → "two\nthree"
		keys(e, 'u')       // undo → "one\ntwo\nthree"
		keys(e, 'x')       // new edit → "ne\ntwo\nthree"
		keys(e, 'U')       // redo should not restore "two\nthree"
		assert.Equal(t, "ne\ntwo\nthree", content(e))
	})
}
