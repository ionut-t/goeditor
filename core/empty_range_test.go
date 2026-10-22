package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// emptyRangeCase is a key sequence and what Vim 9.2 leaves after it. The
// clipboard starts as emptyRangeReg, so reg shows whether the keys wrote to it.
// Normal-mode deletes don't write the clipboard yet, so a case that deletes
// text has reg anyReg, which skips the check.
type emptyRangeCase struct {
	name    string
	content string
	keys    string // key notation
	text    string
	want    Position
	insert  bool // in insert mode after the keys
	reg     string
}

const (
	emptyRangeReg = "REG"
	anyReg        = "\x00"
)

func runEmptyRangeCases(t *testing.T, tests []emptyRangeCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = emptyRangeReg
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.text, content(e), "text")
			assert.Equal(t, tt.want, cursorPos(e), "cursor")
			assert.Equal(t, tt.insert, e.IsInsertMode(), "insert mode")
			if tt.reg != anyReg {
				assert.Equal(t, tt.reg, cb.content, "register")
			}
		})
	}
}

// TestOperatorOverEmptyRange checks operators whose motion succeeds without
// moving (d0 in column 0, dl on an empty line, $dTx onto the cursor). Vim
// still runs them: c starts insert mode, d and c become the change '.'
// repeats, and y empties the register but is never repeated.
func TestOperatorOverEmptyRange(t *testing.T) {
	const (
		abc    = "abc def"
		indent = "  abc def"
		xs     = "aXbXcXdXe"
		empty  = "abc\n\ndef" // an empty line between two others
		last   = "abc\n\n"    // an empty last line (the final \n ends the file)
		last2  = "abc\n"      // last as GetCurrentContent returns it
		reg    = emptyRangeReg
	)

	runEmptyRangeCases(t, []emptyRangeCase{
		// 0 in column 0.
		{name: "d0", content: abc, keys: "d0", text: abc, want: Position{0, 0}, reg: reg},
		{name: "c0", content: abc, keys: "c0", text: abc, want: Position{0, 0}, insert: true, reg: reg},
		{name: "c0 then typing", content: abc, keys: "c0Y<Esc>", text: "Yabc def", want: Position{0, 0}, reg: reg},
		{name: "y0", content: abc, keys: "y0", text: abc, want: Position{0, 0}, reg: ""},
		{name: "3d0", content: abc, keys: "3d0", text: abc, want: Position{0, 0}, reg: reg},
		{name: "d0 is repeated", content: abc, keys: "d0$.", text: "f", want: Position{0, 0}, reg: anyReg},
		{name: "c0 is repeated", content: abc, keys: "c0Y<Esc>$.", text: "Yf", want: Position{0, 0}, reg: anyReg},
		{name: "c0 with nothing typed is repeated", content: abc, keys: "c0<Esc>$.", text: "f", want: Position{0, 0}, reg: anyReg},

		// ^ on the first non-blank, or in column 0 with no indent.
		{name: "d^", content: indent, keys: "^d^", text: indent, want: Position{0, 2}, reg: reg},
		{name: "c^", content: indent, keys: "^c^Y<Esc>", text: "  Yabc def", want: Position{0, 2}, reg: reg},
		{name: "y^", content: indent, keys: "^y^", text: indent, want: Position{0, 2}, reg: ""},
		{name: "d^ is repeated", content: indent, keys: "^d^$.", text: "  f", want: Position{0, 2}, reg: anyReg},
		{name: "d^ in column 0 is repeated", content: abc, keys: "d^$.", text: "f", want: Position{0, 0}, reg: anyReg},

		// T onto the cursor (T is exclusive, so the range is empty).
		{name: "dT", content: xs, keys: "$dTX", text: xs, want: Position{0, 8}, reg: reg},
		{name: "cT", content: xs, keys: "$cTX", text: xs, want: Position{0, 8}, insert: true, reg: reg},
		{name: "cT then typing", content: xs, keys: "$cTXY<Esc>", text: "aXbXcXdXYe", want: Position{0, 8}, reg: reg},
		{name: "yT", content: xs, keys: "$yTX", text: xs, want: Position{0, 8}, reg: ""},
		{name: "dT is repeated", content: xs, keys: "$dTXh.", text: "aXbXcXXe", want: Position{0, 6}, reg: anyReg},
		{name: "dT is repeated with a count", content: xs, keys: "$dTXh2.", text: "aXbXXe", want: Position{0, 4}, reg: anyReg},
		{name: "cT is repeated", content: xs, keys: "$cTXY<Esc>h.", text: "aXbXcXYXYe", want: Position{0, 6}, reg: anyReg},
		{name: "cT is repeated in place", content: xs, keys: "$cTXY<Esc>.", text: "aXbXcXdXYYe", want: Position{0, 8}, reg: anyReg},
		{name: "dT replaces an earlier change", content: xs, keys: "x$dTXh.", text: "XbXcXXe", want: Position{0, 5}, reg: anyReg},

		// h in column 0.
		{name: "dh", content: abc, keys: "dh", text: abc, want: Position{0, 0}, reg: reg},
		{name: "d5h", content: abc, keys: "d5h", text: abc, want: Position{0, 0}, reg: reg},
		{name: "ch", content: abc, keys: "chY<Esc>", text: "Yabc def", want: Position{0, 0}, reg: reg},
		{name: "yh", content: abc, keys: "yh", text: abc, want: Position{0, 0}, reg: ""},
		{name: "dh is repeated", content: abc, keys: "dhl.", text: "bc def", want: Position{0, 0}, reg: anyReg},
		{name: "ch is repeated", content: abc, keys: "chY<Esc>l.", text: "Yabc def", want: Position{0, 0}, reg: anyReg},

		// l and $ on an empty line.
		{name: "dl", content: empty, keys: "jdl", text: empty, want: Position{1, 0}, reg: reg},
		{name: "cl", content: empty, keys: "jclY<Esc>", text: "abc\nY\ndef", want: Position{1, 0}, reg: reg},
		{name: "yl", content: empty, keys: "jyl", text: empty, want: Position{1, 0}, reg: ""},
		{name: "dl is repeated", content: empty, keys: "jdlk.", text: "bc\n\ndef", want: Position{0, 0}, reg: anyReg},
		{name: "d2l is repeated", content: empty, keys: "jd2lk.", text: "c\n\ndef", want: Position{0, 0}, reg: anyReg},
		{name: "cl is repeated", content: empty, keys: "jclY<Esc>k.", text: "Ybc\nY\ndef", want: Position{0, 0}, reg: anyReg},
		{name: "d$", content: empty, keys: "jd$", text: empty, want: Position{1, 0}, reg: reg},
		{name: "y$", content: empty, keys: "jy$", text: empty, want: Position{1, 0}, reg: ""},
		{name: "d$ is repeated", content: empty, keys: "jd$k.", text: "\n\ndef", want: Position{0, 0}, reg: anyReg},

		// w, W, e and E on an empty last line.
		{name: "dw", content: last, keys: "Gdw", text: last2, want: Position{1, 0}, reg: reg},
		{name: "cw", content: last, keys: "GcwY<Esc>", text: "abc\nY", want: Position{1, 0}, reg: reg},
		{name: "yw", content: last, keys: "Gyw", text: last2, want: Position{1, 0}, reg: ""},
		{name: "dw is repeated", content: last, keys: "Gdwk.", text: "\n", want: Position{0, 0}, reg: anyReg},
		{name: "dW is repeated", content: last, keys: "GdWk.", text: "\n", want: Position{0, 0}, reg: anyReg},
		{name: "de is repeated", content: last, keys: "Gdek.", text: "\n", want: Position{0, 0}, reg: anyReg},
		{name: "dE is repeated", content: last, keys: "GdEk.", text: "\n", want: Position{0, 0}, reg: anyReg},
		{name: "ce", content: last, keys: "GceY<Esc>", text: "abc\nY", want: Position{1, 0}, reg: anyReg},

		// { on the first line and } on an empty last line.
		{name: "d{", content: empty, keys: "d{", text: empty, want: Position{0, 0}, reg: reg},
		{name: "c{", content: empty, keys: "c{Y<Esc>", text: "Yabc\n\ndef", want: Position{0, 0}, reg: reg},
		{name: "y{", content: empty, keys: "y{", text: empty, want: Position{0, 0}, reg: ""},
		{name: "d{ is repeated", content: empty, keys: "d{G.", text: "abc\ndef", want: Position{1, 0}, reg: anyReg},
		{name: "d}", content: last, keys: "Gd}", text: last2, want: Position{1, 0}, reg: reg},
		{name: "c}", content: last, keys: "Gc}Y<Esc>", text: "abc\nY", want: Position{1, 0}, reg: reg},
		{name: "y}", content: last, keys: "Gy}", text: last2, want: Position{1, 0}, reg: ""},
		{name: "d} is repeated", content: last, keys: "Gd}gg.", text: "", want: Position{0, 0}, reg: anyReg},

		// A yank is never repeated, empty or not.
		{name: "y0 is not repeated", content: abc, keys: "xy0l.", text: "b def", want: Position{0, 1}, reg: anyReg},
		{name: "vly is not repeated", content: abc, keys: "xvlyl.", text: "b def", want: Position{0, 1}, reg: anyReg},
	})
}

