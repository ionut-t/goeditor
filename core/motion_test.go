package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMoveLeft tests 'h' — move left one character.
// Like Vim's default 'whichwrap', it doesn't leave the line.
func TestMoveLeft(t *testing.T) {
	t.Run("moves left within line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'h')
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("at col 0 stays on the line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'h')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at start of buffer stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'h')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 3h", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '3', 'h') // $ → col 4; 3h → col 1
		assert.Equal(t, Position{0, 1}, cursorPos(e))
	})

	t.Run("count past col 0 stops on col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '9', 'h')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveRight tests 'l' — move right one character.
// Like Vim's default 'whichwrap', it stops on the last character of the line.
func TestMoveRight(t *testing.T) {
	t.Run("moves right within line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l')
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("at end of line stays on the last char", func(t *testing.T) {
		e := newTestEditor("a\nb")
		keys(e, 'l', 'l')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at end of last line stays on the last char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', 'l', 'l')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("count: 3l", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '3', 'l')
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("count past the end stops on the last char", func(t *testing.T) {
		e := newTestEditor("abc")
		keys(e, 'l', '5', 'l')
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

type whichwrapCase struct {
	name    string
	content string
	keys    []KeyEvent
	want    Position
	text    string // buffer after the keys; empty to skip the check
}

func runWhichwrapCases(t *testing.T, tests []whichwrapCase, setup func(Editor)) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			if setup != nil {
				setup(e)
			}
			for _, k := range tt.keys {
				_ = e.HandleKey(k)
			}
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
		})
	}
}

func whichwrapKeys() (left, right, space, bs, esc KeyEvent) {
	return KeyEvent{Key: KeyLeft}, KeyEvent{Key: KeyRight}, KeyEvent{Key: KeySpace, Rune: ' '},
		KeyEvent{Key: KeyBackspace}, KeyEvent{Key: KeyEscape}
}

// keySeq builds a key sequence from strings (one rune per key) and KeyEvents.
func keySeq(parts ...any) []KeyEvent {
	var evs []KeyEvent
	for _, p := range parts {
		switch p := p.(type) {
		case string:
			for _, c := range p {
				evs = append(evs, KeyEvent{Rune: c})
			}
		case KeyEvent:
			evs = append(evs, p)
		}
	}
	return evs
}

// TestWhichwrap checks h/l, the arrow keys, <Space> and <BS> at line
// boundaries against Vim's default 'whichwrap' (b,s): only <BS> and <Space>
// cross lines, in normal and visual mode. Expected positions are from Vim 9.2.
func TestWhichwrap(t *testing.T) {
	left, right, space, bs, esc := whichwrapKeys()
	seq := keySeq
	const t3 = "abcdef\nab\nabcdefghij"

	tests := []whichwrapCase{
		{"normal jh", "abc\ndef", seq("jh"), Position{1, 0}, ""},
		{"normal j<Left>", "abc\ndef", seq("j", left), Position{1, 0}, ""},
		{"normal $l", "abc\ndef", seq("$l"), Position{0, 2}, ""},
		{"normal $<Right>", "abc\ndef", seq("$", right), Position{0, 2}, ""},
		{"normal $<Space>", "abc\ndef", seq("$", space), Position{1, 0}, ""},
		{"normal j<BS>", "abc\ndef", seq("j", bs), Position{0, 2}, ""},
		{"normal <BS> at start of buffer", "abc\ndef", seq(bs), Position{0, 0}, ""},
		{"normal <Space> at end of buffer", "abc\ndef", seq("j$", space), Position{1, 2}, ""},
		{"normal $4<Space> crosses an empty line", "abc\n\ndef\nghi", seq("$4", space), Position{2, 2}, ""},
		{"normal $2<Space>", "abc\n\ndef\nghi", seq("$2", space), Position{2, 0}, ""},
		{"normal G3<BS>", "abc\n\ndef\nghi", seq("G3", bs), Position{2, 0}, ""},
		{"normal jj<BS><BS> crosses an empty line", "abc\n\ndef\nghi", seq("jj", bs, bs), Position{0, 2}, ""},
		{"insert ji<Left>", "abc\ndef", seq("ji", left), Position{1, 0}, ""},
		{"insert A<Right>", "abc\ndef", seq("A", right), Position{0, 3}, ""},
		{"insert A<Right>x", "abc\ndef", seq("A", right, "x", esc), Position{0, 3}, "abcx\ndef"},
		{"visual jvh", "abc\ndef", seq("jvh"), Position{1, 0}, ""},
		{"visual $vl goes onto the line break", "abc\ndef", seq("$vl"), Position{0, 3}, ""},
		{"visual $v<Right>", "abc\ndef", seq("$v", right), Position{0, 3}, ""},
		{"visual $v3l", "abc\ndef", seq("$v3l"), Position{0, 3}, ""},
		{"visual v9l", t3, seq("v9l"), Position{0, 6}, ""},
		{"visual v5ll", t3, seq("v5ll"), Position{0, 6}, ""},
		{"visual v5lljj keeps the column", t3, seq("v5lljj"), Position{2, 6}, ""},
		{"visual v5llld deletes the line break", t3, seq("v5llld"), Position{0, 0}, "ab\nabcdefghij"},
		{"visual v5ll<Esc> goes onto the last character", t3, seq("v5ll", esc, "jj"), Position{2, 5}, ""},
		{"visual jv2l on a short line", t3, seq("jv2l"), Position{1, 2}, ""},
		{"visual Gv19l on the last line", t3, seq("Gv19l"), Position{2, 10}, ""},
		{"visual Glv19ld on the last line", t3, seq("Glv19ld"), Position{2, 0}, "abcdef\nab\na"},
		{"visual vl on a one-character line", "a\nabc", seq("vl"), Position{0, 1}, ""},
		{"visual vl on an empty line", "\nabc", seq("vl"), Position{0, 0}, ""},
		{"visual jjv<BS>d", "abc\n\ndef\nghi", seq("jjv", bs, "d"), Position{1, 0}, "abc\nef\nghi"},
		{"visual $v<Space>", "abc\ndef", seq("$v", space), Position{1, 0}, ""},
		{"visual line jVh", "abc\ndef", seq("jVh"), Position{1, 0}, ""},
		{"visual line $Vl", "abc\ndef", seq("$Vl"), Position{0, 3}, ""},
		{"visual line jVll on a short line", t3, seq("jVll"), Position{1, 2}, ""},
		{"visual line jV<BS>", "abc\ndef", seq("jV", bs), Position{0, 2}, ""},
		{"visual line $V<Space>", "abc\ndef", seq("$V", space), Position{1, 0}, ""},
	}
	runWhichwrapCases(t, tests, nil)

	t.Run("<BS> across lines keeps the column for j", func(t *testing.T) {
		e := newTestEditor("abc\nabcdef\nabcdef")
		keys(e, 'j')
		_ = e.HandleKey(bs)
		keys(e, 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})
}

// TestMoveDown tests 'j' — move down one line.
func TestMoveDown(t *testing.T) {
	t.Run("moves down one line", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'j')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("at last line stays put", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'j')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("preserves column when moving down", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, '$', 'j') // $→col4 (Preferred=4); j→col4 on "world"
		assert.Equal(t, Position{1, 4}, cursorPos(e))
	})

	t.Run("clamps column to shorter line", func(t *testing.T) {
		e := newTestEditor("hello\nhi")
		// $ → col 4, Preferred=4; j → target col 4, clamps to the last char of "hi"
		keys(e, '$', 'j')
		assert.Equal(t, Position{1, 1}, cursorPos(e))
	})

	t.Run("count: 2j", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '2', 'j')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})
}

