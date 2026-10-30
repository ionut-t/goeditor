package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// curswantText has a short middle line, so jj from the end of the first
// line shows the column j and k go back to: (2,9) when it is still $.
const curswantText = "abcdef\nab\nabcdefghij"

// curswantCase is a key sequence and where Vim 9.2 leaves the cursor and
// its curswant (col, -1 for MAXCOL) after it.
type curswantCase struct {
	content string
	keys    string
	mode    Mode
	want    Position
	col     int
}

// TestCancelledCommandKeepsColumn checks that a key which cancels an
// operator, and a command that changes nothing, leave the column j and k go
// to as it was, as Vim; u, U and CTRL-R set it even with nothing to undo or
// redo. Not here: dz, which goeditor cancels but Vim reads as the start of
// a fold motion (dzj).
func TestCancelledCommandKeepsColumn(t *testing.T) {
	runCurswantCases(t, []curswantCase{
		{curswantText, "$dqjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$cxjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$dcjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$ydjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$cyjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$dgqjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$dgxjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$gUqjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$gUxjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$gUgqjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$gUgxjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$y<Esc>jj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$d<C-a>jj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$9rxjj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$r<Esc>jj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$.jj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$3.jj", NormalMode, Position{2, 9}, -1},
		{curswantText, "$d%jj", NormalMode, Position{2, 9}, -1},
		// A text object sets it, even one that isn't found or doesn't exist.
		{curswantText, "$dizjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$di(jj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$yi(jj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$di\"jj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$gUi(jj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$gUizjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$rxjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$ujj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$Ujj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$<C-r>jj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$hujj", NormalMode, Position{2, 4}, 4},
	})
}

