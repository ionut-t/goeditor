package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReplaceChar tests 'r{char}' — replace character under cursor without entering insert mode.
func TestReplaceChar(t *testing.T) {
	t.Run("replaces char under cursor", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r', 'X')
		assert.Equal(t, "Xello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("replaces char at mid-line position", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'r', 'Z') // cursor at col 2; replace 'l' with 'Z'
		assert.Equal(t, "heZlo", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("Escape after r cancels replace, stays in normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r')
		escape(e) // cancel replace
		assert.Equal(t, "hello", content(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("on empty line does nothing", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'r', 'X') // move to blank line; r is a no-op
		assert.Equal(t, "hello\n\nworld", content(e))
	})

	t.Run("count replaces that many chars, cursor on the last", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', 'r', 'X')
		assert.Equal(t, "XXXlo", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("count past the line end replaces nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', '4', 'r', 'X')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("count too large to fit replaces nothing", func(t *testing.T) {
		e := newTestEditor("hello")
		typeKeys(t, e, "l99999999999999999999rX")
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 1}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("undo restores original char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'r', 'X')
		assert.Equal(t, "Xello", content(e))
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))
	})
}

// TestReplaceLineBreak checks r<CR> and r CTRL-J against Vim 9.2 (without
// 'autoindent'): count characters (a wide or composing character counts as
// one) become a single line break, no white space around it is removed, and
// the cursor goes to the start of the new line. With too few characters left,
// or on an empty line, nothing changes and the column kept for j and k stays
// (the line end after $). Undo goes back to where r was, redo to the end of
// the first line, and '.' repeats the count unless given a new one.
func TestReplaceLineBreak(t *testing.T) {
	runVisualEditCases(t, []visualEditCase{
		{"", "r<CR>", "", Position{0, 0}, 0},
		{"abc def", "r<CR>", "\nbc def", Position{1, 0}, 0},
		{"abc def", "lr<CR>", "a\nc def", Position{1, 0}, 0},
		{"abc def", "3lr<CR>", "abc\ndef", Position{1, 0}, 0},
		{"abc def", "$r<CR>", "abc de\n", Position{1, 0}, 0},
		{"abc def\nxyz", "$r<CR>", "abc de\n\nxyz", Position{1, 0}, 0},
		{"a", "r<CR>", "\n", Position{1, 0}, 0},
		{"x\n\ny", "jr<CR>", "x\n\ny", Position{1, 0}, 0},
		{"abc\nlast", "jlr<CR>", "abc\nl\nst", Position{2, 0}, 0},
		{"abc\nlast", "j$r<CR>", "abc\nlas\n", Position{2, 0}, 0},
		{"abc def", "2r<CR>", "\nc def", Position{1, 0}, 0},
		{"abc def", "l2r<CR>", "a\n def", Position{1, 0}, 0},
		{"abc", "5r<CR>", "abc", Position{0, 0}, 0},
		{"abc", "l3r<CR>", "abc", Position{0, 1}, 1},
		{"abc", "l2r<CR>", "a\n", Position{1, 0}, 0},
		{"abc def", "$3r<CR>", "abc def", Position{0, 6}, maxCol},
		{"abc def", "7r<CR>", "\n", Position{1, 0}, 0},
		{"abc def", "8r<CR>", "abc def", Position{0, 0}, 0},
		{"abc   def", "2lr<CR>", "ab\n   def", Position{1, 0}, 0},
		{"abc \t def", "2lr<CR>", "ab\n \t def", Position{1, 0}, 0},
		{"abc   def", "3lr<CR>", "abc\n  def", Position{1, 0}, 0},
		{"abc   def", "4lr<CR>", "abc \n def", Position{1, 0}, 0},
		{"abc \t def", "4lr<CR>", "abc \n def", Position{1, 0}, 0},
		{"abc   ", "2lr<CR>", "ab\n   ", Position{1, 0}, 0},
		{"abc   ", "$r<CR>", "abc  \n", Position{1, 0}, 0},
		{"abc   ", "3lr<CR>", "abc\n  ", Position{1, 0}, 0},
		{"abc  def", "3l2r<CR>", "abc\ndef", Position{1, 0}, 0},
		{"abc   def", "2l2r<CR>", "ab\n  def", Position{1, 0}, 0},
		{"    abc def", "^r<CR>", "    \nbc def", Position{1, 0}, 0},
		{"    abc def", "wwr<CR>", "    abc \nef", Position{1, 0}, 0},
		{"    abc def", "r<CR>", "\n   abc def", Position{1, 0}, 0},
		{"\t\tabc def", "wwr<CR>", "\t\tabc \nef", Position{1, 0}, 0},
		{"aéb", "lr<CR>", "a\nb", Position{1, 0}, 0},
		{"a日b", "lr<CR>", "a\nb", Position{1, 0}, 0},
		{"a日b", "l2r<CR>", "a\n", Position{1, 0}, 0},
		{"ae\u0301b", "lr<CR>", "a\nb", Position{1, 0}, 0},
		{"a😀b c", "lr<CR>", "a\nb c", Position{1, 0}, 0},
		{"a日b", "r<CR>", "\n日b", Position{1, 0}, 0},
		{"abc def", "lr<C-j>", "a\nc def", Position{1, 0}, 0},
		{"abc def", "l2r<C-j>", "a\n def", Position{1, 0}, 0},
		{"abc def", "lr<CR>u", "abc def", Position{0, 1}, 1},
		{"abc def", "lr<CR>u<C-r>", "a\nc def", Position{0, 0}, 0},
		{"abc def", "l2r<CR>u", "abc def", Position{0, 1}, 1},
		{"abc def", "l2r<CR>u<C-r>", "a\n def", Position{0, 0}, 0},
		{"abc def ghi", "lr<CR>.", "a\n\n def ghi", Position{2, 0}, 0},
		{"abc def ghi", "lr<CR>l.", "a\nc\ndef ghi", Position{2, 0}, 0},
		{"abc def ghi", "lr<CR>j.", "a\n\n def ghi", Position{2, 0}, 0},
		{"abc def ghi", "lr<CR>3.", "a\n\nef ghi", Position{2, 0}, 0},
		{"abc def ghi", "l2r<CR>.", "a\n\nef ghi", Position{2, 0}, 0},
		{"abc def ghi", "l2r<CR>3.", "a\n\nf ghi", Position{2, 0}, 0},
		{"abc def ghi", "l2r<CR>w.", "a\n \nf ghi", Position{2, 0}, 0},
		{"abc def ghi\nxyz uvw", "lr<CR>jj.", "a\nc def ghi\n\nyz uvw", Position{3, 0}, 0},
		{"abc def ghi\nxyz uvw", "lr<CR>j.u", "a\nc def ghi\nxyz uvw", Position{2, 0}, 0},
		{"abc def ghi", "lr<CR>.u", "a\nc def ghi", Position{1, 0}, 0},
		{"abc def ghi", "lr<CR>3.u", "a\nc def ghi", Position{1, 0}, 0},
		{"abc def ghi", "l2r<CR>3.u", "a\n def ghi", Position{1, 0}, 0},
		{"abc def ghi", "lr<CR>uu", "abc def ghi", Position{0, 1}, 1},
		{"ab\ncd", "lr<CR>j.", "a\n\n\nd", Position{3, 0}, 0},
		{"abc\ndef", "jlr<CR>ku", "abc\ndef", Position{1, 1}, 1},
		{"abc def\nxyzwvu", "lr<CR>k", "a\nc def\nxyzwvu", Position{0, 0}, 0},
		{"abc def\nxyzwvu", "lr<CR>j", "a\nc def\nxyzwvu", Position{2, 0}, 0},
		{"abc def\nxyzwvu", "lr<CR>uj", "abc def\nxyzwvu", Position{1, 1}, 1},
		{"abc def\nxyz", "lr<CR>u<C-r>j", "a\nc def\nxyz", Position{1, 0}, 0},
		{"abc def", "lr<CR>u.", "a\nc def", Position{1, 0}, 0},
		{"abc def", "l2r<CR>u.", "a\n def", Position{1, 0}, 0},
		{"abc def", "l2r<CR>0.", "a\n\nef", Position{2, 0}, 0},
		{"abc def", "lrx0r<CR>u", "axc def", Position{0, 0}, 0},
		{"abc def", "lr<CR>ix<Esc>u", "a\nc def", Position{1, 0}, 0},
		{"a", "2r<CR>", "a", Position{0, 0}, 0},
		{"a b", "2lr<CR>", "a \n", Position{1, 0}, 0},
		{"abc def", "lr<CR>ix<Esc>.", "a\nxxc def", Position{1, 0}, 0},
		{"  abc def", "4lr<CR>u<C-r>", "  ab\n def", Position{0, 3}, 3},
		{"abc def\nxy", "$3r<CR>j", "abc def\nxy", Position{1, 1}, maxCol},
		{"  abc def", "4lr<CR>u", "  abc def", Position{0, 4}, 4},
	})
}

// TestReplaceLineBreakReadOnly checks that r<CR> changes nothing with insert
// mode disabled.
func TestReplaceLineBreakReadOnly(t *testing.T) {
	e := newTestEditor("abc def")
	e.DisableInsertMode(true)
	typeKeys(t, e, "lr<CR>")
	assert.Equal(t, "abc def", content(e))
	assert.Equal(t, Position{0, 1}, cursorPos(e))
}

// TestVisualReplace checks r{char} after v and V against Vim 9.2: every
// selected character becomes one {char} (a tab, a wide character, a letter
// with its composing characters alike), line breaks stay, the count is
// ignored, and the cursor goes to the selection start (column 0 after V).
// Undo goes back there too, except after V with the cursor above the anchor,
// where it goes back to the cursor, as after Vd. '.' replaces as much from the cursor. A selection
// of only empty lines changes nothing but is still the change '.' repeats.
func TestVisualReplace(t *testing.T) {
	const (
		words = "abc def\nghi jkl\nmno pqr"
		long  = "abcdefgh\nabcdefgh\nabcdefgh"
	)
	runVisualEditCases(t, []visualEditCase{
		{words, "Vr-", "-------\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vlr-", "--c def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vr-", "-bc def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "4lvhhr-", "ab---ef\nghi jkl\nmno pqr", Position{0, 2}, 2},
		{words, "vjr-", "-------\n-hi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "Vjr-", "-------\n-------\nmno pqr", Position{0, 0}, 0},
		{words, "4lvjr-", "abc ---\n-----kl\nmno pqr", Position{0, 4}, 4},
		{words, "j4lvkr-", "abc ---\n-----kl\nmno pqr", Position{0, 4}, 4},
		{words, "j4lVkr-", "-------\n-------\nmno pqr", Position{0, 0}, 0},
		{words, "vlr<Space>", "  c def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "Vr<Space>", "       \nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vl3r-", "--c def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "V2r-", "-------\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vlr<Tab>", "\t\tc def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{"abc\n\ndef", "Vjjr-", "---\n\n---", Position{0, 0}, 0},
		{"abc\n\ndef", "vjjr-", "---\n\n-ef", Position{0, 0}, 0},
		{"\nabc", "Vr-", "\nabc", Position{0, 0}, 0},
		{"\nabc", "vr-", "\nabc", Position{0, 0}, 0},
		{"a\tb", "Vr-", "---", Position{0, 0}, 0},
		{"a\tb\nc", "vjr-", "---\n-", Position{0, 0}, 0},
		{"a日本b", "Vr-", "----", Position{0, 0}, 0},
		{"a日本b", "lvr-", "a-本b", Position{0, 1}, 1},
		{"a日本b", "vlr日", "日日本b", Position{0, 0}, 0},
		{"caféx", "Vr-", "-----", Position{0, 0}, 0},
		{"abc\ndef", "v$r-", "---\ndef", Position{0, 0}, 0},
		{"abc\ndef", "lv$r-", "a--\ndef", Position{0, 1}, 1},
		{"abc\ndef", "vj$r-", "---\n---", Position{0, 0}, 0},

		{long, "4lvlr-j", "abcd--gh\nabcdefgh\nabcdefgh", Position{1, 4}, 4},
		{long, "$vhr-j", "abcdef--\nabcdefgh\nabcdefgh", Position{1, 6}, 6},
		{long, "4lVr-j", "--------\nabcdefgh\nabcdefgh", Position{1, 0}, 0},

		{words, "vlr-u", words, Position{0, 0}, 0},
		{words, "4lvjr-u", words, Position{0, 4}, 4},
		{words, "j4lvkr-u", words, Position{0, 4}, 4},
		{words, "jVkr-u", words, Position{0, 0}, 0},
		{words, "j4lVkr-u", words, Position{0, 4}, 4},
		{words, "jj4lVkkr-u", words, Position{0, 4}, 4},
		{words, "4lVjr-u", words, Position{0, 0}, 0},
		{words, "vlr-u<C-r>", "--c def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vlr-.", "--c def\nghi jkl\nmno pqr", Position{0, 0}, 0},
		{words, "vlr-j.", "--c def\n--i jkl\nmno pqr", Position{1, 0}, 0},
		{words, "Vr-j.", "-------\n-------\nmno pqr", Position{1, 0}, 0},
		{words, "vjr-j.", "-------\n-------\n-no pqr", Position{1, 0}, 0},
		{"abc\ndef", "vlr-uj.", "abc\n--f", Position{1, 0}, 0},
		{"\nabc\nxyz", "jxkVr-j.", "\n--\nxyz", Position{1, 0}, 0},
	})
}

// TestVisualReplaceCancel checks the keys after a visual r that replace
// nothing. Esc and Backspace cancel r and keep the selection, as in Vim.
// Enter and Ctrl keys do the same: Vim puts them in literally (^M, ^H), which
// the editor can't show yet.
func TestVisualReplaceCancel(t *testing.T) {
	for _, keys := range []string{"vlr<Esc>", "vlr<BS>", "vlr<CR>", "vlr<C-h>", "vlr<C-a>"} {
		t.Run(keys, func(t *testing.T) {
			e := newTestEditor("abc def\nghi")
			typeKeys(t, e, keys)
			assert.Equal(t, "abc def\nghi", content(e))
			assert.Equal(t, VisualMode, e.GetMode().Name())
			assert.Equal(t, Position{0, 0}, e.GetState().VisualStart)
			assert.Equal(t, Position{0, 1}, cursorPos(e))

			typeKeys(t, e, "d") // the selection is still there
			assert.Equal(t, "c def\nghi", content(e))
		})
	}

	t.Run("in visual line mode", func(t *testing.T) {
		e := newTestEditor("abc\ndef\nghi")
		typeKeys(t, e, "Vjr<Esc>")
		assert.Equal(t, VisualLineMode, e.GetMode().Name())
		typeKeys(t, e, "d")
		assert.Equal(t, "ghi", content(e))
	})
}

// TestVisualReplaceCharIsLiteral checks that the character after a visual r
// is not mapped, as in Vim.
func TestVisualReplaceCharIsLiteral(t *testing.T) {
	e := newTestEditor("abc def")
	require.NoError(t, e.Map(MapVisual, "-", "d", true))
	typeKeys(t, e, "vlr-")
	assert.Equal(t, "--c def", content(e))
}

// TestVisualJoinReplaceReadOnly checks that J, gJ and r change nothing with
// insert mode disabled.
func TestVisualJoinReplaceReadOnly(t *testing.T) {
	for _, keys := range []string{"VJ", "vjJ", "VgJ", "gJ", "Vr-", "vlr-"} {
		t.Run(keys, func(t *testing.T) {
			e := newTestEditor("abc\ndef")
			e.DisableInsertMode(true)
			typeKeys(t, e, keys)
			assert.Equal(t, "abc\ndef", content(e))
		})
	}
}