// TestMoveUp tests 'k' — move up one line.
func TestMoveUp(t *testing.T) {
	t.Run("moves up one line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at first line stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2k", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j', '2', 'k')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestHorizontalMoveKeepsColumnForVertical checks that l/Right set the
// preferred column the next j/k uses, in every mode that binds them.
func TestHorizontalMoveKeepsColumnForVertical(t *testing.T) {
	const text = "abcdefghij\nabcdefghij"

	t.Run("normal: llj", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'l', 'l', 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("normal: 5llj", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, '5', 'l', 'l', 'j')
		assert.Equal(t, Position{1, 6}, cursorPos(e))
	})

	t.Run("normal: Right Right j", func(t *testing.T) {
		e := newTestEditor(text)
		_ = e.HandleKey(KeyEvent{Key: KeyRight})
		_ = e.HandleKey(KeyEvent{Key: KeyRight})
		keys(e, 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("visual: llvjd", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'l', 'l', 'v', 'j', 'd')
		assert.Equal(t, "abdefghij", content(e))
	})

	t.Run("visual: vllj", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'v', 'l', 'l', 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("visual line: Vllj", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'V', 'l', 'l', 'j')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("insert: Right Right Down", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'i')
		_ = e.HandleKey(KeyEvent{Key: KeyRight})
		_ = e.HandleKey(KeyEvent{Key: KeyRight})
		_ = e.HandleKey(KeyEvent{Key: KeyDown})
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})
}

// TestPreferredIsFullColumn checks that j/k go back to the full column the
// last horizontal motion left, whatever the width the host renders at, as
// Vim's curswant. Expected positions are from Vim 9.2.
func TestPreferredIsFullColumn(t *testing.T) {
	down := KeyEvent{Key: KeyDown}
	const long = "abcdefghijklmno\nabcdefghijklmno"
	tests := []whichwrapCase{
		{"l past the wrap width", long, keySeq("12lj"), Position{1, 12}, ""},
		{"h count past the start", "abcdef\nabcdef", keySeq("$9hj"), Position{1, 0}, ""},
		{"l count past the end", long, keySeq("5l20lj"), Position{1, 14}, ""},
		{"shorter line keeps the column", "abcdefghijkl\nab\nabcdefghijkl", keySeq("10ljj"), Position{2, 10}, ""},
		{"empty line keeps the column", "abcdefghijkl\n\nabcdefghijkl", keySeq("10ljj"), Position{2, 10}, ""},
		{"k keeps the column", "abcdefghijkl\nab\nabcdefghijkl", keySeq("10ljk"), Position{0, 10}, ""},
		{"w", "aaaaaaaaaaaa bbb\naaaaaaaaaaaaaaaa", keySeq("wj"), Position{1, 13}, ""},
		{"visual d", long, keySeq("12lvjd"), Position{0, 12}, "abcdefghijklno"},
		{"visual line", long, keySeq("12lVjlk"), Position{0, 13}, ""},
		{"insert Down", long, keySeq("12li", down, "x"), Position{1, 13}, "abcdefghijklmno\nabcdefghijklxmno"},
		{"k count past the top", "ab\nabcdefghijkl\nabcdefghijkl", keySeq("jj10l5kj"), Position{1, 10}, ""},
		{"j count past the end", "abcdefghijkl\nabcdefghijkl\nab", keySeq("10l5jk"), Position{1, 10}, ""},
		{"insert Down onto a shorter line", "abcdefghijkl\nab\nabcdefghijkl", keySeq("10li", down, "x"), Position{1, 3}, "abcdefghijkl\nabx\nabcdefghijkl"},
		{"insert Down past a shorter line", "abcdefghijkl\nab\nabcdefghijkl", keySeq("10li", down, down, "x"), Position{2, 11}, "abcdefghijkl\nab\nabcdefghijxkl"},
	}
	for _, width := range []int{0, 10, 80} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			runWhichwrapCases(t, tests, func(e Editor) {
				s := e.GetState()
				s.AvailableWidth = width
				e.SetState(s)
			})
		})
	}
}

