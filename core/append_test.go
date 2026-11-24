package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// userState is what AppendBytes must leave alone.
type userState struct {
	Cursor        Cursor
	TopLine       int
	Mode          Mode
	StatusLine    string
	CommandLine   string
	CommandCursor int
	SearchEcho    string
	VisualStart   Position
	PendingCount  *int
}

func snapshotUserState(e Editor) userState {
	s := e.GetState()
	return userState{
		Cursor:        e.GetBuffer().GetCursor(),
		TopLine:       s.TopLine,
		Mode:          s.Mode,
		StatusLine:    s.StatusLine,
		CommandLine:   s.CommandLine,
		CommandCursor: s.CommandCursor,
		SearchEcho:    s.SearchEcho,
		VisualStart:   s.VisualStart,
		PendingCount:  s.PendingCount,
	}
}

// TestAppendLeavesUserStateAlone checks that arriving lines move nothing the
// user owns, from each state the user can be in, and that the keys typed next
// carry on as if nothing had arrived.
func TestAppendLeavesUserStateAlone(t *testing.T) {
	tests := []struct {
		name  string
		keys  string // typed before the append
		after string // typed after it
		want  string // the first line afterwards
		mode  Mode
	}{
		{"normal, scrolled, mid line", "20Gll<C-e>", "", "    line 0", NormalMode},
		{"normal, last line", "G", "", "    line 0", NormalMode},
		{"normal, last line end", "G$", "", "    line 0", NormalMode},
		{"pending count and operator", "gg2d", "d", "    line 2", NormalMode},
		{"visual", "25Glvjl", "d", "    line 0", NormalMode},
		{"visual line on the last line", "GV", "", "    line 0", VisualLineMode},
		{"insert", "ggAxy", "z<Esc>", "    line 0xyz", NormalMode},
		{"typing a search", "gg/line 1", "<CR>", "    line 0", NormalMode},
		{"typing a command", ":se", "<BS><BS><Esc>", "    line 0", NormalMode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newScrollEditor(30, 10)
			typeKeys(t, e, tt.keys)
			before := snapshotUserState(e)
			rows := e.GetBuffer().LineCount()
			modified := e.GetBuffer().IsModified()

			e.AppendBytes([]byte("new 1\nnew 2\nnew 3\n"))

			assert.Equal(t, before, snapshotUserState(e))
			assert.Equal(t, rows+3, e.GetBuffer().LineCount())
			assert.Equal(t, modified, e.GetBuffer().IsModified())

			typeKeys(t, e, tt.after)
			assert.Equal(t, tt.want, e.GetBuffer().GetLines()[0])
			assert.Equal(t, tt.mode, e.GetState().Mode)
		})
	}

	t.Run("the search typed through an append finds its match", func(t *testing.T) {
		e := newScrollEditor(30, 10)
		typeKeys(t, e, "gg/new 2")
		e.AppendBytes([]byte("new 1\nnew 2\n"))
		typeKeys(t, e, "<CR>")
		assert.Equal(t, Position{Row: 31, Col: 0}, cursorPos(e))
	})
}

