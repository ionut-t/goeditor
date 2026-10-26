package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPasteCharacterWise tests 'p' after a character-wise yank.
// Content is inserted AFTER the cursor char — matching Vim's 'p' behaviour.
// The cursor lands on the last character of the pasted text.
func TestPasteCharacterWise(t *testing.T) {
	t.Run("yw then p inserts after cursor char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'w') // yank "hello " to clipboard; cursor stays at col 0 ('h')
		assert.Equal(t, "hello ", cb.content)
		keys(e, 'p') // insert "hello " after 'h' → "hhello ello world"
		assert.Equal(t, "hhello ello world", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("ye then p inserts word after cursor char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello" (no trailing space); cursor at col 0
		assert.Equal(t, "hello", cb.content)
		keys(e, 'p') // insert "hello" after 'h' → "hhelloello world"
		assert.Equal(t, "hhelloello world", content(e))
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("p mid-line inserts after cursor char", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'w')      // move to col 6 ('w' of "world")
		keys(e, 'p')      // insert "hello" after 'w' → "hello whelloorld"
		assert.Equal(t, "hello whelloorld", content(e))
		assert.Equal(t, Position{0, 11}, cursorPos(e))
	})
}

// TestPasteCharacterWiseOnEmptyLine tests 'p' and 'P' on an empty line.
// On an empty line there is no character to insert after, so the text is inserted at column 0.
func TestPasteCharacterWiseOnEmptyLine(t *testing.T) {
	t.Run("p on empty line inserts text at col 0", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\n\nworld")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'j')      // move to empty line (row 1)
		keys(e, 'p')      // paste on empty line
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 4}, cursorPos(e))
	})

	t.Run("P on empty line inserts text at col 0", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\n\nworld")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'j')      // move to empty line (row 1)
		keys(e, 'P')      // paste before on empty line
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 4}, cursorPos(e))
	})
}