// TestShorterLine checks j, k, CTRL-D and CTRL-U onto a line shorter than the
// column. Normal mode stops on its last character; visual mode can stop after
// it, on the line break, and leaving visual puts the cursor back on that
// character. <Esc> also resets the column j and k go back to; v and V don't.
// Expected positions are from Vim 9.2 (CTRL-D/CTRL-U with 'nosol').
func TestShorterLine(t *testing.T) {
	_, _, _, _, esc := whichwrapKeys()
	ctrlU := KeyEvent{Rune: 'u', Modifiers: ModCtrl}
	const text = "abcdefghijkl\nab\nabcdefghijkl"
	const empty = "abcdefghijkl\n\nabcdefghijkl"

	runWhichwrapCases(t, []whichwrapCase{
		{"normal j", text, keySeq("10lj"), Position{1, 1}, ""},
		{"normal $j", "hello\nhi", keySeq("$j"), Position{1, 1}, ""},
		{"normal k", text, keySeq("2j10lk"), Position{1, 1}, ""},
		{"normal j onto an empty line", empty, keySeq("10lj"), Position{1, 0}, ""},
		{"normal l after j", text, keySeq("10ljl"), Position{1, 1}, ""},
		{"normal x after j", text, keySeq("10ljx"), Position{1, 0}, "abcdefghijkl\na\nabcdefghijkl"},
		{"normal CTRL-U", "ab\nabcdefghijkl", keySeq("j10l", ctrlU), Position{0, 1}, ""},
		{"visual j", text, keySeq("10lvj"), Position{1, 2}, ""},
		{"visual j onto an empty line", empty, keySeq("10lvj"), Position{1, 0}, ""},
		{"visual line j", text, keySeq("10lVj"), Position{1, 2}, ""},
		{"visual CTRL-U", "ab\nabcdefghijkl", keySeq("j10lv", ctrlU), Position{0, 2}, ""},
		{"visual jh", text, keySeq("10lvjh"), Position{1, 1}, ""},
		{"visual jd deletes the line break", text, keySeq("10lvjd"), Position{0, 10}, "abcdefghijabcdefghijkl"},
		{"visual jhd", text, keySeq("10lvjhd"), Position{0, 9}, "abcdefghij\nabcdefghijkl"},
		{"visual <Esc>", text, keySeq("10lvj", esc), Position{1, 1}, ""},
		{"visual <Esc> resets the column", text, keySeq("10lvj", esc, "j"), Position{2, 1}, ""},
		{"visual <Esc> on an empty line", empty, keySeq("10lvj", esc, "j"), Position{2, 0}, ""},
		{"visual <Esc> after k", text, keySeq("10ljjvk", esc, "j"), Position{2, 1}, ""},
		{"visual <Esc> on a long line", text, keySeq("10ljvj", esc, "k"), Position{1, 1}, ""},
		{"visual line <Esc>", text, keySeq("10lVj", esc), Position{1, 1}, ""},
		{"visual line <Esc> resets the column", text, keySeq("10lVj", esc, "j"), Position{2, 1}, ""},
		{"visual v", text, keySeq("10lvjv"), Position{1, 1}, ""},
		{"visual v keeps the column", text, keySeq("10lvjvj"), Position{2, 10}, ""},
		{"visual line V keeps the column", text, keySeq("10lVjVj"), Position{2, 10}, ""},
		{"visual V keeps the column", text, keySeq("10lvjVj"), Position{2, 10}, ""},
	}, nil)
}