// TestAppendUndoRedo checks that u and CTRL-R after an append undo and redo
// the user's own changes, never the arrived lines, as the design says: old
// undo steps stay valid because an append adds rows only after the old end.
func TestAppendUndoRedo(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		keys      string // the user's change; the append comes after it
		mid       string // typed after the append, before u (an insert's rest)
		changed   string
		undone    string
		appendNow string
	}{
		{"x on the first line", "abc\ndef", "x", "", "bc\ndef\nnew", "abc\ndef\nnew", "new\n"},
		{"dd on the last line", "a\nb\nc", "Gdd", "", "a\nb\nnew", "a\nb\nc\nnew", "new\n"},
		{"o on the last line", "a\nb", "Gonew<Esc>", "", "a\nb\nnew\nx", "a\nb\nx", "x\n"},
		{"o on the last line, append mid-insert", "a\nb", "Gone", "w<Esc>", "a\nb\nnew\nx", "a\nb\nx", "x\n"},
		{"A on the last line, append mid-insert", "a\nb", "GAyy", "z<Esc>", "a\nbyyz\nx", "a\nb\nx", "x\n"},
		{"p of a line below the last", "a\nb", "ggyyGp", "", "a\nb\na\nx", "a\nb\nx", "x\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			e.AppendBytes([]byte(tt.appendNow))
			typeKeys(t, e, tt.mid)
			require.Equal(t, tt.changed, content(e))

			typeKeys(t, e, "u")
			assert.Equal(t, tt.undone, content(e), "after u")
			assert.False(t, e.GetBuffer().IsModified(), "undone back to the saved text")

			typeKeys(t, e, "<C-r>")
			assert.Equal(t, tt.changed, content(e), "after CTRL-R")

			typeKeys(t, e, "uu")
			assert.Equal(t, tt.undone, content(e), "u with nothing left to undo keeps arrived lines")
		})
	}

	t.Run("redo after an append", func(t *testing.T) {
		e := newTestEditor("a\nb")
		typeKeys(t, e, "Gddu")
		e.AppendBytes([]byte("x\n"))
		typeKeys(t, e, "<C-r>")
		assert.Equal(t, "a\nx", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "a\nb\nx", content(e))
	})
}

// TestAppendKeepsDot checks that . still repeats the user's last change.
func TestAppendKeepsDot(t *testing.T) {
	e := newTestEditor("foo\nbar")
	typeKeys(t, e, "x")
	e.AppendBytes([]byte("baz\n"))
	typeKeys(t, e, "j.")
	assert.Equal(t, "oo\nar\nbaz", content(e))

	e = newTestEditor("foo\nbar")
	typeKeys(t, e, "Aa<Esc>")
	e.AppendBytes([]byte("baz\n"))
	typeKeys(t, e, "j.")
	assert.Equal(t, "fooa\nbara\nbaz", content(e))
}

// hostBuffer is a Buffer from a host: textBuffer without appendSaved, so the
// editor takes its fallback path.
type hostBuffer struct{ inner Buffer }

func (b *hostBuffer) GetLines() []string                     { return b.inner.GetLines() }
func (b *hostBuffer) GetLineRunes(n int) []rune              { return b.inner.GetLineRunes(n) }
func (b *hostBuffer) LineRuneCount(n int) int                { return b.inner.LineRuneCount(n) }
func (b *hostBuffer) GetSavedContent() string                { return b.inner.GetSavedContent() }
func (b *hostBuffer) GetCurrentContent() string              { return b.inner.GetCurrentContent() }
func (b *hostBuffer) LineCount() int                         { return b.inner.LineCount() }
func (b *hostBuffer) InsertRunesAt(r, c int, x []rune) error { return b.inner.InsertRunesAt(r, c, x) }
func (b *hostBuffer) DeleteRunesAt(r, c, n int) *EditorError { return b.inner.DeleteRunesAt(r, c, n) }
func (b *hostBuffer) GetCursor() Cursor                      { return b.inner.GetCursor() }
func (b *hostBuffer) SetCursor(c Cursor)                     { b.inner.SetCursor(c) }
func (b *hostBuffer) IsModified() bool                       { return b.inner.IsModified() }
func (b *hostBuffer) SaveContent()                           { b.inner.SaveContent() }
func (b *hostBuffer) SetContent(c []byte)                    { b.inner.SetContent(c) }
func (b *hostBuffer) IsEmpty() bool                          { return b.inner.IsEmpty() }
func (b *hostBuffer) Find(p string, s Position, o SearchOptions) (Position, bool) {
	return b.inner.Find(p, s, o)
}

