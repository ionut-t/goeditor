package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Visual mode (character-wise) ---

// TestVisualModeEnterExit tests entering and exiting character-wise visual mode.
func TestVisualModeEnterExit(t *testing.T) {
	t.Run("v enters visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		assert.True(t, e.IsVisualMode())
	})

	t.Run("Escape exits visual mode to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v while in visual mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'v')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V switches from visual to visual-line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'V')
		assert.True(t, e.IsVisualLineMode())
	})
}

// TestVisualModeDelete tests 'd'/'x' in character-wise visual mode.
func TestVisualModeDelete(t *testing.T) {
	t.Run("d deletes single selected char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'd')
		assert.Equal(t, "ello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v+l+l then d deletes 3 chars", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'd')
		assert.Equal(t, "lo world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+$ then d deletes to end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', '$', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+w then d deletes up to and including first char of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w', 'd')
		// w is a plain motion in visual mode: cursor lands on 'w' of "world" (col 6),
		// so the inclusive selection covers "hello w" (cols 0–6).
		assert.Equal(t, "orld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeDeleteMultiLine tests multi-line deletion in character-wise visual mode.
func TestVisualModeDeleteMultiLine(t *testing.T) {
	t.Run("v+j then d deletes from cursor to same col on next line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'v', 'j', 'd')
		// selection: {0,0}→{1,0}; deletes "first" from row 0 and 's' from row 1, then merges
		assert.Equal(t, "econd", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeYank tests 'y' in character-wise visual mode.
func TestVisualModeYank(t *testing.T) {
	t.Run("y yanks selected text to clipboard", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'y') // select "hello"
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, "hello world", content(e)) // content unchanged
	})

	t.Run("y with $ selects to end of line", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		keys(e, 'v', '$', 'y')
		assert.Equal(t, "hello", cb.content)
	})

	t.Run("y stays in visual mode when the copy fails", func(t *testing.T) {
		e := newTestEditor("abcdef") // no clipboard
		keys(e, 'v', 'l')
		assert.NotNil(t, e.HandleKey(KeyEvent{Rune: 'y'}))
		assert.True(t, e.IsVisualMode())
	})
}

// TestVisualYankEndsVisual checks that y in either visual mode returns to
// normal mode with the cursor where Vim leaves it: the start of a charwise
// selection; for a linewise one, column 0 of the anchor's line unless the
// cursor was above the anchor, where it stays.
func TestVisualYankEndsVisual(t *testing.T) {
	const text = "abcdef\nabcdef\nabcdef"
	tests := []struct {
		keys string
		want Position
	}{
		{"vly", Position{0, 0}},
		{"lllvhhy", Position{0, 1}},
		{"lllvy", Position{0, 3}},
		{"lllvjhy", Position{0, 3}},
		{"jlllvkly", Position{0, 4}},
		{"lllvhhyj", Position{1, 1}}, // j keeps the column y moved to
		{"lllVy", Position{0, 0}},
		{"lllVhy", Position{0, 0}},
		{"lllVly", Position{0, 0}},
		{"lllVjy", Position{0, 0}},
		{"lljVjy", Position{1, 0}},
		{"jjllVky", Position{1, 2}},
		{"jlllVkhy", Position{0, 2}},
		{"jjlllVkkly", Position{0, 4}},
		// After o the cursor and the anchor have swapped.
		{"Vjoky", Position{0, 0}},
		{"lllVjoy", Position{0, 3}},
		{"jlllVkoy", Position{0, 0}},
		{"jjlllVkoky", Position{1, 0}},
		{"lllvjhhoy", Position{0, 3}},
		{"jlllvkhhoy", Position{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(text)
			typeKeys(t, e, tt.keys)
			assert.True(t, e.IsNormalMode())
			assert.Equal(t, tt.want, cursorPos(e))
		})
	}

	t.Run("the yanked text stays highlighted", func(t *testing.T) {
		tests := []struct {
			keys string
			want SelectionType
			on   []Position // highlighted
			off  []Position // not highlighted
		}{
			{"lvly", SelectionCharacter, []Position{{0, 1}, {0, 2}}, []Position{{0, 0}, {0, 3}}},
			{"lllvjhy", SelectionCharacter, []Position{{0, 3}, {1, 2}}, []Position{{0, 2}, {1, 3}}},
			{"jVjy", SelectionLine, []Position{{1, 0}, {2, 5}}, []Position{{0, 0}}},
			{"jjVky", SelectionLine, []Position{{1, 0}, {2, 5}}, []Position{{0, 0}}},
		}
		for _, tt := range tests {
			t.Run(tt.keys, func(t *testing.T) {
				e, _ := newTestEditorWithClipboard(text)
				typeKeys(t, e, tt.keys)
				assert.Equal(t, tt.want, e.GetState().YankSelection)
				for _, p := range tt.on {
					assert.Equal(t, tt.want, e.GetSelectionStatus(p), "%v", p)
				}
				for _, p := range tt.off {
					assert.Equal(t, SelectionNone, e.GetSelectionStatus(p), "%v", p)
				}
			})
		}
	})

	t.Run("a new selection is not taken for the last yank", func(t *testing.T) {
		for _, yank := range []string{"yy", "Vy", "vly"} {
			e, cb := newTestEditorWithClipboard(text)
			typeKeys(t, e, yank+"vly")
			assert.Equal(t, "ab", cb.content, yank)
		}
	})

	t.Run("keys after y act in normal mode", func(t *testing.T) {
		// What a mapping such as Q → vlyx types in one go: x deletes one
		// character at the cursor, not the selection.
		e, cb := newTestEditorWithClipboard("abcdef")
		typeKeys(t, e, "vlyx")
		assert.Equal(t, "ab", cb.content)
		assert.Equal(t, "bcdef", content(e))
	})
}

// TestVisualOperatorResetsColumn checks that the visual operators leave j and
// k on the column the cursor ends on, not the one of the last horizontal
// motion, and where the linewise case operators put the cursor. Each case
// without a trailing j or k is followed by one with it. Expected results are
// from Vim 9.2.
func TestVisualOperatorResetsColumn(t *testing.T) {
	const six = "abcdef\nabcdef\nabcdef"
	const short = "abcdefghijkl\nab\nabcdefghijkl"
	const end = "abcdef\nabcdefgh"
	const indent = "hello world\n  second line\nthird line"
	tests := []struct {
		content string
		keys    string
		want    Position
		text    string
	}{
		{six, "lllvjhhd", Position{0, 3}, "abccdef\nabcdef"},
		{six, "lllvjhhdj", Position{1, 3}, ""},
		{six, "jlllvkhhd", Position{0, 1}, "aef\nabcdef"},
		{six, "jlllvkhhdj", Position{1, 1}, ""},
		{six, "lllvjhhxj", Position{1, 3}, "abccdef\nabcdef"},
		{short, "jj10lvkkhd", Position{0, 9}, "abcdefghil"},
		{short, "10lvjhdj", Position{1, 9}, "abcdefghij\nabcdefghijkl"},
		{end, "$vd", Position{0, 4}, "abcde\nabcdefgh"},
		{end, "$vdj", Position{1, 4}, ""},
		{six, "lllvjhhU", Position{0, 3}, "abcDEF\nABcdef\nabcdef"},
		{six, "lllvjhhUj", Position{1, 3}, ""},
		{six, "lllvjhhuj", Position{1, 3}, ""},
		{six, "lllvjhh~j", Position{1, 3}, "abcDEF\nABcdef\nabcdef"},
		{six, "jlllvkhhUj", Position{1, 1}, "aBCDEF\nABCDef\nabcdef"},
		{six, "yllllvjhhp", Position{0, 3}, "abcacdef\nabcdef"},
		{six, "yllllvjhhpj", Position{1, 3}, ""},
		{six, "yllllvjhhPj", Position{1, 3}, ""},
		{six, "3ylllvjhhp", Position{0, 4}, "ababcbcdef\nabcdef"},
		{six, "3ylllvjhhpj", Position{1, 4}, ""},
		{end, "yl$vp", Position{0, 5}, "abcdea\nabcdefgh"},
		{end, "yl$vpj", Position{1, 5}, ""},
		{six, "lllvjhhcX<Esc>j", Position{1, 3}, "abcXcdef\nabcdef"},
		{indent, "lllVd", Position{0, 2}, "  second line\nthird line"},
		{indent, "lllVdj", Position{1, 2}, ""},
		{indent, "lllVxj", Position{1, 2}, ""},
		{indent, "jjlllVd", Position{1, 2}, "hello world\n  second line"},
		{indent, "jlllVkdj", Position{0, 0}, "third line"},
		{indent, "jlllVU", Position{1, 0}, "hello world\n  SECOND LINE\nthird line"},
		{indent, "jlllVUj", Position{2, 0}, ""},
		{indent, "lllVu", Position{0, 0}, ""},
		{indent, "lllVuj", Position{1, 0}, ""},
		{indent, "lllVjU", Position{0, 0}, "HELLO WORLD\n  SECOND LINE\nthird line"},
		// Linewise u, U and ~ leave the cursor where it is when it is above
		// the anchor, as linewise y.
		{indent, "jlllVkU", Position{0, 3}, "HELLO WORLD\n  SECOND LINE\nthird line"},
		{indent, "jlllVkUj", Position{1, 3}, ""},
		{indent, "jlllVk~", Position{0, 3}, "HELLO WORLD\n  SECOND LINE\nthird line"},
		{indent, "jjlllVkU", Position{1, 3}, "hello world\n  SECOND LINE\nTHIRD LINE"},
		{indent, "jjlllVkkUj", Position{1, 3}, ""},
		{indent, "jjlllVkhhUj", Position{2, 1}, ""},
		{indent, "jyyklllVp", Position{0, 2}, "  second line\n  second line\nthird line"},
		{indent, "jyyklllVpj", Position{1, 2}, ""},
		{indent, "yyjjlllVp", Position{2, 0}, "hello world\n  second line\nhello world"},
		{indent, "yyjjlllVpk", Position{1, 0}, ""},
		{indent, "yyjlllVpj", Position{2, 0}, "hello world\nhello world\nthird line"},
		// After o the cursor and the anchor have swapped.
		{six, "VjokU", Position{0, 0}, "ABCDEF\nABCDEF\nabcdef"},
		{indent, "jlllVjokU", Position{0, 3}, "HELLO WORLD\n  SECOND LINE\nTHIRD LINE"},
		{indent, "jlllVjoU", Position{1, 3}, "hello world\n  SECOND LINE\nTHIRD LINE"},
		{indent, "jlllVjoUj", Position{2, 3}, ""},
		{indent, "jjlllVkoU", Position{1, 0}, "hello world\n  SECOND LINE\nTHIRD LINE"},
		{six, "lllvjhhoUj", Position{1, 3}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.True(t, e.IsNormalMode())
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
		})
	}
}

// TestVisualModeChange tests 'c' in character-wise visual mode.
func TestVisualModeChange(t *testing.T) {
	t.Run("c deletes selection and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c') // select "hello"
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c replaces selection", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c')
		keys(e, 'h', 'i')
		assert.Equal(t, "hi world", content(e))
	})
}

// TestVisualModeTextObjects tests text object selections in character-wise visual mode.
func TestVisualModeTextObjects(t *testing.T) {
	// viw / vaw

	t.Run("viw selects inside word from start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("viw selects inside word from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw selects around word including trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'a', 'w', 'd')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw on middle word includes leading space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'v', 'a', 'w', 'd')
		assert.Equal(t, "one three", content(e))
	})

	// vip / vap

	t.Run("vip switches to visual line mode", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("vip then d deletes paragraph", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vap then d deletes paragraph and surrounding blanks", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\n\nfoo")
		keys(e, 'v', 'a', 'p', 'd')
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip from mid-paragraph selects whole paragraph", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\n\nfoo")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip on blank line selects just the blank line", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "hello\nworld", content(e))
	})
}

