package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- findCompletionPrefixLength ---

// TestFindCompletionPrefixLength tests the pure prefix-matching helper.
func TestFindCompletionPrefixLength(t *testing.T) {
	t.Run("exact full match", func(t *testing.T) {
		assert.Equal(t, 5, findCompletionPrefixLength("hello", "hello"))
	})

	t.Run("typed suffix matches completion prefix", func(t *testing.T) {
		// text before cursor ends with "sel"; completion starts with "sel"
		assert.Equal(t, 3, findCompletionPrefixLength("SELECT sel", "select"))
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		assert.Equal(t, 3, findCompletionPrefixLength("SEL", "select"))
	})

	t.Run("no match returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findCompletionPrefixLength("hello", "world"))
	})

	t.Run("empty text before cursor returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findCompletionPrefixLength("", "select"))
	})

	t.Run("empty completion text returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findCompletionPrefixLength("sel", ""))
	})

	t.Run("both empty returns 0", func(t *testing.T) {
		assert.Equal(t, 0, findCompletionPrefixLength("", ""))
	})

	t.Run("completion longer than typed text: partial match", func(t *testing.T) {
		// typed "sel" is a prefix of "select" → length 3
		assert.Equal(t, 3, findCompletionPrefixLength("sel", "select"))
	})

	t.Run("typed text longer than completion: full completion matches", func(t *testing.T) {
		// completion is "sel" (len 3); longest suffix of "mysel" that is a prefix of "sel" is "sel"
		assert.Equal(t, 3, findCompletionPrefixLength("mysel", "sel"))
	})

	t.Run("prefers longest match", func(t *testing.T) {
		// "fro" is a 3-char suffix of "fro" and prefix of "from" → 3, not 1
		assert.Equal(t, 3, findCompletionPrefixLength("fro", "from"))
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

	t.Run("inserts into empty buffer", func(t *testing.T) {
		e := newTestEditor("a")
		keys(e, 'i') // insert mode at col 0
		err := e.InsertCompletion(Completion{Text: "SELECT"})
		assert.NoError(t, err)
		assert.Equal(t, "SELECTa", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestInsertCompletionWithPrefix tests that the typed prefix is replaced by the completion.
func TestInsertCompletionWithPrefix(t *testing.T) {
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

	t.Run("case-insensitive prefix replacement", func(t *testing.T) {
		e := newTestEditor("SEL")
		keys(e, '$', 'a') // col 3
		err := e.InsertCompletion(Completion{Text: "select"})
		assert.NoError(t, err)
		assert.Equal(t, "select", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestInsertCompletionPreservesRemainder tests that text after the cursor is kept.
func TestInsertCompletionPreservesRemainder(t *testing.T) {
	t.Run("text after cursor is preserved", func(t *testing.T) {
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
