package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepeatChange covers '.' for normal-mode commands, row by row from the
// edge-case table in the design. Expected results are Vim's.
func TestRepeatChange(t *testing.T) {
	tests := []struct {
		name    string
		content string
		keys    string
		want    string
	}{
		{"dw", "one two three four", "dw.", "three four"},
		{"dd", "1\n2\n3\n4", "dd.", "3\n4"},
		{"count on operator", "a b c d e f g", "2dw.", "e f g"},
		{"counts multiply", "a b c d e f g h i j k l m", "2d3w.", "m"},
		{"count replaces 2d3w's 6", "a b c d e f g h i j k l m", "2d3w3.", "j k l m"},
		{"replacing count sticks", "a b c d e f g h i j k l m", "2d3w3..", "m"},
		{"x", "abcdefgh", "x.", "cdefgh"},
		{"x then 3.", "abcdefgh", "x3..", "h"},
		{"multi-digit count", "abcdefghijklmnopqrstuvwxyz", "12x.", "yz"},
		{"X", "abcdef", "$X.", "abcf"},
		{"D", "abc\ndef", "lDj0l.", "a\nd"},
		{"3rx", "abcdefgh", "3rx.", "xxxxxfgh"},
		{"J", "a\nb\nc\nd", "J.", "a b c\nd"},
		{"~", "abcd", "~.", "ABcd"},
		{"gUiw", "foo bar", "gUiww.", "FOO BAR"},
		{"guu", "AB\nCD", "guuj.", "ab\ncd"},
		{"dfx", "axbxcx", "dfx..", ""},
		{"d; uses the search current at the time of .", "ab1cd1ef2gh2ij", "f1d;f2.", "abefij"},
		{"yank is not a change", "one two three", "dwyw.", "three"},
		{"motion is not a change", "one two three", "dww.", "two "},
		{"undo is not a change", "one two three", "dwu.", "two three"},
		{"U is not a change", "abc", "xxU.", "bc"},
		{"nothing to repeat", "abc", ".", "abc"},
		{"cw with its insert", "one two three", "cwfoo<Esc>w.", "foo foo three"},
		{"o with its insert", "a", "ob<Esc>.", "a\nb\nb"},
		{"s with its insert", "abc", "sX<Esc>l.", "XXc"},
		{"insert counts are not count digits", "ab", "i12<Esc>.", "1122ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.True(t, e.IsNormalMode())
		})
	}
}

func TestRepeatChangePaste(t *testing.T) {
	e, _ := newTestEditorWithClipboard("ab")
	typeKeys(t, e, "ylp.")
	assert.Equal(t, "aaab", content(e))
}

func TestRepeatChangeMappings(t *testing.T) {
	t.Run("a mapping repeats as the keys it expands to", func(t *testing.T) {
		e := newTestEditor("a b c d e f")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		typeKeys(t, e, "Q.")
		assert.Equal(t, "c d e f", content(e))
		typeKeys(t, e, "3.")
		assert.Equal(t, "f", content(e))
	})

	t.Run("a two-change mapping undoes as one step and repeats one change", func(t *testing.T) {
		e := newTestEditor("a b c d e f")
		require.NoError(t, e.Map(MapNormal, "Q", "dwdw", true))
		typeKeys(t, e, "Q")
		assert.Equal(t, "c d e f", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "a b c d e f", content(e))
		typeKeys(t, e, "Q.")
		assert.Equal(t, "d e f", content(e))
	})

	t.Run("replay ignores mappings", func(t *testing.T) {
		e := newTestEditor("a b c d")
		typeKeys(t, e, "dw")
		require.NoError(t, e.Map(MapNormal, "d", "x", true))
		typeKeys(t, e, ".")
		assert.Equal(t, "c d", content(e))
	})
}

func TestRepeatChangeUndo(t *testing.T) {
	t.Run("u undoes a whole repeat", func(t *testing.T) {
		e := newTestEditor("one two three")
		typeKeys(t, e, "cwfoo<Esc>w.")
		assert.Equal(t, "foo foo three", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "foo two three", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "one two three", content(e))
	})

	t.Run("a cursor key inside the replayed insert doesn't split the step", func(t *testing.T) {
		// Typing it splits the insert into two undo steps; replaying it doesn't.
		e := newTestEditor("cd")
		typeKeys(t, e, "Aab<Left>X<Esc>")
		assert.Equal(t, "cdaXb", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "cdab", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "cd", content(e))

		typeKeys(t, e, ".")
		assert.Equal(t, "cdaXb", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "cd", content(e))
	})

	t.Run("each repeat is its own step", func(t *testing.T) {
		e := newTestEditor("a b c d")
		typeKeys(t, e, "dw..u")
		assert.Equal(t, "c d", content(e))
	})
}

func TestRepeatChangeFailure(t *testing.T) {
	t.Run("a failed motion drops the rest of the replay", func(t *testing.T) {
		// If the replay went on, "ab" would run as normal-mode commands:
		// a would start an insert and b would be typed.
		e := newTestEditor("1x2\nnone")
		typeKeys(t, e, "cfxab<Esc>")
		assert.Equal(t, "ab2\nnone", content(e))
		typeKeys(t, e, "j0.")
		assert.Equal(t, "ab2\nnone", content(e))
		assert.True(t, e.IsNormalMode())
		assert.Equal(t, Position{Row: 1, Col: 0}, cursorPos(e))
	})

	t.Run("the record survives a failed replay", func(t *testing.T) {
		e := newTestEditor("axb\nnone\ncxd")
		typeKeys(t, e, "dfxj.j0.")
		assert.Equal(t, "b\nnone\nd", content(e))
	})
}

func TestRepeatChangeCountClamped(t *testing.T) {
	// Vim clamps counts at 999999999, both a long run of digits and the
	// product of the counts around an operator.
	e := newTestEditor("a b c\nd e f")
	typeKeys(t, e, "99999999999999999999")
	assert.Equal(t, maxCount, *e.GetState().PendingCount)
	escape(e)

	typeKeys(t, e, "50000d50000w")
	assert.Equal(t, "", content(e))
	assert.Equal(t, maxCount, e.(*editor).repeat.count)
	assert.Equal(t, []KeyEvent{{Rune: 'd'}, {Rune: 'w'}}, e.(*editor).repeat.keys)
}

func TestRepeatChangeEditingDisabled(t *testing.T) {
	e := newTestEditor("one two three four five")
	typeKeys(t, e, "x")
	e.DisableInsertMode(true)
	typeKeys(t, e, "3.")
	assert.Equal(t, "ne two three four five", content(e))
	assert.Nil(t, e.GetState().PendingCount)
	typeKeys(t, e, "w")
	assert.Equal(t, Position{Row: 0, Col: 3}, cursorPos(e))
}

func TestRepeatChangeSetContentForgets(t *testing.T) {
	e := newTestEditor("one two")
	typeKeys(t, e, "dw")
	e.SetContent([]byte("one two"))
	typeKeys(t, e, ".")
	assert.Equal(t, "one two", content(e))
}
