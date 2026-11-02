package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOperatorHLAcrossLines checks dh, dl, yh, yl, ch, cl and X with
// SetMoveAcrossLines against Vim 9.2 with 'set whichwrap+=h,l'. From column
// 0, d and c go past the end of the line above and join the lines, while y
// stops on its last character; the last character of a line is one step of
// l and its line break the next, so $d2l deletes only that character.
func TestOperatorHLAcrossLines(t *testing.T) {
	const (
		two   = "abc\ndef"
		blank = "abc\n\ndef"
		four  = "abc\ndef\nghi\njkl"
	)
	tests := []struct {
		keys, content string
		text          string
		pos           Position
		reg           string // the clipboard after the keys, for y; "" to skip
	}{
		{"$dl", two, "ab\ndef", Position{0, 1}, ""},
		{"$d2l", two, "ab\ndef", Position{0, 1}, ""},
		{"$d3l", two, "abef", Position{0, 2}, ""},
		{"$d4l", two, "abf", Position{0, 2}, ""},
		{"ld9l", two, "a", Position{0, 0}, ""},
		{"$d2l", blank, "ab\n\ndef", Position{0, 1}, ""},
		{"jdl", blank, "abc\ndef", Position{1, 0}, ""},
		{"jd2l", blank, "abc\nef", Position{1, 0}, ""},
		{"jdh", two, "abcdef", Position{0, 3}, ""},
		{"jd2h", two, "abdef", Position{0, 2}, ""},
		{"jld3h", two, "abef", Position{0, 2}, ""},
		{"jd4h", two, "def", Position{0, 0}, ""},
		{"jd9h", two, "def", Position{0, 0}, ""},
		{"j$d9h", two, "f", Position{0, 0}, ""},
		{"jdh", blank, "abc\ndef", Position{0, 2}, ""},
		{"jjdh", blank, "abc\ndef", Position{1, 0}, ""},
		{"jjd2h", blank, "abcdef", Position{0, 3}, ""},
		{"jjd4h", four, "abc\nghi\njkl", Position{1, 0}, ""},
		{"jjd5h", four, "abcghi\njkl", Position{0, 3}, ""},
		{"X", two, two, Position{0, 0}, ""},
		{"jX", two, "abcdef", Position{0, 3}, ""},
		{"j2X", two, "abdef", Position{0, 2}, ""},
		{"jlX", two, "abc\nef", Position{1, 0}, ""},
		{"jl3X", two, "abef", Position{0, 2}, ""},
		{"jjX", blank, "abc\ndef", Position{1, 0}, ""},
		{"jj2X", blank, "abcdef", Position{0, 3}, ""},
		{"$y2l", two, two, Position{0, 2}, "c"},
		{"$y3l", two, two, Position{0, 2}, "c\nd"},
		{"$y9l", two, two, Position{0, 2}, "c\ndef"},
		{"jyh", two, two, Position{0, 2}, "c"},
		{"jy2h", two, two, Position{0, 1}, "bc"},
		{"jly3h", two, two, Position{0, 1}, "bc\nd"},
		{"jy4h", two, two, Position{0, 0}, "abc\n"},
		{"jjyh", blank, blank, Position{1, 0}, "\n"},
		{"jyl", blank, blank, Position{1, 0}, "\n"},
		{"jy2l", blank, blank, Position{1, 0}, "\nd"},
		{"$c2lX<Esc>", two, "abX\ndef", Position{0, 2}, ""},
		{"$c3lX<Esc>", two, "abXef", Position{0, 2}, ""},
		{"jchX<Esc>", two, "abcXdef", Position{0, 3}, ""},
		{"jc2hX<Esc>", two, "abXdef", Position{0, 2}, ""},
		{"jc4hX<Esc>", two, "Xdef", Position{0, 0}, ""},
		{"jjchX<Esc>", blank, "abc\nX\ndef", Position{1, 0}, ""},
		{"jjc2hX<Esc>", blank, "abcXdef", Position{0, 3}, ""},
		{"jclX<Esc>", blank, "abc\nX\ndef", Position{1, 0}, ""},
		{"jc2lX<Esc>", blank, "abc\nXef", Position{1, 0}, ""},
		{"$d3lj$.", four, "abef\nghkl", Position{1, 2}, ""},
		{"jdhj.", four, "abcdef\ngi\njkl", Position{1, 1}, ""},
		{"jXj.", four, "abcdef\ngi\njkl", Position{1, 1}, ""},
		{"jchX<Esc>j.", four, "abcXdef\ngXi\njkl", Position{1, 1}, ""},
		{"jjdh.", "abc\n\ndef\n\nghi", "abcdef\n\nghi", Position{0, 3}, ""},
		{"jdl.", blank, "abc\nef", Position{1, 0}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			e.SetMoveAcrossLines(true)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.pos, cursorPos(e))
			if tt.reg != "" {
				assert.Equal(t, tt.reg, cb.content)
			}
		})
	}

	// Without the option they stay on the line, as with Vim's default.
	for _, tt := range []struct {
		keys, content, text string
		pos                 Position
	}{
		{"$d3l", two, "ab\ndef", Position{0, 1}},
		{"jdh", two, two, Position{1, 0}},
		{"jjdh", blank, blank, Position{2, 0}},
		{"jX", two, two, Position{1, 0}},
		{"jl3X", two, "abc\nef", Position{1, 0}},
		{"jchX<Esc>", two, "abc\nXdef", Position{1, 0}},
	} {
		t.Run("default "+tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.pos, cursorPos(e))
		})
	}
}

