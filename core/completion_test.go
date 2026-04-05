package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- findWordLengthBeforeCursor ---

func TestFindWordLengthBeforeCursor(t *testing.T) {
	isWord := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
	}

	t.Run("returns length of trailing word", func(t *testing.T) {
		assert.Equal(t, 3, findWordLengthBeforeCursor("FROM sel", isWord))
	})

	t.Run("empty string returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findWordLengthBeforeCursor("", isWord))
	})

	t.Run("cursor after space returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findWordLengthBeforeCursor("hello ", isWord))
	})

	t.Run("full word with no preceding text", func(t *testing.T) {
		assert.Equal(t, 6, findWordLengthBeforeCursor("SELECT", isWord))
	})

	t.Run("stops at non-word character", func(t *testing.T) {
		assert.Equal(t, 3, findWordLengthBeforeCursor("table.col", isWord))
	})
}

// --- InsertCompletion ---

// TestInsertCompletionNoPrefix tests inserting a completion when nothing has been typed yet.
func TestInsertCompletionNoPrefix(t *testing.T) {
	t.Run("inserts full completion text at cursor", func(t *testing.T) {
		e := newTestEditor("hello ")
		keys(e, '$', 'a') // enter insert mode at end of line (col 6)
		err := e.InsertCompletion(Completion{Text: "world"})
		assert.NoError(t, err)
		assert.Equal(t, "hello world", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})

	t.Run("cursor at start of word replaces whole word", func(t *testing.T) {
		// col 0, cursor before "a" — full-word replacement removes "a" too
		e := newTestEditor("a")
		keys(e, 'i') // insert mode at col 0
		err := e.InsertCompletion(Completion{Text: "SELECT"})
		assert.NoError(t, err)
		assert.Equal(t, "SELECT", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestInsertCompletionReplacesWord tests that the typed word fragment is always
// fully replaced, regardless of whether it shares a prefix with the completion.
func TestInsertCompletionReplacesWord(t *testing.T) {
	t.Run("replaces typed prefix with full completion", func(t *testing.T) {
		e := newTestEditor("sel")
		keys(e, '$', 'a') // cursor after last char → col 3
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("replaces partial prefix mid-line", func(t *testing.T) {
		e := newTestEditor("FROM sel")
		keys(e, '$', 'a') // col 8
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "FROM select", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})

	t.Run("replaces word even when it shares no prefix with completion", func(t *testing.T) {
		e := newTestEditor("se")
		keys(e, '$', 'a') // col 2
		err := e.InsertCompletion(Completion{Text: "insert"})
		assert.NoError(t, err)
		assert.Equal(t, "insert", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("replaces word case-insensitively", func(t *testing.T) {
		e := newTestEditor("SEL")
		keys(e, '$', 'a') // col 3
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestInsertCompletionFullWordReplacement tests that the whole word under the
// cursor is replaced, not just the fragment before it.
func TestInsertCompletionFullWordReplacement(t *testing.T) {
	t.Run("replaces entire word when cursor is at end", func(t *testing.T) {
		e := newTestEditor("sele world")
		keys(e, '$') // normal mode, end of line — but we want col 4
		// move to col 4 then enter insert mode
		keys(e, '0', 'l', 'l', 'l', 'l', 'i') // col 4, after "sele", before " world"
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("replaces entire word when cursor is mid-word", func(t *testing.T) {
		// cursor between "sel" and "e": should replace the whole "sele"
		e := newTestEditor("sele")
		keys(e, '0', 'l', 'l', 'l', 'i') // insert mode at col 3, "e" is after cursor
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("non-word text after cursor is preserved", func(t *testing.T) {
		e := newTestEditor("sel world")
		keys(e, 'l', 'l', 'l', 'i') // insert mode at col 3 (after "sel", before " world")
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestInsertCompletionSavesHistory tests that InsertCompletion records an undo snapshot.
func TestInsertCompletionSavesHistory(t *testing.T) {
	t.Run("completion can be undone", func(t *testing.T) {
		e := newTestEditor("sel")
		keys(e, '$', 'a')
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select", content(e))
		escape(e)
		keys(e, 'u')
		assert.Equal(t, "sel", content(e))
	})
}

// --- TriggerCompletion ---

// TestTriggerCompletion tests that TriggerCompletion dispatches a CompletionRequestSignal
// with the correct context fields.
func TestTriggerCompletion(t *testing.T) {
	t.Run("manual trigger dispatches CompletionRequestSignal", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'i') // insert mode at col 6
		drainSignals(e)
		e.TriggerCompletion(CompletionTriggerManual, "")
		sig := nextSignal(e)
		req, ok := sig.(CompletionRequestSignal)
		assert.True(t, ok)
		ctx := req.Context()
		assert.Equal(t, CompletionTriggerManual, ctx.TriggerKind)
		assert.Equal(t, Position{0, 6}, ctx.Position)
		assert.Equal(t, "hello world", ctx.CurrentLine)
		assert.Equal(t, "hello ", ctx.TextBeforeCursor)
		assert.Equal(t, "world", ctx.TextAfterCursor)
	})

	t.Run("auto trigger carries trigger character", func(t *testing.T) {
		e := newTestEditor("hello.")
		keys(e, '$', 'a') // col 6
		drainSignals(e)
		e.TriggerCompletion(CompletionTriggerAuto, ".")
		sig := nextSignal(e)
		req, ok := sig.(CompletionRequestSignal)
		assert.True(t, ok)
		ctx := req.Context()
		assert.Equal(t, CompletionTriggerAuto, ctx.TriggerKind)
		assert.Equal(t, ".", ctx.TriggerCharacter)
	})

	t.Run("context includes surrounding lines", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'i') // row 1, insert mode
		drainSignals(e)
		e.TriggerCompletion(CompletionTriggerManual, "")
		sig := nextSignal(e)
		req, ok := sig.(CompletionRequestSignal)
		assert.True(t, ok)
		ctx := req.Context()
		assert.Equal(t, []string{"one"}, ctx.LinesBefore)
		assert.Equal(t, []string{"three"}, ctx.LinesAfter)
	})
}