// --- Visual line mode ---

// TestVisualLineModeEnterExit tests entering and exiting visual line mode.
func TestVisualLineModeEnterExit(t *testing.T) {
	t.Run("V enters visual line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("Escape exits to normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V while in visual-line mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'V')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v switches from visual-line to character-wise visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'v')
		assert.True(t, e.IsVisualMode())
	})
}

// TestCountV checks [count]V: it selects count lines, as far as the last
// one, going down as j does (the column j and k use is kept), as Vim does
// when there was no earlier Visual operation. A count typed in visual mode
// before V is dropped. Expected results are from Vim 9.2 (`-u NONE`). The
// clipboard starts as "REG"; after d and p, Vim's unnamed register gets the
// deleted or replaced text where goeditor's clipboard keeps what it held (the
// registers gap); those rows give Vim's register in a comment.
func TestCountV(t *testing.T) {
	const (
		four  = "abc\ndef\nghi\njkl"
		short = "abc\nd\nefg"
	)
	tests := []struct {
		content string
		keys    string
		want    Position
		col     int
		text    string
		reg     string
		mode    Mode
	}{
		{four, "yl2Vp", Position{0, 0}, 0, "a\nghi\njkl", "a", NormalMode}, // Vim: "abc\ndef\n"
		{four, "yl2VP", Position{0, 0}, 0, "a\nghi\njkl", "a", NormalMode},
		{four, "yl2Vpu", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "a", NormalMode}, // Vim: "abc\ndef\n"
		{four, "2Vd", Position{0, 0}, 0, "ghi\njkl", "REG", NormalMode},            // Vim: "abc\ndef\n"
		{four, "2Vy", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "abc\ndef\n", NormalMode},
		{four, "3Vd", Position{0, 0}, 0, "jkl", "REG", NormalMode},        // Vim: "abc\ndef\nghi\n"
		{four, "j3Vd", Position{0, 0}, 0, "abc", "REG", NormalMode},       // Vim: "def\nghi\njkl\n"
		{four, "jj3Vd", Position{1, 0}, 0, "abc\ndef", "REG", NormalMode}, // Vim: "ghi\njkl\n"
		{four, "2Vj", Position{2, 0}, 0, "abc\ndef\nghi\njkl", "REG", VisualLineMode},
		{four, "2Vk", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "REG", VisualLineMode},
		{four, "2Vo", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "REG", VisualLineMode},
		{four, "2Vd.", Position{0, 0}, 0, "", "REG", NormalMode},     // Vim: "ghi\njkl\n"
		{four, "2Vdj.", Position{0, 0}, 0, "ghi", "REG", NormalMode}, // Vim: "jkl\n"
		{four, "v2V", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "REG", VisualLineMode},
		{four, "V2V", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "REG", NormalMode},
		{four, "v2Vd", Position{0, 0}, 0, "def\nghi\njkl", "REG", NormalMode}, // Vim: "abc\n"
		{four, "2V", Position{1, 0}, 0, "abc\ndef\nghi\njkl", "REG", VisualLineMode},
		{four, "j2V<Esc>", Position{2, 0}, 0, "abc\ndef\nghi\njkl", "REG", NormalMode},
		{four, "yl2V2p", Position{0, 0}, 0, "a\na\nghi\njkl", "a", NormalMode}, // Vim: "abc\ndef\n"
		{four, "2Vjd", Position{0, 0}, 0, "jkl", "REG", NormalMode},            // Vim: "abc\ndef\nghi\n"
		{four, "G2Vd", Position{2, 0}, 0, "abc\ndef\nghi", "REG", NormalMode},  // Vim: "jkl\n"
		{four, "G2V<Esc>", Position{3, 0}, 0, "abc\ndef\nghi\njkl", "REG", NormalMode},
		{four, "Gl2Vd", Position{2, 0}, 0, "abc\ndef\nghi", "REG", NormalMode}, // Vim: "jkl\n"
		{four, "ll2V<Esc>", Position{1, 2}, 2, "abc\ndef\nghi\njkl", "REG", NormalMode},
		{four, "ll2Vy", Position{0, 0}, 0, "abc\ndef\nghi\njkl", "abc\ndef\n", NormalMode},
		{four, "5Vd", Position{0, 0}, 0, "", "REG", NormalMode},               // Vim: "abc\ndef\nghi\njkl\n"
		{four, "G5Vd", Position{2, 0}, 0, "abc\ndef\nghi", "REG", NormalMode}, // Vim: "jkl\n"
		{four, "2Vd.u", Position{0, 0}, 0, "ghi\njkl", "REG", NormalMode},     // Vim: "ghi\njkl\n"
		{four, "l2V<Esc>j", Position{2, 1}, 1, "abc\ndef\nghi\njkl", "REG", NormalMode},
		{four, "ll3Vd", Position{0, 0}, 0, "jkl", "REG", NormalMode}, // Vim: "abc\ndef\nghi\n"
		{short, "ll2V<Esc>", Position{1, 0}, 0, "abc\nd\nefg", "REG", NormalMode},
		{short, "ll2V<Esc>j", Position{2, 0}, 0, "abc\nd\nefg", "REG", NormalMode},
		{short, "ll3V<Esc>", Position{2, 2}, 2, "abc\nd\nefg", "REG", NormalMode},
		{short, "$2V<Esc>j", Position{2, 0}, 0, "abc\nd\nefg", "REG", NormalMode},
		{short, "ll2Vj", Position{2, 2}, 2, "abc\nd\nefg", "REG", VisualLineMode},
		{short, "$2Vj", Position{2, 3}, math.MaxInt, "abc\nd\nefg", "REG", VisualLineMode},
		{short, "ll2V", Position{1, 1}, 2, "abc\nd\nefg", "REG", VisualLineMode},
		{short, "$2V", Position{1, 1}, math.MaxInt, "abc\nd\nefg", "REG", VisualLineMode},
	}
	for _, tt := range tests {
		t.Run(tt.content+"/"+tt.keys, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.mode, e.GetState().Mode)
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.want, cursorPos(e))
			assert.Equal(t, tt.col, e.GetBuffer().GetCursor().Preferred, "column for j and k")
			assert.Equal(t, tt.reg, cb.content)
		})
	}
}