// TestCaseHLAcrossLines checks gU, gu and g~ with h and l and
// SetMoveAcrossLines against Vim 9.2 with 'set whichwrap+=h,l'. They take
// the same steps as dh and dl, except that h stops on the last character of
// the line above, as y does; an end in column 0 of a later line goes back to
// the end of the line above, so a range over only line breaks changes
// nothing. The cursor goes to the start of the range.
func TestCaseHLAcrossLines(t *testing.T) {
	const (
		two    = "abc\ndef"
		blank  = "abc\n\ndef"
		three  = "abc\ndef\nghi"
		indent = "  abc\ndef"
	)
	tests := []struct {
		keys, content, text string
		pos                 Position
	}{
		{"$gU3l", two, "abC\nDef", Position{0, 2}},
		{"$gUl", two, "abC\ndef", Position{0, 2}},
		{"$gU2l", two, "abC\ndef", Position{0, 2}},
		{"lgU5l", two, "aBC\nDEf", Position{0, 1}},
		{"lgU9l", two, "aBC\nDEF", Position{0, 1}},
		{"jgUh", two, "abC\ndef", Position{0, 2}},
		{"jgU2h", two, "aBC\ndef", Position{0, 1}},
		{"jlgU3h", two, "aBC\nDef", Position{0, 1}},
		{"jgU9h", two, "ABC\ndef", Position{0, 0}},
		{"j$gU9h", two, "ABC\nDEf", Position{0, 0}},
		{"$gUl", blank, "abC\n\ndef", Position{0, 2}},
		{"$gU2l", blank, "abC\n\ndef", Position{0, 2}},
		{"$gU3l", blank, "abC\n\ndef", Position{0, 2}},
		{"jgUl", blank, blank, Position{1, 0}},
		{"jgU2l", blank, "abc\n\nDef", Position{1, 0}},
		{"jgUh", blank, "abC\n\ndef", Position{0, 2}},
		{"jgU2h", blank, "aBC\n\ndef", Position{0, 1}},
		{"2jgUh", blank, blank, Position{1, 0}},
		{"2jgU2h", blank, "abC\n\ndef", Position{0, 2}},
		{"2jgU3h", blank, "aBC\n\ndef", Position{0, 1}},
		{"jgU3h", indent, "  ABC\ndef", Position{0, 2}},
		{"jgU9h", indent, "  ABC\ndef", Position{0, 0}},
		{"llgU9l", indent, "  ABC\nDEF", Position{0, 2}},
		{"gU9l", indent, "  ABC\nDEF", Position{0, 0}},
		{"Gg~9l", two, "abc\nDEF", Position{1, 0}},
		{"$g~3l", two, "abC\nDef", Position{0, 2}},
		{"jg~h", "ABC\ndef", "ABc\ndef", Position{0, 2}},
		{"$gu3l", "ABC\nDEF", "ABc\ndEF", Position{0, 2}},
		{"$gU2lj.", three, "abC\ndeF\nghi", Position{1, 2}},
		{"jgU2hj.", three, "aBC\nDef\nghi", Position{0, 2}},
		{"lgU5lu", three, three, Position{0, 1}},
		{"jgU2hu", three, three, Position{0, 1}},
		{"lgU5luj", three, three, Position{1, 1}},
		{"lgU5lj", three, "aBC\nDEf\nghi", Position{1, 1}},
		{"$gU3lj", three, "abC\nDef\nghi", Position{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			e.SetMoveAcrossLines(true)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.pos, cursorPos(e))
		})
	}

	// Without the option they stay on the line, as with Vim's default.
	for _, tt := range []struct {
		keys, content, text string
		pos                 Position
	}{
		{"$gU3l", two, "abC\ndef", Position{0, 2}},
		{"jlgU3h", two, "abc\nDef", Position{1, 0}},
		{"lgU5l", two, "aBC\ndef", Position{0, 1}},
		{"jgUl", blank, blank, Position{1, 0}},
	} {
		t.Run("default "+tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.pos, cursorPos(e))
		})
	}
}