// TestMoveAcrossLines checks SetMoveAcrossLines against Vim with
// ':set whichwrap=b,s,<,>,[,],h,l'. Expected positions are from Vim 9.2.
func TestMoveAcrossLines(t *testing.T) {
	left, right, space, bs, esc := whichwrapKeys()
	seq := keySeq
	const text = "abc\n\ndef"

	runWhichwrapCases(t, []whichwrapCase{
		{"normal jjh", text, seq("jjh"), Position{1, 0}, ""},
		{"normal jj<Left>", text, seq("jj", left), Position{1, 0}, ""},
		{"normal $l", text, seq("$l"), Position{1, 0}, ""},
		{"normal $<Right>", text, seq("$", right), Position{1, 0}, ""},
		{"normal $3l", text, seq("$3l"), Position{2, 1}, ""},
		{"normal G$l at end of buffer", text, seq("G$l"), Position{2, 2}, ""},
		{"normal <Space> and <BS> unchanged", text, seq("$", space, space, bs), Position{1, 0}, ""},
		{"insert A<Right>x", text, seq("A", right, "x", esc), Position{1, 0}, "abc\nx\ndef"},
		{"insert A<Right><Right>x", text, seq("A", right, right, "x", esc), Position{2, 0}, "abc\n\nxdef"},
		{"insert jji<Left>x", text, seq("jji", left, "x", esc), Position{1, 0}, "abc\nx\ndef"},
		{"insert jji<Left><Left>x", text, seq("jji", left, left, "x", esc), Position{0, 3}, "abcx\n\ndef"},
		{"visual jjvh", text, seq("jjvh"), Position{1, 0}, ""},
		{"visual $vl goes onto the line break", text, seq("$vl"), Position{0, 3}, ""},
		{"visual $vll", text, seq("$vll"), Position{1, 0}, ""},
		{"visual $v2l", text, seq("$v2l"), Position{1, 0}, ""},
		{"visual $v3l", text, seq("$v3l"), Position{2, 0}, ""},
		{"visual jv$l from an empty line", text, seq("jv$l"), Position{2, 0}, ""},
		{"visual lv$l", "abcdef\nab\nabcdefghij", seq("lv$l"), Position{1, 0}, ""},
		{"visual lv$ljj", "abcdef\nab\nabcdefghij", seq("lv$ljj"), Position{2, 0}, ""},
		{"visual Glv$l at end of buffer keeps the column", "abcdefghijkl\nab\nabcdef", seq("Glv$lkk"), Position{0, 12}, ""},
		{"normal G$l at end of buffer keeps the column", "abcdefghijkl\nab\nabcdef", seq("G$lkk"), Position{0, 11}, ""},
		{"visual line jjVhd", text, seq("jjVhd"), Position{0, 0}, "abc"},
	}, func(e Editor) { e.SetMoveAcrossLines(true) })

	t.Run("off by default", func(t *testing.T) {
		e := newTestEditor(text)
		assert.False(t, e.MoveAcrossLines())
	})

	t.Run("can be turned off again", func(t *testing.T) {
		e := newTestEditor(text)
		e.SetMoveAcrossLines(true)
		e.SetMoveAcrossLines(false)
		keys(e, '$', 'l')
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	// Without Vim mode the editor is a plain text box, where the arrow keys
	// cross lines as in any other editor.
	runWhichwrapCases(t, []whichwrapCase{
		{"non-vim <Left> at col 0", text, seq(right, right, right, right, right, left, "x"), Position{1, 1}, "abc\nx\ndef"},
		{"non-vim <Right> at end of line", text, seq(right, right, right, right, "x"), Position{1, 1}, "abc\nx\ndef"},
	}, func(e Editor) {
		e.DisableVimMode(true)
		e.SetInsertMode()
	})
}

// TestMoveToLineStart tests '0' — move to column 0.
func TestMoveToLineStart(t *testing.T) {
	t.Run("moves to col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '0')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already at col 0 stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '0')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestHomeIsLineStart checks that <Home> goes to column 0, as Vim's 0, in
// normal and visual mode: not to the first non-blank, whatever the count, and
// j/k afterwards keep column 0. Expected positions are from Vim 9.2.
func TestHomeIsLineStart(t *testing.T) {
	seq := keySeq
	home := KeyEvent{Key: KeyHome}
	const text = "    hello world\n  second line\n      \nthird"

	runDollarCases(t, []dollarCase{
		// Normal mode
		{name: "<Home> on an indented line", content: text, keys: seq("$", home), want: Position{0, 0}},
		{name: "<Home>j keeps column 0", content: text, keys: seq("$", home, "j"), want: Position{1, 0}},
		{name: "<Home> on a line of only blanks", content: text, keys: seq("jj$", home), want: Position{2, 0}},
		{name: "<Home>k from a line of only blanks", content: text, keys: seq("jj$", home, "k"), want: Position{1, 0}},
		{name: "count is ignored", content: text, keys: seq("$3", home), want: Position{0, 0}},
		{name: "count is ignored, then j", content: text, keys: seq("$3", home, "j"), want: Position{1, 0}},
		{
			name: "<Home>x", content: text, keys: seq("$", home, "x"), want: Position{0, 0},
			text: "   hello world\n  second line\n      \nthird",
		},
		{name: "^ still goes to the first non-blank", content: text, keys: seq("$^"), want: Position{0, 4}},

		// Visual mode
		{name: "v<Home>", content: text, keys: seq("12lv", home), want: Position{0, 0}},
		{name: "v<Home>j", content: text, keys: seq("12lv", home, "j"), want: Position{1, 0}},
		{
			name: "v<Home>d", content: text, keys: seq("12lv", home, "d"), want: Position{0, 0},
			text: "ld\n  second line\n      \nthird",
		},
		{name: "v<Home> on a line of only blanks", content: text, keys: seq("jj$v", home), want: Position{2, 0}},
		{name: "V<Home>", content: text, keys: seq("12lV", home), want: Position{0, 0}},
	})
}

// TestMoveToLineEnd tests '$' — move to last character of line.
func TestMoveToLineEnd(t *testing.T) {
	t.Run("moves to last char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("empty line stays at col 0", func(t *testing.T) {
		e := newTestEditor("")
		keys(e, '$')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already at end stays put", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '$')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
}

// TestMoveToFirstNonBlank tests '^' — move to first non-whitespace character.
func TestMoveToFirstNonBlank(t *testing.T) {
	t.Run("skips leading spaces", func(t *testing.T) {
		e := newTestEditor("   hello")
		keys(e, '^')
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("no leading spaces goes to col 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, '$', '^')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("all spaces goes to the last space", func(t *testing.T) {
		e := newTestEditor("   ")
		keys(e, '^')
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

// TestEnterMovesDown tests <Enter> — move down [count] lines to the first non-blank.
func TestEnterMovesDown(t *testing.T) {
	const text = "one\n  two\n    three\n four\nfive"

	t.Run("no count moves down one line", func(t *testing.T) {
		e := newTestEditor(text)
		setCursorRow(e, 1)
		enter(e)
		assert.Equal(t, Position{2, 4}, cursorPos(e))
	})

	t.Run("count moves down count lines", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, '2')
		enter(e)
		assert.Equal(t, Position{2, 4}, cursorPos(e))
		assert.Nil(t, e.GetState().PendingCount)
	})

	t.Run("count past the end stops on the last line", func(t *testing.T) {
		e := newTestEditor(text)
		setCursorRow(e, 2)
		keys(e, '9')
		enter(e)
		assert.Equal(t, Position{4, 0}, cursorPos(e))
	})

	t.Run("on the last line stays put", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'G', 'l', 'l')
		enter(e)
		assert.Equal(t, Position{4, 2}, cursorPos(e))
	})
}

// TestMoveToBufferStart tests 'g' — move to first line.
func TestMoveToBufferStart(t *testing.T) {
	t.Run("moves to row 0 col 0", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'j', 'j', 'g', 'g')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("already on first line stays at row 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'l', 'l', 'g', 'g')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveToBufferEnd tests 'G' — move to last line (first non-blank).
func TestMoveToBufferEnd(t *testing.T) {
	t.Run("moves to last line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'G')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("last line with leading spaces lands on first non-blank", func(t *testing.T) {
		e := newTestEditor("one\n   indented")
		keys(e, 'G')
		assert.Equal(t, Position{1, 3}, cursorPos(e))
	})

	t.Run("single line buffer stays at row 0", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'G')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2G goes to line 2 first non-blank", func(t *testing.T) {
		e := newTestEditor("one\n  two\nthree")
		keys(e, '2', 'G')
		assert.Equal(t, Position{1, 2}, cursorPos(e))
	})

	t.Run("count: 1G goes to the first line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'G', '1', 'G')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count past the end clamps to the last line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '9', 'G')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})
}

// TestMoveWordForward tests 'w' — move to start of next word.
func TestMoveWordForward(t *testing.T) {
	t.Run("moves to start of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("from mid-word jumps to next word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("count: 2w", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'w')
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("wraps to next line", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'w')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestMoveWordBackward tests 'b' — move to start of current or previous word.
func TestMoveWordBackward(t *testing.T) {
	t.Run("from mid-word jumps to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'l', 'l', 'b') // col 8; b → col 6
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("from start of word jumps to previous word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'b') // col 6; b → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2b", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'w', '2', 'b') // col 8; 2b → col 0
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestMoveWordToEnd tests 'e' — move to end of current or next word.
func TestMoveWordToEnd(t *testing.T) {
	t.Run("from start of word moves to end of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("from end of word jumps to end of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'e', 'e')
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})

	t.Run("count: 2e", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, '2', 'e')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestMoveParagraphForward tests '}' — move to the next blank line (paragraph boundary).
// Like Vim: from a non-blank line, lands on the next blank line (or last line if none).
// From a blank line, skips the blank gap first, then lands on the following blank line.
func TestMoveParagraphForward(t *testing.T) {
	t.Run("lands on the blank line between paragraphs", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, '}') // from row 0: skip "one","two", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from within paragraph lands on blank separator", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'j', '}') // row 1 → skip "two", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from blank line skips blank gap and lands on next blank", func(t *testing.T) {
		e := newTestEditor("one\n\n\nthree\nfour\n\nsix")
		keys(e, 'j', '}') // row 1 (blank): skip blanks (1,2), skip non-blank (3,4), land on row 5 (blank)
		assert.Equal(t, Position{5, 0}, cursorPos(e))
	})

	t.Run("at last line stays put", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, 'G', '}') // G → last line (row 3); } is a no-op
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})

	t.Run("no blank line: lands on last line", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, '}') // skip non-blank until last line; no blank found
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("two } jumps land on successive blank lines", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, '}', '}') // row 0 → row 1 (blank) → row 3 (blank)
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})

	t.Run("count: 2} jumps two paragraph boundaries", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, '2', '}') // row 0 → row 1 (blank) → row 3 (blank)
		assert.Equal(t, Position{3, 0}, cursorPos(e))
	})
}

