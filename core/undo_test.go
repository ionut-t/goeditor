package core

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		_ = e.HandleKey(KeyEvent{Key: KeyCtrlU})
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

// TestUndoStepsMatchVim pins down what one 'u' undoes, as in Vim: a whole
// insert session, a change and the insert that follows it, and each paste or
// visual operation are one undo step.
func TestUndoStepsMatchVim(t *testing.T) {
	t.Run("A world<Esc>u undoes the whole insert", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, []rune("A world")...)
		escape(e)
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		ctrlKey(e, 'r')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("insert with Backspace, Enter and Tab is one step", func(t *testing.T) {
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
		e := newTestEditor("one\ntwo")
		keys(e, []rune("oX")...)
		escape(e)
		assert.Equal(t, "one\nX\ntwo", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("OX<Esc>u undoes the new line and its text", func(t *testing.T) {
		e := newTestEditor("one\ntwo")
		keys(e, []rune("jOX")...)
		escape(e)
		assert.Equal(t, "one\nX\ntwo", content(e))
		keys(e, 'u')
		assert.Equal(t, "one\ntwo", content(e))
	})

	t.Run("SX<Esc>u undoes the substitute", func(t *testing.T) {
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

// TestUndoStepBoundaries covers where an undo step ends besides a completed
// command: cursor keys in insert mode, mappings, and edits made outside a key.
func TestUndoStepBoundaries(t *testing.T) {
	t.Run("a cursor key in insert mode starts a new step", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, []rune("A a")...)
		_ = e.HandleKey(KeyEvent{Key: KeyLeft})
		keys(e, 'b')
		escape(e)
		assert.Equal(t, "hello ba", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello a", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("a mapping is one step", func(t *testing.T) {
		e := newTestEditor("one two three four")
		require.NoError(t, e.Map(MapNormal, ",x", "dwdw", true))
		keys(e, ',', 'x')
		assert.Equal(t, "three four", content(e))
		keys(e, 'u')
		assert.Equal(t, "one two three four", content(e))
	})

	t.Run("steps still close after a recursive mapping is stopped", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "a", "b", false))
		require.NoError(t, e.Map(MapNormal, "b", "a", false))
		require.NotNil(t, e.HandleKey(KeyEvent{Rune: 'a'}))

		keys(e, 'x', 'x')
		keys(e, 'u')
		assert.Equal(t, "ello", content(e))
	})

	t.Run("host edits group into one step that closes on the next key", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.GetBuffer().InsertRunesAt(0, 0, []rune("a")))
		require.NoError(t, e.GetBuffer().InsertRunesAt(0, 0, []rune("b")))
		keys(e, 'x')
		assert.Equal(t, "ahello", content(e))
		keys(e, 'u')
		assert.Equal(t, "bahello", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("a completion joins the insert session", func(t *testing.T) {
		e := newTestEditor("hello ")
		keys(e, 'A', 'w')
		require.NoError(t, e.InsertCompletion(Completion{Text: "world"}))
		escape(e)
		assert.Equal(t, "hello world", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello ", content(e))
	})

	t.Run("without Vim mode every key is a step", func(t *testing.T) {
		e := newTestEditor("x")
		e.DisableVimMode(true)
		keys(e, 'a', 'b')
		assert.Equal(t, "abx", content(e))
		_, err := e.Undo()
		require.NoError(t, err)
		assert.Equal(t, "ax", content(e))
	})
}

// TestClampCursor checks that clampCursor keeps both ends of the row and
// column inside the text.
func TestClampCursor(t *testing.T) {
	tests := []struct {
		name    string
		content string
		in      Position
		want    Position
	}{
		{"inside stays put", "ab\ncd", Position{1, 1}, Position{1, 1}},
		{"just past the line end stays put", "ab\ncd", Position{0, 2}, Position{0, 2}},
		{"row below the last line", "ab\ncd", Position{5, 1}, Position{1, 1}},
		{"column past the line end", "ab\ncd", Position{0, 9}, Position{0, 2}},
		{"negative row and column", "ab\ncd", Position{-1, -1}, Position{0, 0}},
		{"empty buffer", "", Position{3, 3}, Position{0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEditor(tc.content).(*editor)
			got := e.clampCursor(Cursor{Position: tc.in})
			assert.Equal(t, tc.want, got.Position)
		})
	}
}

// TestUndoHistory covers the history itself: its limit, resets, and exact
// restoration of the text.
func TestUndoHistory(t *testing.T) {
	t.Run("SetMaxHistory limits the number of undoable changes", func(t *testing.T) {
		e := newTestEditor("hello")
		e.SetMaxHistory(2)
		keys(e, 'x', 'x', 'x')
		keys(e, 'u', 'u', 'u')
		assert.Equal(t, "ello", content(e))
	})

	t.Run("SetContent on the buffer clears the history", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x')
		e.GetBuffer().SetContent([]byte("new"))
		keys(e, 'u')
		assert.Equal(t, "new", content(e))
	})

	t.Run("undo restores a trailing empty line", func(t *testing.T) {
		e := newTestEditor("a")
		keys(e, 'o')
		escape(e)
		assert.Equal(t, "a\n", content(e))
		keys(e, 'x', 'k', 'x')
		assert.Equal(t, "\n", content(e))
		keys(e, 'u')
		assert.Equal(t, "a\n", content(e))
	})

	t.Run("a step that no longer fits the text clears the history", func(t *testing.T) {
		e := New(nil)
		raw := NewBufferFromBytes([]byte("a\nb\nc"))
		e.SetBuffer(raw)
		keys(e, 'j', 'j', 'x') // records deleting "c" on row 2

		// Edit behind the recording wrapper, so row 2 no longer exists.
		require.Nil(t, raw.DeleteRunesAt(0, 1, 3))
		require.Equal(t, "a", content(e))

		_, err := e.Undo()
		assert.ErrorIs(t, err, ErrUndoCorrupt)
		assert.Equal(t, "a", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e), "cursor moved back inside the text")

		_, err = e.Undo()
		assert.EqualError(t, err, "already at oldest change", "history was cleared")
	})

	t.Run("a U that no longer fits the text leaves no step open", func(t *testing.T) {
		e := New(nil)
		raw := NewBufferFromBytes([]byte("a\nb\nc"))
		e.SetBuffer(raw)
		keys(e, 'j', 'j', 'x') // starts a U run on row 2

		// Edit behind the recording wrapper, so row 2 no longer exists.
		require.Nil(t, raw.DeleteRunesAt(0, 1, 3))

		_, err := e.UndoLine()
		assert.ErrorIs(t, err, ErrUndoCorrupt)
		assert.Nil(t, e.(*editor).openStep)

		// The next change is a step of its own.
		keys(e, 'x')
		assert.Equal(t, "", content(e))
		keys(e, 'u')
		assert.Equal(t, "a", content(e))
	})

	t.Run("redo after several undos replays in order", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'd', 'w')
		keys(e, []rune("ifoo ")...)
		escape(e)
		assert.Equal(t, "foo two three", content(e))
		keys(e, 'u', 'u')
		assert.Equal(t, "one two three", content(e))
		ctrlKey(e, 'r')
		ctrlKey(e, 'r')
		assert.Equal(t, "foo two three", content(e))
	})
}