// TestVisualLineModeDelete tests 'd' in visual line mode.
func TestVisualLineModeDelete(t *testing.T) {
	t.Run("d deletes current line (single line buffer)", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("d deletes current line (multi-line buffer)", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'd')
		assert.Equal(t, "second\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j then d deletes two lines", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'd')
		assert.Equal(t, "third", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j+j then d deletes three lines", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'V', 'j', 'j', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("d from last line moves cursor to previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'V', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualLineModeYank tests 'y' in visual line mode.
func TestVisualLineModeYank(t *testing.T) {
	t.Run("y yanks current line with trailing newline", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'V', 'y')
		assert.Equal(t, "hello\n", cb.content)
		assert.Equal(t, "hello\nworld", content(e)) // unchanged
	})

	t.Run("V+j then y yanks two lines", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, 'V', 'j', 'y')
		assert.Equal(t, "one\ntwo\n", cb.content)
		assert.Equal(t, "one\ntwo\nthree", content(e)) // unchanged
	})
}

// TestVisualLineModeChange tests 'c' in visual line mode.
func TestVisualLineModeChange(t *testing.T) {
	t.Run("c empties selected lines to one and enters insert mode", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		assert.Equal(t, "\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c fills the emptied line", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		keys(e, 'n', 'e', 'w')
		assert.Equal(t, "new\nthird", content(e))
	})
}