// TestAppendModifiedState checks that arrived text counts as saved: an
// unmodified buffer stays unmodified, a user edit still shows, and undoing
// back reads as unmodified again, for the built-in buffer and, except for
// that last case, for a host's.
func TestAppendModifiedState(t *testing.T) {
	for _, host := range []bool{false, true} {
		name := "textBuffer"
		if host {
			name = "host buffer"
		}
		t.Run(name, func(t *testing.T) {
			e := newTestEditor("a\nb")
			if host {
				e.SetBuffer(&hostBuffer{inner: NewBufferFromBytes([]byte("a\nb"))})
			}
			b := e.GetBuffer()

			e.AppendBytes([]byte("c\n"))
			assert.False(t, b.IsModified(), "append to an unmodified buffer")
			assert.Equal(t, "a\nb\nc", b.GetSavedContent())
			assert.Nil(t, e.ExecuteCommand("q"), ":q works when only text arrived")

			typeKeys(t, e, "x")
			assert.True(t, b.IsModified(), "after a user edit")
			e.AppendBytes([]byte("d\n"))
			assert.True(t, b.IsModified(), "append after a user edit")
			assert.Equal(t, "\nb\nc\nd", content(e))

			typeKeys(t, e, "u")
			assert.Equal(t, "a\nb\nc\nd", content(e))
			if host {
				// Documented: a host buffer can't add to its saved content.
				assert.True(t, b.IsModified(), "host buffer, undone back after an append")
			} else {
				assert.False(t, b.IsModified(), "undone back after an append")
			}
		})
	}

	t.Run(":q works after appends", func(t *testing.T) {
		e := newTestEditor("a")
		e.AppendBytes([]byte("b\nc\n"))
		assert.Nil(t, e.ExecuteCommand("q"))
		assert.True(t, e.GetState().Quit)
	})
}

// TestAppendPartialLines checks that only whole lines are added and the rest
// is held, as Vim holds a job's unfinished line.
func TestAppendPartialLines(t *testing.T) {
	e := newTestEditor("x")
	e.AppendBytes([]byte("ab"))
	assert.Equal(t, "x", content(e), "no line yet")
	e.AppendBytes([]byte("c\nd"))
	assert.Equal(t, "x\nabc", content(e))
	e.AppendBytes([]byte("\n"))
	assert.Equal(t, "x\nabc\nd", content(e))

	e.FlushAppend()
	assert.Equal(t, "x\nabc\nd", content(e), "FlushAppend with nothing held")

	e.AppendBytes([]byte("\n\ne"))
	assert.Equal(t, "x\nabc\nd\n\n", content(e), "empty lines are lines")
	e.FlushAppend()
	assert.Equal(t, "x\nabc\nd\n\n\ne", content(e))
	e.AppendBytes([]byte("f\n"))
	assert.Equal(t, "x\nabc\nd\n\n\ne\nf", content(e), "after FlushAppend a new line starts")

	e.AppendBytes([]byte("cr\r\n"))
	assert.Equal(t, "cr\r", e.GetBuffer().GetLines()[7], "\\r is kept, as SetContent keeps it")

	t.Run("SetContent drops held bytes", func(t *testing.T) {
		e := newTestEditor("x")
		e.AppendBytes([]byte("held"))
		e.SetContent([]byte("y"))
		e.AppendBytes([]byte("z\n"))
		assert.Equal(t, "y\nz", content(e))
		e.FlushAppend()
		assert.Equal(t, "y\nz", content(e))
	})

	t.Run("SetBuffer drops held bytes", func(t *testing.T) {
		e := newTestEditor("x")
		e.AppendBytes([]byte("held"))
		e.SetBuffer(NewBufferFromBytes([]byte("y")))
		e.FlushAppend()
		assert.Equal(t, "y", content(e))
	})

	t.Run("the caller may reuse its slice", func(t *testing.T) {
		e := newTestEditor("x")
		p := []byte("ab")
		e.AppendBytes(p)
		copy(p, "zz")
		e.AppendBytes([]byte("\n"))
		assert.Equal(t, "x\nab", content(e))
	})
}

