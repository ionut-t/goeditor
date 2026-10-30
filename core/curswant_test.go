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