// TestVisualLineTextObjectSelects checks text objects after V against Vim 9.2:
// ip/ap keep the selection linewise; every other object that is found switches
// to charwise visual mode and selects what it would there, from V's anchor
// (column included, so 2lViw selects the whole word and Vjiw extends). An
// object that isn't found keeps V; a word count that runs out still moves the
// cursor (V9aw). The column j and k go to is where the cursor ends.
func TestVisualLineTextObjectSelects(t *testing.T) {
	const (
		words  = "abc def\nghi jkl"
		blanks = "abc   def\nghi jkl"
		empty  = "abc\n\ndef"
		big    = "a.b c.d\nxy zw"
		parens = "f(a, b) x\nyz"
		pairs  = "f(a[b]c) {d} <e>\nz"
		block  = "f(\n  a\n  b\n)\nz"
		quotes = "a \"b c\" d\nx"
		paras  = "a b\nc d\n\ne f\ng h\n\ni"
		none   = "abc def\nghijklmn"
	)
	tests := []struct {
		content, keys  string
		mode           Mode
		anchor, cursor Position
		preferred      int
	}{
		{words, "Viw", VisualMode, Position{0, 0}, Position{0, 2}, 2},
		{words, "Vaw", VisualMode, Position{0, 0}, Position{0, 3}, 3},
		{words, "4lViw", VisualMode, Position{0, 4}, Position{0, 6}, 6},
		{words, "4lVaw", VisualMode, Position{0, 3}, Position{0, 6}, 6},
		{words, "2lViw", VisualMode, Position{0, 0}, Position{0, 2}, 2},
		{words, "$Viw", VisualMode, Position{0, 4}, Position{0, 6}, 6},
		{words, "$Vaw", VisualMode, Position{0, 3}, Position{0, 6}, 6},
		{words, "V2aw", VisualMode, Position{0, 0}, Position{0, 6}, 6},
		{words, "V4iw", VisualMode, Position{0, 0}, Position{1, 2}, 2},
		{words, "V9aw", VisualLineMode, Position{0, 0}, Position{1, 7}, 7},
		{words, "Vjiw", VisualMode, Position{0, 0}, Position{1, 2}, 2},
		{words, "4lVjiw", VisualMode, Position{0, 4}, Position{1, 6}, 6},
		{words, "j$Vkiw", VisualMode, Position{1, 6}, Position{0, 4}, 4},
		{words, "jVkiw", VisualLineMode, Position{1, 0}, Position{0, 0}, 0},
		{words, "$Vjiw", VisualLineMode, Position{0, 6}, Position{1, 7}, 7},
		{words, "Vjiwiw", VisualMode, Position{0, 0}, Position{1, 3}, 3},
		{words, "ViwV", VisualLineMode, Position{0, 0}, Position{0, 2}, 2},
		{words, "4lViwj", VisualMode, Position{0, 4}, Position{1, 6}, 6},
		{blanks, "3lViw", VisualMode, Position{0, 3}, Position{0, 5}, 5},
		{blanks, "3lVaw", VisualMode, Position{0, 3}, Position{0, 8}, 8},
		{empty, "jViw", VisualMode, Position{1, 0}, Position{1, 0}, 0},
		{empty, "jVaw", VisualMode, Position{1, 0}, Position{2, 2}, 2},
		{big, "ViW", VisualMode, Position{0, 0}, Position{0, 2}, 2},
		{big, "4lVaW", VisualMode, Position{0, 3}, Position{0, 6}, 6},

		{parens, "2lVi(", VisualMode, Position{0, 2}, Position{0, 5}, 5},
		{parens, "2lVa)", VisualMode, Position{0, 1}, Position{0, 6}, 6},
		{parens, "2lVab", VisualMode, Position{0, 1}, Position{0, 6}, 6},
		{parens, "Vi(", VisualMode, Position{0, 2}, Position{0, 5}, 5},
		{pairs, "4lVa[", VisualMode, Position{0, 3}, Position{0, 5}, 5},
		{pairs, "10lVaB", VisualMode, Position{0, 9}, Position{0, 11}, 11},
		{pairs, "14lVa>", VisualMode, Position{0, 13}, Position{0, 15}, 15},
		{block, "jVi(", VisualMode, Position{1, 0}, Position{2, 3}, 3},
		{block, "jVa(", VisualMode, Position{0, 1}, Position{3, 0}, 0},
		{quotes, "3lVi\"", VisualMode, Position{0, 3}, Position{0, 5}, 5},
		{quotes, "3lVa\"", VisualMode, Position{0, 2}, Position{0, 7}, 7},
		{quotes, "lVi\"", VisualMode, Position{0, 3}, Position{0, 5}, 5},
		{"a 'b c' d\nx", "3lVi'", VisualMode, Position{0, 3}, Position{0, 5}, 5},
		{"a ``\nx", "2lVi`", VisualMode, Position{0, 3}, Position{0, 2}, 2},
		{quotes, "3lViq", VisualMode, Position{0, 3}, Position{0, 5}, 5}, // iq isn't in Vim

		{paras, "Vip", VisualLineMode, Position{0, 0}, Position{1, 0}, 0},
		{paras, "Vap", VisualLineMode, Position{0, 0}, Position{2, 0}, 0},
		{paras, "jVip", VisualLineMode, Position{0, 0}, Position{1, 0}, 0},
		{paras, "$Vip", VisualLineMode, Position{0, 2}, Position{1, 0}, 0},
		{paras, "$Vipj", VisualLineMode, Position{0, 2}, Position{2, 0}, 0},
		{paras, "$vip", VisualLineMode, Position{0, 2}, Position{1, 0}, 0},
		{paras, "3jVap", VisualLineMode, Position{3, 0}, Position{5, 0}, 0},
		{paras, "6jVap", VisualLineMode, Position{5, 0}, Position{6, 0}, 0},
		{"x\nabcdef\n\ny", "j$Vip", VisualLineMode, Position{0, 0}, Position{1, 0}, 0},

		{none, "2lVi(", VisualLineMode, Position{0, 2}, Position{0, 2}, 2},
		{none, "$Va\"", VisualLineMode, Position{0, 6}, Position{0, 6}, 6},
		{none, "2lVib", VisualLineMode, Position{0, 2}, Position{0, 2}, 2},
		{"f()\nz", "lVi(", VisualLineMode, Position{0, 1}, Position{0, 1}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.mode, e.GetMode().Name())
			assert.Equal(t, tt.anchor, e.GetState().VisualStart, "anchor")
			assert.Equal(t, tt.cursor, cursorPos(e), "cursor")
			assert.Equal(t, tt.preferred, e.GetBuffer().GetCursor().Preferred, "column for j/k")
		})
	}
}

