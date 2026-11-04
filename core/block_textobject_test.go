package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The rows below come from Vim 9.2 (current_block in textobject.c). The
// clipboard starts as emptyRangeReg; normal-mode d and c don't write it yet,
// so cases that delete text have reg anyReg. A c case stops after the typed
// Z, in insert mode.

// TestInnerBlockOverLines checks d, y and c on i( over blocks of several
// lines. When the close bracket starts its line (after blanks or in column
// 0), the blanks before it aren't part of the block and the block ends at
// the start of that line, so a block that starts a line becomes linewise
// (:h exclusive-linewise): f(\n  x\n  ) di( deletes the line "  x". A
// body of only blanks is empty.
func TestInnerBlockOverLines(t *testing.T) {
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "indented body: j di(", content: "f(\n  x\n  )", keys: "jdi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi(", content: "f(\n  x\n  )", keys: "jyi(", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci(", content: "f(\n  x\n  )", keys: "jci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: l di(", content: "f(\n  x\n  )", keys: "ldi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: l yi(", content: "f(\n  x\n  )", keys: "lyi(", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: l ci(", content: "f(\n  x\n  )", keys: "lci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: G$ di(", content: "f(\n  x\n  )", keys: "G$di(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: G$ yi(", content: "f(\n  x\n  )", keys: "G$yi(", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: G$ ci(", content: "f(\n  x\n  )", keys: "G$ci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di(", content: "f(\n  )", keys: "ldi(", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi(", content: "f(\n  )", keys: "lyi(", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci(", content: "f(\n  )", keys: "lci(Z", text: "f(\nZ  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: j di(", content: "f(\n  )", keys: "jdi(", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: j yi(", content: "f(\n  )", keys: "jyi(", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: j ci(", content: "f(\n  )", keys: "jci(Z", text: "f(\nZ  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: G$ di(", content: "f(\n  )", keys: "G$di(", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: G$ yi(", content: "f(\n  )", keys: "G$yi(", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: G$ ci(", content: "f(\n  )", keys: "G$ci(Z", text: "f(\nZ  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, close in column 0: l di(", content: "f(\n)", keys: "ldi(", text: "f(\n)", want: Position{1, 0}, reg: reg},
		{name: "empty, close in column 0: l yi(", content: "f(\n)", keys: "lyi(", text: "f(\n)", want: Position{1, 0}, reg: ""},
		{name: "empty, close in column 0: l ci(", content: "f(\n)", keys: "lci(Z", text: "f(\nZ)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di(", content: "f() (\n  x\n)", keys: "jdi(", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi(", content: "f() (\n  x\n)", keys: "jyi(", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci(", content: "f() (\n  x\n)", keys: "jci(Z", text: "f() (\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: $ di(", content: "f() (\n  x\n)", keys: "$di(", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: $ yi(", content: "f() (\n  x\n)", keys: "$yi(", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: $ ci(", content: "f() (\n  x\n)", keys: "$ci(Z", text: "f() (\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: G di(", content: "f() (\n  x\n)", keys: "Gdi(", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: G yi(", content: "f() (\n  x\n)", keys: "Gyi(", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: G ci(", content: "f() (\n  x\n)", keys: "Gci(Z", text: "f() (\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two lines: j di(", content: "f(\n  x\n  y\n  )", keys: "jdi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "two lines: j yi(", content: "f(\n  x\n  y\n  )", keys: "jyi(", text: "f(\n  x\n  y\n  )", want: Position{1, 0}, reg: "  x\n  y\n"},
		{name: "two lines: j ci(", content: "f(\n  x\n  y\n  )", keys: "jci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two lines: jj di(", content: "f(\n  x\n  y\n  )", keys: "jjdi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "two lines: jj yi(", content: "f(\n  x\n  y\n  )", keys: "jjyi(", text: "f(\n  x\n  y\n  )", want: Position{1, 0}, reg: "  x\n  y\n"},
		{name: "two lines: jj ci(", content: "f(\n  x\n  y\n  )", keys: "jjci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two lines: l di(", content: "f(\n  x\n  y\n  )", keys: "ldi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "two lines: l yi(", content: "f(\n  x\n  y\n  )", keys: "lyi(", text: "f(\n  x\n  y\n  )", want: Position{1, 0}, reg: "  x\n  y\n"},
		{name: "two lines: l ci(", content: "f(\n  x\n  y\n  )", keys: "lci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "blank line: l di(", content: "f(\n  \n  )", keys: "ldi(", text: "f(\n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "blank line: l yi(", content: "f(\n  \n  )", keys: "lyi(", text: "f(\n  \n  )", want: Position{1, 0}, reg: ""},
		{name: "blank line: l ci(", content: "f(\n  \n  )", keys: "lci(Z", text: "f(\nZ  \n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "blank line: j di(", content: "f(\n  \n  )", keys: "jdi(", text: "f(\n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "blank line: j yi(", content: "f(\n  \n  )", keys: "jyi(", text: "f(\n  \n  )", want: Position{1, 0}, reg: ""},
		{name: "blank line: j ci(", content: "f(\n  \n  )", keys: "jci(Z", text: "f(\nZ  \n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two blank lines: l di(", content: "f(\n  \n  \n  )", keys: "ldi(", text: "f(\n  \n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "two blank lines: l yi(", content: "f(\n  \n  \n  )", keys: "lyi(", text: "f(\n  \n  \n  )", want: Position{1, 0}, reg: ""},
		{name: "two blank lines: l ci(", content: "f(\n  \n  \n  )", keys: "lci(Z", text: "f(\nZ  \n  \n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two blank lines: j di(", content: "f(\n  \n  \n  )", keys: "jdi(", text: "f(\n  \n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "two blank lines: j yi(", content: "f(\n  \n  \n  )", keys: "jyi(", text: "f(\n  \n  \n  )", want: Position{1, 0}, reg: ""},
		{name: "two blank lines: j ci(", content: "f(\n  \n  \n  )", keys: "jci(Z", text: "f(\nZ  \n  \n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty line: l di(", content: "f(\n\n  )", keys: "ldi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line: l yi(", content: "f(\n\n  )", keys: "lyi(", text: "f(\n\n  )", want: Position{1, 0}, reg: "\n"},
		{name: "empty line: l ci(", content: "f(\n\n  )", keys: "lci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty line: j di(", content: "f(\n\n  )", keys: "jdi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line: j yi(", content: "f(\n\n  )", keys: "jyi(", text: "f(\n\n  )", want: Position{1, 0}, reg: "\n"},
		{name: "empty line: j ci(", content: "f(\n\n  )", keys: "jci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty line, close in column 0: l di(", content: "f(\n\n)", keys: "ldi(", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "empty line, close in column 0: l yi(", content: "f(\n\n)", keys: "lyi(", text: "f(\n\n)", want: Position{1, 0}, reg: "\n"},
		{name: "empty line, close in column 0: l ci(", content: "f(\n\n)", keys: "lci(Z", text: "f(\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty line before the close: l di(", content: "f(\n  x\n\n  )", keys: "ldi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line before the close: l yi(", content: "f(\n  x\n\n  )", keys: "lyi(", text: "f(\n  x\n\n  )", want: Position{1, 0}, reg: "  x\n\n"},
		{name: "empty line before the close: l ci(", content: "f(\n  x\n\n  )", keys: "lci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty line before the close: j di(", content: "f(\n  x\n\n  )", keys: "jdi(", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line before the close: j yi(", content: "f(\n  x\n\n  )", keys: "jyi(", text: "f(\n  x\n\n  )", want: Position{1, 0}, reg: "  x\n\n"},
		{name: "empty line before the close: j ci(", content: "f(\n  x\n\n  )", keys: "jci(Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "text after the open: l di(", content: "f(a,\n  b\n  )", keys: "ldi(", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "text after the open: l yi(", content: "f(a,\n  b\n  )", keys: "lyi(", text: "f(a,\n  b\n  )", want: Position{0, 2}, reg: "a,\n  b"},
		{name: "text after the open: l ci(", content: "f(a,\n  b\n  )", keys: "lci(Z", text: "f(Z\n  )", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "text after the open: j di(", content: "f(a,\n  b\n  )", keys: "jdi(", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "text after the open: j yi(", content: "f(a,\n  b\n  )", keys: "jyi(", text: "f(a,\n  b\n  )", want: Position{0, 2}, reg: "a,\n  b"},
		{name: "text after the open: j ci(", content: "f(a,\n  b\n  )", keys: "jci(Z", text: "f(Z\n  )", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "blanks after the open: l di(", content: "f(  \n  x\n  )", keys: "ldi(", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "blanks after the open: l yi(", content: "f(  \n  x\n  )", keys: "lyi(", text: "f(  \n  x\n  )", want: Position{0, 2}, reg: "  \n  x"},
		{name: "blanks after the open: l ci(", content: "f(  \n  x\n  )", keys: "lci(Z", text: "f(Z\n  )", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "blanks after the open: j di(", content: "f(  \n  x\n  )", keys: "jdi(", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "blanks after the open: j yi(", content: "f(  \n  x\n  )", keys: "jyi(", text: "f(  \n  x\n  )", want: Position{0, 2}, reg: "  \n  x"},
		{name: "blanks after the open: j ci(", content: "f(  \n  x\n  )", keys: "jci(Z", text: "f(Z\n  )", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "indented block: j di(", content: "  if (\n    x\n  )", keys: "jdi(", text: "  if (\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented block: j yi(", content: "  if (\n    x\n  )", keys: "jyi(", text: "  if (\n    x\n  )", want: Position{1, 0}, reg: "    x\n"},
		{name: "indented block: j ci(", content: "  if (\n    x\n  )", keys: "jci(Z", text: "  if (\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "no indent: j di(", content: "(\nx\n)", keys: "jdi(", text: "(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "no indent: j yi(", content: "(\nx\n)", keys: "jyi(", text: "(\nx\n)", want: Position{1, 0}, reg: "x\n"},
		{name: "no indent: j ci(", content: "(\nx\n)", keys: "jci(Z", text: "(\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "text before the close: j di(", content: "f(\n  x)", keys: "jdi(", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "text before the close: j yi(", content: "f(\n  x)", keys: "jyi(", text: "f(\n  x)", want: Position{1, 0}, reg: "  x"},
		{name: "text before the close: j ci(", content: "f(\n  x)", keys: "jci(Z", text: "f(\nZ)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "two lines, text before the close: j di(", content: "f(\n  x\n  y)", keys: "jdi(", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "two lines, text before the close: j yi(", content: "f(\n  x\n  y)", keys: "jyi(", text: "f(\n  x\n  y)", want: Position{1, 0}, reg: "  x\n  y"},
		{name: "two lines, text before the close: j ci(", content: "f(\n  x\n  y)", keys: "jci(Z", text: "f(\nZ)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "text only after the open: l di(", content: "(x\n  )", keys: "ldi(", text: "(\n  )", want: Position{0, 0}, reg: anyReg},
		{name: "text only after the open: l yi(", content: "(x\n  )", keys: "lyi(", text: "(x\n  )", want: Position{0, 1}, reg: "x"},
		{name: "text only after the open: l ci(", content: "(x\n  )", keys: "lci(Z", text: "(Z\n  )", want: Position{0, 2}, insert: true, reg: anyReg},
		{name: "tab indent: j di(", content: "f(\n\tx\n\t)", keys: "jdi(", text: "f(\n\t)", want: Position{1, 1}, reg: anyReg},
		{name: "tab indent: j yi(", content: "f(\n\tx\n\t)", keys: "jyi(", text: "f(\n\tx\n\t)", want: Position{1, 0}, reg: "\tx\n"},
		{name: "tab indent: j ci(", content: "f(\n\tx\n\t)", keys: "jci(Z", text: "f(\nZ\n\t)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "text after the close: j di(", content: "f(\n  x\n  ) + 1", keys: "jdi(", text: "f(\n  ) + 1", want: Position{1, 2}, reg: anyReg},
		{name: "text after the close: j yi(", content: "f(\n  x\n  ) + 1", keys: "jyi(", text: "f(\n  x\n  ) + 1", want: Position{1, 0}, reg: "  x\n"},
		{name: "text after the close: j ci(", content: "f(\n  x\n  ) + 1", keys: "jci(Z", text: "f(\nZ\n  ) + 1", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "line after the block: j di(", content: "f(\n  x\n  )\nend", keys: "jdi(", text: "f(\n  )\nend", want: Position{1, 2}, reg: anyReg},
		{name: "line after the block: j yi(", content: "f(\n  x\n  )\nend", keys: "jyi(", text: "f(\n  x\n  )\nend", want: Position{1, 0}, reg: "  x\n"},
		{name: "line after the block: j ci(", content: "f(\n  x\n  )\nend", keys: "jci(Z", text: "f(\nZ\n  )\nend", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "one line: l di(", content: "f(x)", keys: "ldi(", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line: l yi(", content: "f(x)", keys: "lyi(", text: "f(x)", want: Position{0, 2}, reg: "x"},
		{name: "one line: l ci(", content: "f(x)", keys: "lci(Z", text: "f(Z)", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "one line with blanks: l di(", content: "f( x )", keys: "ldi(", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line with blanks: l yi(", content: "f( x )", keys: "lyi(", text: "f( x )", want: Position{0, 2}, reg: " x "},
		{name: "one line with blanks: l ci(", content: "f( x )", keys: "lci(Z", text: "f(Z)", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "blank line before the close: j di(", content: "f(\n  x\n  \n  )", keys: "jdi(", text: "f(\n  \n  )", want: Position{1, 1}, reg: anyReg},
		{name: "blank line before the close: j yi(", content: "f(\n  x\n  \n  )", keys: "jyi(", text: "f(\n  x\n  \n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "blank line before the close: j ci(", content: "f(\n  x\n  \n  )", keys: "jci(Z", text: "f(\nZ\n  \n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "from the indent: nested di(", content: "(\n  (\n    x\n  )\n)", keys: "jdi(", text: "(\n)", want: Position{1, 0}, reg: anyReg},
	})
}

// TestInnerBlockObjects checks that every bracket object (i), ib, i[, i],
// i{, i}, iB, i<, i>) works as i( above.
func TestInnerBlockObjects(t *testing.T) {
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "indented body: j di)", content: "f(\n  x\n  )", keys: "jdi)", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi)", content: "f(\n  x\n  )", keys: "jyi)", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci)", content: "f(\n  x\n  )", keys: "jci)Z", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j dib", content: "f(\n  x\n  )", keys: "jdib", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yib", content: "f(\n  x\n  )", keys: "jyib", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j cib", content: "f(\n  x\n  )", keys: "jcibZ", text: "f(\nZ\n  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di[", content: "f[\n  x\n  ]", keys: "jdi[", text: "f[\n  ]", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi[", content: "f[\n  x\n  ]", keys: "jyi[", text: "f[\n  x\n  ]", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci[", content: "f[\n  x\n  ]", keys: "jci[Z", text: "f[\nZ\n  ]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di]", content: "f[\n  x\n  ]", keys: "jdi]", text: "f[\n  ]", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi]", content: "f[\n  x\n  ]", keys: "jyi]", text: "f[\n  x\n  ]", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci]", content: "f[\n  x\n  ]", keys: "jci]Z", text: "f[\nZ\n  ]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di{", content: "f{\n  x\n  }", keys: "jdi{", text: "f{\n  }", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi{", content: "f{\n  x\n  }", keys: "jyi{", text: "f{\n  x\n  }", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci{", content: "f{\n  x\n  }", keys: "jci{Z", text: "f{\nZ\n  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di}", content: "f{\n  x\n  }", keys: "jdi}", text: "f{\n  }", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi}", content: "f{\n  x\n  }", keys: "jyi}", text: "f{\n  x\n  }", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci}", content: "f{\n  x\n  }", keys: "jci}Z", text: "f{\nZ\n  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j diB", content: "f{\n  x\n  }", keys: "jdiB", text: "f{\n  }", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yiB", content: "f{\n  x\n  }", keys: "jyiB", text: "f{\n  x\n  }", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ciB", content: "f{\n  x\n  }", keys: "jciBZ", text: "f{\nZ\n  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di<", content: "f<\n  x\n  >", keys: "jdi<lt>", text: "f<\n  >", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi<", content: "f<\n  x\n  >", keys: "jyi<lt>", text: "f<\n  x\n  >", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci<", content: "f<\n  x\n  >", keys: "jci<lt>Z", text: "f<\nZ\n  >", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "indented body: j di>", content: "f<\n  x\n  >", keys: "jdi>", text: "f<\n  >", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j yi>", content: "f<\n  x\n  >", keys: "jyi>", text: "f<\n  x\n  >", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: j ci>", content: "f<\n  x\n  >", keys: "jci>Z", text: "f<\nZ\n  >", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di)", content: "f() (\n  x\n)", keys: "jdi)", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi)", content: "f() (\n  x\n)", keys: "jyi)", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci)", content: "f() (\n  x\n)", keys: "jci)Z", text: "f() (\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j dib", content: "f() (\n  x\n)", keys: "jdib", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yib", content: "f() (\n  x\n)", keys: "jyib", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j cib", content: "f() (\n  x\n)", keys: "jcibZ", text: "f() (\nZ\n)", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di[", content: "f[] [\n  x\n]", keys: "jdi[", text: "f[] [\n]", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi[", content: "f[] [\n  x\n]", keys: "jyi[", text: "f[] [\n  x\n]", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci[", content: "f[] [\n  x\n]", keys: "jci[Z", text: "f[] [\nZ\n]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di]", content: "f[] [\n  x\n]", keys: "jdi]", text: "f[] [\n]", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi]", content: "f[] [\n  x\n]", keys: "jyi]", text: "f[] [\n  x\n]", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci]", content: "f[] [\n  x\n]", keys: "jci]Z", text: "f[] [\nZ\n]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di{", content: "f{} {\n  x\n}", keys: "jdi{", text: "f{} {\n}", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi{", content: "f{} {\n  x\n}", keys: "jyi{", text: "f{} {\n  x\n}", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci{", content: "f{} {\n  x\n}", keys: "jci{Z", text: "f{} {\nZ\n}", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di}", content: "f{} {\n  x\n}", keys: "jdi}", text: "f{} {\n}", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi}", content: "f{} {\n  x\n}", keys: "jyi}", text: "f{} {\n  x\n}", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci}", content: "f{} {\n  x\n}", keys: "jci}Z", text: "f{} {\nZ\n}", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j diB", content: "f{} {\n  x\n}", keys: "jdiB", text: "f{} {\n}", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yiB", content: "f{} {\n  x\n}", keys: "jyiB", text: "f{} {\n  x\n}", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ciB", content: "f{} {\n  x\n}", keys: "jciBZ", text: "f{} {\nZ\n}", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di<", content: "f<> <\n  x\n>", keys: "jdi<lt>", text: "f<> <\n>", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi<", content: "f<> <\n  x\n>", keys: "jyi<lt>", text: "f<> <\n  x\n>", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci<", content: "f<> <\n  x\n>", keys: "jci<lt>Z", text: "f<> <\nZ\n>", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "close in column 0: j di>", content: "f<> <\n  x\n>", keys: "jdi>", text: "f<> <\n>", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j yi>", content: "f<> <\n  x\n>", keys: "jyi>", text: "f<> <\n  x\n>", want: Position{1, 0}, reg: "  x\n"},
		{name: "close in column 0: j ci>", content: "f<> <\n  x\n>", keys: "jci>Z", text: "f<> <\nZ\n>", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di)", content: "f(\n  )", keys: "ldi)", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi)", content: "f(\n  )", keys: "lyi)", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci)", content: "f(\n  )", keys: "lci)Z", text: "f(\nZ  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l dib", content: "f(\n  )", keys: "ldib", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yib", content: "f(\n  )", keys: "lyib", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l cib", content: "f(\n  )", keys: "lcibZ", text: "f(\nZ  )", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di[", content: "f[\n  ]", keys: "ldi[", text: "f[\n  ]", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi[", content: "f[\n  ]", keys: "lyi[", text: "f[\n  ]", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci[", content: "f[\n  ]", keys: "lci[Z", text: "f[\nZ  ]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di]", content: "f[\n  ]", keys: "ldi]", text: "f[\n  ]", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi]", content: "f[\n  ]", keys: "lyi]", text: "f[\n  ]", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci]", content: "f[\n  ]", keys: "lci]Z", text: "f[\nZ  ]", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di{", content: "f{\n  }", keys: "ldi{", text: "f{\n  }", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi{", content: "f{\n  }", keys: "lyi{", text: "f{\n  }", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci{", content: "f{\n  }", keys: "lci{Z", text: "f{\nZ  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di}", content: "f{\n  }", keys: "ldi}", text: "f{\n  }", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi}", content: "f{\n  }", keys: "lyi}", text: "f{\n  }", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci}", content: "f{\n  }", keys: "lci}Z", text: "f{\nZ  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l diB", content: "f{\n  }", keys: "ldiB", text: "f{\n  }", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yiB", content: "f{\n  }", keys: "lyiB", text: "f{\n  }", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ciB", content: "f{\n  }", keys: "lciBZ", text: "f{\nZ  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di<", content: "f<\n  >", keys: "ldi<lt>", text: "f<\n  >", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi<", content: "f<\n  >", keys: "lyi<lt>", text: "f<\n  >", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci<", content: "f<\n  >", keys: "lci<lt>Z", text: "f<\nZ  >", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "empty, indented close: l di>", content: "f<\n  >", keys: "ldi>", text: "f<\n  >", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l yi>", content: "f<\n  >", keys: "lyi>", text: "f<\n  >", want: Position{1, 0}, reg: ""},
		{name: "empty, indented close: l ci>", content: "f<\n  >", keys: "lci>Z", text: "f<\nZ  >", want: Position{1, 1}, insert: true, reg: anyReg},
	})
}

// TestInnerBlockSetsColumn checks where the cursor and the remembered column
// end up after i( over lines: d on the first non-blank of the line after
// the block, y and c (after <Esc>) at the start of the block.
func TestInnerBlockSetsColumn(t *testing.T) {
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "indented body: j di( then j", content: "f(\n  x\n  )", keys: "jdi(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: j di( then k", content: "f(\n  x\n  )", keys: "jdi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "indented body: j yi( then j", content: "f(\n  x\n  )", keys: "jyi(j", text: "f(\n  x\n  )", want: Position{2, 0}, reg: "  x\n"},
		{name: "indented body: l di( then j", content: "f(\n  x\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: l di( then k", content: "f(\n  x\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "indented body: l yi( then j", content: "f(\n  x\n  )", keys: "lyi(j", text: "f(\n  x\n  )", want: Position{2, 0}, reg: "  x\n"},
		{name: "indented body: G$ di( then j", content: "f(\n  x\n  )", keys: "G$di(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented body: G$ di( then k", content: "f(\n  x\n  )", keys: "G$di(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "indented body: G$ yi( then j", content: "f(\n  x\n  )", keys: "G$yi(j", text: "f(\n  x\n  )", want: Position{2, 0}, reg: "  x\n"},
		{name: "empty, indented close: l di( then j", content: "f(\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, indented close: l di( then k", content: "f(\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 0}, reg: reg},
		{name: "empty, indented close: l yi( then j", content: "f(\n  )", keys: "lyi(j", text: "f(\n  )", want: Position{1, 0}, reg: ""},
		{name: "empty, close in column 0: l di( then j", content: "f(\n)", keys: "ldi(j", text: "f(\n)", want: Position{1, 0}, reg: reg},
		{name: "empty, close in column 0: l di( then k", content: "f(\n)", keys: "ldi(k", text: "f(\n)", want: Position{0, 0}, reg: reg},
		{name: "empty, close in column 0: l yi( then j", content: "f(\n)", keys: "lyi(j", text: "f(\n)", want: Position{1, 0}, reg: ""},
		{name: "close in column 0: j di( then j", content: "f() (\n  x\n)", keys: "jdi(j", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j di( then k", content: "f() (\n  x\n)", keys: "jdi(k", text: "f() (\n)", want: Position{0, 0}, reg: anyReg},
		{name: "close in column 0: j yi( then j", content: "f() (\n  x\n)", keys: "jyi(j", text: "f() (\n  x\n)", want: Position{2, 0}, reg: "  x\n"},
		{name: "two lines: j di( then j", content: "f(\n  x\n  y\n  )", keys: "jdi(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "two lines: j di( then k", content: "f(\n  x\n  y\n  )", keys: "jdi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "two lines: j yi( then j", content: "f(\n  x\n  y\n  )", keys: "jyi(j", text: "f(\n  x\n  y\n  )", want: Position{2, 0}, reg: "  x\n  y\n"},
		{name: "blank line: l di( then j", content: "f(\n  \n  )", keys: "ldi(j", text: "f(\n  \n  )", want: Position{2, 0}, reg: reg},
		{name: "blank line: l di( then k", content: "f(\n  \n  )", keys: "ldi(k", text: "f(\n  \n  )", want: Position{0, 0}, reg: reg},
		{name: "blank line: l yi( then j", content: "f(\n  \n  )", keys: "lyi(j", text: "f(\n  \n  )", want: Position{2, 0}, reg: ""},
		{name: "two blank lines: l di( then j", content: "f(\n  \n  \n  )", keys: "ldi(j", text: "f(\n  \n  \n  )", want: Position{2, 0}, reg: reg},
		{name: "two blank lines: l di( then k", content: "f(\n  \n  \n  )", keys: "ldi(k", text: "f(\n  \n  \n  )", want: Position{0, 0}, reg: reg},
		{name: "two blank lines: l yi( then j", content: "f(\n  \n  \n  )", keys: "lyi(j", text: "f(\n  \n  \n  )", want: Position{2, 0}, reg: ""},
		{name: "empty line: l di( then j", content: "f(\n\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line: l di( then k", content: "f(\n\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "empty line: l yi( then j", content: "f(\n\n  )", keys: "lyi(j", text: "f(\n\n  )", want: Position{2, 0}, reg: "\n"},
		{name: "empty line, close in column 0: l di( then j", content: "f(\n\n)", keys: "ldi(j", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "empty line, close in column 0: l di( then k", content: "f(\n\n)", keys: "ldi(k", text: "f(\n)", want: Position{0, 0}, reg: anyReg},
		{name: "empty line, close in column 0: l yi( then j", content: "f(\n\n)", keys: "lyi(j", text: "f(\n\n)", want: Position{2, 0}, reg: "\n"},
		{name: "empty line before the close: l di( then j", content: "f(\n  x\n\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "empty line before the close: l di( then k", content: "f(\n  x\n\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "empty line before the close: l yi( then j", content: "f(\n  x\n\n  )", keys: "lyi(j", text: "f(\n  x\n\n  )", want: Position{2, 0}, reg: "  x\n\n"},
		{name: "text after the open: l di( then j", content: "f(a,\n  b\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 1}, reg: anyReg},
		{name: "text after the open: l di( then k", content: "f(a,\n  b\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "text after the open: l yi( then j", content: "f(a,\n  b\n  )", keys: "lyi(j", text: "f(a,\n  b\n  )", want: Position{1, 2}, reg: "a,\n  b"},
		{name: "blanks after the open: l di( then j", content: "f(  \n  x\n  )", keys: "ldi(j", text: "f(\n  )", want: Position{1, 1}, reg: anyReg},
		{name: "blanks after the open: l di( then k", content: "f(  \n  x\n  )", keys: "ldi(k", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "blanks after the open: l yi( then j", content: "f(  \n  x\n  )", keys: "lyi(j", text: "f(  \n  x\n  )", want: Position{1, 2}, reg: "  \n  x"},
		{name: "indented block: j di( then j", content: "  if (\n    x\n  )", keys: "jdi(j", text: "  if (\n  )", want: Position{1, 2}, reg: anyReg},
		{name: "indented block: j di( then k", content: "  if (\n    x\n  )", keys: "jdi(k", text: "  if (\n  )", want: Position{0, 2}, reg: anyReg},
		{name: "indented block: j yi( then j", content: "  if (\n    x\n  )", keys: "jyi(j", text: "  if (\n    x\n  )", want: Position{2, 0}, reg: "    x\n"},
		{name: "no indent: j di( then j", content: "(\nx\n)", keys: "jdi(j", text: "(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "no indent: j di( then k", content: "(\nx\n)", keys: "jdi(k", text: "(\n)", want: Position{0, 0}, reg: anyReg},
		{name: "no indent: j yi( then j", content: "(\nx\n)", keys: "jyi(j", text: "(\nx\n)", want: Position{2, 0}, reg: "x\n"},
		{name: "text before the close: j di( then j", content: "f(\n  x)", keys: "jdi(j", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "text before the close: j di( then k", content: "f(\n  x)", keys: "jdi(k", text: "f(\n)", want: Position{0, 0}, reg: anyReg},
		{name: "text before the close: j yi( then j", content: "f(\n  x)", keys: "jyi(j", text: "f(\n  x)", want: Position{1, 0}, reg: "  x"},
		{name: "two lines, text before the close: j di( then j", content: "f(\n  x\n  y)", keys: "jdi(j", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "two lines, text before the close: j di( then k", content: "f(\n  x\n  y)", keys: "jdi(k", text: "f(\n)", want: Position{0, 0}, reg: anyReg},
		{name: "two lines, text before the close: j yi( then j", content: "f(\n  x\n  y)", keys: "jyi(j", text: "f(\n  x\n  y)", want: Position{2, 0}, reg: "  x\n  y"},
		{name: "text only after the open: l di( then j", content: "(x\n  )", keys: "ldi(j", text: "(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "text only after the open: l di( then k", content: "(x\n  )", keys: "ldi(k", text: "(\n  )", want: Position{0, 0}, reg: anyReg},
		{name: "text only after the open: l yi( then j", content: "(x\n  )", keys: "lyi(j", text: "(x\n  )", want: Position{1, 1}, reg: "x"},
		{name: "tab indent: j di( then j", content: "f(\n\tx\n\t)", keys: "jdi(j", text: "f(\n\t)", want: Position{1, 1}, reg: anyReg},
		{name: "tab indent: j di( then k", content: "f(\n\tx\n\t)", keys: "jdi(k", text: "f(\n\t)", want: Position{0, 1}, reg: anyReg},
		{name: "tab indent: j yi( then j", content: "f(\n\tx\n\t)", keys: "jyi(j", text: "f(\n\tx\n\t)", want: Position{2, 0}, reg: "\tx\n"},
		{name: "text after the close: j di( then j", content: "f(\n  x\n  ) + 1", keys: "jdi(j", text: "f(\n  ) + 1", want: Position{1, 2}, reg: anyReg},
		{name: "text after the close: j di( then k", content: "f(\n  x\n  ) + 1", keys: "jdi(k", text: "f(\n  ) + 1", want: Position{0, 1}, reg: anyReg},
		{name: "text after the close: j yi( then j", content: "f(\n  x\n  ) + 1", keys: "jyi(j", text: "f(\n  x\n  ) + 1", want: Position{2, 0}, reg: "  x\n"},
		{name: "line after the block: j di( then j", content: "f(\n  x\n  )\nend", keys: "jdi(j", text: "f(\n  )\nend", want: Position{2, 2}, reg: anyReg},
		{name: "line after the block: j di( then k", content: "f(\n  x\n  )\nend", keys: "jdi(k", text: "f(\n  )\nend", want: Position{0, 1}, reg: anyReg},
		{name: "line after the block: j yi( then j", content: "f(\n  x\n  )\nend", keys: "jyi(j", text: "f(\n  x\n  )\nend", want: Position{2, 0}, reg: "  x\n"},
		{name: "one line: l di( then j", content: "f(x)", keys: "ldi(j", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line: l di( then k", content: "f(x)", keys: "ldi(k", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line: l yi( then j", content: "f(x)", keys: "lyi(j", text: "f(x)", want: Position{0, 2}, reg: "x"},
		{name: "one line with blanks: l di( then j", content: "f( x )", keys: "ldi(j", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line with blanks: l di( then k", content: "f( x )", keys: "ldi(k", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line with blanks: l yi( then j", content: "f( x )", keys: "lyi(j", text: "f( x )", want: Position{0, 2}, reg: " x "},
		{name: "blank line before the close: j di( then j", content: "f(\n  x\n  \n  )", keys: "jdi(j", text: "f(\n  \n  )", want: Position{2, 1}, reg: anyReg},
		{name: "blank line before the close: j di( then k", content: "f(\n  x\n  \n  )", keys: "jdi(k", text: "f(\n  \n  )", want: Position{0, 1}, reg: anyReg},
		{name: "blank line before the close: j yi( then j", content: "f(\n  x\n  \n  )", keys: "jyi(j", text: "f(\n  x\n  \n  )", want: Position{2, 0}, reg: "  x\n"},
		{name: "indented body: j ci( esc", content: "f(\n  x\n  )", keys: "jci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented body: l ci( esc", content: "f(\n  x\n  )", keys: "lci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented body: G$ ci( esc", content: "f(\n  x\n  )", keys: "G$ci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "empty, indented close: l ci( esc", content: "f(\n  )", keys: "lci(<Esc>", text: "f(\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty, close in column 0: l ci( esc", content: "f(\n)", keys: "lci(<Esc>", text: "f(\n)", want: Position{1, 0}, reg: reg},
		{name: "close in column 0: j ci( esc", content: "f() (\n  x\n)", keys: "jci(<Esc>", text: "f() (\n\n)", want: Position{1, 0}, reg: anyReg},
		{name: "two lines: j ci( esc", content: "f(\n  x\n  y\n  )", keys: "jci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "blank line: l ci( esc", content: "f(\n  \n  )", keys: "lci(<Esc>", text: "f(\n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "two blank lines: l ci( esc", content: "f(\n  \n  \n  )", keys: "lci(<Esc>", text: "f(\n  \n  \n  )", want: Position{1, 0}, reg: reg},
		{name: "empty line: l ci( esc", content: "f(\n\n  )", keys: "lci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: reg},
		{name: "empty line, close in column 0: l ci( esc", content: "f(\n\n)", keys: "lci(<Esc>", text: "f(\n\n)", want: Position{1, 0}, reg: reg},
		{name: "empty line before the close: l ci( esc", content: "f(\n  x\n\n  )", keys: "lci(<Esc>", text: "f(\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "text after the open: l ci( esc", content: "f(a,\n  b\n  )", keys: "lci(<Esc>", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "blanks after the open: l ci( esc", content: "f(  \n  x\n  )", keys: "lci(<Esc>", text: "f(\n  )", want: Position{0, 1}, reg: anyReg},
		{name: "indented block: j ci( esc", content: "  if (\n    x\n  )", keys: "jci(<Esc>", text: "  if (\n\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "no indent: j ci( esc", content: "(\nx\n)", keys: "jci(<Esc>", text: "(\n\n)", want: Position{1, 0}, reg: anyReg},
		{name: "text before the close: j ci( esc", content: "f(\n  x)", keys: "jci(<Esc>", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "two lines, text before the close: j ci( esc", content: "f(\n  x\n  y)", keys: "jci(<Esc>", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "text only after the open: l ci( esc", content: "(x\n  )", keys: "lci(<Esc>", text: "(\n  )", want: Position{0, 0}, reg: anyReg},
		{name: "tab indent: j ci( esc", content: "f(\n\tx\n\t)", keys: "jci(<Esc>", text: "f(\n\n\t)", want: Position{1, 0}, reg: anyReg},
		{name: "text after the close: j ci( esc", content: "f(\n  x\n  ) + 1", keys: "jci(<Esc>", text: "f(\n\n  ) + 1", want: Position{1, 0}, reg: anyReg},
		{name: "line after the block: j ci( esc", content: "f(\n  x\n  )\nend", keys: "jci(<Esc>", text: "f(\n\n  )\nend", want: Position{1, 0}, reg: anyReg},
		{name: "one line: l ci( esc", content: "f(x)", keys: "lci(<Esc>", text: "f()", want: Position{0, 1}, reg: anyReg},
		{name: "one line with blanks: l ci( esc", content: "f( x )", keys: "lci(<Esc>", text: "f()", want: Position{0, 1}, reg: anyReg},
		{name: "blank line before the close: j ci( esc", content: "f(\n  x\n  \n  )", keys: "jci(<Esc>", text: "f(\n\n  \n  )", want: Position{1, 0}, reg: anyReg},
	})
}

// TestInnerBlockInVisualMode checks vi( over lines. When the close bracket
// starts its line, the selection ends on the line break before it, so d and
// y act on whole lines but charwise. An empty block leaves the selection as
// it is (Vim gets there by selecting from after the open bracket backwards,
// to the same effect).
func TestInnerBlockInVisualMode(t *testing.T) {
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "indented body: j vi(", content: "f(\n  x\n  )", keys: "jvi(", text: "f(\n  x\n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "indented body: j vi(d", content: "f(\n  x\n  )", keys: "jvi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented body: j vi(y", content: "f(\n  x\n  )", keys: "jvi(y", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: l vi(", content: "f(\n  x\n  )", keys: "lvi(", text: "f(\n  x\n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "indented body: l vi(d", content: "f(\n  x\n  )", keys: "lvi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented body: l vi(y", content: "f(\n  x\n  )", keys: "lvi(y", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "indented body: G$ vi(", content: "f(\n  x\n  )", keys: "G$vi(", text: "f(\n  x\n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "indented body: G$ vi(d", content: "f(\n  x\n  )", keys: "G$vi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented body: G$ vi(y", content: "f(\n  x\n  )", keys: "G$vi(y", text: "f(\n  x\n  )", want: Position{1, 0}, reg: "  x\n"},
		{name: "empty, indented close: l vi(", content: "f(\n  )", keys: "lvi(", text: "f(\n  )", want: Position{0, 1}, visual: true, reg: reg},
		{name: "empty, indented close: l vi(d", content: "f(\n  )", keys: "lvi(d", text: "f\n  )", want: Position{0, 0}, reg: anyReg},
		{name: "empty, indented close: l vi(y", content: "f(\n  )", keys: "lvi(y", text: "f(\n  )", want: Position{0, 1}, reg: "("},
		{name: "empty, close in column 0: l vi(", content: "f(\n)", keys: "lvi(", text: "f(\n)", want: Position{0, 1}, visual: true, reg: reg},
		{name: "empty, close in column 0: l vi(d", content: "f(\n)", keys: "lvi(d", text: "f\n)", want: Position{0, 0}, reg: anyReg},
		{name: "empty, close in column 0: l vi(y", content: "f(\n)", keys: "lvi(y", text: "f(\n)", want: Position{0, 1}, reg: "("},
		{name: "close in column 0: j vi(", content: "f() (\n  x\n)", keys: "jvi(", text: "f() (\n  x\n)", want: Position{1, 3}, visual: true, reg: reg},
		{name: "close in column 0: j vi(d", content: "f() (\n  x\n)", keys: "jvi(d", text: "f() (\n)", want: Position{1, 0}, reg: anyReg},
		{name: "close in column 0: j vi(y", content: "f() (\n  x\n)", keys: "jvi(y", text: "f() (\n  x\n)", want: Position{1, 0}, reg: "  x\n"},
		{name: "two lines: j vi(", content: "f(\n  x\n  y\n  )", keys: "jvi(", text: "f(\n  x\n  y\n  )", want: Position{2, 3}, visual: true, reg: reg},
		{name: "two lines: j vi(d", content: "f(\n  x\n  y\n  )", keys: "jvi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "two lines: j vi(y", content: "f(\n  x\n  y\n  )", keys: "jvi(y", text: "f(\n  x\n  y\n  )", want: Position{1, 0}, reg: "  x\n  y\n"},
		{name: "blank line: l vi(", content: "f(\n  \n  )", keys: "lvi(", text: "f(\n  \n  )", want: Position{0, 1}, visual: true, reg: reg},
		{name: "blank line: l vi(d", content: "f(\n  \n  )", keys: "lvi(d", text: "f\n  \n  )", want: Position{0, 0}, reg: anyReg},
		{name: "blank line: l vi(y", content: "f(\n  \n  )", keys: "lvi(y", text: "f(\n  \n  )", want: Position{0, 1}, reg: "("},
		{name: "two blank lines: l vi(", content: "f(\n  \n  \n  )", keys: "lvi(", text: "f(\n  \n  \n  )", want: Position{0, 1}, visual: true, reg: reg},
		{name: "two blank lines: l vi(d", content: "f(\n  \n  \n  )", keys: "lvi(d", text: "f\n  \n  \n  )", want: Position{0, 0}, reg: anyReg},
		{name: "two blank lines: l vi(y", content: "f(\n  \n  \n  )", keys: "lvi(y", text: "f(\n  \n  \n  )", want: Position{0, 1}, reg: "("},
		{name: "empty line: l vi(", content: "f(\n\n  )", keys: "lvi(", text: "f(\n\n  )", want: Position{1, 0}, visual: true, reg: reg},
		{name: "empty line: l vi(d", content: "f(\n\n  )", keys: "lvi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "empty line: l vi(y", content: "f(\n\n  )", keys: "lvi(y", text: "f(\n\n  )", want: Position{1, 0}, reg: "\n"},
		{name: "empty line, close in column 0: l vi(", content: "f(\n\n)", keys: "lvi(", text: "f(\n\n)", want: Position{1, 0}, visual: true, reg: reg},
		{name: "empty line, close in column 0: l vi(d", content: "f(\n\n)", keys: "lvi(d", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "empty line, close in column 0: l vi(y", content: "f(\n\n)", keys: "lvi(y", text: "f(\n\n)", want: Position{1, 0}, reg: "\n"},
		{name: "empty line before the close: l vi(", content: "f(\n  x\n\n  )", keys: "lvi(", text: "f(\n  x\n\n  )", want: Position{2, 0}, visual: true, reg: reg},
		{name: "empty line before the close: l vi(d", content: "f(\n  x\n\n  )", keys: "lvi(d", text: "f(\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "empty line before the close: l vi(y", content: "f(\n  x\n\n  )", keys: "lvi(y", text: "f(\n  x\n\n  )", want: Position{1, 0}, reg: "  x\n\n"},
		{name: "text after the open: l vi(", content: "f(a,\n  b\n  )", keys: "lvi(", text: "f(a,\n  b\n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "text after the open: l vi(d", content: "f(a,\n  b\n  )", keys: "lvi(d", text: "f(  )", want: Position{0, 2}, reg: anyReg},
		{name: "text after the open: l vi(y", content: "f(a,\n  b\n  )", keys: "lvi(y", text: "f(a,\n  b\n  )", want: Position{0, 2}, reg: "a,\n  b\n"},
		{name: "blanks after the open: l vi(", content: "f(  \n  x\n  )", keys: "lvi(", text: "f(  \n  x\n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "blanks after the open: l vi(d", content: "f(  \n  x\n  )", keys: "lvi(d", text: "f(  )", want: Position{0, 2}, reg: anyReg},
		{name: "blanks after the open: l vi(y", content: "f(  \n  x\n  )", keys: "lvi(y", text: "f(  \n  x\n  )", want: Position{0, 2}, reg: "  \n  x\n"},
		{name: "indented block: j vi(", content: "  if (\n    x\n  )", keys: "jvi(", text: "  if (\n    x\n  )", want: Position{1, 5}, visual: true, reg: reg},
		{name: "indented block: j vi(d", content: "  if (\n    x\n  )", keys: "jvi(d", text: "  if (\n  )", want: Position{1, 0}, reg: anyReg},
		{name: "indented block: j vi(y", content: "  if (\n    x\n  )", keys: "jvi(y", text: "  if (\n    x\n  )", want: Position{1, 0}, reg: "    x\n"},
		{name: "no indent: j vi(", content: "(\nx\n)", keys: "jvi(", text: "(\nx\n)", want: Position{1, 1}, visual: true, reg: reg},
		{name: "no indent: j vi(d", content: "(\nx\n)", keys: "jvi(d", text: "(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "no indent: j vi(y", content: "(\nx\n)", keys: "jvi(y", text: "(\nx\n)", want: Position{1, 0}, reg: "x\n"},
		{name: "text before the close: j vi(", content: "f(\n  x)", keys: "jvi(", text: "f(\n  x)", want: Position{1, 2}, visual: true, reg: reg},
		{name: "text before the close: j vi(d", content: "f(\n  x)", keys: "jvi(d", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "text before the close: j vi(y", content: "f(\n  x)", keys: "jvi(y", text: "f(\n  x)", want: Position{1, 0}, reg: "  x"},
		{name: "two lines, text before the close: j vi(", content: "f(\n  x\n  y)", keys: "jvi(", text: "f(\n  x\n  y)", want: Position{2, 2}, visual: true, reg: reg},
		{name: "two lines, text before the close: j vi(d", content: "f(\n  x\n  y)", keys: "jvi(d", text: "f(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "two lines, text before the close: j vi(y", content: "f(\n  x\n  y)", keys: "jvi(y", text: "f(\n  x\n  y)", want: Position{1, 0}, reg: "  x\n  y"},
		{name: "text only after the open: l vi(", content: "(x\n  )", keys: "lvi(", text: "(x\n  )", want: Position{0, 2}, visual: true, reg: reg},
		{name: "text only after the open: l vi(d", content: "(x\n  )", keys: "lvi(d", text: "(  )", want: Position{0, 1}, reg: anyReg},
		{name: "text only after the open: l vi(y", content: "(x\n  )", keys: "lvi(y", text: "(x\n  )", want: Position{0, 1}, reg: "x\n"},
		{name: "tab indent: j vi(", content: "f(\n\tx\n\t)", keys: "jvi(", text: "f(\n\tx\n\t)", want: Position{1, 2}, visual: true, reg: reg},
		{name: "tab indent: j vi(d", content: "f(\n\tx\n\t)", keys: "jvi(d", text: "f(\n\t)", want: Position{1, 0}, reg: anyReg},
		{name: "tab indent: j vi(y", content: "f(\n\tx\n\t)", keys: "jvi(y", text: "f(\n\tx\n\t)", want: Position{1, 0}, reg: "\tx\n"},
		{name: "text after the close: j vi(", content: "f(\n  x\n  ) + 1", keys: "jvi(", text: "f(\n  x\n  ) + 1", want: Position{1, 3}, visual: true, reg: reg},
		{name: "text after the close: j vi(d", content: "f(\n  x\n  ) + 1", keys: "jvi(d", text: "f(\n  ) + 1", want: Position{1, 0}, reg: anyReg},
		{name: "text after the close: j vi(y", content: "f(\n  x\n  ) + 1", keys: "jvi(y", text: "f(\n  x\n  ) + 1", want: Position{1, 0}, reg: "  x\n"},
		{name: "line after the block: j vi(", content: "f(\n  x\n  )\nend", keys: "jvi(", text: "f(\n  x\n  )\nend", want: Position{1, 3}, visual: true, reg: reg},
		{name: "line after the block: j vi(d", content: "f(\n  x\n  )\nend", keys: "jvi(d", text: "f(\n  )\nend", want: Position{1, 0}, reg: anyReg},
		{name: "line after the block: j vi(y", content: "f(\n  x\n  )\nend", keys: "jvi(y", text: "f(\n  x\n  )\nend", want: Position{1, 0}, reg: "  x\n"},
		{name: "one line: l vi(", content: "f(x)", keys: "lvi(", text: "f(x)", want: Position{0, 2}, visual: true, reg: reg},
		{name: "one line: l vi(d", content: "f(x)", keys: "lvi(d", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line: l vi(y", content: "f(x)", keys: "lvi(y", text: "f(x)", want: Position{0, 2}, reg: "x"},
		{name: "one line with blanks: l vi(", content: "f( x )", keys: "lvi(", text: "f( x )", want: Position{0, 4}, visual: true, reg: reg},
		{name: "one line with blanks: l vi(d", content: "f( x )", keys: "lvi(d", text: "f()", want: Position{0, 2}, reg: anyReg},
		{name: "one line with blanks: l vi(y", content: "f( x )", keys: "lvi(y", text: "f( x )", want: Position{0, 2}, reg: " x "},
		{name: "blank line before the close: j vi(", content: "f(\n  x\n  \n  )", keys: "jvi(", text: "f(\n  x\n  \n  )", want: Position{1, 3}, visual: true, reg: reg},
		{name: "blank line before the close: j vi(d", content: "f(\n  x\n  \n  )", keys: "jvi(d", text: "f(\n  \n  )", want: Position{1, 0}, reg: anyReg},
		{name: "blank line before the close: j vi(y", content: "f(\n  x\n  \n  )", keys: "jvi(y", text: "f(\n  x\n  \n  )", want: Position{1, 0}, reg: "  x\n"},
	})
}

// TestInnerBlockRepeat checks that . repeats di( and ci( over lines.
func TestInnerBlockRepeat(t *testing.T) {
	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "di(", content: "f(\n  x\n  )\ng(\n  y\n  )", keys: "jdi(jj.", text: "f(\n  )\ng(\n  )", want: Position{3, 2}, reg: anyReg},
		{name: "ci(", content: "f(\n  x\n  )\ng(\n  y\n  )", keys: "jci(Z<Esc>jjl.", text: "f(\nZ\n  )\ng(\nZ\n  )", want: Position{4, 0}, reg: anyReg},
		{name: "di( on an empty block", content: "f(\n  )\ng(\n  y\n  )", keys: "jdi(jj.", text: "f(\n  )\ng(\n  )", want: Position{3, 2}, reg: anyReg},
		{name: "di( with the close in column 0", content: "f(\n  x\n)\ng(\n  y\n)", keys: "jdi(jj.", text: "f(\n)\ng(\n)", want: Position{3, 0}, reg: anyReg},
		{name: "ci{ with the close in column 0", content: "f() {\n  x\n}\ng() {\n  y\n}", keys: "jci{Z<Esc>jj$.", text: "f() {\nZ\n}\ng() {\nZ\n}", want: Position{4, 0}, reg: anyReg},
	})
}

// TestCurlyBlockFromIndent checks i{ and a{ with the cursor in the indent
// before a {: as Vim, the cursor first moves past the indent, so it picks
// the block that { opens. i( doesn't do that.
func TestCurlyBlockFromIndent(t *testing.T) {
	const reg = emptyRangeReg

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "from the indent: di{", content: "  {\n    x\n  }", keys: "di{", text: "  {\n  }", want: Position{1, 2}, reg: anyReg},
		{name: "from the indent: da{", content: "  {\n    x\n  }", keys: "da{", text: "", want: Position{0, 0}, reg: anyReg},
		{name: "from the indent: yi{", content: "  {\n    x\n  }", keys: "yi{", text: "  {\n    x\n  }", want: Position{1, 0}, reg: "    x\n"},
		{name: "from the indent: ya{", content: "  {\n    x\n  }", keys: "ya{", text: "  {\n    x\n  }", want: Position{0, 2}, reg: "{\n    x\n  }"},
		{name: "from the indent: ci{", content: "  {\n    x\n  }", keys: "ci{Z", text: "  {\nZ\n  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "from the indent: vi{", content: "  {\n    x\n  }", keys: "vi{", text: "  {\n    x\n  }", want: Position{1, 5}, visual: true, reg: reg},
		{name: "from the indent: nested di{", content: "{\n  {\n    x\n  }\n}", keys: "jdi{", text: "{\n  {\n  }\n}", want: Position{2, 2}, reg: anyReg},
		{name: "from the indent: nested di(", content: "(\n  (\n    x\n  )\n)", keys: "jdi(", text: "(\n)", want: Position{1, 0}, reg: anyReg},
		{name: "from the indent: diB", content: "  {\n    x\n  }", keys: "diB", text: "  {\n  }", want: Position{1, 2}, reg: anyReg},
		{name: "from the indent: daB", content: "  {\n    x\n  }", keys: "daB", text: "", want: Position{0, 0}, reg: anyReg},
		{name: "from the indent: yiB", content: "  {\n    x\n  }", keys: "yiB", text: "  {\n    x\n  }", want: Position{1, 0}, reg: "    x\n"},
		{name: "from the indent: yaB", content: "  {\n    x\n  }", keys: "yaB", text: "  {\n    x\n  }", want: Position{0, 2}, reg: "{\n    x\n  }"},
		{name: "from the indent: ciB", content: "  {\n    x\n  }", keys: "ciBZ", text: "  {\nZ\n  }", want: Position{1, 1}, insert: true, reg: anyReg},
		{name: "from the indent: viB", content: "  {\n    x\n  }", keys: "viB", text: "  {\n    x\n  }", want: Position{1, 5}, visual: true, reg: reg},
		{name: "from the indent: nested diB", content: "{\n  {\n    x\n  }\n}", keys: "jdiB", text: "{\n  {\n  }\n}", want: Position{2, 2}, reg: anyReg},
	})
}

// TestAroundBlockOverLines checks a{ over lines. A delete from the indent to
// the end of a line is linewise (op_delete), as for any other motion.
func TestAroundBlockOverLines(t *testing.T) {
	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "after text: jda{", content: "f() {\n  x\n}", keys: "jda{", text: "f() ", want: Position{0, 3}, reg: anyReg},
		{name: "after text: jya{", content: "f() {\n  x\n}", keys: "jya{", text: "f() {\n  x\n}", want: Position{0, 4}, reg: "{\n  x\n}"},
		{name: "after text: jca{", content: "f() {\n  x\n}", keys: "jca{Z", text: "f() Z", want: Position{0, 5}, insert: true, reg: anyReg},
		{name: "after text: jva{d", content: "f() {\n  x\n}", keys: "jva{d", text: "f() ", want: Position{0, 3}, reg: anyReg},
		{name: "own lines: jda{", content: "  {\n    x\n  }", keys: "jda{", text: "", want: Position{0, 0}, reg: anyReg},
		{name: "own lines: jya{", content: "  {\n    x\n  }", keys: "jya{", text: "  {\n    x\n  }", want: Position{0, 2}, reg: "{\n    x\n  }"},
		{name: "own lines: jca{", content: "  {\n    x\n  }", keys: "jca{Z", text: "  Z", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "own lines: jva{d", content: "  {\n    x\n  }", keys: "jva{d", text: "  ", want: Position{0, 1}, reg: anyReg},
		{name: "text after the close: jda{", content: "  {\n    x\n  } y", keys: "jda{", text: "   y", want: Position{0, 2}, reg: anyReg},
		{name: "text after the close: jya{", content: "  {\n    x\n  } y", keys: "jya{", text: "  {\n    x\n  } y", want: Position{0, 2}, reg: "{\n    x\n  }"},
		{name: "text after the close: jca{", content: "  {\n    x\n  } y", keys: "jca{Z", text: "  Z y", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "text after the close: jva{d", content: "  {\n    x\n  } y", keys: "jva{d", text: "   y", want: Position{0, 2}, reg: anyReg},
		{name: "line after the block: jda{", content: "  {\n    x\n  }\nend", keys: "jda{", text: "end", want: Position{0, 0}, reg: anyReg},
		{name: "line after the block: jya{", content: "  {\n    x\n  }\nend", keys: "jya{", text: "  {\n    x\n  }\nend", want: Position{0, 2}, reg: "{\n    x\n  }"},
		{name: "line after the block: jca{", content: "  {\n    x\n  }\nend", keys: "jca{Z", text: "  Z\nend", want: Position{0, 3}, insert: true, reg: anyReg},
		{name: "line after the block: jva{d", content: "  {\n    x\n  }\nend", keys: "jva{d", text: "  \nend", want: Position{0, 1}, reg: anyReg},
	})
}

// selectionBlockCase is a block object typed on a selection over more than
// one character, and the selection it leaves (anchor and cursor, charwise).
type selectionBlockCase struct {
	content, keys  string
	anchor, cursor Position
}

// TestBlockObjectFromSelection checks i(/a( and the other block objects on a
// selection over more than one character, as Vim 9.2's current_block: the
// block is searched from the selection start, the character there not
// counted; i( that doesn't grow the selection goes out a level; a( takes the
// block found even when smaller; with no block the selection is kept.
func TestBlockObjectFromSelection(t *testing.T) {
	const nest = "x(ab(cd)ef)y"
	const lines = "a(bc\nde\nfg)h"
	const curly = "{\n  a {\n    b\n  }\n}"
	tests := []selectionBlockCase{
		// From inside the inner block: i( that doesn't grow goes out a level.
		{nest, "5lvli(", Position{0, 2}, Position{0, 9}},
		{nest, "6lvhi(", Position{0, 2}, Position{0, 9}},
		{nest, "5lvli)", Position{0, 2}, Position{0, 9}},
		{nest, "5lvi(i(", Position{0, 2}, Position{0, 9}},
		{nest, "5lvi(i(i(", Position{0, 2}, Position{0, 9}}, // no third level: kept
		{nest, "5lva(i(", Position{0, 2}, Position{0, 9}},
		{nest, "5lv3li(", Position{0, 2}, Position{0, 9}},
		{nest, "5lvla(", Position{0, 4}, Position{0, 7}},
		{nest, "5lva(a(", Position{0, 1}, Position{0, 10}},
		{nest, "5lva(a(a(", Position{0, 4}, Position{0, 7}}, // from the outer (: the next ( after it
		{nest, "5lv3la(", Position{0, 4}, Position{0, 7}},   // a( may shrink
		// Searched from the start, not the cursor.
		{nest, "2lv3li(", Position{0, 2}, Position{0, 9}},
		{nest, "8lv4hi(", Position{0, 2}, Position{0, 9}},
		{nest, "2lv3la(", Position{0, 1}, Position{0, 10}},
		{nest, "7lvli(", Position{0, 5}, Position{0, 6}}, // from ): the block it closes, i( may shrink
		{nest, "10lvli(", Position{0, 2}, Position{0, 9}},
		// The bracket under the start isn't counted.
		{nest, "4lvli(", Position{0, 2}, Position{0, 9}},
		{nest, "4lv3la(", Position{0, 1}, Position{0, 10}},
		{nest, "lvli(", Position{0, 5}, Position{0, 6}}, // on the outer (: the next ( after it
		{nest, "2lvhi(", Position{0, 5}, Position{0, 6}},
		{nest, "lv9la(", Position{0, 4}, Position{0, 7}},
		{nest, "vli(", Position{0, 2}, Position{0, 9}},
		// No bigger block: kept.
		{nest, "lv9li(", Position{0, 1}, Position{0, 10}},
		{nest, "10lv9hi(", Position{0, 10}, Position{0, 1}},
		{"abc def", "lvli(", Position{0, 1}, Position{0, 2}},
		{"x()y", "vli(", Position{0, 0}, Position{0, 1}},
		{"ab ()", "vli(", Position{0, 0}, Position{0, 1}}, // an empty block isn't found
		{"a(b()c)", "4lvli(", Position{0, 4}, Position{0, 5}},
		{"x(()z)y", "2lvli(", Position{0, 2}, Position{0, 4}},
		{"a(b)c(d)e", "2lvlli(", Position{0, 2}, Position{0, 2}}, // one character: not grown
		{"a(b)c(d)e", "3lvli(", Position{0, 2}, Position{0, 2}},
		// As Vim, going out from the character before the start skips a ) there.
		{"(a)b c(de)f", "3lv7li(", Position{0, 1}, Position{0, 1}},
		{"(a)b c(de)f", "3lv7la(", Position{0, 6}, Position{0, 9}},
		// Over lines.
		{"f(x)\nz", "vji(", Position{0, 2}, Position{0, 2}},
		{"a(b\nc)d", "lvji(", Position{0, 1}, Position{1, 1}},
		{"a(b\nc)d", "jvki(", Position{1, 0}, Position{0, 0}},
		{lines, "2lvji(", Position{0, 2}, Position{2, 1}},
		{lines, "2jvki(", Position{0, 2}, Position{2, 1}},
		{lines, "2lvja(", Position{0, 1}, Position{2, 2}},
		{lines, "2jllvkki(", Position{2, 2}, Position{0, 2}},
		{lines, "vjjllli(", Position{0, 0}, Position{2, 3}},
		{lines, "vjjllla(", Position{0, 1}, Position{2, 2}},
		{lines, "lvjjli(", Position{0, 1}, Position{2, 2}},
		{"x(ab)\n\ncd(ef)", "3lv2jli(", Position{0, 2}, Position{0, 3}},
		{"x(ab)\n\n(ef)", "2jv3li(", Position{2, 0}, Position{2, 3}},
		{"f(\n  x\n)", "jvli(", Position{1, 0}, Position{1, 3}},
		{"f(\n  x\n)", "jvla(", Position{0, 1}, Position{2, 0}},
		{"f(\n  x\n)", "jllvhhi(", Position{1, 2}, Position{1, 0}},
		{"f(\n\n)x", "jjvhi(", Position{1, 0}, Position{1, 0}},
		{"g(f(\n  x\n))", "2jvli(", Position{1, 0}, Position{1, 3}},
		{"g(f(\n  x\n))", "2jvi(i(", Position{0, 2}, Position{2, 0}},
		{curly, "2jvli{", Position{2, 0}, Position{2, 5}},
		{curly, "2jvi{i{", Position{1, 0}, Position{3, 3}},
		{curly, "2jvi{i{i{", Position{1, 0}, Position{3, 3}},
		{curly, "2jva{a{", Position{0, 0}, Position{4, 0}},
		{curly, "jvjli{", Position{1, 0}, Position{3, 3}},
		{curly, "jvjla{", Position{0, 0}, Position{4, 0}},
		// The other brackets.
		{"x[ab[cd]ef]y", "5lvli[", Position{0, 2}, Position{0, 9}},
		{"x[ab[cd]ef]y", "5lvli]", Position{0, 2}, Position{0, 9}},
		{"x[ab[cd]ef]y", "5lvi[i[", Position{0, 2}, Position{0, 9}},
		{"a[bc\nde\nfg]h", "lvjjli[", Position{0, 1}, Position{2, 2}},
		{"x{ab{cd}ef}y", "5lvli{", Position{0, 2}, Position{0, 9}},
		{"x{ab{cd}ef}y", "5lvliB", Position{0, 2}, Position{0, 9}},
		{"x{ab{cd}ef}y", "5lvi{a{", Position{0, 4}, Position{0, 7}},
		{"x{ab{cd}ef}y", "5lva{aB", Position{0, 1}, Position{0, 10}},
		{"a{bc\nde\nfg}h", "lvjjli{", Position{0, 1}, Position{2, 2}},
		{"x<ab<cd>ef>y", "5lvli<lt>", Position{0, 2}, Position{0, 9}},
		{"x<ab<cd>ef>y", "5lvli>", Position{0, 2}, Position{0, 9}},
		{"x<ab<cd>ef>y", "5lvi<lt>i<lt>", Position{0, 2}, Position{0, 9}},
		{"a<bc\nde\nfg>h", "lvjjli<lt>", Position{0, 1}, Position{2, 2}},
		// ib/ab: the same rules, with blocks of any type (a divergence).
		{nest, "5lvlib", Position{0, 2}, Position{0, 9}},
		{nest, "5lvlab", Position{0, 4}, Position{0, 7}},
		{"x(ab[cd]ef)y", "5lvlib", Position{0, 2}, Position{0, 9}},
		{"x(ab[cd]ef)y", "5lvlab", Position{0, 4}, Position{0, 7}},
		{"x(ab[cd]ef)y", "5lvab", Position{0, 4}, Position{0, 7}},
		{"x(ab[cd]ef)y", "5lvabab", Position{0, 1}, Position{0, 10}},
		{"(a)b c[de]f", "3lv7lib", Position{0, 1}, Position{0, 1}},
		// From V, once the selection is more than one character.
		{nest, "5lVi(i(", Position{0, 2}, Position{0, 9}},
		{lines, "jVji(", Position{0, 2}, Position{2, 1}},
		{lines, "2jVki(", Position{0, 2}, Position{2, 1}},
		{curly, "2jVi{i{", Position{1, 0}, Position{3, 3}},
		{curly, "jVji{", Position{1, 0}, Position{3, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e := newTestEditor(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.content, content(e))
			assert.Equal(t, VisualMode, e.GetMode().Name())
			assert.Equal(t, tt.anchor, e.GetState().VisualStart, "anchor")
			assert.Equal(t, tt.cursor, cursorPos(e), "cursor")
			assert.Equal(t, tt.cursor.Col, e.GetBuffer().GetCursor().Preferred, "preferred")
		})
	}
}

// TestBlockObjectFromSelectionOperators checks y, d, c and . on the selection
// a block object gives from a selection over more than one character. Visual
// . repeats on as many characters from the cursor, as Vim.
func TestBlockObjectFromSelectionOperators(t *testing.T) {
	const nest = "x(ab(cd)ef)y"
	const lines = "a(bc\nde\nfg)h"
	tests := []struct {
		content, keys, want string
		cursor              Position
		reg                 string // "" when not checked (d and c don't write the clipboard)
	}{
		{nest, "5lvli(y", nest, Position{0, 2}, "ab(cd)ef"},
		{nest, "5lvli(d", "x()y", Position{0, 2}, ""},
		{nest, "5lvli(cZ<Esc>", "x(Z)y", Position{0, 2}, ""},
		{nest, "5lvla(d", "x(abef)y", Position{0, 4}, ""},
		{nest, "2lv3li(d", "x()y", Position{0, 2}, ""},
		{nest, "5lvli(dlu", nest, Position{0, 2}, ""},
		{nest + "\n" + nest, "5lvli(cZ<Esc>j05l.", "x(Z)y\nx(ab(Z", Position{1, 5}, ""},
		{nest + "\n" + nest, "5lvli(dj05l.", "x()y\nx(ab(", Position{1, 4}, ""},
		{lines, "2lvji(y", lines, Position{0, 2}, "bc\nde\nfg"},
		{lines, "jvki(d", "a()h", Position{0, 2}, ""},
		{lines, "jVji(d", "a()h", Position{0, 2}, ""},
		{lines + "\n" + lines, "2lvji(cZ<Esc>3j02l.", "a(Z)h\na(bc\nde\nfZh", Position{3, 1}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys+" on "+tt.content, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.want, content(e))
			assert.Equal(t, NormalMode, e.GetMode().Name())
			assert.Equal(t, tt.cursor, cursorPos(e), "cursor")
			if tt.reg != "" {
				assert.Equal(t, tt.reg, cb.content)
			}
		})
	}
}