// TestUndoCoalescesTyping verifies an insert session is recorded as one edit
// when each key inserts where the previous one ended, line breaks included.
func TestUndoCoalescesTyping(t *testing.T) {
	t.Run("typing across a line break is one edit", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, []rune("Aab")...)
		enter(e)
		keys(e, 'c', 'd')
		escape(e)
		require.Equal(t, "helloab\ncd", content(e))

		ed := e.(*editor)
		require.Len(t, ed.undoSteps, 1)
		assert.Equal(t, []edit{{at: Position{0, 5}, inserted: "ab\ncd"}}, ed.undoSteps[0].edits)

		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		ctrlKey(e, 'r')
		assert.Equal(t, "helloab\ncd", content(e))
	})

	t.Run("an insert that does not follow the last one is a new edit", func(t *testing.T) {
		e := newTestEditor("hello")
		buf := e.GetBuffer()
		require.NoError(t, buf.InsertRunesAt(0, 0, []rune("x\ny")))
		require.NoError(t, buf.InsertRunesAt(0, 1, []rune("z"))) // inside the first insert
		require.Equal(t, "xz\nyhello", content(e))

		keys(e, 'l') // closes the host step
		ed := e.(*editor)
		require.Len(t, ed.undoSteps, 1)
		assert.Len(t, ed.undoSteps[0].edits, 2)

		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})

	t.Run("a Backspace is not merged into the insert", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, []rune("Aab")...)
		backspace(e)
		keys(e, 'c')
		escape(e)
		require.Equal(t, "helloac", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
		ctrlKey(e, 'r')
		assert.Equal(t, "helloac", content(e))
	})
}