// TestVisualLineTextObjectThenOperator checks the edits after a text object
// typed in V against Vim 9.2: they are charwise, as after v, including undo
// and the size . repeats, except after ip/ap or an object that isn't found.
func TestVisualLineTextObjectThenOperator(t *testing.T) {
	const (
		text  = "abc def ghi\njkl mno pqr\nstu"
		calls = "f(a, b) x\ng(c, d) y"
		block = "f(\n  a\n  b\n)\nz"
		paras = "a b\nc d\n\ne f\ng h\n\ni"
	)
	tests := []struct {
		content, keys, want string
		cursor              Position
	}{
		{text, "4lViwd", "abc  ghi\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwy", text, Position{0, 4}},
		{text, "4lViwcX<Esc>", "abc X ghi\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwp", "abc REG ghi\njkl mno pqr\nstu", Position{0, 6}},
		{text, "4lVawd", "abc ghi\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwU", "abc DEF ghi\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwdu", text, Position{0, 4}},
		{text, "4lViwdu<C-r>", "abc  ghi\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwd.", "abc i\njkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwdj.", "abc  ghi\njkl  pqr\nstu", Position{1, 4}},
		{text, "4lViwdj0.", "abc  ghi\n mno pqr\nstu", Position{1, 0}},
		{text, "4lVawd.", "abc jkl mno pqr\nstu", Position{0, 4}},
		{text, "4lViwjd", "abc  pqr\nstu", Position{0, 4}},
		{text, "4lViwVd", "jkl mno pqr\nstu", Position{0, 0}},
		{text, "Vjiwd", " mno pqr\nstu", Position{0, 0}},
		{text, "Vjiwd.", "", Position{0, 0}},
		{"abc def\nghi jkl", "V9awd", "", Position{0, 0}},
		{calls, "2lVi(d", "f() x\ng(c, d) y", Position{0, 2}},
		{calls, "2lVi(dj.", "f() x\ng() y", Position{1, 2}},
		{calls, "2lVi(cQ<Esc>j.", "f(Q) x\ng(Q) y", Position{1, 2}},
		{block, "jVi(d", "f(\n)\nz", Position{1, 0}},
		{block, "jVa(d", "f\nz", Position{0, 0}},
		{paras, "Vipd", "\ne f\ng h\n\ni", Position{0, 0}},
		{paras, "Vipd.", "g h\n\ni", Position{0, 0}},
		{paras, "jVapd", "e f\ng h\n\ni", Position{0, 0}},
		{"abc def\nghijklmn", "lVi(d", "ghijklmn", Position{0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.Equal(t, NormalMode, e.GetMode().Name())
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, tt.cursor, cursorPos(e))
		})
	}

	t.Run("yank puts the object in the register", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard(text)
		typeKeys(t, e, "4lViwy")
		assert.Equal(t, "def", cb.content)
	})
}

