package core

import (
	"errors"
	"fmt"
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

	t.Run("u undoes a repeated insert count as one step", func(t *testing.T) {
		e := newTestEditor("abc def")
		typeKeys(t, e, "3ix<Esc>w.")
		assert.Equal(t, "xxxabc xxxdef", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "xxxabc def", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "abc def", content(e))
	})

	t.Run("u undoes a repeated visual change as one step", func(t *testing.T) {
		e := newTestEditor("abcdef abcdef")
		typeKeys(t, e, "vllcX<Esc>w.")
		assert.Equal(t, "Xdef Xdef", content(e))
		typeKeys(t, e, "u")
		assert.Equal(t, "Xdef abcdef", content(e))
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

// TestRepeatInsert covers counts on i, a, I, A, o and O, and how a cursor key
// in insert mode ends the change '.' repeats. Expected results are Vim's.
func TestRepeatInsert(t *testing.T) {
	tests := []struct {
		name    string
		content string
		keys    string
		want    string
		cursor  Position
	}{
		{"3i", "ab", "3ix<Esc>", "xxxab", Position{0, 2}},
		{"3a", "ab", "3ax<Esc>", "axxxb", Position{0, 3}},
		{"3A", "ab", "3Ax<Esc>", "abxxx", Position{0, 4}},
		{"3I", "  ab", "$3Ix<Esc>", "  xxxab", Position{0, 4}},
		{"3o", "a\nz", "3ohi<Esc>", "a\nhi\nhi\nhi\nz", Position{3, 1}},
		{"3O", "a\nz", "3Ohi<Esc>", "hi\nhi\nhi\na\nz", Position{2, 1}},
		{"backspace is repeated", "ab", "3iab<BS>c<Esc>", "acacacab", Position{0, 5}},
		{"enter is repeated", "b", "3ia<CR><Esc>", "a\na\na\nb", Position{3, 0}},
		{"tab is repeated", "b", "3ia<Tab><Esc>", "a\ta\ta\tb", Position{0, 5}},
		{"ctrl keys type nothing", "b", "3ix<C-a>y<Esc>", "xyxyxyb", Position{0, 5}},
		{". keeps the insert count", "ab", "3ix<Esc>0.", "xxxxxxab", Position{0, 2}},
		{"count on . replaces it", "ab", "2ix<Esc>03.", "xxxxxab", Position{0, 2}},
		{"count on . after a plain insert", "ab", "ix<Esc>03.", "xxxxab", Position{0, 2}},
		{". keeps the o count", "a\nz", "2ohi<Esc>j.", "a\nhi\nhi\nz\nhi\nhi", Position{5, 1}},
		{"a cursor key drops the count", "abc", "3ifoo<Left>bar<Esc>", "fobaroabc", Position{0, 4}},
		{"a cursor key then Esc drops the count", "abc", "3ifoo<Left><Esc>", "fooabc", Position{0, 1}},
		{"a cursor key drops the o count", "abc", "3ohi<Left>X<Esc>", "abc\nhXi", Position{1, 1}},
		{". repeats only what was typed after a cursor key", "cd", "Aab<Left>X<Esc>0.", "XcdaXb", Position{0, 0}},
		{". after a cursor key typed nothing repeats what came before", "cd", "Aab<Left><Esc>0.", "cdabab", Position{0, 5}},
		{"text typed after a cursor key repeats as i", "abc def", "vlcX<Left>Y<Esc>w.", "YXc Ydef", Position{0, 4}},
		{". leaves out a backspace at the buffer start", "abc", "i<BS>z<Esc>.", "zzabc", Position{0, 0}},
		{". after a backspace at the buffer start, moved", "abc", "i<BS>z<Esc>l.", "zzabc", Position{0, 1}},
		{". keeps a backspace that deleted, not one that didn't", "abc", "iy<BS><BS>z<Esc>l.", "zzabc", Position{0, 1}},
		{". after two backspaces at the buffer start", "abc", "i<BS><BS>z<Esc>l.", "zzabc", Position{0, 1}},
		{"count after a backspace at the buffer start", "abc", "3i<BS>z<Esc>", "zzzabc", Position{0, 2}},
		{"count and . after backspaces at the buffer start", "abc", "2i<BS><BS>zq<Esc>l.", "zqzqzqzqabc", Position{0, 7}},
		{"CTRL-H at the buffer start is left out too", "abc", "i<BS>z<C-h><C-h>y<Esc>l.", "yyabc", Position{0, 1}},
		{"backspaces that join lines are kept", "abc\ndef", "ji<BS><BS>z<Esc>.", "zzdef", Position{0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.True(t, e.IsNormalMode())
		})
	}

	t.Run("a count is not kept for an insert started another way", func(t *testing.T) {
		e := newTestEditor("abc")
		typeKeys(t, e, "3ix<Esc>")
		typeKeys(t, e, "syz<Esc>")
		assert.Equal(t, "xxyzabc", content(e))
	})
}

// TestRepeatVisual covers '.' after a change made from visual mode: it
// applies to the same amount of text from the cursor, and ignores a count.
// Expected results are Vim's.
func TestRepeatVisual(t *testing.T) {
	const tenTwice = "abcdefghij\nabcdefghij"
	tests := []struct {
		name    string
		content string
		keys    string
		want    string
	}{
		{"characters on one line", tenTwice, "v2ld.", "ghij\nabcdefghij"},
		{"count on . is ignored", tenTwice, "lv2ldj03.", "aefghij\ndefghij"},
		{"clamped to the line end", "abcdefghij\nab", "5lv2ldj0.", "abcdeij\n"},
		{"several lines end on the same column", "abcdefghij\nabcdefghij\nabcdefghij\nabcdefghij", "llvjfedj0.", "abfghij\nfghij"},
		{"end column clamped to the last line", "abcdefghij\nabcdefghij\nabcdefghij\nab", "llvjfedj0l.", "abfghij\na"},
		{"lines", "1\n2\n3\n4\n5\n6", "Vjd.", "5\n6"},
		{"lines ignore a count", "1\n2\n3\n4\n5\n6", "Vjd3.", "5\n6"},
		{"lines clamped to the buffer end", "1\n2\n3\n4\n5", "jjjVkd.", "1\n2"},
		{"change with its insert", "abcdef abcdef", "vllcX<Esc>w.", "Xdef Xdef"},
		{"case", "abcdef abcdef", "vllUw.", "ABCdef ABCdef"},
		{"linewise change", "abc\ndef", "VcX<Esc>j.", "X\nX"},
		{"yank is not a change", "abcdef", "xvly.", "cdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.True(t, e.IsNormalMode())
		})
	}

	t.Run("the span is the one the operator used, whatever the host set", func(t *testing.T) {
		// The operator deletes from the mode's anchor; State.VisualStart is
		// only a copy the host can overwrite.
		e := newTestEditor("abcdefgh")
		typeKeys(t, e, "lvl")
		state := e.GetState()
		state.VisualStart = Position{Row: 0, Col: 7}
		e.SetState(state)
		typeKeys(t, e, "d")
		assert.Equal(t, "adefgh", content(e))
		typeKeys(t, e, ".")
		assert.Equal(t, "afgh", content(e))
	})

	t.Run("nothing happens with visual mode disabled", func(t *testing.T) {
		e := newTestEditor("abcdef")
		typeKeys(t, e, "vld")
		e.DisableVisualMode(true)
		typeKeys(t, e, ".")
		assert.Equal(t, "cdef", content(e))
		assert.True(t, e.(*editor).normalIdle())
	})
}

// TestRepeatCompletion covers an accepted completion, which repeats as the
// text it produced, as in Vim.
func TestRepeatCompletion(t *testing.T) {
	t.Run(". types the completed text", func(t *testing.T) {
		e := newTestEditor("x")
		typeKeys(t, e, "ofo")
		require.NoError(t, e.InsertCompletion(Completion{Text: "foobar"}))
		typeKeys(t, e, "<Esc>.")
		assert.Equal(t, "x\nfoobar\nfoobar", content(e))
	})

	t.Run("an insert count repeats the completed text", func(t *testing.T) {
		e := newTestEditor("x")
		typeKeys(t, e, "3ofo")
		require.NoError(t, e.InsertCompletion(Completion{Text: "foobar"}))
		typeKeys(t, e, "<Esc>")
		assert.Equal(t, "x\nfoobar\nfoobar\nfoobar", content(e))
	})

	t.Run("a completion after a cursor key starts the new change", func(t *testing.T) {
		e := newTestEditor("ab cd")
		typeKeys(t, e, "A x<Left>")
		require.NoError(t, e.InsertCompletion(Completion{Text: "zz"}))
		typeKeys(t, e, "<Esc>0.")
		assert.Equal(t, "zzab cd zz", content(e))
	})
}

// TestRepeatInsertLimit covers the cap on the text an insert count adds,
// which keeps a mistyped count from hanging the host.
func TestRepeatInsertLimit(t *testing.T) {
	insertWithCount := func(t *testing.T, count, text string) (Editor, *EditorError) {
		t.Helper()
		e := newTestEditor("b")
		typeKeys(t, e, count+"i"+text)
		return e, e.HandleKey(KeyEvent{Key: KeyEscape})
	}

	t.Run("a count past the limit adds nothing more", func(t *testing.T) {
		e, err := insertWithCount(t, "999999999", "x")
		require.NotNil(t, err)
		assert.Equal(t, ErrTextTooLongId, err.ID())
		assert.True(t, errors.Is(err, ErrTextTooLong))
		assert.Equal(t, "resulting text too long: 999999998 copies of 1 rune", err.Error())
		assert.Equal(t, "xb", content(e))
		assert.True(t, e.IsNormalMode())
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("backspace within the text is worked out up front", func(t *testing.T) {
		// Replaying 2.5 million copies of x y <BS> key by key would take
		// minutes; the net text, x, goes in with one edit.
		e, err := insertWithCount(t, "2500000", "xy<BS>")
		require.Nil(t, err)
		assert.Equal(t, 2500000+1, e.GetBuffer().LineRuneCount(0))
	})

	t.Run("backspace past the insert start replays the keys", func(t *testing.T) {
		e := newTestEditor("abc")
		typeKeys(t, e, "$3a<BS>x<Esc>")
		assert.Equal(t, "abx", content(e))
	})

	t.Run("replayed keys have their own lower limit", func(t *testing.T) {
		// Three keys a copy (Backspace, x, and the copy itself): 33333
		// copies is the most that fits in maxReplayedKeys.
		e := newTestEditor("abc")
		typeKeys(t, e, "$33334a<BS>x")
		require.Nil(t, e.HandleKey(KeyEvent{Key: KeyEscape}))
		assert.Equal(t, "abx", content(e))

		e = newTestEditor("abc")
		typeKeys(t, e, "$33335a<BS>x")
		err := e.HandleKey(KeyEvent{Key: KeyEscape})
		assert.True(t, errors.Is(err, ErrTextTooLong))
		assert.Equal(t, "resulting text too long: 33334 copies of 3 keys", err.Error())
		assert.Equal(t, "abx", content(e))
	})

	t.Run("text up to the limit goes in", func(t *testing.T) {
		// 5000000 repeats of two runes is exactly the limit.
		e, err := insertWithCount(t, "5000001", "ab")
		require.Nil(t, err)
		assert.Equal(t, 2*5000001+1, e.GetBuffer().LineRuneCount(0))
		assert.Equal(t, Position{0, 2*5000001 - 1}, cursorPos(e))
	})

	t.Run("one copy past the limit is refused", func(t *testing.T) {
		e, err := insertWithCount(t, "5000002", "ab")
		assert.True(t, errors.Is(err, ErrTextTooLong))
		assert.Equal(t, "abb", content(e))
	})
}

// TestRepeatInsertMatchesTyping checks that an insert count adds what typing
// the keys again would, for every key insert mode can receive in a count.
// It guards insertedText against drifting from insertMode.HandleKey.
func TestRepeatInsertMatchesTyping(t *testing.T) {
	keys := []KeyEvent{
		{Rune: 'x'}, {Rune: 'é'}, {Rune: ' ', Key: KeySpace}, {Rune: 'a', Modifiers: ModCtrl},
		{Rune: 'X', Modifiers: ModShift},
	}
	for code := KeyUnknown; code <= KeyInsert; code++ {
		if code == KeyEscape || breaksInsert(KeyEvent{Key: code}) {
			continue // they end the insert or drop its count
		}
		keys = append(keys, KeyEvent{Key: code})
	}

	for _, key := range keys {
		for _, start := range []string{"i", "o", "a"} {
			t.Run(fmt.Sprintf("%s %v", start, key), func(t *testing.T) {
				typed := []KeyEvent{{Rune: 'a'}, key, {Rune: 'b'}}

				counted := newTestEditor("pq\nrs")
				typeKeys(t, counted, "$3"+start)
				for _, k := range typed {
					_ = counted.HandleKey(k)
				}
				_ = counted.HandleKey(KeyEvent{Key: KeyEscape})

				byHand := newTestEditor("pq\nrs")
				typeKeys(t, byHand, "$"+start)
				for i := range 3 {
					if i > 0 && start == "o" {
						_ = byHand.HandleKey(KeyEvent{Key: KeyEnter})
					}
					for _, k := range typed {
						_ = byHand.HandleKey(k)
					}
				}
				_ = byHand.HandleKey(KeyEvent{Key: KeyEscape})

				assert.Equal(t, content(byHand), content(counted))
				assert.Equal(t, cursorPos(byHand), cursorPos(counted))
			})
		}
	}
}

// TestInsertHostCursorMove covers the host moving the cursor during an
// insert, as it does for a mouse click: like a cursor key, it ends the
// insert for the count, for '.', and for undo, as in Vim.
func TestInsertHostCursorMove(t *testing.T) {
	moveTo := func(e Editor, row, col int) {
		cursor := e.GetBuffer().GetCursor()
		cursor.Position = Position{Row: row, Col: col}
		e.GetBuffer().SetCursor(cursor)
	}

	t.Run("drops the count", func(t *testing.T) {
		e := newTestEditor("ab\ncd")
		typeKeys(t, e, "3ix")
		moveTo(e, 1, 0)
		typeKeys(t, e, "y<Esc>")
		assert.Equal(t, "xab\nycd", content(e))
	})

	t.Run(". repeats only what was typed after it", func(t *testing.T) {
		e := newTestEditor("ab\ncd")
		typeKeys(t, e, "Ax")
		moveTo(e, 1, 0)
		typeKeys(t, e, "y<Esc>$.")
		assert.Equal(t, "abx\nycyd", content(e))
	})

	t.Run("ends the undo step", func(t *testing.T) {
		e := newTestEditor("ab\ncd")
		typeKeys(t, e, "Ax")
		moveTo(e, 1, 0)
		typeKeys(t, e, "y<Esc>u")
		assert.Equal(t, "abx\ncd", content(e))
	})

	t.Run("a host edit after a cursor key doesn't leave an empty record", func(t *testing.T) {
		e := newTestEditor("ab\ncd")
		typeKeys(t, e, "Ax<Left>")
		require.NoError(t, e.GetBuffer().InsertRunesAt(1, 0, []rune("Z")))
		typeKeys(t, e, "<Esc>j.")
		assert.Equal(t, "abx\nZcdx", content(e))
	})
}
