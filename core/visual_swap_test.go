package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Buffers for the visual o/O tests.
const (
	swapRow    = "abcdef"
	swapEight  = "abcdefgh"
	swapDigits = "1\n2\n3\n4"
	swapSix    = "abcdef\nabcdef\nabcdef"
	swapIndent = "hello world\n  second line\nthird line"
	swapWords  = "foo bar baz"
	swapParen  = "x(abc)y"
	swapBlock  = "f(\n  x\n  )"
	swapLines  = "abc\ndef\nghi"
	swapEmpty  = "abc\n\ndef"
	swapLens   = "abcdefgh\nab\nabcdefgh"
)

// swapCase is a key sequence and what Vim 9.2 leaves after it.
type swapCase struct {
	name    string
	content string
	keys    string
	mode    Mode
	want    Position
	anchor  Position // the other end of the selection, in visual mode
	col     int      // the column j and k go to; -1 for $ (MAXCOL)
	text    string   // buffer after the keys
	yanked  string   // clipboard after the keys (set to "REG" first); empty to skip
}

// TestVisualSwapEnds checks o and O in charwise and linewise visual mode
// against Vim 9.2: the cursor goes to the anchor and the anchor to where the
// cursor was, both keeping their columns (also in visual line mode, and on
// a line break), and j and k go back to the column the cursor lands on. The
// operators after it act on the same selection from its other end: linewise
// y, u, U and ~ leave the cursor where it is when it is above the anchor.
// d and c don't write the clipboard in goeditor yet, so those rows don't
// check it. Left out because of other bugs: o after switching between v and
// V (the switch moves the anchor to the cursor), the register after visual
// p, and the column after y from a line break.
func TestVisualSwapEnds(t *testing.T) {
	tests := []swapCase{
		{"row lvlo", swapRow, "lvlo", VisualMode, Position{0, 1}, Position{0, 2}, 1, "abcdef", "REG"},
		{"row lvloh", swapRow, "lvloh", VisualMode, Position{0, 0}, Position{0, 2}, 0, "abcdef", "REG"},
		{"row lvlohd", swapRow, "lvlohd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"row lvlO", swapRow, "lvlO", VisualMode, Position{0, 1}, Position{0, 2}, 1, "abcdef", "REG"},
		{"row lvlOhd", swapRow, "lvlOhd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"row lvloo", swapRow, "lvloo", VisualMode, Position{0, 2}, Position{0, 1}, 2, "abcdef", "REG"},
		{"row lvlool", swapRow, "lvlool", VisualMode, Position{0, 3}, Position{0, 1}, 3, "abcdef", "REG"},
		{"row lvlohy", swapRow, "lvlohy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef", "abc"},
		{"row lvlohc<Esc>", swapRow, "lvlohc<Esc>", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"row lvl2ohd", swapRow, "lvl2ohd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"row lvlohU", swapRow, "lvlohU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ABCdef", "REG"},
		{"row lvlo<Esc>", swapRow, "lvlo<Esc>", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abcdef", "REG"},
		{"row lvlov", swapRow, "lvlov", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abcdef", "REG"},
		{"eight lvlohdl.", swapEight, "lvlohdl.", NormalMode, Position{0, 1}, Position{0, 1}, 1, "dh", ""},
		{"eight lllvlohd", swapEight, "lllvlohd", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abfgh", ""},
		{"eight lllvhho", swapEight, "lllvhho", VisualMode, Position{0, 3}, Position{0, 1}, 3, "abcdefgh", "REG"},
		{"eight lllvhhold", swapEight, "lllvhhold", NormalMode, Position{0, 1}, Position{0, 1}, 1, "afgh", ""},
		{"digits jVjo", swapDigits, "jVjo", VisualLineMode, Position{1, 0}, Position{2, 0}, 0, "1\n2\n3\n4", "REG"},
		{"digits jVjokd", swapDigits, "jVjokd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "4", ""},
		{"digits jVjO", swapDigits, "jVjO", VisualLineMode, Position{1, 0}, Position{2, 0}, 0, "1\n2\n3\n4", "REG"},
		{"digits jVjOkd", swapDigits, "jVjOkd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "4", ""},
		{"digits jVjok", swapDigits, "jVjok", VisualLineMode, Position{0, 0}, Position{2, 0}, 0, "1\n2\n3\n4", "REG"},
		{"digits jVjoky", swapDigits, "jVjoky", NormalMode, Position{0, 0}, Position{0, 0}, 0, "1\n2\n3\n4", "1\n2\n3\n"},
		{"digits jVjokU", swapDigits, "jVjokU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "1\n2\n3\n4", "REG"},
		{"digits jVjo<Esc>", swapDigits, "jVjo<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "1\n2\n3\n4", "REG"},
		{"six lllvjhho", swapSix, "lllvjhho", VisualMode, Position{0, 3}, Position{1, 1}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllvjhhod", swapSix, "lllvjhhod", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abccdef\nabcdef", ""},
		{"six lllvjhhoy", swapSix, "lllvjhhoy", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "def\nab"},
		{"six lllvjhhoU", swapSix, "lllvjhhoU", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcDEF\nABcdef\nabcdef", "REG"},
		{"six jlllvkhho", swapSix, "jlllvkhho", VisualMode, Position{1, 3}, Position{0, 1}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six jlllvkhhoy", swapSix, "jlllvkhhoy", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abcdef\nabcdef\nabcdef", "bcdef\nabcd"},
		{"six jlllvkhhod", swapSix, "jlllvkhhod", NormalMode, Position{0, 1}, Position{0, 1}, 1, "aef\nabcdef", ""},
		{"six lllvjoj", swapSix, "lllvjoj", VisualMode, Position{1, 3}, Position{1, 3}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllvjhhok", swapSix, "lllvjhhok", VisualMode, Position{0, 3}, Position{1, 1}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllvjhho<Esc>", swapSix, "lllvjhho<Esc>", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllvjhhoyj", swapSix, "lllvjhhoyj", NormalMode, Position{1, 3}, Position{1, 3}, 3, "abcdef\nabcdef\nabcdef", "def\nab"},
		{"six lllvjhhocX<Esc>", swapSix, "lllvjhhocX<Esc>", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcXcdef\nabcdef", ""},
		{"six Vjoky", swapSix, "Vjoky", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"six Vjoy", swapSix, "Vjoy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"six lllVjoy", swapSix, "lllVjoy", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"six lllVjoky", swapSix, "lllVjoky", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"six jlllVkoy", swapSix, "jlllVkoy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"six jjlllVkoky", swapSix, "jjlllVkoky", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abcdef\nabcdef\nabcdef", "abcdef\n"},
		{"six jjlllVkkoy", swapSix, "jjlllVkkoy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\nabcdef\n"},
		{"six VjokU", swapSix, "VjokU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ABCDEF\nABCDEF\nabcdef", "REG"},
		{"six lllVjoU", swapSix, "lllVjoU", NormalMode, Position{0, 3}, Position{0, 3}, 3, "ABCDEF\nABCDEF\nabcdef", "REG"},
		{"six jlllVkoU", swapSix, "jlllVkoU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ABCDEF\nABCDEF\nabcdef", "REG"},
		{"six jjlllVkhhoU", swapSix, "jjlllVkhhoU", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abcdef\nABCDEF\nABCDEF", "REG"},
		{"six lllVjhhoU", swapSix, "lllVjhhoU", NormalMode, Position{0, 3}, Position{0, 3}, 3, "ABCDEF\nABCDEF\nabcdef", "REG"},
		{"six lllVjhho<Esc>", swapSix, "lllVjhho<Esc>", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllVjhhoj", swapSix, "lllVjhhoj", VisualLineMode, Position{1, 3}, Position{1, 1}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllVjhhoyj", swapSix, "lllVjhhoyj", NormalMode, Position{1, 3}, Position{1, 3}, 3, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
		{"indent jlllVjoky", swapIndent, "jlllVjoky", NormalMode, Position{0, 3}, Position{0, 3}, 3, "hello world\n  second line\nthird line", "hello world\n  second line\nthird line\n"},
		{"indent jlllVjoy", swapIndent, "jlllVjoy", NormalMode, Position{1, 3}, Position{1, 3}, 3, "hello world\n  second line\nthird line", "  second line\nthird line\n"},
		{"indent jlllVjokU", swapIndent, "jlllVjokU", NormalMode, Position{0, 3}, Position{0, 3}, 3, "HELLO WORLD\n  SECOND LINE\nTHIRD LINE", "REG"},
		{"indent jlllVjoU", swapIndent, "jlllVjoU", NormalMode, Position{1, 3}, Position{1, 3}, 3, "hello world\n  SECOND LINE\nTHIRD LINE", "REG"},
		{"indent jlllVjo~", swapIndent, "jlllVjo~", NormalMode, Position{1, 3}, Position{1, 3}, 3, "hello world\n  SECOND LINE\nTHIRD LINE", "REG"},
		{"indent jlllVjou", swapIndent, "jlllVjou", NormalMode, Position{1, 3}, Position{1, 3}, 3, "hello world\n  second line\nthird line", "REG"},
		{"indent jjlllVkoU", swapIndent, "jjlllVkoU", NormalMode, Position{1, 0}, Position{1, 0}, 0, "hello world\n  SECOND LINE\nTHIRD LINE", "REG"},
		{"indent jlllVjoUj", swapIndent, "jlllVjoUj", NormalMode, Position{2, 3}, Position{2, 3}, 3, "hello world\n  SECOND LINE\nTHIRD LINE", "REG"},
		{"indent jlllVjod", swapIndent, "jlllVjod", NormalMode, Position{0, 0}, Position{0, 0}, 0, "hello world", ""},
		{"indent jlllVjodj", swapIndent, "jlllVjodj", NormalMode, Position{0, 0}, Position{0, 0}, 0, "hello world", ""},
		{"indent jlllVjocX<Esc>", swapIndent, "jlllVjocX<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "hello world\nX", ""},
		{"indent jlllVjokd", swapIndent, "jlllVjokd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "", ""},
		{"indent jjlllVkkhho<Esc>", swapIndent, "jjlllVkkhho<Esc>", NormalMode, Position{2, 3}, Position{2, 3}, 3, "hello world\n  second line\nthird line", "REG"},
		{"indent jjlllVkkhhoy", swapIndent, "jjlllVkkhhoy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "hello world\n  second line\nthird line", "hello world\n  second line\nthird line\n"},
		{"words wviwo", swapWords, "wviwo", VisualMode, Position{0, 4}, Position{0, 6}, 4, "foo bar baz", "REG"},
		{"words wviwod", swapWords, "wviwod", NormalMode, Position{0, 4}, Position{0, 4}, 4, "foo  baz", ""},
		{"words wviwoh", swapWords, "wviwoh", VisualMode, Position{0, 3}, Position{0, 6}, 3, "foo bar baz", "REG"},
		{"words wviwohd", swapWords, "wviwohd", NormalMode, Position{0, 3}, Position{0, 3}, 3, "foo baz", ""},
		{"words wvawo", swapWords, "wvawo", VisualMode, Position{0, 4}, Position{0, 7}, 4, "foo bar baz", "REG"},
		{"words wvawohd", swapWords, "wvawohd", NormalMode, Position{0, 3}, Position{0, 3}, 3, "foobaz", ""},
		{"words wviwoo", swapWords, "wviwoo", VisualMode, Position{0, 6}, Position{0, 4}, 6, "foo bar baz", "REG"},
		{"words wviwoiw", swapWords, "wviwoiw", VisualMode, Position{0, 3}, Position{0, 6}, 3, "foo bar baz", "REG"},
		{"words wviwoaw", swapWords, "wviwoaw", VisualMode, Position{0, 0}, Position{0, 6}, 0, "foo bar baz", "REG"},
		{"words wviwoy", swapWords, "wviwoy", NormalMode, Position{0, 4}, Position{0, 4}, 4, "foo bar baz", "bar"},
		{"words wv2awo", swapWords, "wv2awo", VisualMode, Position{0, 3}, Position{0, 10}, 3, "foo bar baz", "REG"},
		{"paren llvi(o", swapParen, "llvi(o", VisualMode, Position{0, 2}, Position{0, 4}, 2, "x(abc)y", "REG"},
		{"paren llvi(ohd", swapParen, "llvi(ohd", NormalMode, Position{0, 1}, Position{0, 1}, 1, "x)y", ""},
		{"paren llva(o", swapParen, "llva(o", VisualMode, Position{0, 1}, Position{0, 5}, 1, "x(abc)y", "REG"},
		{"paren llva(ohd", swapParen, "llva(ohd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "y", ""},
		{"paren llvi(oy", swapParen, "llvi(oy", NormalMode, Position{0, 2}, Position{0, 2}, 2, "x(abc)y", "abc"},
		{"block jlllvi(o", swapBlock, "jlllvi(o", VisualMode, Position{1, 0}, Position{1, 3}, 0, "f(\n  x\n  )", "REG"},
		{"block jlllvi(o<Esc>", swapBlock, "jlllvi(o<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "f(\n  x\n  )", "REG"},
		{"block jlllvi(od", swapBlock, "jlllvi(od", NormalMode, Position{1, 0}, Position{1, 0}, 0, "f(\n  )", ""},
		{"block jlllvi(oy", swapBlock, "jlllvi(oy", NormalMode, Position{1, 0}, Position{1, 0}, 0, "f(\n  x\n  )", "  x\n"},
		{"block jlllva(o", swapBlock, "jlllva(o", VisualMode, Position{0, 1}, Position{2, 2}, 1, "f(\n  x\n  )", "REG"},
		{"block jlllva(od", swapBlock, "jlllva(od", NormalMode, Position{0, 0}, Position{0, 0}, 0, "f", ""},
		{"block jlllvi(ohd", swapBlock, "jlllvi(ohd", NormalMode, Position{1, 0}, Position{1, 0}, 0, "f(\n  )", ""},
		{"lines jv<BS>o", swapLines, "jv<BS>o", VisualMode, Position{1, 0}, Position{0, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>o<Esc>", swapLines, "jv<BS>o<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>od", swapLines, "jv<BS>od", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcef\nghi", ""},
		{"lines jv<BS>oU", swapLines, "jv<BS>oU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\nDef\nghi", "REG"},
		{"lines jv<BS>oj", swapLines, "jv<BS>oj", VisualMode, Position{2, 0}, Position{0, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>ok", swapLines, "jv<BS>ok", VisualMode, Position{0, 0}, Position{0, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>oh", swapLines, "jv<BS>oh", VisualMode, Position{1, 0}, Position{0, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>ol", swapLines, "jv<BS>ol", VisualMode, Position{1, 1}, Position{0, 3}, 1, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>oo", swapLines, "jv<BS>oo", VisualMode, Position{0, 3}, Position{1, 0}, 3, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>oo<Esc>", swapLines, "jv<BS>oo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines jv<BS>ooj", swapLines, "jv<BS>ooj", VisualMode, Position{1, 3}, Position{1, 0}, 3, "abc\ndef\nghi", "REG"},
		{"lines lv<BS>o", swapLines, "lv<BS>o", VisualMode, Position{0, 1}, Position{0, 0}, 1, "abc\ndef\nghi", "REG"},
		{"lines lv<BS>o<Esc>", swapLines, "lv<BS>o<Esc>", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abc\ndef\nghi", "REG"},
		{"lines $v<BS>o", swapLines, "$v<BS>o", VisualMode, Position{0, 2}, Position{0, 1}, 2, "abc\ndef\nghi", "REG"},
		{"lines $v<BS>o<Esc>", swapLines, "$v<BS>o<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $v<BS>od", swapLines, "$v<BS>od", NormalMode, Position{0, 0}, Position{0, 0}, 0, "a\ndef\nghi", ""},
		{"lines jV<BS>o", swapLines, "jV<BS>o", VisualLineMode, Position{1, 0}, Position{0, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines jV<BS>o<Esc>", swapLines, "jV<BS>o<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines jV<BS>oy", swapLines, "jV<BS>oy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\ndef\nghi", "abc\ndef\n"},
		{"lines jV<BS>oU", swapLines, "jV<BS>oU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ABC\nDEF\nghi", "REG"},
		{"lines jV<BS>od", swapLines, "jV<BS>od", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ghi", ""},
		{"lines jV<BS>oo<Esc>", swapLines, "jV<BS>oo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines lV<BS>o", swapLines, "lV<BS>o", VisualLineMode, Position{0, 1}, Position{0, 0}, 1, "abc\ndef\nghi", "REG"},
		{"lines lV<BS>o<Esc>", swapLines, "lV<BS>o<Esc>", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abc\ndef\nghi", "REG"},
		{"lines $V<BS>o", swapLines, "$V<BS>o", VisualLineMode, Position{0, 2}, Position{0, 1}, 2, "abc\ndef\nghi", "REG"},
		{"lines $V<BS>o<Esc>", swapLines, "$V<BS>o<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $vlo", swapLines, "$vlo", VisualMode, Position{0, 2}, Position{0, 3}, 2, "abc\ndef\nghi", "REG"},
		{"lines $vlo<Esc>", swapLines, "$vlo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $vlod", swapLines, "$vlod", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abdef\nghi", ""},
		{"lines $vloo<Esc>", swapLines, "$vloo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $vlooj", swapLines, "$vlooj", VisualMode, Position{1, 3}, Position{0, 2}, 3, "abc\ndef\nghi", "REG"},
		{"lines $vlohd", swapLines, "$vlohd", NormalMode, Position{0, 1}, Position{0, 1}, 1, "adef\nghi", ""},
		{"lines $vlok<Esc>", swapLines, "$vlok<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $Vlo<Esc>", swapLines, "$Vlo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines $Vlooy", swapLines, "$Vlooy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\ndef\nghi", "abc\n"},
		{"lines $vl<BS>o", swapLines, "$vl<BS>o", VisualMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines v<BS>o", swapLines, "v<BS>o", VisualMode, Position{0, 0}, Position{0, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines v<BS>o<Esc>", swapLines, "v<BS>o<Esc>", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines V<BS>o<Esc>", swapLines, "V<BS>o<Esc>", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines G$vlo", swapLines, "G$vlo", VisualMode, Position{2, 2}, Position{2, 3}, 2, "abc\ndef\nghi", "REG"},
		{"lines G$vlo<Esc>", swapLines, "G$vlo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines G$vlod", swapLines, "G$vlod", NormalMode, Position{2, 1}, Position{2, 1}, 1, "abc\ndef\ngh", ""},
		{"lines G$vloo<Esc>", swapLines, "G$vloo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines G$v<Space>o<Esc>", swapLines, "G$v<Space>o<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines Gv<BS>o", swapLines, "Gv<BS>o", VisualMode, Position{2, 0}, Position{1, 3}, 0, "abc\ndef\nghi", "REG"},
		{"lines Gv<BS>o<Esc>", swapLines, "Gv<BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines Gv<BS>od", swapLines, "Gv<BS>od", NormalMode, Position{1, 3}, Position{1, 3}, 3, "abc\ndefhi", ""},
		{"lines GV<BS>o<Esc>", swapLines, "GV<BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines G$Vlo<Esc>", swapLines, "G$Vlo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines jjv<BS><BS>o<Esc>", swapLines, "jjv<BS><BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\ndef\nghi", "REG"},
		{"lines jjv<BS><BS>od", swapLines, "jjv<BS><BS>od", NormalMode, Position{1, 2}, Position{1, 2}, 2, "abc\ndehi", ""},
		{"empty jv<BS>o", swapEmpty, "jv<BS>o", VisualMode, Position{1, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>o<Esc>", swapEmpty, "jv<BS>o<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>od", swapEmpty, "jv<BS>od", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef", ""},
		{"empty jv<BS>oU", swapEmpty, "jv<BS>oU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty jv<BS>oj", swapEmpty, "jv<BS>oj", VisualMode, Position{2, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>ok", swapEmpty, "jv<BS>ok", VisualMode, Position{0, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>oh", swapEmpty, "jv<BS>oh", VisualMode, Position{1, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>ol", swapEmpty, "jv<BS>ol", VisualMode, Position{1, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jv<BS>oo", swapEmpty, "jv<BS>oo", VisualMode, Position{0, 3}, Position{1, 0}, 3, "abc\n\ndef", "REG"},
		{"empty jv<BS>oo<Esc>", swapEmpty, "jv<BS>oo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty jv<BS>ooj", swapEmpty, "jv<BS>ooj", VisualMode, Position{1, 0}, Position{1, 0}, 3, "abc\n\ndef", "REG"},
		{"empty lv<BS>o", swapEmpty, "lv<BS>o", VisualMode, Position{0, 1}, Position{0, 0}, 1, "abc\n\ndef", "REG"},
		{"empty lv<BS>o<Esc>", swapEmpty, "lv<BS>o<Esc>", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abc\n\ndef", "REG"},
		{"empty $v<BS>o", swapEmpty, "$v<BS>o", VisualMode, Position{0, 2}, Position{0, 1}, 2, "abc\n\ndef", "REG"},
		{"empty $v<BS>o<Esc>", swapEmpty, "$v<BS>o<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $v<BS>od", swapEmpty, "$v<BS>od", NormalMode, Position{0, 0}, Position{0, 0}, 0, "a\n\ndef", ""},
		{"empty jV<BS>o", swapEmpty, "jV<BS>o", VisualLineMode, Position{1, 0}, Position{0, 3}, 0, "abc\n\ndef", "REG"},
		{"empty jV<BS>o<Esc>", swapEmpty, "jV<BS>o<Esc>", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abc\n\ndef", "REG"},
		{"empty jV<BS>oy", swapEmpty, "jV<BS>oy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\n\ndef", "abc\n\n"},
		{"empty jV<BS>oU", swapEmpty, "jV<BS>oU", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ABC\n\ndef", "REG"},
		{"empty jV<BS>od", swapEmpty, "jV<BS>od", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"empty jV<BS>oo<Esc>", swapEmpty, "jV<BS>oo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty lV<BS>o", swapEmpty, "lV<BS>o", VisualLineMode, Position{0, 1}, Position{0, 0}, 1, "abc\n\ndef", "REG"},
		{"empty lV<BS>o<Esc>", swapEmpty, "lV<BS>o<Esc>", NormalMode, Position{0, 1}, Position{0, 1}, 1, "abc\n\ndef", "REG"},
		{"empty $V<BS>o", swapEmpty, "$V<BS>o", VisualLineMode, Position{0, 2}, Position{0, 1}, 2, "abc\n\ndef", "REG"},
		{"empty $V<BS>o<Esc>", swapEmpty, "$V<BS>o<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $vlo", swapEmpty, "$vlo", VisualMode, Position{0, 2}, Position{0, 3}, 2, "abc\n\ndef", "REG"},
		{"empty $vlo<Esc>", swapEmpty, "$vlo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $vlod", swapEmpty, "$vlod", NormalMode, Position{0, 1}, Position{0, 1}, 1, "ab\ndef", ""},
		{"empty $vloo<Esc>", swapEmpty, "$vloo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $vlooj", swapEmpty, "$vlooj", VisualMode, Position{1, 0}, Position{0, 2}, 3, "abc\n\ndef", "REG"},
		{"empty $vlohd", swapEmpty, "$vlohd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "a\ndef", ""},
		{"empty $vlok<Esc>", swapEmpty, "$vlok<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $Vlo<Esc>", swapEmpty, "$Vlo<Esc>", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty $Vlooy", swapEmpty, "$Vlooy", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\n\ndef", "abc\n"},
		{"empty $vl<BS>o", swapEmpty, "$vl<BS>o", VisualMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty v<BS>o", swapEmpty, "v<BS>o", VisualMode, Position{0, 0}, Position{0, 0}, 0, "abc\n\ndef", "REG"},
		{"empty v<BS>o<Esc>", swapEmpty, "v<BS>o<Esc>", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\n\ndef", "REG"},
		{"empty V<BS>o<Esc>", swapEmpty, "V<BS>o<Esc>", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abc\n\ndef", "REG"},
		{"empty G$vlo", swapEmpty, "G$vlo", VisualMode, Position{2, 2}, Position{2, 3}, 2, "abc\n\ndef", "REG"},
		{"empty G$vlo<Esc>", swapEmpty, "G$vlo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\n\ndef", "REG"},
		{"empty G$vlod", swapEmpty, "G$vlod", NormalMode, Position{2, 1}, Position{2, 1}, 1, "abc\n\nde", ""},
		{"empty G$vloo<Esc>", swapEmpty, "G$vloo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\n\ndef", "REG"},
		{"empty G$v<Space>o<Esc>", swapEmpty, "G$v<Space>o<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\n\ndef", "REG"},
		{"empty Gv<BS>o", swapEmpty, "Gv<BS>o", VisualMode, Position{2, 0}, Position{1, 0}, 0, "abc\n\ndef", "REG"},
		{"empty Gv<BS>o<Esc>", swapEmpty, "Gv<BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\n\ndef", "REG"},
		{"empty Gv<BS>od", swapEmpty, "Gv<BS>od", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abc\nef", ""},
		{"empty GV<BS>o<Esc>", swapEmpty, "GV<BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\n\ndef", "REG"},
		{"empty G$Vlo<Esc>", swapEmpty, "G$Vlo<Esc>", NormalMode, Position{2, 2}, Position{2, 2}, 2, "abc\n\ndef", "REG"},
		{"empty jjv<BS><BS>o<Esc>", swapEmpty, "jjv<BS><BS>o<Esc>", NormalMode, Position{2, 0}, Position{2, 0}, 0, "abc\n\ndef", "REG"},
		{"empty jjv<BS><BS>od", swapEmpty, "jjv<BS><BS>od", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcef", ""},
		{"lens $vjo", swapLens, "$vjo", VisualMode, Position{0, 7}, Position{1, 2}, 7, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $vjod", swapLens, "$vjod", NormalMode, Position{0, 7}, Position{0, 7}, 7, "abcdefgabcdefgh", ""},
		{"lens $vjoj", swapLens, "$vjoj", VisualMode, Position{1, 2}, Position{1, 2}, 7, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $vjoo", swapLens, "$vjoo", VisualMode, Position{1, 2}, Position{0, 7}, 2, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $vjooj", swapLens, "$vjooj", VisualMode, Position{2, 2}, Position{0, 7}, 2, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $vjoyj", swapLens, "$vjoyj", NormalMode, Position{1, 1}, Position{1, 1}, 7, "abcdefgh\nab\nabcdefgh", "h\nab\n"},
		{"lens lvjo", swapLens, "lvjo", VisualMode, Position{0, 1}, Position{1, 1}, 1, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens lvjoj", swapLens, "lvjoj", VisualMode, Position{1, 1}, Position{1, 1}, 1, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $Vjo<Esc>", swapLens, "$Vjo<Esc>", NormalMode, Position{0, 7}, Position{0, 7}, 7, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $Vjok", swapLens, "$Vjok", VisualLineMode, Position{0, 7}, Position{1, 2}, 7, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens $Vjoj", swapLens, "$Vjoj", VisualLineMode, Position{1, 2}, Position{1, 2}, 7, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens jv$o", swapLens, "jv$o", VisualMode, Position{1, 0}, Position{1, 2}, 0, "abcdefgh\nab\nabcdefgh", "REG"},
		{"lens jv$od", swapLens, "jv$od", NormalMode, Position{1, 0}, Position{1, 0}, 0, "abcdefgh\nabcdefgh", ""},
		{"lens jv$oj", swapLens, "jv$oj", VisualMode, Position{2, 0}, Position{1, 2}, 0, "abcdefgh\nab\nabcdefgh", "REG"},
		{"six lvjlVokd", swapSix, "lvjlVokd", NormalMode, Position{0, 0}, Position{0, 0}, 0, "abcdef", ""},
		{"lines jV<BS>ooU", swapLines, "jV<BS>ooU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "ABC\nDEF\nghi", "REG"},
		{"lines jV<BS>oou", swapLines, "jV<BS>oou", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\ndef\nghi", "REG"},
		{"lines jV<BS>ooUj", swapLines, "jV<BS>ooUj", NormalMode, Position{1, 2}, Position{1, 2}, 2, "ABC\nDEF\nghi", "REG"},
		{"lines jV<BS>ood", swapLines, "jV<BS>ood", NormalMode, Position{0, 0}, Position{0, 0}, 0, "ghi", ""},
		{"lines jV<BS>U", swapLines, "jV<BS>U", NormalMode, Position{0, 2}, Position{0, 2}, 2, "ABC\nDEF\nghi", "REG"},
		{"lines jv<BS>ooU", swapLines, "jv<BS>ooU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\nDef\nghi", "REG"},
		{"lines jv<BS>ood", swapLines, "jv<BS>ood", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcef\nghi", ""},
		{"empty jV<BS>ooU", swapEmpty, "jV<BS>ooU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "ABC\n\ndef", "REG"},
		{"empty jV<BS>oou", swapEmpty, "jV<BS>oou", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty jV<BS>ooUj", swapEmpty, "jV<BS>ooUj", NormalMode, Position{1, 0}, Position{1, 0}, 2, "ABC\n\ndef", "REG"},
		{"empty jV<BS>ood", swapEmpty, "jV<BS>ood", NormalMode, Position{0, 0}, Position{0, 0}, 0, "def", ""},
		{"empty jV<BS>U", swapEmpty, "jV<BS>U", NormalMode, Position{0, 2}, Position{0, 2}, 2, "ABC\n\ndef", "REG"},
		{"empty jv<BS>ooU", swapEmpty, "jv<BS>ooU", NormalMode, Position{0, 2}, Position{0, 2}, 2, "abc\n\ndef", "REG"},
		{"empty jv<BS>ood", swapEmpty, "jv<BS>ood", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef", ""},
		{"six lllvjhhO<Esc>", swapSix, "lllvjhhO<Esc>", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six jlllVkO<Esc>", swapSix, "jlllVkO<Esc>", NormalMode, Position{1, 3}, Position{1, 3}, 3, "abcdef\nabcdef\nabcdef", "REG"},
		{"six lllVjOy", swapSix, "lllVjOy", NormalMode, Position{0, 3}, Position{0, 3}, 3, "abcdef\nabcdef\nabcdef", "abcdef\nabcdef\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.mode, e.GetMode().Name())
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.mode != NormalMode {
				assert.Equal(t, tt.anchor, e.GetState().VisualStart)
			}
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