// TestOperatorWithFailedMotion checks operators whose motion fails: Vim beeps
// and does nothing, c stays in normal mode, the register is left alone, and
// '.' still repeats the change before.
func TestOperatorWithFailedMotion(t *testing.T) {
	const (
		abc   = "abc def"
		lines = "abc\nx\ndef" // no blank lines
		xs    = "aXbXcXdXe"
		reg   = emptyRangeReg
	)

	runEmptyRangeCases(t, []emptyRangeCase{
		// b, B, ge and gE at the start of the buffer.
		{name: "db", content: abc, keys: "db", text: abc, want: Position{0, 0}, reg: reg},
		{name: "d3b", content: abc, keys: "d3b", text: abc, want: Position{0, 0}, reg: reg},
		{name: "cb", content: abc, keys: "cb", text: abc, want: Position{0, 0}, reg: reg},
		{name: "yb", content: abc, keys: "yb", text: abc, want: Position{0, 0}, reg: reg},
		{name: "dB", content: abc, keys: "dB", text: abc, want: Position{0, 0}, reg: reg},
		{name: "cB", content: abc, keys: "cB", text: abc, want: Position{0, 0}, reg: reg},
		{name: "dge", content: abc, keys: "dge", text: abc, want: Position{0, 0}, reg: reg},
		{name: "cge", content: abc, keys: "cge", text: abc, want: Position{0, 0}, reg: reg},
		{name: "yge", content: abc, keys: "yge", text: abc, want: Position{0, 0}, reg: reg},
		{name: "dgE", content: abc, keys: "dgE", text: abc, want: Position{0, 0}, reg: reg},
		{name: "db is not repeated", content: abc, keys: "xdb.", text: "c def", want: Position{0, 0}, reg: anyReg},
		{name: "dge is not repeated", content: abc, keys: "xdge.", text: "c def", want: Position{0, 0}, reg: anyReg},

		// % with no bracket.
		{name: "d%", content: abc, keys: "d%", text: abc, want: Position{0, 0}, reg: reg},
		{name: "c%", content: abc, keys: "c%", text: abc, want: Position{0, 0}, reg: reg},
		{name: "y%", content: abc, keys: "y%", text: abc, want: Position{0, 0}, reg: reg},
		{name: "d% is not repeated", content: abc, keys: "xd%.", text: "c def", want: Position{0, 0}, reg: anyReg},

		// f, F, t and T that find nothing, or fewer than the count.
		{name: "dfz", content: abc, keys: "ldfz", text: abc, want: Position{0, 1}, reg: reg},
		{name: "cFz", content: abc, keys: "lcFz", text: abc, want: Position{0, 1}, reg: reg},
		{name: "ytz", content: abc, keys: "lytz", text: abc, want: Position{0, 1}, reg: reg},
		{name: "dTz is not repeated", content: abc, keys: "xldTz.", text: "b def", want: Position{0, 1}, reg: anyReg},
		{name: "d5T", content: xs, keys: "$d5TX", text: xs, want: Position{0, 8}, reg: reg},

		// ; and , with no search to repeat.
		{name: "d;", content: abc, keys: "ld;", text: abc, want: Position{0, 1}, reg: reg},
		{name: "c,", content: abc, keys: "lc,", text: abc, want: Position{0, 1}, reg: reg},
		{name: "d; is not repeated", content: abc, keys: "xld;.", text: "b def", want: Position{0, 1}, reg: anyReg},

		// j and k past the buffer edges.
		{name: "dk", content: lines, keys: "dk", text: lines, want: Position{0, 0}, reg: reg},
		{name: "ck", content: lines, keys: "ck", text: lines, want: Position{0, 0}, reg: reg},
		{name: "dj", content: lines, keys: "Gdj", text: lines, want: Position{2, 0}, reg: reg},
		{name: "dk is not repeated", content: lines, keys: "xdk.", text: "c\nx\ndef", want: Position{0, 0}, reg: anyReg},
	})
}