// TestVisualYankSetsColumn checks that visual y sets the column j and k go
// to where the cursor ends, back on the last character when the selection
// started on a line break, as Vim. The rows after o are in
// TestVisualSwapEnds, the ones after switching between v and V in
// TestVisualSwitchKeepsAnchor.
func TestVisualYankSetsColumn(t *testing.T) {
	runCurswantCases(t, []curswantCase{
		{"abc\ndef", "jv<BS>y", NormalMode, Position{0, 2}, 2},
		{"abc\ndefgh", "jv<BS>yj", NormalMode, Position{1, 2}, 2},
		{"abc\ndefgh", "jvh<BS>yj", NormalMode, Position{1, 2}, 2},
		{"abc\ndefgh", "jV<BS>yj", NormalMode, Position{1, 2}, 2},
		{"abc\ndef\nghi", "jV<BS>y", NormalMode, Position{0, 2}, 2},
		{"abc\ndefgh", "jv<BS><Esc>j", NormalMode, Position{1, 2}, 2},
		{"abc\ndefgh", "jv<BS>vj", NormalMode, Position{1, 3}, 3},
		{"abc\ndefgh", "jV<BS>Vj", NormalMode, Position{1, 3}, 3},
		{curswantText, "$vyjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$vjyjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "j$vkyjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "j$vkoyjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "j$Vkyjj", NormalMode, Position{2, 5}, 5},
		{curswantText, "$Vyjj", NormalMode, Position{2, 0}, 0},
		{curswantText, "$Vjyjj", NormalMode, Position{2, 0}, 0},
	})
}

// TestCharSearchSetsColumn checks that f/F/t/T, ; and , set the column j and
// k go to where they land, as Vim, and that a search which fails keeps it.
func TestCharSearchSetsColumn(t *testing.T) {
	const text = "abcdcf\nabcdefghij"
	runCurswantCases(t, []curswantCase{
		{text, "fcj", NormalMode, Position{1, 2}, 2},
		{text, "2fcj", NormalMode, Position{1, 4}, 4},
		{text, "$Fcj", NormalMode, Position{1, 4}, 4},
		{text, "tcj", NormalMode, Position{1, 1}, 1},
		{text, "lTaj", NormalMode, Position{1, 1}, 1},
		{text, "fc;j", NormalMode, Position{1, 4}, 4},
		{text, "tc;j", NormalMode, Position{1, 3}, 3},
		{text, "fc;,j", NormalMode, Position{1, 2}, 2},
		{text, "fcvj", VisualMode, Position{1, 2}, 2},
		{text, "fcv;j", VisualMode, Position{1, 4}, 4},
		{text, "lVfcj", VisualLineMode, Position{1, 2}, 2},
		{text, "$fzj", NormalMode, Position{1, 9}, -1},
		{text, "$9fcj", NormalMode, Position{1, 9}, -1},
		{text, "$;j", NormalMode, Position{1, 9}, -1},
		{text, "fc$;j", NormalMode, Position{1, 9}, -1},
		{text, "$vfzj", VisualMode, Position{1, 10}, -1},
		{text, "$Vfcj", VisualLineMode, Position{1, 10}, -1},
		{text, "yfcj", NormalMode, Position{1, 0}, 0},
		{text, "$yFcj", NormalMode, Position{1, 4}, 4},
		{text, "dfcj", NormalMode, Position{1, 0}, 0},
		{text, "$dFcj", NormalMode, Position{1, 4}, 4},
	})
}

// TestVisualEscapeCancelsTextObject checks that Esc after a visual i/a only
// cancels it and its count, as Vim: the mode, the selection and the column
// j and k go to stay, and a second Esc leaves visual mode.
func TestVisualEscapeCancelsTextObject(t *testing.T) {
	runCurswantCases(t, []curswantCase{
		{"abcdef\nabcdefghij", "$vi<Esc>", VisualMode, Position{0, 5}, -1},
		{"abcdef\nabcdefghij", "$vi<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$va<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$vi<Esc>jk", VisualMode, Position{0, 6}, -1},
		{"abcdef\nabcdefghij", "lvli<Esc>j", VisualMode, Position{1, 2}, 2},
		{"abcdef\nabcdefghij", "$vi<Esc><Esc>", NormalMode, Position{0, 5}, 5},
		{"abcdef\nabcdefghij", "$vi<Esc><Esc>j", NormalMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$v2i<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "lv2i<Esc>l", VisualMode, Position{0, 2}, 2},
		{"abcdef\nabcdefghij", "$Vi<Esc>", VisualLineMode, Position{0, 5}, -1},
		{"abcdef\nabcdefghij", "$Vi<Esc>j", VisualLineMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$Va<Esc>j", VisualLineMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "lVi<Esc>j", VisualLineMode, Position{1, 1}, 1},
		{"abcdef\nabcdefghij", "$Vi<Esc><Esc>j", NormalMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "lVa<Esc><Esc>j", NormalMode, Position{1, 1}, 1},
		{"abcdef\nabcdefghij", "V2i<Esc>j", VisualLineMode, Position{1, 0}, 0},
		{"abcdef\nabcdefghij", "lVix", VisualLineMode, Position{0, 1}, 1},
	})
}

// TestVisualEscapeCancelsPrefix checks that Esc after a visual g, z or
// f/F/t/T only cancels it and its count, as Vim, and that ; still repeats
// the search before it.
func TestVisualEscapeCancelsPrefix(t *testing.T) {
	runCurswantCases(t, []curswantCase{
		{"abcdef\nabcdefghij", "$vg<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$vz<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$vf<Esc>j", VisualMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$Vg<Esc>j", VisualLineMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "$VF<Esc>j", VisualLineMode, Position{1, 10}, -1},
		{"abcdef\nabcdefghij", "lvt<Esc>", VisualMode, Position{0, 1}, 1},
		{"abcdef\nabcdefghij", "lvT<Esc>", VisualMode, Position{0, 1}, 1},
		{"abcdef\nabcdefghij", "lVz<Esc>", VisualLineMode, Position{0, 1}, 1},
		{"abcdef\nabcdefghij", "lv2g<Esc>l", VisualMode, Position{0, 2}, 2},
		{"abcdef\nabcdefghij", "lv2f<Esc>l", VisualMode, Position{0, 2}, 2},
		{"abcdef\nabcdefghij", "lV2z<Esc>j", VisualLineMode, Position{1, 1}, 1},
		{"abcdef\nabcdefghij", "lvg<Esc><Esc>", NormalMode, Position{0, 1}, 1},
		{"abcdef\nabcdefghij", "lVf<Esc><Esc>", NormalMode, Position{0, 1}, 1},
		{"abcdef\nabcdefghij", "lv2<Esc>", NormalMode, Position{0, 1}, 1},
		{"abcdcf\nabcdefghij", "fcvF<Esc>;", VisualMode, Position{0, 4}, 4}, // ; repeats fc, not the cancelled F
	})
}

// TestVisualTextObjectSetsColumn checks that a text object in visual mode
// sets the column j and k go to where the cursor ends, as Vim, also when
// the object isn't found or doesn't exist.
func TestVisualTextObjectSetsColumn(t *testing.T) {
	runCurswantCases(t, []curswantCase{
		{"abc (y)\ndefghijkl", "vi(j", VisualMode, Position{1, 5}, 5},
		{"abc (y)\ndefghijkl", "$vi(j", VisualMode, Position{1, 5}, 5},
		{"abc (y)\ndefghijkl", "va(j", VisualMode, Position{1, 6}, 6},
		{"abc (y)\ndefghijkl", "$va(j", VisualMode, Position{1, 6}, 6},
		{"abc (y)\ndefghijkl", "vi(i(j", VisualMode, Position{1, 5}, 5},
		{"abc (y)\ndefghijkl", "$vi(o<Esc>j", NormalMode, Position{1, 5}, 5},
		{"abc (y)\ndefghijkl", "$vi(<Esc>j", NormalMode, Position{1, 5}, 5},
		{"a (b (c) d)\ndefghijklmnop", "fcvi(i(j", VisualMode, Position{1, 6}, 6},
		{"a (b (c) d)\ndefghijklmnop", "fcvi(a(j", VisualMode, Position{1, 7}, 7},
		{"abc\n  (\n  x\n  )", "vi(j", VisualMode, Position{3, 3}, 3},
		{"abc\n  (\n  x\n  )\nabcdefgh", "jjva(j", VisualMode, Position{4, 2}, 2},
		{"abc ()\ndefghijkl", "$vi(j", VisualMode, Position{1, 5}, 5},
		{"abc ()\ndefghijkl", "$hvi(j", VisualMode, Position{1, 4}, 4},
		{"abc [y]\ndefghijkl", "$vi[j", VisualMode, Position{1, 5}, 5},
		{"abc {y}\ndefghijkl", "$va{j", VisualMode, Position{1, 6}, 6},
		{"abc <y>\ndefghijkl", "$vi<j", VisualMode, Position{1, 5}, 5},
		{"ab \"y\" z\ndefghijkl", "vi\"j", VisualMode, Position{1, 4}, 4},
		{"ab \"y\" z\ndefghijkl", "$vi\"j", VisualMode, Position{1, 7}, 7},
		{"ab \"y\" z\ndefghijkl", "$va\"j", VisualMode, Position{1, 7}, 7},
		{"ab \"y\" z\ndefghijkl", "va\"j", VisualMode, Position{1, 6}, 6},
		{"ab 'y' z\ndefghijkl", "$vi'j", VisualMode, Position{1, 7}, 7},
		{"ab `y` z\ndefghijkl", "$vi`j", VisualMode, Position{1, 7}, 7},
		{"abc de\nfghijklmn", "$hviwj", VisualMode, Position{1, 5}, 5},
		{"abc\nde\n\nfghijkl", "$vipj", VisualLineMode, Position{2, 0}, 0},
		{"abc\nde\n\nfghijkl", "lvipj", VisualLineMode, Position{2, 0}, 0},
		{"abc\nde\n\nfghijkl", "$vapj", VisualLineMode, Position{3, 0}, 0},
		{"abcdef\n\nabcdefghij", "$vipjj", VisualLineMode, Position{2, 0}, 0},
		{"abcdef\nabcdefghij", "$vi(j", VisualMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$vi\"j", VisualMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$vibj", VisualMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$vizj", VisualMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "lvi(j", VisualMode, Position{1, 1}, 1},
		{"abcdef\nabcdefghij", "$Vixj", VisualLineMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$Vizj", VisualLineMode, Position{1, 5}, 5},
		{"abcdef\nabcdefghij", "$Vi(j", VisualLineMode, Position{1, 5}, 5},
	})
}

func runCurswantCases(t *testing.T, tests []curswantCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.mode, e.GetMode().Name())
			assert.Equal(t, tt.want, cursorPos(e))
			col := e.GetBuffer().GetCursor().Preferred
			if col == maxCol {
				col = -1
			}
			assert.Equal(t, tt.col, col, "column for j and k")
		})
	}
}
