package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUndoLineSingleLineEdits covers Vim's 'U' for changes confined to one line:
// all of them are undone at once, however many keystrokes they took.
func TestUndoLineSingleLineEdits(t *testing.T) {
	t.Run("restores a line after several x deletions", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'x', 'x', 'x')
		assert.Equal(t, "lo world", content(e))
		keys(e, 'U')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("restores a line after an insert-mode run", func(t *testing.T) {
		e := newTestEditor("abc")
		keys(e, 'i')
		assertInsertMode(t, e)
		keys(e, 'X', 'Y', 'Z')
		escape(e)
		assert.Equal(t, "XYZabc", content(e))
		keys(e, 'U')
		assert.Equal(t, "abc", content(e))
	})

	t.Run("restores a line after cw", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'w')
		keys(e, 'b', 'y', 'e')
		escape(e)
		assert.Equal(t, "bye world", content(e))
		keys(e, 'U')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("only affects the last changed line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'x')           // "ne\ntwo\nthree"
		keys(e, 'j', 'x')      // "ne\nwo\nthree"
		keys(e, 'j', 'x', 'x') // "ne\nwo\nree"
		assert.Equal(t, "ne\nwo\nree", content(e))

		keys(e, 'U') // restores only row 2
		assert.Equal(t, "ne\nwo\nthree", content(e))
	})

	t.Run("errors when nothing has changed", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'U')
		assert.Equal(t, "hello", content(e))
	})
}

// TestUndoLineLineCountChanges covers the operations that add or remove lines.
// Snapshot history captures these the same way as in-line edits, so 'U' restores
// a deleted line and removes an inserted one.
func TestUndoLineLineCountChanges(t *testing.T) {
	t.Run("restores a line deleted with dd", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'd', 'd')
		assert.Equal(t, "one\nthree", content(e))
		keys(e, 'U')
		assert.Equal(t, "one\ntwo\nthree", content(e))
	})

	t.Run("removes a line opened with o", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'o')
		keys(e, 'n', 'e', 'w')
		escape(e)
		assert.Equal(t, "one\nnew\ntwo", content(e))
		keys(e, 'U')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("undoes a join", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'J')
		assert.Equal(t, "one two", content(e))
		keys(e, 'U')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("restores multiple lines deleted with a count", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\nfour")
		keys(e, 'j', '2', 'd', 'd')
		assert.Equal(t, "one\nfour", content(e))
		keys(e, 'U')
		assert.Equal(t, "one\ntwo\nthree\nfour", content(e))
	})
}

// TestUndoLineToggle verifies that 'U' is itself a change: pressing it twice
// returns to the modified text, and 'u' undoes a 'U'.
func TestUndoLineToggle(t *testing.T) {
	t.Run("second U reapplies the changes", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x', 'x')
		assert.Equal(t, "llo", content(e))

		keys(e, 'U')
		assert.Equal(t, "hello", content(e))

		keys(e, 'U')
		assert.Equal(t, "llo", content(e))

		keys(e, 'U')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("u undoes a U", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x', 'x')
		keys(e, 'U')
		assert.Equal(t, "hello", content(e))

		keys(e, 'u')
		assert.Equal(t, "llo", content(e))
	})
}

// TestUndoLineRunBoundaries verifies when the tracked run restarts. Moving to a
// different line and changing it re-anchors 'U'; stepping through history
// abandons the run entirely, as there is no longer a trailing sequence of
// same-line changes to undo.
func TestUndoLineRunBoundaries(t *testing.T) {
	t.Run("changing another line re-anchors the run", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'x')      // row 0 → "ne\ntwo"
		keys(e, 'j', 'x') // row 1 → "ne\nwo"

		keys(e, 'U')
		assert.Equal(t, "ne\ntwo", content(e), "U should restore row 1 only")
	})

	t.Run("returning to a line starts a fresh run", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, 'x')      // row 0 → "ne\ntwo"
		keys(e, 'j', 'x') // row 1 → "ne\nwo"
		keys(e, 'k', 'x') // row 0 again → "e\nwo"
		assert.Equal(t, "e\nwo", content(e))

		keys(e, 'U') // only the latest row-0 change is undone
		assert.Equal(t, "ne\nwo", content(e))
	})

	t.Run("undo abandons the run", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x', 'x')
		keys(e, 'u') // steps back through history
		assert.Equal(t, "ello", content(e))

		keys(e, 'U') // no active run — nothing happens
		assert.Equal(t, "ello", content(e))
	})
}

// TestUndoLineDispatchesSignal verifies consumers are notified.
func TestUndoLineDispatchesSignal(t *testing.T) {
	e := newTestEditor("hello")
	keys(e, 'x')
	drainSignals(e)

	keys(e, 'U')
	assert.Equal(t, "hello", content(e))

	var got *UndoLineSignal
	for s := nextSignal(e); s != nil; s = nextSignal(e) {
		if sig, ok := s.(UndoLineSignal); ok {
			got = &sig
		}
	}
	if assert.NotNil(t, got, "expected an UndoLineSignal") {
		assert.Equal(t, "ello", got.Value(), "signal carries the content before the restore")
	}
}