// TestEmptyRangeOperatorColumn checks curswant after an operator: one over an
// empty range sets it, as any operator does, and one whose motion fails keeps
// it (the line end after $).
func TestEmptyRangeOperatorColumn(t *testing.T) {
	const xs2 = "aXbXcXdXe\nabcdefghijk"
	const emptyFirst = "\nabcdefghijk"
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "dT sets it", content: xs2, keys: "$dTXj", text: xs2, want: Position{1, 8}, reg: reg},
		{name: "yT sets it", content: xs2, keys: "$yTXj", text: xs2, want: Position{1, 8}, reg: ""},
		{name: "failed df keeps it", content: xs2, keys: "$dfzj", text: xs2, want: Position{1, 10}, reg: reg},
		{name: "failed d% keeps it", content: xs2, keys: "$d%j", text: xs2, want: Position{1, 10}, reg: reg},
		{name: "failed db sets it", content: emptyFirst, keys: "$dbj", text: emptyFirst, want: Position{1, 0}, reg: reg},
		{name: "failed dge sets it", content: emptyFirst, keys: "$dgej", text: emptyFirst, want: Position{1, 0}, reg: reg},
		{name: "failed dk keeps it", content: xs2, keys: "$dkj", text: xs2, want: Position{1, 10}, reg: reg},
		{name: "failed dj keeps it", content: xs2, keys: "j$djk", text: xs2, want: Position{0, 8}, reg: reg},
		{name: "dl on an empty line sets it", content: emptyFirst, keys: "$dlj", text: emptyFirst, want: Position{1, 0}, reg: reg},
	})
}