// applyEdits applies edits to text in order, as a host tracking the buffer
// would.
func applyEdits(text string, edits []TextEdit) string {
	runes := []rune(text)
	offset := func(p Position) int {
		i := 0
		for row := 0; row < p.Row; row++ {
			for runes[i] != '\n' {
				i++
			}
			i++
		}
		return i + p.Col
	}
	for _, te := range edits {
		start, end := offset(te.Start), offset(te.End)
		runes = slices.Concat(runes[:start], []rune(te.Text), runes[end:])
	}
	return string(runes)
}

// TestUndoReturnsEdits checks that the edits undo, redo and 'U' return take
// the text as it was before them to the text they leave.
func TestUndoReturnsEdits(t *testing.T) {
	undo := func(e Editor) ([]TextEdit, error) { return e.Undo() }
	redo := func(e Editor) ([]TextEdit, error) { return e.Redo() }
	undoLine := func(e Editor) ([]TextEdit, error) { return e.UndoLine() }

	tests := []struct {
		name    string
		content string
		setup   func(e Editor)
		apply   func(e Editor) ([]TextEdit, error)
	}{
		{"undo x", "hello", func(e Editor) { keys(e, 'x') }, undo},
		{"undo dd in the middle", "one\ntwo\nthree", func(e Editor) { keys(e, 'j', 'd', 'd') }, undo},
		{"undo dd of the last line", "one\ntwo", func(e Editor) { keys(e, 'j', 'd', 'd') }, undo},
		{"undo J", "one\ntwo", func(e Editor) { keys(e, 'J') }, undo},
		{"undo o", "one\ntwo", func(e Editor) { keys(e, []rune("onew")...); escape(e) }, undo},
		{"undo cw", "one two", func(e Editor) { keys(e, []rune("cwfoo")...); escape(e) }, undo},
		{"undo typing across lines", "héllo", func(e Editor) {
			keys(e, []rune("Aé")...)
			enter(e)
			keys(e, 'ü')
			backspace(e)
			escape(e)
		}, undo},
		{"undo a mapping that edits twice", "one two three", func(e Editor) {
			require.NoError(t, e.Map(MapNormal, ",x", "dwdw", true))
			keys(e, ',', 'x')
		}, undo},
		{"redo dd", "one\ntwo\nthree", func(e Editor) { keys(e, 'j', 'd', 'd', 'u') }, redo},
		{"redo cw", "one two", func(e Editor) { keys(e, []rune("cwfoo")...); escape(e); keys(e, 'u') }, redo},
		{"U after several changes", "one two three", func(e Editor) { keys(e, 'x', 'w', 'd', 'w', 'x') }, undoLine},
		{"U after o", "one\ntwo", func(e Editor) { keys(e, []rune("onew")...); escape(e) }, undoLine},
		{"U twice", "hello", func(e Editor) { keys(e, 'x', 'x', 'U') }, undoLine},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEditor(tc.content)
			tc.setup(e)
			before := content(e)

			edits, err := tc.apply(e)
			require.NoError(t, err)
			require.NotEmpty(t, edits)
			assert.Equal(t, content(e), applyEdits(before, edits))
		})
	}

	t.Run("edits describe the change exactly", func(t *testing.T) {
		e := newTestEditor("one two")
		keys(e, []rune("cwfoo")...)
		escape(e)

		edits, err := e.Undo()
		require.NoError(t, err)
		assert.Equal(t, []TextEdit{
			{Start: Position{0, 0}, End: Position{0, 3}, Text: ""},
			{Start: Position{0, 0}, End: Position{0, 0}, Text: "one"},
		}, edits)

		edits, err = e.Redo()
		require.NoError(t, err)
		assert.Equal(t, []TextEdit{
			{Start: Position{0, 0}, End: Position{0, 3}, Text: ""},
			{Start: Position{0, 0}, End: Position{0, 0}, Text: "foo"},
		}, edits)
	})

	t.Run("a step that fails part way returns the edits it applied", func(t *testing.T) {
		e := New(nil)
		raw := NewBufferFromBytes([]byte("a\nb\nc"))
		e.SetBuffer(raw)
		require.NoError(t, e.GetBuffer().InsertRunesAt(2, 0, []rune("x")))
		require.NoError(t, e.GetBuffer().InsertRunesAt(0, 0, []rune("y")))
		keys(e, 'l') // closes the host step: two edits, undone row 0 first

		// Edit behind the recording wrapper, so row 2 no longer exists.
		require.Nil(t, raw.DeleteRunesAt(1, 1, 3))
		before := content(e)
		require.Equal(t, "ya\nb", before)
		drainSignals(e)

		err := e.HandleKey(KeyEvent{Rune: 'u'})
		assert.ErrorIs(t, err, ErrUndoCorrupt)
		assert.Equal(t, "a\nb", content(e))

		var undo *UndoSignal
		for s := nextSignal(e); s != nil; s = nextSignal(e) {
			if sig, ok := s.(UndoSignal); ok {
				undo = &sig
			}
		}
		if assert.NotNil(t, undo, "expected an UndoSignal for the partial undo") {
			assert.Equal(t, []TextEdit{{End: Position{0, 1}}}, undo.Value())
			assert.Equal(t, content(e), applyEdits(before, undo.Value()))
		}
	})

	t.Run("a replacement whose insert fails reports the delete", func(t *testing.T) {
		buf := failingInsertBuffer{NewBufferFromBytes([]byte("hello"))}
		applied, err := replaceAt(buf, Position{0, 0}, "he", "HE", nil)
		assert.ErrorIs(t, err, errInsertFailed)
		assert.Equal(t, "llo", buf.GetCurrentContent())
		assert.Equal(t, []TextEdit{{End: Position{0, 2}}}, applied)
	})

	t.Run("nothing to undo returns no edits", func(t *testing.T) {
		e := newTestEditor("hello")
		edits, err := e.Undo()
		assert.Error(t, err)
		assert.Nil(t, edits)
	})

	t.Run("u and Ctrl-R dispatch the edits", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x')
		drainSignals(e)

		keys(e, 'u')
		ctrlKey(e, 'r')
		var got []Signal
		for s := nextSignal(e); s != nil; s = nextSignal(e) {
			switch s.(type) {
			case UndoSignal, RedoSignal:
				got = append(got, s)
			}
		}
		assert.Equal(t, []Signal{
			UndoSignal{edits: []TextEdit{{Text: "h"}}},
			RedoSignal{edits: []TextEdit{{End: Position{0, 1}}}},
		}, got)
	})
}

var errInsertFailed = errors.New("insert failed")

// failingInsertBuffer is a Buffer whose inserts always fail.
type failingInsertBuffer struct{ Buffer }

func (failingInsertBuffer) InsertRunesAt(int, int, []rune) error { return errInsertFailed }