// TestMoveParagraphBackward tests '{' — move to the previous blank line (paragraph boundary).
// Like Vim: from a non-blank line, lands on the previous blank line (or row 0 if none).
// From a blank line, skips the blank gap first, then lands on the preceding blank line.
func TestMoveParagraphBackward(t *testing.T) {
	t.Run("lands on the blank line between paragraphs", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'G', '{') // from row 4: skip "four","three", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from start of paragraph lands on blank separator", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree\nfour")
		keys(e, 'j', 'j', 'j', '{') // row 3 → skip "three", land on row 2 (blank)
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("from blank line skips blank gap and lands on previous blank", func(t *testing.T) {
		e := newTestEditor("one\n\nthree\nfour\n\nsix")
		keys(e, 'G', '{', '{') // row 5 (blank) → skip blank, skip "four","three", land row 1 (blank)
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	t.Run("no blank before first paragraph: lands on row 0", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, 'j', '{') // row 1 → skip "two","one", hit row 0 (start of buffer)
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("at first line stays put", func(t *testing.T) {
		e := newTestEditor("one\ntwo\n\nthree")
		keys(e, '{') // row 0; { is a no-op
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("two { jumps land on successive blank lines", func(t *testing.T) {
		e := newTestEditor("a\n\nb\n\nc")
		keys(e, 'G', '{', '{') // row 4 → row 3 (blank) → row 1 (blank)
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})
}

// TestGPrefix tests the 'g' prefix commands (gg, ge).
func TestGPrefix(t *testing.T) {
	// gg
	t.Run("gg goes to top of buffer", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'G')
		keys(e, 'g', 'g')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("5gg goes to line 5", func(t *testing.T) {
		e := newTestEditor("a\nb\nc\nd\ne\nf")
		keys(e, '5', 'g', 'g')
		assert.Equal(t, Position{4, 0}, cursorPos(e))
	})

	t.Run("gg count beyond last line clamps to last line", func(t *testing.T) {
		e := newTestEditor("a\nb")
		keys(e, '9', 'g', 'g')
		assert.Equal(t, Position{1, 0}, cursorPos(e))
	})

	// ge
	t.Run("ge moves to end of previous word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w') // cursor at 'w' of "world" (col 6)
		keys(e, 'g', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e)) // 'o' of "hello"
	})

	t.Run("ge from start of word moves to end of previous word", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'w') // cursor at 't' of "three" (col 8)
		keys(e, 'g', 'e')
		assert.Equal(t, Position{0, 6}, cursorPos(e)) // 'o' of "two"
	})

	t.Run("ge at start of buffer returns error and stays", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'g', 'e')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("ge crosses line boundary", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'j') // row 1
		keys(e, 'g', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e)) // 'o' of "hello"
	})

	// Escape cancels pending g
	t.Run("Escape after g cancels prefix", func(t *testing.T) {
		e := newTestEditor("hello\nworld")
		keys(e, 'j') // row 1
		_ = e.HandleKey(KeyEvent{Rune: 'g'})
		escape(e)
		assert.Equal(t, Position{1, 0}, cursorPos(e)) // cursor unchanged
		assert.True(t, e.IsNormalMode())
	})
}