// TestPasteLinewise tests 'p' after a line-wise yank ('yy').
// Linewise paste inserts the yanked line below the current line and moves the cursor
// to the first non-blank of the newly inserted line — matching Vim's behaviour.
func TestPasteLinewise(t *testing.T) {
	t.Run("yy then p on single line duplicates it below", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy then p on first of two lines inserts copy between them", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'y', 'y', 'p')
		assert.Equal(t, "first\nfirst\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy on second line then p duplicates it below", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y', 'p')
		assert.Equal(t, "first\nsecond\nsecond", content(e))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("cursor column is irrelevant: paste always goes below current line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, '$', 'y', 'y', 'p') // cursor at end of line 0; should still paste below row 0
		assert.Equal(t, "hello\nhello\nworld", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("repeated p pastes on successive lines", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'p', 'p') // yy then p, p
		assert.Equal(t, "hello\nhello\nhello", content(e))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("count: 3p pastes three copies below, cursor on the first", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', '3', 'p')
		assert.Equal(t, "hello\nhello\nhello\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestPasteLinewiseFirstNonBlank checks that linewise p and P, in normal and
// visual-line mode, put the cursor on the first non-blank of the first pasted
// line, and that j and k keep that column. Expected results are from Vim 9.2.
func TestPasteLinewiseFirstNonBlank(t *testing.T) {
	const text = "hello world\n  second line\n    third line\nfourth"
	tests := []struct {
		keys string
		want Position
		text string
	}{
		{"jyyp", Position{2, 2}, "hello world\n  second line\n  second line\n    third line\nfourth"},
		{"jyypj", Position{3, 2}, ""},
		{"jyyP", Position{1, 2}, "hello world\n  second line\n  second line\n    third line\nfourth"},
		{"jyyPj", Position{2, 2}, ""},
		{"jyyGp", Position{4, 2}, "hello world\n  second line\n    third line\nfourth\n  second line"},
		{"jyyggP", Position{0, 2}, "  second line\nhello world\n  second line\n    third line\nfourth"},
		{"jyyklllp", Position{1, 2}, ""},
		{"jyyklllP", Position{0, 2}, ""},
		{"yyjp", Position{2, 0}, "hello world\n  second line\nhello world\n    third line\nfourth"},
		// Multi-line: the first non-blank of the first pasted line.
		{"jVjyp", Position{2, 2}, "hello world\n  second line\n  second line\n    third line\n    third line\nfourth"},
		{"jVjypj", Position{3, 2}, ""},
		{"jVjyP", Position{1, 2}, "hello world\n  second line\n    third line\n  second line\n    third line\nfourth"},
		{"jVjyGp", Position{4, 2}, "hello world\n  second line\n    third line\nfourth\n  second line\n    third line"},
		{"jjVkyggp", Position{1, 2}, "hello world\n  second line\n    third line\n  second line\n    third line\nfourth"},
		{"yjjp", Position{2, 0}, "hello world\n  second line\nhello world\n  second line\n    third line\nfourth"},
		// Counts: the copies stay in order and the cursor goes to the first.
		{"jyy3p", Position{2, 2}, "hello world\n  second line\n  second line\n  second line\n  second line\n    third line\nfourth"},
		{"jyy3pj", Position{3, 2}, ""},
		{"jyy3P", Position{1, 2}, "hello world\n  second line\n  second line\n  second line\n  second line\n    third line\nfourth"},
		{"jVjy2p", Position{2, 2}, "hello world\n  second line\n  second line\n    third line\n  second line\n    third line\n    third line\nfourth"},
		{"jVjy2P", Position{1, 2}, "hello world\n  second line\n    third line\n  second line\n    third line\n  second line\n    third line\nfourth"},
		// Visual-line mode.
		{"jyyklllVp", Position{0, 2}, "  second line\n  second line\n    third line\nfourth"},
		{"jyyklllVP", Position{0, 2}, "  second line\n  second line\n    third line\nfourth"},
		{"jVjyklllVp", Position{0, 2}, "  second line\n    third line\n  second line\n    third line\nfourth"},
		{"jyyjVjp", Position{2, 2}, "hello world\n  second line\n  second line"},
		{"jyyGVp", Position{3, 2}, "hello world\n  second line\n    third line\n  second line"},
		{"jyyGVpk", Position{2, 2}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(text)
			typeKeys(t, e, tt.keys)
			assert.True(t, e.IsNormalMode())
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
		})
	}
}

// TestPasteCharacterwiseCount checks charwise p and P with a count, multi-line
// charwise text, and where they leave the cursor: on the last pasted char for
// text within a line, on the first for text spanning lines. Expected results
// are from Vim 9.2.
func TestPasteCharacterwiseCount(t *testing.T) {
	const text = "hello world\nsecond line\nthird"
	tests := []struct {
		content string
		keys    string
		want    Position
		text    string
	}{
		{text, "yl3p", Position{0, 3}, "hhhhello world\nsecond line\nthird"},
		{text, "yl3pj", Position{1, 3}, ""},
		{text, "yl3P", Position{0, 2}, "hhhhello world\nsecond line\nthird"},
		{text, "y2l3p", Position{0, 6}, "hheheheello world\nsecond line\nthird"},
		{text, "y2l3P", Position{0, 5}, "hehehehello world\nsecond line\nthird"},
		{text, "ye3p", Position{0, 15}, "hhellohellohelloello world\nsecond line\nthird"},
		{text, "ye3P", Position{0, 14}, "hellohellohellohello world\nsecond line\nthird"},
		{text, "ylp", Position{0, 1}, "hhello world\nsecond line\nthird"},
		{text, "ylP", Position{0, 0}, "hhello world\nsecond line\nthird"},
		{text, "yl$3p", Position{0, 13}, "hello worldhhh\nsecond line\nthird"},
		{text, "yl2p.", Position{0, 4}, "hhhhhello world\nsecond line\nthird"},
		{text, "yl3pu", Position{0, 0}, text},
		// Text spanning lines: the cursor goes to its first char.
		{text, "lvjyp", Position{0, 2}, "heello world\nsello world\nsecond line\nthird"},
		{text, "lvjyP", Position{0, 1}, "hello world\nseello world\nsecond line\nthird"},
		{text, "lvjy2p", Position{0, 2}, "heello world\nseello world\nsello world\nsecond line\nthird"},
		{text, "lvjy2pj", Position{1, 2}, ""},
		{text, "lvjy2P", Position{0, 1}, "hello world\nseello world\nseello world\nsecond line\nthird"},
		{text, "lvjy2pu", Position{0, 1}, text},
		// Visual p: the same cursor rules.
		{text, "lvjyjjlllvlp", Position{2, 4}, "hello world\nsecond line\nthirello world\nse"},
		{text, "lvjyjjlllvlpk", Position{1, 4}, ""},
		{text, "yejjlllvlp", Position{2, 7}, "hello world\nsecond line\nthihello"},
		// Runes, not bytes.
		{"héllo", "lylp", Position{0, 2}, "hééllo"},
		{"héllo", "lylP", Position{0, 1}, "hééllo"},
		// Empty line: the text goes at col 0.
		{"hello\n\nworld", "yej3p", Position{1, 14}, "hello\nhellohellohello\nworld"},
		{"hello\n\nworld", "yej3P", Position{1, 14}, "hello\nhellohellohello\nworld"},
	}
	for _, tt := range tests {
		t.Run(tt.content+"/"+tt.keys, func(t *testing.T) {
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

// TestPasteCountLimit checks that a count that would paste more than
// maxRepeatedInsert runes pastes nothing and fails with ErrTextTooLong, as the
// insert count limit does.
func TestPasteCountLimit(t *testing.T) {
	tests := []struct {
		name, yank, count, msg string
	}{
		{"charwise", "yl", "10000001p", "resulting text too long: 10000001 copies of 1 rune"},
		{"charwise before", "yl", "10000001P", "resulting text too long: 10000001 copies of 1 rune"},
		{"linewise", "yy", "1666667p", "resulting text too long: 1666667 copies of 6 runes"},
		{"linewise before", "yy", "1666667P", "resulting text too long: 1666667 copies of 6 runes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard("hello")
			typeKeys(t, e, tt.yank+tt.count[:len(tt.count)-1])
			err := e.HandleKey(KeyEvent{Rune: rune(tt.count[len(tt.count)-1])})
			require.NotNil(t, err)
			assert.Equal(t, ErrTextTooLongId, err.ID())
			assert.True(t, errors.Is(err, ErrTextTooLong))
			assert.Equal(t, tt.msg, err.Error())
			assert.Equal(t, "hello", content(e))
			assert.Equal(t, Position{0, 0}, cursorPos(e))
		})
	}

	t.Run("one copy has no limit", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		cb.content = strings.Repeat("x", 10_000_001)
		require.Nil(t, e.HandleKey(KeyEvent{Rune: 'p'}))
		assert.Equal(t, 10_000_006, e.GetBuffer().LineRuneCount(0))
	})

	t.Run("up to the limit pastes", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		typeKeys(t, e, "yl10000000p")
		assert.Equal(t, 10_000_005, e.GetBuffer().LineRuneCount(0))
		assert.Equal(t, Position{0, 10_000_000}, cursorPos(e))
	})
}

// TestPasteCharacterWiseBefore tests 'P' after a character-wise yank.
// Content is inserted at the cursor position (same column) — the cursor lands on the last pasted character.
func TestPasteCharacterWiseBefore(t *testing.T) {
	t.Run("ye then P inserts word at cursor position", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello" (no trailing space)
		assert.Equal(t, "hello", cb.content)
		keys(e, 'P')
		assert.Equal(t, "hellohello world", content(e))
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("P mid-line inserts at cursor column", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'e') // yank "hello"
		keys(e, 'w')      // move to col 6 ("world")
		keys(e, 'P')      // paste "hello" at col 6
		assert.Equal(t, "hello helloworld", content(e))
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})
}

// TestPasteLinewiseBefore tests 'P' after a line-wise yank ('yy').
// Linewise P inserts the yanked line above the current line; cursor stays at the newly inserted line.
func TestPasteLinewiseBefore(t *testing.T) {
	t.Run("yy then P on single line inserts copy above", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'P')
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("yy then P on second line inserts copy above it", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'j', 'y', 'y', 'P') // yank "second", paste above row 1
		assert.Equal(t, "first\nsecond\nsecond", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("yy then P on first of two lines inserts copy at top", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond")
		keys(e, 'y', 'y', 'P') // yank "first", paste above row 0
		assert.Equal(t, "first\nfirst\nsecond", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("repeated P pastes accumulate above original line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', 'P', 'P') // yy then P, P
		assert.Equal(t, "hello\nhello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 3P pastes three copies above", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello")
		keys(e, 'y', 'y', '3', 'P')
		assert.Equal(t, "hello\nhello\nhello\nhello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestPasteOverSelection tests 'p' in visual character mode (v).
// The selected text is deleted and the clipboard is pasted in its place.
func TestPasteOverSelection(t *testing.T) {
	t.Run("p over character-wise selection replaces it", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'v', 'e')
		keys(e, 'p')
		assert.Equal(t, "bye world", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("p over mid-line selection replaces selected word", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'w', 'v', 'e') // move to "world", select it
		keys(e, 'p')
		assert.Equal(t, "hello bye", content(e))
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("p over single char at end of line", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, '$', 'v', 'p') // select last char 'd', paste
		assert.Equal(t, "hello worlbye", content(e))
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})
}

// TestPasteBeforeOverSelection tests 'P' in visual modes — identical behaviour to 'p'.
func TestPasteBeforeOverSelection(t *testing.T) {
	t.Run("P over character-wise selection replaces it", func(t *testing.T) {
		e, cp := newTestEditorWithClipboard("hello world")
		cp.content = "bye"
		keys(e, 'v', 'e', 'P')
		assert.Equal(t, "bye world", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("VP replaces line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'j', 'y', 'y') // yank "world\n" from row 1
		keys(e, 'k', 'V', 'P') // select row 0, paste
		assert.Equal(t, "world\nworld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestPasteLinewiseOverSelection tests 'p' in visual line mode (V).
// The selected lines are deleted and the clipboard is pasted in their place.
func TestPasteLinewiseOverSelection(t *testing.T) {
	t.Run("Vp replaces last line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'y', 'y')      // yank "hello\n" from row 0
		keys(e, 'j', 'V', 'p') // select row 1, paste
		assert.Equal(t, "hello\nhello", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("Vp replaces first line with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'j', 'y', 'y') // yank "world\n" from row 1
		keys(e, 'k', 'V', 'p') // select row 0, paste
		assert.Equal(t, "world\nworld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e)) // cursor on the pasted line
	})

	t.Run("Vp puts pasted content at the selected row, not below it", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("first\nsecond\nthird")
		keys(e, 'j', 'j', 'y', 'y') // yank "third\n" from row 2
		keys(e, 'k', 'k', 'V', 'p') // select row 0, paste
		assert.Equal(t, "third\nsecond\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("Vjp replaces multi-line selection with yanked line", func(t *testing.T) {
		e, _ := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, 'y', 'y')           // yank "one\n" from row 0
		keys(e, 'j', 'V', 'j', 'p') // select rows 1+2, paste
		assert.Equal(t, "one\none", content(e))
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestPasteCharwiseOverLines checks p and P over a linewise selection: text
// within a line, or spanning lines, replaces the lines as lines of its own
// instead of being joined onto the next line, and the cursor, u and <C-r>
// go to column 0 of the first pasted line. Replacing every line leaves no
// empty line behind, with linewise text too. Expected results are from Vim
// 9.2; counts and . after visual p are left out (they differ from Vim on
// their own).
func TestPasteCharwiseOverLines(t *testing.T) {
	const (
		three    = "abcdef\nabcdef\nabcdef"
		one      = "abcdef"
		indented = "  abc\n    def\nghi\njkl"
		four     = "abc\ndef\nghi\njkl"
		blank    = "abc\n\ndef"
	)
	tests := []struct {
		content string
		keys    string
		want    Position
		col     int
		text    string
	}{
		{three, "ylVp", Position{0, 0}, 0, "a\nabcdef\nabcdef"},
		{three, "ylVP", Position{0, 0}, 0, "a\nabcdef\nabcdef"},
		{three, "yljVp", Position{1, 0}, 0, "abcdef\na\nabcdef"},
		{three, "ylGVp", Position{2, 0}, 0, "abcdef\nabcdef\na"},
		{three, "ylVjp", Position{0, 0}, 0, "a\nabcdef"},
		{three, "ylVGp", Position{0, 0}, 0, "a"},
		{three, "ylggVGp", Position{0, 0}, 0, "a"},
		{three, "ylVpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "ylVjpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "ylVpu<C-r>", Position{0, 0}, 0, "a\nabcdef\nabcdef"},
		{three, "yeVp", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "y$Vp", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "lllyeVp", Position{0, 0}, 0, "def\nabcdef\nabcdef"},
		{three, "ylVpj", Position{1, 0}, 0, "a\nabcdef\nabcdef"},
		{three, "ylVpk", Position{0, 0}, 0, "a\nabcdef\nabcdef"},
		{three, "ylVpjk", Position{0, 0}, 0, "a\nabcdef\nabcdef"},
		{four, "lvjyVp", Position{0, 0}, 0, "bc\nde\ndef\nghi\njkl"},
		{four, "lvjyjjVp", Position{2, 0}, 0, "abc\ndef\nbc\nde\njkl"},
		{four, "lvjyGVp", Position{3, 0}, 0, "abc\ndef\nghi\nbc\nde"},
		{four, "lvjyVjp", Position{0, 0}, 0, "bc\nde\nghi\njkl"},
		{four, "lvjyVpu", Position{0, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "lvjyVP", Position{0, 0}, 0, "bc\nde\ndef\nghi\njkl"},
		{four, "lvjyggVGp", Position{0, 0}, 0, "bc\nde"},
		{four, "lvjyVpj", Position{1, 0}, 0, "bc\nde\ndef\nghi\njkl"},
		{four, "lvjyVpu<C-r>", Position{0, 0}, 0, "bc\nde\ndef\nghi\njkl"},
		{four, "llvjyVp", Position{0, 0}, 0, "c\ndef\ndef\nghi\njkl"},
		{four, "vjhyVp", Position{0, 0}, 0, "abc\nd\ndef\nghi\njkl"},
		{indented, "jwyeVp", Position{1, 0}, 0, "  abc\ndef\nghi\njkl"},
		{indented, "jwyekVp", Position{0, 0}, 0, "def\n    def\nghi\njkl"},
		{indented, "jwyeGVp", Position{3, 0}, 0, "  abc\n    def\nghi\ndef"},
		{indented, "ylVp", Position{0, 0}, 0, " \n    def\nghi\njkl"},
		{indented, "jwyejVp", Position{2, 0}, 0, "  abc\n    def\ndef\njkl"},
		{indented, "wvjyjjVp", Position{2, 0}, 0, "  abc\n    def\nabc\n   \njkl"},
		{one, "ylVp", Position{0, 0}, 0, "a"},
		{one, "yeVp", Position{0, 0}, 0, "abcdef"},
		{one, "ylVpu", Position{0, 0}, 0, "abcdef"},
		{one, "ylVP", Position{0, 0}, 0, "a"},
		{one, "lvlyVp", Position{0, 0}, 0, "bc"},
		{three, "yyVp", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{blank, "yljVp", Position{1, 0}, 0, "abc\na\ndef"},
		{three, "yyjVpu", Position{1, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "yyVGp", Position{0, 0}, 0, "abcdef"},
		{three, "jyyggVGp", Position{0, 0}, 0, "abcdef"},
		{three, "yyVGpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "ylVGpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "ylVGpu<C-r>", Position{0, 0}, 0, "a"},
		{three, "yyVGP", Position{0, 0}, 0, "abcdef"},
		{three, "ylggVGpj", Position{0, 0}, 0, "a"},
		{three, "jyyggVGpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{one, "yyVp", Position{0, 0}, 0, "abcdef"},
		{one, "yyVP", Position{0, 0}, 0, "abcdef"},
		{one, "yyVpu", Position{0, 0}, 0, "abcdef"},
		{one, "ylVpu<C-r>", Position{0, 0}, 0, "a"},
		{one, "yyVpj", Position{0, 0}, 0, "abcdef"},
		{one, "lvlyVpu", Position{0, 0}, 0, "abcdef"},
		{indented, "wyeggVGp", Position{0, 0}, 0, "abc"},
		{indented, "jwyeggVGp", Position{0, 0}, 0, "def"},
		{indented, "wvjyggVGp", Position{0, 0}, 0, "abc\n   "},
		{indented, "wvjyggVGpu", Position{0, 0}, 0, "  abc\n    def\nghi\njkl"},
		{four, "lvjyjjVpu", Position{2, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "lvjyGVpu", Position{3, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "lvjyVjpu", Position{0, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "ylGVpu", Position{3, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "yljjVpu", Position{2, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "yyjjVpu", Position{2, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "jyyVjpu", Position{1, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "ylGllVkpu", Position{2, 2}, 2, "abc\ndef\nghi\njkl"},
		{four, "yljjlVkkpu", Position{0, 1}, 1, "abc\ndef\nghi\njkl"},
		{four, "yyjlVkpu", Position{0, 1}, 1, "abc\ndef\nghi\njkl"},
		{four, "lvjyjjlVkpu", Position{1, 2}, 2, "abc\ndef\nghi\njkl"},
		{four, "ylllVpu", Position{0, 0}, 0, "abc\ndef\nghi\njkl"},
		{four, "ylllVpu<C-r>", Position{0, 0}, 0, "a\ndef\nghi\njkl"},
		{four, "yljjlVkpu<C-r>", Position{1, 0}, 0, "abc\na\njkl"},
		{four, "lvjyGlVggpu", Position{0, 0}, 0, "abc\ndef\nghi\njkl"},
		{three, "ylllVpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "yylllVGpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
		{three, "ylGlllVggpu", Position{0, 0}, 0, "abcdef\nabcdef\nabcdef"},
	}
	for _, tt := range tests {
		t.Run(tt.content+"/"+tt.keys, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.True(t, e.IsNormalMode())
			assert.Equal(t, tt.text, content(e))
			assert.Equal(t, tt.want, cursorPos(e))
			assert.Equal(t, tt.col, e.GetBuffer().GetCursor().Preferred, "column for j and k")
		})
	}
}

type failClipboard struct{}

func (failClipboard) Write(string) error    { return nil }
func (failClipboard) Read() (string, error) { return "", errors.New("no clipboard") }

// TestPasteWithoutClipboard checks p and P when the clipboard can't be read,
// or there is none. Vim (nothing in the register, E353) still deletes a visual
// selection, leaves the cursor on a character, and j then uses the column it
// ends on. Expected values are from Vim 9.2.
func TestPasteWithoutClipboard(t *testing.T) {
	const text = "hello world\nsecond line long\n  third\nfourth line"
	tests := []struct {
		name   string
		keys   string
		text   string
		want   Position
		afterJ Position
	}{
		{"$vp", "$vp", "hello worl\nsecond line long\n  third\nfourth line", Position{0, 9}, Position{1, 9}},
		{"$vP", "$vP", "hello worl\nsecond line long\n  third\nfourth line", Position{0, 9}, Position{1, 9}},
		{"lvlp", "lvlp", "hlo world\nsecond line long\n  third\nfourth line", Position{0, 1}, Position{1, 1}},
		{"jlllvjp", "jlllvjp", "hello world\nsecird\nfourth line", Position{1, 3}, Position{2, 3}},
		{"Vp", "Vp", "second line long\n  third\nfourth line", Position{0, 0}, Position{1, 0}},
		{"lllVp", "lllVp", "second line long\n  third\nfourth line", Position{0, 0}, Position{1, 0}},
		{"jVp", "jVp", "hello world\n  third\nfourth line", Position{1, 2}, Position{2, 2}},
		{"jjjVp last line", "jjjVp", "hello world\nsecond line long\n  third", Position{2, 2}, Position{2, 2}},
		{"jlllVjP", "jlllVjP", "hello world\nfourth line", Position{1, 0}, Position{1, 0}},
		{"normal p", "lllp", text, Position{0, 3}, Position{1, 3}},
		{"normal P", "lllP", text, Position{0, 3}, Position{1, 3}},
	}
	clipboards := []struct {
		name string
		cb   Clipboard
	}{
		{"failing", failClipboard{}},
		{"nil", nil},
	}
	for _, c := range clipboards {
		for _, tt := range tests {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				e := New(c.cb)
				e.SetContent([]byte(text))
				var err *EditorError
				for _, r := range tt.keys {
					err = e.HandleKey(KeyEvent{Rune: r})
				}
				require.NotNil(t, err)
				assert.Equal(t, ErrFailedToPasteId, err.ID())
				assert.True(t, e.IsNormalMode())
				assert.Equal(t, tt.text, content(e))
				assert.Equal(t, tt.want, cursorPos(e))
				keys(e, 'j')
				assert.Equal(t, tt.afterJ, cursorPos(e))
			})
		}
	}
}
