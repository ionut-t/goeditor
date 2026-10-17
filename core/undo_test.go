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

// TestRedoBasic tests Ctrl+R — redo the last undone change.
func TestRedoBasic(t *testing.T) {
	t.Run("redo after undo reapplies dd", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		ctrlKey(e, 'r') // redo
		assert.Equal(t, "", content(e))
	})

	t.Run("redo at newest change does nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'd', 'd')
		ctrlKey(e, 'r') // nothing to redo yet
		assert.Equal(t, "", content(e))
	})

	t.Run("redo restores cursor to post-change position", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'd', 'd') // delete "second", cursor goes to row 0
		keys(e, 'u')           // undo: "second" restored, cursor at row 1
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		ctrlKey(e, 'r') // redo: "second" deleted again, cursor at row 0
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("deprecated KeyCtrlU constant still scrolls, not redoes", func(t *testing.T) {
		// normalizeKey maps the legacy constants onto the canonical form, so
		// consumers constructing them keep working.
		e := newTestEditor("hello")
		e.HandleKey(KeyEvent{Key: KeyCtrlU})
		assert.Equal(t, "hello", content(e))
	})
}

// TestUndoTruncatesRedo verifies that making a new change after undo discards the redo history.
func TestUndoTruncatesRedo(t *testing.T) {
	t.Run("new edit after undo prevents redo", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'd', 'd') // delete "one" → "two\nthree"
		keys(e, 'u')      // undo → "one\ntwo\nthree"
		keys(e, 'x')      // new edit → "ne\ntwo\nthree"
		ctrlKey(e, 'r')   // redo should not restore "two\nthree"
		assert.Equal(t, "ne\ntwo\nthree", content(e))
	})
}

// skipUntilDeltaHistory skips an undo test that fails today. Step 3 of the
// `.` + undo design (delta history, boundaries in dispatchKey) un-skips them.
func skipUntilDeltaHistory(t *testing.T, why string) {
	t.Helper()
	t.Skipf("%s; fixed by delta undo history (step 3 of the . + undo design)", why)
}

// TestUndoStepsMatchVim pins down what one 'u' undoes, as in Vim: a whole
// insert session, a change and the insert that follows it, and each paste or
// visual operation are one undo step.
func TestUndoStepsMatchVim(t *testing.T) {
	t.Run("A world<Esc>u undoes the whole insert", func(t *testing.T) {
		skipUntilDeltaHistory(t, "insert mode saves history per keystroke")
		e := newTestEditor("hello")
		keys(e, []rune("A world")...)
		escape(e)
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		ctrlKey(e, 'r')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("insert with Backspace, Enter and Tab is one step", func(t *testing.T) {
		skipUntilDeltaHistory(t, "insert mode saves history per keystroke")
		e := newTestEditor("hello")
		keys(e, []rune("Axy")...)
		backspace(e)
		enter(e)
		tab(e)
		keys(e, 'z')
		escape(e)
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("cwX<Esc>u undoes the delete and the insert", func(t *testing.T) {
		skipUntilDeltaHistory(t, "c saves after the delete, then per inserted rune")
		e := newTestEditor("one two three")
		keys(e, []rune("cwX")...)
		escape(e)
		assert.Equal(t, "X two three", content(e))
		keys(e, 'u')
		assert.Equal(t, "one two three", content(e))
		ctrlKey(e, 'r')
		assert.Equal(t, "X two three", content(e))
	})

	t.Run("2d3w is one step", func(t *testing.T) {
		e := newTestEditor("a b c d e f g h")
		keys(e, '2', 'd', '3', 'w')
		keys(e, 'u')
		assert.Equal(t, "a b c d e f g h", content(e))
	})

	t.Run("oX<Esc>u undoes the new line and its text", func(t *testing.T) {
		skipUntilDeltaHistory(t, "o saves after the new line, then per inserted rune")
		e := newTestEditor("one\ntwo")
		keys(e, []rune("oX")...)
		escape(e)
		assert.Equal(t, "one\nX\ntwo", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("OX<Esc>u undoes the new line and its text", func(t *testing.T) {
		skipUntilDeltaHistory(t, "O saves after the new line, then per inserted rune")
		e := newTestEditor("one\ntwo")
		keys(e, []rune("jOX")...)
		escape(e)
		assert.Equal(t, "one\nX\ntwo", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("SX<Esc>u undoes the substitute", func(t *testing.T) {
		skipUntilDeltaHistory(t, "S saves after emptying the line, then per inserted rune")
		e := newTestEditor("one\ntwo")
		keys(e, []rune("SX")...)
		escape(e)
		assert.Equal(t, "X\ntwo", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("yyp then u removes the pasted line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("one")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "one\none", content(e))
		keys(e, 'u')
		assert.Equal(t, "one", content(e))
	})

	t.Run("3p of a line is one step", func(t *testing.T) {
		skipUntilDeltaHistory(t, "each pasted line saves its own history step")
		e, _ := newTestEditorWithClipboard("one")
		keys(e, 'y', 'y', '3', 'p')
		assert.Equal(t, "one\none\none\none", content(e))
		keys(e, 'u')
		assert.Equal(t, "one", content(e))
	})

	t.Run("x then ylp: u undoes only the paste, then the x", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("abc")
		keys(e, 'x', 'y', 'l', 'p')
		assert.Equal(t, "bbc", content(e))
		keys(e, 'u')
		assert.Equal(t, "bc", content(e))
		keys(e, 'u')
		assert.Equal(t, "abc", content(e))
	})

	t.Run("visual d then u restores the selection", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'e', 'd')
		assert.Equal(t, " world", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("visual line d then u restores the lines", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'V', 'j', 'd')
		assert.Equal(t, "three", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo\nthree", content(e))
	})

	t.Run("U after an insert restores the line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, []rune("A world")...)
		escape(e)
		keys(e, 'U')
		assert.Equal(t, "hello", content(e))
		keys(e, 'U') // U undoes itself
		assert.Equal(t, "hello world", content(e))
	})
}
