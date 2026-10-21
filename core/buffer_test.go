package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteRunesAtAcrossLines pins down deletions that cross line breaks,
// which undo relies on to remove exactly the text it inserted.
func TestDeleteRunesAtAcrossLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		row, col int
		count    int
		want     string
	}{
		{"line break only", "a\nb\nc", 0, 1, 1, "ab\nc"},
		{"ends at the end of the next line", "a\nb\nc", 0, 1, 2, "a\nc"},
		{"ends inside the next line", "a\nbc\nd", 0, 1, 2, "ac\nd"},
		{"spans whole lines", "a\nb\nc\nd", 0, 1, 4, "a\nd"},
		{"ends at the end of the buffer", "a\nb", 0, 1, 2, "a"},
		{"runs past the end of the buffer", "a\nb", 0, 1, 10, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferFromBytes([]byte(tt.content))
			require.Nil(t, b.DeleteRunesAt(tt.row, tt.col, tt.count))
			assert.Equal(t, tt.want, b.GetCurrentContent())
		})
	}
}

// TestSetContentEmpty checks that empty content gives one empty line, like
// NewBuffer, so the buffer can still be edited. Expected values are from Vim
// 9.2 on an empty file.
func TestSetContentEmpty(t *testing.T) {
	t.Run("buffer", func(t *testing.T) {
		for _, content := range [][]byte{nil, {}} {
			b := NewBufferFromBytes(content)
			assert.Equal(t, 1, b.LineCount())
			assert.True(t, b.IsEmpty())
			assert.Equal(t, []string{""}, b.GetLines())
		}
	})

	t.Run("replacing content", func(t *testing.T) {
		b := NewBufferFromBytes([]byte("hello\nworld"))
		b.SetContent(nil)
		assert.Equal(t, 1, b.LineCount())
		assert.True(t, b.IsEmpty())
	})

	tests := []struct {
		name string
		keys string
		text string
		want Position
	}{
		{"ihi<Esc>", "ihi\x1b", "hi", Position{0, 1}},
		{"ohi<Esc>", "ohi\x1b", "\nhi", Position{1, 1}},
		{"Ohi<Esc>", "Ohi\x1b", "hi\n", Position{0, 1}},
		{"dd", "dd", "", Position{0, 0}},
		{"G", "G", "", Position{0, 0}},
		{"ihi<Esc>u", "ihi\x1bu", "", Position{0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard("")
			for _, r := range tt.keys {
				if r == '\x1b' {
					escape(e)
					continue
				}
				require.Nil(t, e.HandleKey(KeyEvent{Rune: r}))
			}
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.want, cursorPos(e))
			assert.Equal(t, 1+strings.Count(tt.text, "\n"), e.GetBuffer().LineCount())
		})
	}
}