// TestVisualModeMovementSequences tests that navigation keys work after shared-motion keys.
func TestVisualModeMovementSequences(t *testing.T) {
	t.Run("v+j+l: j then l both move cursor", func(t *testing.T) {
		e := newTestEditor("abcd\nefgh")
		keys(e, 'v', 'j') // enter visual, move down
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		keys(e, 'l') // move right
		assert.Equal(t, Position{1, 1}, cursorPos(e))
	})
	t.Run("v+w: word forward moves cursor to start of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
	t.Run("v+w+w: second w advances to start of second next word", func(t *testing.T) {
		e := newTestEditor("hello world foo")
		keys(e, 'v', 'w', 'w')
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})
	t.Run("v+e: word end moves to last char of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("v+b: word backward from middle moves to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'v', 'b') // cursor at col 3, then v+b
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
	t.Run("V+j: j in visual line mode extends selection", func(t *testing.T) {
		e := newTestEditor("line1\nline2\nline3")
		keys(e, 'V', 'j', 'd') // visual line, extend down, delete 2 lines
		assert.Equal(t, "line3", content(e))
	})
	t.Run("V+w: w moves cursor forward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world\nfoo")
		keys(e, 'V', 'w') // enter visual line, move forward one word
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assert.True(t, e.IsVisualLineMode())
	})
	t.Run("V+e: e moves cursor to word end in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'V', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("V+b: b moves cursor word backward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'V', 'b') // cursor at col 8 (mid "world"), then V+b → start of "world"
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