// TestAppendSplitRunes feeds multi-byte characters one byte per call.
func TestAppendSplitRunes(t *testing.T) {
	text := "é 世 😀 é\n"
	e := newTestEditor("x")
	for i := range len(text) {
		e.AppendBytes([]byte{text[i]})
	}
	assert.Equal(t, "x\né 世 😀 é", content(e))

	e.AppendBytes([]byte("ok\xe4\xb8")) // 世 cut short
	e.FlushAppend()
	assert.Equal(t, "ok\uFFFD\uFFFD", e.GetBuffer().GetLines()[2],
		"an unfinished rune at flush: one U+FFFD per byte, as SetContent decodes it")
}

// TestAppendEmptyBuffer checks Vim's rule: the single empty line of an
// unmodified empty buffer is replaced, a modified one keeps it.
func TestAppendEmptyBuffer(t *testing.T) {
	e := newTestEditor("")
	e.AppendBytes([]byte("a\nb\n"))
	assert.Equal(t, "a\nb", content(e))
	assert.False(t, e.GetBuffer().IsModified())
	assert.Equal(t, Position{}, cursorPos(e))

	// Only the first append replaces: the result doesn't depend on how the
	// stream is cut into chunks.
	e = newTestEditor("")
	e.AppendBytes([]byte("\n"))
	assert.Equal(t, []string{""}, e.GetBuffer().GetLines(), "an empty line replaces the empty line")
	e.AppendBytes([]byte("a\n"))
	assert.Equal(t, []string{"", "a"}, e.GetBuffer().GetLines(), "the arrived empty line is kept")

	e = newTestEditor("")
	e.AppendBytes([]byte("\na\n"))
	assert.Equal(t, []string{"", "a"}, e.GetBuffer().GetLines(), "the same in one chunk")

	e = newTestEditor("x")
	e.AppendBytes([]byte("a\n"))
	e.SetContent(nil)
	e.AppendBytes([]byte("b\n"))
	assert.Equal(t, "b", content(e), "SetContent starts over")

	e = newTestEditor("x")
	typeKeys(t, e, "x")
	require.True(t, e.GetBuffer().IsEmpty())
	e.AppendBytes([]byte("a\n"))
	assert.Equal(t, "\na", content(e), "a modified empty buffer keeps its line")
	assert.True(t, e.GetBuffer().IsModified())

	e = newTestEditor("")
	e.SetBuffer(&hostBuffer{inner: NewBufferFromBytes(nil)})
	e.AppendBytes([]byte("a\n\n"))
	assert.Equal(t, []string{"a", ""}, e.GetBuffer().GetLines(), "host buffer, empty")
	assert.False(t, e.GetBuffer().IsModified())
}

// TestAppendSearch checks the search state: the current match stays
// highlighted, and n finds matches in arrived lines.
func TestAppendSearch(t *testing.T) {
	e := newTestEditor("foo\nbar")
	typeKeys(t, e, "/foo<CR>")
	require.Equal(t, []Position{{0, 0}}, e.SearchResults())

	e.AppendBytes([]byte("x foo\n"))
	assert.Equal(t, []Position{{0, 0}}, e.SearchResults(), "the current match is kept")
	assert.Equal(t, "/foo", e.GetState().SearchEcho)

	typeKeys(t, e, "n")
	assert.Equal(t, Position{2, 2}, cursorPos(e))
	assert.Equal(t, []Position{{2, 2}}, e.SearchResults())
}

// TestAppendLargeChunk checks a chunk of many lines lands in order.
func TestAppendLargeChunk(t *testing.T) {
	var b strings.Builder
	for range 1000 {
		b.WriteString("line\n")
	}
	e := newTestEditor("first")
	e.AppendBytes([]byte(b.String()))
	assert.Equal(t, 1001, e.GetBuffer().LineCount())
	assert.False(t, e.GetBuffer().IsModified())
}