// TestWORDMotions tests 'W', 'E', 'B' — whitespace-delimited WORD motions.
func TestWORDMotions(t *testing.T) {
	t.Run("W skips punctuation to next WORD", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'W')
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("E moves to end of WORD", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'E')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("B moves back to start of WORD", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'W', 'B')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("gE moves to end of previous WORD", func(t *testing.T) {
		e := newTestEditor("foo-bar baz")
		keys(e, 'W', 'g', 'E')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestMatchingBracketMotion tests '%' — jump to the matching bracket.
func TestMatchingBracketMotion(t *testing.T) {
	t.Run("from open to close bracket", func(t *testing.T) {
		e := newTestEditor("(foo [bar])")
		keys(e, '%')
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})

	t.Run("from close back to open bracket", func(t *testing.T) {
		e := newTestEditor("(foo [bar])")
		keys(e, '%', '%')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("finds first bracket after cursor on the line", func(t *testing.T) {
		e := newTestEditor("a (b) c")
		keys(e, '%')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("across lines", func(t *testing.T) {
		e := newTestEditor("func {\n\tbody\n}")
		keys(e, '%')
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run("no bracket on line is a no-op", func(t *testing.T) {
		e := newTestEditor("plain text")
		keys(e, '%')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

func TestOperatorWithUnknownGKeyIsCancelled(t *testing.T) {
	e := newTestEditor("one two three")

	keys(e, 'd', 'g', 'w')
	assert.False(t, e.GetMode().OperatorPending(), "dgw cancels the d, as in Vim")

	keys(e, 'w')
	assert.Equal(t, "one two three", content(e))
	assert.Equal(t, 4, cursorPos(e).Col)
}

// TestFirstNonBlankBlankLine checks ^ and everything else that goes to the
// first non-blank, on lines of only blanks: as Vim's beginline(BL_WHITE|BL_FIX),
// the cursor stops on the last blank, not col 0. I inserts after the blanks.
// Only space and tab are blanks. Expected results are from Vim 9.2.
func TestFirstNonBlankBlankLine(t *testing.T) {
	const (
		spaces = "hello\n   \nworld"
		tabs   = "hello\n\t\t\nworld"
		endsBl = "ab\n   \ncd\n   "
	)
	tests := []struct {
		content string
		keys    string
		want    Position
		text    string // checked if not empty
		mode    Mode   // checked if not empty
	}{
		// Motions.
		{spaces, "j^", Position{1, 2}, "", NormalMode},
		{spaces, "j$^", Position{1, 2}, "", NormalMode},
		{tabs, "j^", Position{1, 1}, "", NormalMode},
		{"x\n \t \ny", "j^", Position{1, 2}, "", NormalMode},
		{spaces, "<CR>", Position{1, 2}, "", NormalMode},
		{spaces, "<CR>j", Position{2, 2}, "", NormalMode},
		{endsBl, "G", Position{3, 2}, "", NormalMode},
		{endsBl, "2G", Position{1, 2}, "", NormalMode},
		{"   \nx", "Ggg", Position{0, 2}, "", NormalMode},
		// gg goes to the first non-blank too, not col 0.
		{"  x\n    y\nz", "Ggg", Position{0, 2}, "", NormalMode},
		{"  x\n    y\nz", "G2gg", Position{1, 4}, "", NormalMode},
		{"  x\n    y\nz", "G9gg", Position{2, 0}, "", NormalMode},
		{"  x\n    y\nz", "Gvgg", Position{0, 2}, "", VisualMode},
		{"  x\n    y\nz", "Gv2gg", Position{1, 4}, "", VisualMode},
		{"  x\n    y\nz", "GVgg", Position{0, 2}, "", VisualLineMode},
		{"  x\n    y\nz", "GV2gg", Position{1, 4}, "", VisualLineMode},
		{spaces, "2z<CR>", Position{1, 2}, "", NormalMode},
		{spaces, "2z.", Position{1, 2}, "", NormalMode},
		{spaces, "2z-", Position{1, 2}, "", NormalMode},
		{spaces, "jv^", Position{1, 2}, "", VisualMode},
		{spaces, "jV^", Position{1, 2}, "", VisualLineMode},
		{spaces, "j$hv^d", Position{1, 0}, "hello\n \nworld", NormalMode},
		// Linewise delete lands on a blank line.
		{"x\nhello\n   \nw", "jdd", Position{1, 2}, "x\n   \nw", NormalMode},
		{endsBl, "Gkdd", Position{2, 2}, "ab\n   \n   ", NormalMode},
		// Operators with ^.
		{spaces, "j0d^", Position{1, 0}, "hello\n \nworld", NormalMode},
		{spaces, "j$d^", Position{1, 2}, spaces, NormalMode},
		{spaces, "jld^", Position{1, 1}, "hello\n  \nworld", NormalMode},
		{spaces, "j0y^$p", Position{1, 4}, "hello\n     \nworld", NormalMode},
		{spaces, "j0c^X<Esc>", Position{1, 0}, "hello\nX \nworld", NormalMode},
		{"a\n b c\n  ", "jj$g~^", Position{2, 1}, "a\n b c\n  ", NormalMode},
		// I inserts after the blanks.
		{spaces, "jIX<Esc>", Position{1, 3}, "hello\n   X\nworld", NormalMode},
		{tabs, "jIX<Esc>", Position{1, 2}, "hello\n\t\tX\nworld", NormalMode},
		// Linewise paste of a blank line.
		{spaces, "jyyp", Position{2, 2}, "hello\n   \n   \nworld", NormalMode},
		{spaces, "jyyP", Position{1, 2}, "hello\n   \n   \nworld", NormalMode},
		{spaces, "jyyggVp", Position{0, 2}, "   \n   \nworld", NormalMode},
		{"hello\n   \nworld\nend", "jyyjjVp", Position{3, 2}, "hello\n   \nworld\n   ", NormalMode},
		// Unaffected: new lines are empty, cc and S clear the line.
		{spaces, "joX<Esc>", Position{2, 0}, "hello\n   \nX\nworld", NormalMode},
		{spaces, "jOX<Esc>", Position{1, 0}, "hello\nX\n   \nworld", NormalMode},
		{spaces, "jccX<Esc>", Position{1, 0}, "hello\nX\nworld", NormalMode},
		{spaces, "jSX<Esc>", Position{1, 0}, "hello\nX\nworld", NormalMode},
		// Only space and tab are blanks.
		{"  x", "$^", Position{0, 0}, "", NormalMode},
		{"\fx", "$^", Position{0, 0}, "", NormalMode},
		{"\vx", "$^", Position{0, 0}, "", NormalMode},
		{"  ", "$^", Position{0, 0}, "", NormalMode},
		{" x", "$IX<Esc>", Position{0, 0}, "X x", NormalMode},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q on %q", tt.keys, tt.content), func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
			assert.Equal(t, tt.mode, e.GetState().Mode)
		})
	}
}

// TestWordBackwardOverBlanks checks that b and B don't stop on the blanks at
// the start of a line: they carry on to the last word of the line above. All
// cases are from Vim 9.2.
func TestWordBackwardOverBlanks(t *testing.T) {
	const (
		tabs      = "one\nab cd\n\t\t\t\tLorem ipsum\nCras"
		spaces    = "one\nab cd\n    Lorem ipsum\nCras"
		blankLine = "ab cd\n   \n  x y"
		emptyLine = "ab cd\n\n  x y"
		first     = "  foo bar\nbaz"
	)
	tests := []struct {
		content string
		keys    string
		want    Position
		text    string // checked if not empty
	}{
		{tabs, "jj0llllb", Position{1, 3}, ""},
		{tabs, "jj0llllB", Position{1, 3}, ""},
		{tabs, "jj0llll2b", Position{1, 0}, ""},
		{tabs, "jj0llb", Position{1, 3}, ""},
		{tabs, "jj0b", Position{1, 3}, ""},
		{tabs, "jj0lllllb", Position{2, 4}, ""},
		{tabs, "jj010l3b", Position{1, 0}, ""},
		{tabs, "jj0lllldb", Position{1, 3}, "one\nab Lorem ipsum\nCras"},
		{tabs, "jj0lllldB", Position{1, 3}, "one\nab Lorem ipsum\nCras"},
		{tabs, "jj0lllld2b", Position{1, 0}, "one\nLorem ipsum\nCras"},
		{tabs, "jj0lldb", Position{1, 3}, "one\nab \t\tLorem ipsum\nCras"},
		{tabs, "jj0llllcbX<Esc>", Position{1, 3}, "one\nab XLorem ipsum\nCras"},
		{tabs, "jj0llllybP", Position{1, 3}, "one\nab cd\n\t\t\t\tcd\n\t\t\t\tLorem ipsum\nCras"},
		{tabs, "jj0llllg~b", Position{1, 3}, "one\nab CD\n\t\t\t\tLorem ipsum\nCras"},
		{spaces, "jj0llllb", Position{1, 3}, ""},
		{spaces, "jj0lllldb", Position{1, 3}, "one\nab Lorem ipsum\nCras"},
		// A line of blanks is skipped; an empty line is a word.
		{blankLine, "jj0llb", Position{0, 3}, ""},
		{blankLine, "jj0ll2b", Position{0, 0}, ""},
		{blankLine, "jj0lldb", Position{0, 3}, "ab x y"},
		{emptyLine, "jj0llb", Position{1, 0}, ""},
		{emptyLine, "jj0lldb", Position{1, 0}, "ab cd\nx y"},
		// Blanks at the start of the buffer: b stops at its start.
		{first, "llb", Position{0, 0}, ""},
		{first, "lb", Position{0, 0}, ""},
		{first, "lldb", Position{0, 0}, "foo bar\nbaz"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q on %q", tt.keys, tt.content), func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
			assert.Equal(t, NormalMode, e.GetState().Mode)
		})
	}
}