func TestVisualGGWithCountGoesToLine(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo\nthree\nfour\nfive\nsix")

			keys(e, mode, '4', 'g', 'g')

			assert.Equal(t, 3, cursorPos(e).Row)
			assert.Nil(t, e.GetState().PendingCount)
		})
	}
}

func TestVisualGWithCountGoesToLine(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo\nthree\nfour\nfive\nsix")

			keys(e, mode, '3', 'G')
			assert.Equal(t, 2, cursorPos(e).Row)
			assert.Nil(t, e.GetState().PendingCount)

			keys(e, 'G')
			assert.Equal(t, 5, cursorPos(e).Row, "G with no count goes to the last line")
		})
	}
}

func TestVisualEnterMovesDown(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\n  two\n    three\n four\nfive")
			setCursorRow(e, 1)

			keys(e, mode)
			enter(e)
			assert.Equal(t, Position{2, 4}, cursorPos(e), "no count moves down one line")

			keys(e, '2')
			enter(e)
			assert.Equal(t, Position{4, 0}, cursorPos(e), "count moves down count lines")
			assert.Nil(t, e.GetState().PendingCount)
			assert.Equal(t, Position{Row: 1, Col: 0}, e.GetState().VisualStart)
		})
	}
}

func TestPrefixDoesNotSurviveModeSwitch(t *testing.T) {
	e := newTestEditor("one\ntwo\nthree\nfour")
	setCursorRow(e, 2)

	keys(e, 'g')
	e.SetInsertMode()
	escape(e)
	keys(e, 'g')

	assert.Equal(t, 2, cursorPos(e).Row, "the g typed before the switch must not complete a gg")
}

