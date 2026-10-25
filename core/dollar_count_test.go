package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Buffers for the [count]$ operator tests.
const (
	countLong   = "abcdefghijkl\nab\nabcdefghijkl\nhello world"
	countGap    = "abc\n\ndef\nghi"
	countIndent = "  abc\n  def\nghi"
	countTwo    = "abc\ndef"
)

// TestOperatorCountOnDollar checks counts on d$, c$, y$, D, C and gU$/gu$/g~$
// against Vim 9.2: they go through the end of the line count-1 lines down
// (the last line if there are fewer), charwise and not including its line
// break; a count on the operator multiplies the one on $. A delete from the
// indent over several lines deletes the lines (Vim's op_delete); c and y
// stay charwise. On the last line a count above 1 fails as $ does: nothing
// changes, and j and k then go to the end of every line. d and c don't write
// the clipboard in goeditor yet, so those rows don't check it.
func TestOperatorCountOnDollar(t *testing.T) {
	tests := []struct {
		name    string
		content string
		keys    string
		want    Position
		col     int    // the column j and k go to; -1 for $ (MAXCOL)
		text    string // buffer after the keys
		yanked  string // clipboard after the keys (set to "REG" first); empty to skip
	}{
		{"long 5l2d$", countLong, "5l2d$", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5ld2$", countLong, "5ld2$", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5l2D", countLong, "5l2D", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5l2y$", countLong, "5l2y$", Position{0, 5}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "fghijkl\nab"},
		{"long 5l2y$j", countLong, "5l2y$j", Position{1, 1}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "fghijkl\nab"},
		{"long 5lc2$X<Esc>", countLong, "5lc2$X<Esc>", Position{0, 5}, 5, "abcdeX\nabcdefghijkl\nhello world", ""},
		{"long 5l2CX<Esc>", countLong, "5l2CX<Esc>", Position{0, 5}, 5, "abcdeX\nabcdefghijkl\nhello world", ""},
		{"long 5l2C<Esc>", countLong, "5l2C<Esc>", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5l3D", countLong, "5l3D", Position{0, 4}, 4, "abcde\nhello world", ""},
		{"long 5l9D", countLong, "5l9D", Position{0, 4}, 4, "abcde", ""},
		{"long 5l2d3$", countLong, "5l2d3$", Position{0, 4}, 4, "abcde", ""},
		{"long 5l2D.", countLong, "5l2D.", Position{0, 3}, 3, "abcd\nhello world", ""},
		{"long 5l2Dj.", countLong, "5l2Dj.", Position{1, 3}, 3, "abcde\nabcd", ""},
		{"long 5lDj2.", countLong, "5lDj2.", Position{1, 0}, 0, "abcde\na\nhello world", ""},
		{"long 5l2Du", countLong, "5l2Du", Position{0, 5}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", ""},
		{"long 5l2gU$", countLong, "5l2gU$", Position{0, 5}, 5, "abcdeFGHIJKL\nAB\nabcdefghijkl\nhello world", "REG"},
		{"long 5lgU2$", countLong, "5lgU2$", Position{0, 5}, 5, "abcdeFGHIJKL\nAB\nabcdefghijkl\nhello world", "REG"},
		{"long 5l2g~$", countLong, "5l2g~$", Position{0, 5}, 5, "abcdeFGHIJKL\nAB\nabcdefghijkl\nhello world", "REG"},
		{"long 5l2gu$", countLong, "5l2gu$", Position{0, 5}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "REG"},
		{"long jjj2D", countLong, "jjj2D", Position{3, 0}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", ""},
		{"long jjj5l2D", countLong, "jjj5l2D", Position{3, 5}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", ""},
		{"long jjj5l2y$", countLong, "jjj5l2y$", Position{3, 5}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "REG"},
		{"long jjj5l2C<Esc>", countLong, "jjj5l2C<Esc>", Position{3, 5}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", ""},
		{"long jjj2d$", countLong, "jjj2d$", Position{3, 0}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", ""},
		{"long 2D", countLong, "2D", Position{0, 0}, 0, "abcdefghijkl\nhello world", ""},
		{"long 2d$", countLong, "2d$", Position{0, 0}, 0, "abcdefghijkl\nhello world", ""},
		{"long 2C<Esc>", countLong, "2C<Esc>", Position{0, 0}, 0, "\nabcdefghijkl\nhello world", ""},
		{"long 2y$", countLong, "2y$", Position{0, 0}, 0, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "abcdefghijkl\nab"},
		{"long j2D", countLong, "j2D", Position{1, 0}, 0, "abcdefghijkl\nhello world", ""},
		{"long $2D", countLong, "$2D", Position{0, 10}, 10, "abcdefghijk\nabcdefghijkl\nhello world", ""},
		{"long 5l2Dj", countLong, "5l2Dj", Position{1, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5l2Dk", countLong, "5l2Dk", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long 5l2y$k", countLong, "5l2y$k", Position{0, 5}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "fghijkl\nab"},
		{"long 5l2CX<Esc>j", countLong, "5l2CX<Esc>j", Position{1, 5}, 5, "abcdeX\nabcdefghijkl\nhello world", ""},
		{"long 5l1D", countLong, "5l1D", Position{0, 4}, 4, "abcde\nab\nabcdefghijkl\nhello world", ""},
		{"long 5l2c$<Esc>", countLong, "5l2c$<Esc>", Position{0, 4}, 4, "abcde\nabcdefghijkl\nhello world", ""},
		{"long jjj5l2gU$", countLong, "jjj5l2gU$", Position{3, 5}, -1, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "REG"},
		{"long 5l9y$", countLong, "5l9y$", Position{0, 5}, 5, "abcdefghijkl\nab\nabcdefghijkl\nhello world", "fghijkl\nab\nabcdefghijkl\nhello world"},
		{"long 5l2D2.", countLong, "5l2D2.", Position{0, 3}, 3, "abcd\nhello world", ""},
		{"long 2gU$", countLong, "2gU$", Position{0, 0}, 0, "ABCDEFGHIJKL\nAB\nabcdefghijkl\nhello world", "REG"},
		{"gap 2D", countGap, "2D", Position{0, 0}, 0, "def\nghi", ""},
		{"gap l2D", countGap, "l2D", Position{0, 0}, 0, "a\ndef\nghi", ""},
		{"gap 2C<Esc>", countGap, "2C<Esc>", Position{0, 0}, 0, "\ndef\nghi", ""},
		{"gap j2D", countGap, "j2D", Position{1, 0}, 0, "abc\nghi", ""},
		{"gap j2C<Esc>", countGap, "j2C<Esc>", Position{1, 0}, 0, "abc\n\nghi", ""},
		{"gap j2y$", countGap, "j2y$", Position{1, 0}, 0, "abc\n\ndef\nghi", "\ndef"},
		{"gap l2y$", countGap, "l2y$", Position{0, 1}, 1, "abc\n\ndef\nghi", "bc\n"},
		{"gap l3D", countGap, "l3D", Position{0, 0}, 0, "a\nghi", ""},
		{"gap l2gU$", countGap, "l2gU$", Position{0, 1}, 1, "aBC\n\ndef\nghi", "REG"},
		{"gap j3D", countGap, "j3D", Position{0, 0}, 0, "abc", ""},
		{"gap jj2D", countGap, "jj2D", Position{1, 0}, 0, "abc\n", ""},
		{"ind 2D", countIndent, "2D", Position{0, 0}, 0, "ghi", ""},
		{"ind l2D", countIndent, "l2D", Position{0, 0}, 0, "ghi", ""},
		{"ind ll2D", countIndent, "ll2D", Position{0, 0}, 0, "ghi", ""},
		{"ind 3l2D", countIndent, "3l2D", Position{0, 2}, 2, "  a\nghi", ""},
		{"ind ll2C<Esc>", countIndent, "ll2C<Esc>", Position{0, 1}, 1, "  \nghi", ""},
		{"ind ll2y$", countIndent, "ll2y$", Position{0, 2}, 2, "  abc\n  def\nghi", "abc\n  def"},
		{"ind ll2d$", countIndent, "ll2d$", Position{0, 0}, 0, "ghi", ""},
		{"ind 2gU$", countIndent, "2gU$", Position{0, 0}, 0, "  ABC\n  DEF\nghi", "REG"},
		{"ind j2D", countIndent, "j2D", Position{0, 2}, 2, "  abc", ""},
		{"ind jll2D", countIndent, "jll2D", Position{0, 2}, 2, "  abc", ""},
		{"two j2D", countTwo, "j2D", Position{1, 0}, -1, "abc\ndef", ""},
		{"two j2y$", countTwo, "j2y$", Position{1, 0}, -1, "abc\ndef", "REG"},
		{"two j2C<Esc>", countTwo, "j2C<Esc>", Position{1, 0}, -1, "abc\ndef", ""},
		{"two 2D", countTwo, "2D", Position{0, 0}, 0, "", ""},
		{"two l2D", countTwo, "l2D", Position{0, 0}, 0, "a", ""},
		{"two l5D", countTwo, "l5D", Position{0, 0}, 0, "a", ""},
		{"two l2y$", countTwo, "l2y$", Position{0, 1}, 1, "abc\ndef", "bc\ndef"},
		{"two j2gU$", countTwo, "j2gU$", Position{1, 0}, -1, "abc\ndef", "REG"},
		{"two j$2D", countTwo, "j$2D", Position{1, 2}, -1, "abc\ndef", ""},
		{"two 2CX<Esc>", countTwo, "2CX<Esc>", Position{0, 0}, 0, "X", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.Equal(t, NormalMode, e.GetMode().Name())
			assert.Equal(t, tt.want, cursorPos(e))
			col := e.GetBuffer().GetCursor().Preferred
			if col == maxCol {
				col = -1
			}
			assert.Equal(t, tt.col, col, "column for j and k")
			assert.Equal(t, tt.text, content(e))
			if tt.yanked != "" {
				assert.Equal(t, tt.yanked, cb.content)
			}
		})
	}
}

// TestDeleteToLineEndSignal checks that d$ and D send the text they delete,
// over several lines too, and that a linewise one sends it once.
func TestDeleteToLineEndSignal(t *testing.T) {
	tests := []struct {
		name    string
		content string
		keys    string
		want    []string
	}{
		{"D", countLong, "5lD", []string{"fghijkl"}},
		{"2D", countLong, "5l2D", []string{"fghijkl\nab"}},
		{"d2$", countLong, "5ld2$", []string{"fghijkl\nab"}},
		{"2D through an empty line", countGap, "l2D", []string{"bc\n"}},
		{"2D from the indent deletes the lines", countIndent, "ll2D", []string{"  abc\n  def\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			var got []string
			for s := nextSignal(e); s != nil; s = nextSignal(e) {
				if d, ok := s.(DeleteSignal); ok {
					got = append(got, d.Value())
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
