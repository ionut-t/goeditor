package goeditor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelAppendRefreshesCaches checks that lines added by AppendBytes are
// highlighted and laid out as if the text had been loaded with them, while
// the cursor and the view stay where they were.
func TestModelAppendRefreshesCaches(t *testing.T) {
	m := cacheBottomOfFile(newHighlightedModel(alternatingGoLines(300)))
	require.NoError(t, m.SetCursorPosition(120, 0))
	cursor := m.editor.GetBuffer().GetCursor()
	topLine := m.editor.GetState().TopLine
	view := m.viewport.View()

	// Same alternation, continued, cut mid-line between chunks.
	more := strings.TrimPrefix(alternatingGoLines(340), alternatingGoLines(300)+"\n")
	m.AppendBytes([]byte(more[:len(more)/2]))
	m.AppendBytes([]byte(more[len(more)/2:] + "\n"))
	require.Equal(t, 340, m.editor.GetBuffer().LineCount())

	assert.Equal(t, cursor, m.editor.GetBuffer().GetCursor())
	assert.Equal(t, topLine, m.editor.GetState().TopLine)
	assert.Equal(t, view, m.viewport.View(), "nothing on screen changes")
	assert.False(t, m.HasChanges())

	require.NoError(t, m.SetCursorPosition(330, 0))
	requireCachedTokensMatchBuffer(t, m, 330)
}

// TestModelAppendWrappedLines checks the layout of arrived lines that wrap.
func TestModelAppendWrappedLines(t *testing.T) {
	m := New(30, 12)
	m.Focus()
	m.SetContent("short")
	long := strings.Repeat("wrapped words ", 6)
	m.AppendBytes([]byte(long + "\nend\n"))

	require.NoError(t, m.SetCursorPosition(2, 0))
	fresh := New(30, 12)
	fresh.Focus()
	fresh.SetContent("short\n" + long + "\nend")
	require.NoError(t, fresh.SetCursorPosition(2, 0))
	assert.Equal(t, fresh.viewport.View(), m.viewport.View())
}

// TestModelAppendWidensGutter checks the line number gutter grows when the
// line count gains a digit past the minimum width (four digits), and that
// the new width is used straight away.
func TestModelAppendWidensGutter(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 9999; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	text := strings.TrimSuffix(b.String(), "\n")
	m := New(40, 10)
	m.Focus()
	m.SetContent(text)
	before := m.calculateLineNumberWidth(m.editor.GetBuffer().LineCount())

	m.AppendBytes([]byte("line 10000\n"))
	after := m.calculateLineNumberWidth(m.editor.GetBuffer().LineCount())
	assert.Equal(t, before+1, after)
	assert.Equal(t, 40-after, m.editor.GetState().AvailableWidth)

	fresh := New(40, 10)
	fresh.Focus()
	fresh.SetContent(text + "\nline 10000")
	assert.Equal(t, fresh.viewport.View(), m.viewport.View(), "screen at the top")

	require.NoError(t, m.SetCursorPosition(9999, 0))
	require.NoError(t, fresh.SetCursorPosition(9999, 0))
	assert.Equal(t, fresh.viewport.View(), m.viewport.View(), "screen at the end")
}

// TestModelAppendPartialAndFlush checks that a chunk with no complete line
// shows nothing, and FlushAppend shows it.
func TestModelAppendPartialAndFlush(t *testing.T) {
	m := New(30, 8)
	m.Focus()
	m.SetContent("first")
	view := m.viewport.View()

	m.AppendBytes([]byte("par"))
	assert.Equal(t, view, m.viewport.View())
	assert.Equal(t, "first", m.GetCurrentContent())

	m.AppendBytes([]byte("tial"))
	m.FlushAppend()
	assert.Equal(t, "first\npartial", m.GetCurrentContent())
	assert.Contains(t, ansi.Strip(m.viewport.View()), "partial")
	assert.False(t, m.HasChanges())
}

// TestModelAppendFollow is the host's follow rule from the doc comment: move
// to the new last line only when the cursor was on the old one.
func TestModelAppendFollow(t *testing.T) {
	m := New(30, 6)
	m.Focus()
	m.SetContent("a\nb\nc")
	m = typeText("G")(m)

	for i := range 20 {
		buf := m.GetEditor().GetBuffer()
		follow := buf.GetCursor().Position.Row == buf.LineCount()-1
		m.AppendBytes(fmt.Appendf(nil, "line %d\n", i))
		if follow {
			require.NoError(t, m.SetCursorPosition(buf.LineCount()-1, 0))
		}
	}
	assert.Equal(t, 22, m.GetEditor().GetBuffer().GetCursor().Position.Row)
	assert.Contains(t, ansi.Strip(m.viewport.View()), "line 19")

	m = typeText("k")(m)
	m.AppendBytes([]byte("more\n"))
	assert.Equal(t, 21, m.GetEditor().GetBuffer().GetCursor().Position.Row, "not on the last line: stays")
}