func TestVisualTextObjectKeyIsNotAPrefix(t *testing.T) {
	e := newTestEditor("one\ntwo three\nfour")
	setCursorRow(e, 1)

	keys(e, 'v', 'i', 'g', 'w')

	assert.Equal(t, Position{Row: 1, Col: 4}, cursorPos(e), "g after i is the (unknown) text object, so w is a plain motion")
	assert.Equal(t, Position{Row: 1, Col: 0}, e.GetState().VisualStart)
}

func TestPrefixShowsPendingCount(t *testing.T) {
	e := newTestEditor("one\ntwo")
	keys(e, '1', '0', 'g')
	assert.Equal(t, "10g", e.GetState().CommandLine)
	escape(e)

	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo")
			keys(e, mode, '1', '0', 'z')
			assert.Equal(t, "10z", e.GetState().CommandLine)
		})
	}
}

// Re-entering visual mode (e.g. a host refocusing) must not reset the anchor.
func TestSetVisualModeWhileInVisualKeepsSelection(t *testing.T) {
	e := newTestEditor("abc")
	keys(e, 'v', 'l')
	e.SetVisualMode()
	assert.True(t, e.IsVisualMode())
	assert.Equal(t, Position{0, 0}, e.GetState().VisualStart)
	assert.Equal(t, Position{0, 1}, cursorPos(e))
}
