package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unclampedBuffer is a host Buffer that, unlike textBuffer, never clamps the
// cursor, so the editor has to.
type unclampedBuffer struct {
	Buffer
	cursor Cursor
}

func (b *unclampedBuffer) GetCursor() Cursor  { return b.cursor }
func (b *unclampedBuffer) SetCursor(c Cursor) { b.cursor = c }

// TestGetBufferEditsClampCursor checks that edits through GetBuffer() keep
// the cursor inside the text, as SetCursor does: on an existing line and no
// further than its end, with Preferred left as it is. The cursor is only
// clamped, not shifted with the text.
func TestGetBufferEditsClampCursor(t *testing.T) {
	tests := []struct {
		name    string
		content string
		cursor  Position
		edit    func(b Buffer)
		want    Position
	}{
		{"delete empties the cursor line", "abc\ndef\nghi", Position{2, 1},
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(2, 0, 3)) }, Position{2, 0}},
		{"delete shortens the cursor line", "abcdef", Position{0, 5},
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 2, 3)) }, Position{0, 3}},
		{"delete removes the cursor line", "a\nhello", Position{1, 4},
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 1, 6)) }, Position{0, 1}},
		{"delete before the cursor in range", "abcdef", Position{0, 2},
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 4, 2)) }, Position{0, 2}},
		{"insert doesn't shift the cursor down", "abc\ndef", Position{1, 2},
			func(b Buffer) { require.NoError(t, b.InsertRunesAt(0, 0, []rune("x\ny"))) }, Position{1, 2}},
		{"failed edit leaves the cursor", "abc", Position{0, 2},
			func(b Buffer) { require.NotNil(t, b.DeleteRunesAt(5, 0, 1)) }, Position{0, 2}},
		{"SetContent with fewer lines", "abc\ndef\nghi", Position{2, 2},
			func(b Buffer) { b.SetContent([]byte("x\ny")) }, Position{1, 1}},
		{"SetContent with nothing", "abc\ndef", Position{1, 2},
			func(b Buffer) { b.SetContent(nil) }, Position{0, 0}},
	}
	for _, tt := range tests {
		for _, unclamped := range []bool{false, true} {
			name := tt.name
			if unclamped {
				name += ", buffer that doesn't clamp"
			}
			t.Run(name, func(t *testing.T) {
				e := newTestEditor(tt.content)
				if unclamped {
					e.SetBuffer(&unclampedBuffer{Buffer: NewBufferFromBytes([]byte(tt.content))})
				}
				b := e.GetBuffer()
				b.SetCursor(Cursor{Position: tt.cursor, Preferred: 7})
				tt.edit(b)
				assert.Equal(t, Cursor{Position: tt.want, Preferred: 7}, b.GetCursor())
			})
		}
	}
}

// TestHostEditKeepsCursorOnText checks the cursor after a host edits the text
// behind it through GetBuffer(). Expected results are from Vim 9.2, editing
// with setline() between normal-mode commands.
func TestHostEditKeepsCursorOnText(t *testing.T) {
	tests := []struct {
		name    string
		content string
		before  string
		edit    func(b Buffer)
		after   string
		want    string
		wantPos Position
	}{
		{"cursor line emptied", "abc\ndef\nghi", "Gl",
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(2, 0, 3)) },
			"", "abc\ndef\n", Position{2, 0}},
		{"then k keeps the column", "abc\ndef\nghi", "Gl",
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(2, 0, 3)) },
			"k", "abc\ndef\n", Position{1, 1}},
		{"x after the line is shortened under $", "abc\ndefgh", "$",
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 2, 1)) },
			"x", "a\ndefgh", Position{0, 0}},
		{"x sets the column", "abc\ndefgh", "$",
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 2, 1)) },
			"xj", "a\ndefgh", Position{1, 0}},
		{"j goes back to the column", "abcd\ndefgh", "3l",
			func(b Buffer) { require.Nil(t, b.DeleteRunesAt(0, 2, 2)) },
			"j", "ab\ndefgh", Position{1, 3}},
	}
	for _, tt := range tests {
		for _, unclamped := range []bool{false, true} {
			name := tt.name
			if unclamped {
				name += ", buffer that doesn't clamp"
			}
			t.Run(name, func(t *testing.T) {
				e := newTestEditor(tt.content)
				if unclamped {
					e.SetBuffer(&unclampedBuffer{Buffer: NewBufferFromBytes([]byte(tt.content))})
				}
				keys(e, []rune(tt.before)...)
				tt.edit(e.GetBuffer())
				keys(e, []rune(tt.after)...)
				assert.Equal(t, tt.want, content(e))
				assert.Equal(t, tt.wantPos, cursorPos(e))
			})
		}
	}
}

// TestHostEditInInsertMode checks that in insert mode the cursor may stay
// after the last character of a line the host shortened.
func TestHostEditInInsertMode(t *testing.T) {
	e := newTestEditor("abcdef")
	keys(e, 'A')
	require.Nil(t, e.GetBuffer().DeleteRunesAt(0, 2, 4))
	assert.Equal(t, Position{0, 2}, cursorPos(e))
	keys(e, 'x')
	assert.Equal(t, "abx", content(e))
	assert.Equal(t, Position{0, 3}, cursorPos(e))
}
