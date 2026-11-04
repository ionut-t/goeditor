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
