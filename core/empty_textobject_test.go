package core

import (
	"fmt"
	"testing"
)

// TestOperatorOnEmptyQuotes checks d, c and y on an empty i", i' and i`.
// Vim 9.2 runs the operator on the empty range just after the opening quote:
// d changes nothing but is repeated by '.', c starts insert mode there, and y
// empties the register. The cursor goes to the closing quote.
func TestOperatorOnEmptyQuotes(t *testing.T) {
	const reg = emptyRangeReg

	var tests []emptyRangeCase
	for _, q := range []string{`"`, `'`, "`"} {
		buf := fmt.Sprintf("a%s%sb x%sy%s", q, q, q, q)
		filled := fmt.Sprintf("a%sY%sb x%sy%s", q, q, q, q)
		col := fmt.Sprintf("a%s%sbcdef\nabcdefgh", q, q)
		tests = append(tests,
			emptyRangeCase{name: q + " d before the pair", content: buf, keys: "di" + q, text: buf, want: Position{0, 2}, reg: reg},
			emptyRangeCase{name: q + " d on the open quote", content: buf, keys: "ldi" + q, text: buf, want: Position{0, 2}, reg: reg},
			emptyRangeCase{name: q + " d on the close quote", content: buf, keys: "lldi" + q, text: buf, want: Position{0, 2}, reg: reg},
			emptyRangeCase{name: q + " y", content: buf, keys: "lyi" + q, text: buf, want: Position{0, 2}, reg: ""},
			emptyRangeCase{name: q + " y on the close quote", content: buf, keys: "llyi" + q, text: buf, want: Position{0, 2}, reg: ""},
			emptyRangeCase{name: q + " c", content: buf, keys: "lci" + q, text: buf, want: Position{0, 2}, insert: true, reg: reg},
			emptyRangeCase{name: q + " c then typing", content: buf, keys: "lci" + q + "Y<Esc>", text: filled, want: Position{0, 2}, reg: reg},
			emptyRangeCase{name: q + " d is repeated", content: buf, keys: "ldi" + q + "5l.",
				text: fmt.Sprintf("a%s%sb x%s%s", q, q, q, q), want: Position{0, 7}, reg: anyReg},
			emptyRangeCase{name: q + " c is repeated", content: buf, keys: "lci" + q + "Y<Esc>5l.",
				text: fmt.Sprintf("a%sY%sb x%sY%s", q, q, q, q), want: Position{0, 8}, reg: anyReg},
			emptyRangeCase{name: q + " d sets the column", content: col, keys: "$hhhhhhdi" + q + "j", text: col, want: Position{1, 2}, reg: reg},
			emptyRangeCase{name: q + " y sets the column", content: col, keys: "$hhhhhhyi" + q + "j", text: col, want: Position{1, 2}, reg: ""},
		)
	}
	// iq (any quote) isn't a Vim text object; it acts as the quote it finds.
	tests = append(tests,
		emptyRangeCase{name: "iq d", content: `a""b x"y"`, keys: "ldiq", text: `a""b x"y"`, want: Position{0, 2}, reg: reg},
		emptyRangeCase{name: "iq c then typing", content: `a''b`, keys: "lciqY<Esc>", text: `a'Y'b`, want: Position{0, 2}, reg: reg},
	)

	runEmptyRangeCases(t, tests)
}

// TestOperatorOnEmptyBrackets checks d, c and y on an empty i(, ib, i[, i{,
// iB and i<, on one line and with the brackets on two lines. As with quotes,
// the operator runs on the empty range just after the open bracket.
func TestOperatorOnEmptyBrackets(t *testing.T) {
	const reg = emptyRangeReg

	var tests []emptyRangeCase
	for _, p := range []struct{ open, close string }{{"(", ")"}, {"[", "]"}, {"{", "}"}, {"<", ">"}} {
		o, c := p.open, p.close
		buf := fmt.Sprintf("a%s%sb %sy%s", o, c, o, c)
		objs := []string{o, c}
		switch o {
		case "(":
			objs = append(objs, "b")
		case "{":
			objs = append(objs, "B")
		}
		for _, obj := range objs {
			k := keyName(obj)
			tests = append(tests,
				emptyRangeCase{name: "i" + obj + " d on the open bracket", content: buf, keys: "ldi" + k, text: buf, want: Position{0, 2}, reg: reg},
				emptyRangeCase{name: "i" + obj + " d on the close bracket", content: buf, keys: "lldi" + k, text: buf, want: Position{0, 2}, reg: reg},
				emptyRangeCase{name: "i" + obj + " y", content: buf, keys: "lyi" + k, text: buf, want: Position{0, 2}, reg: ""},
				emptyRangeCase{name: "i" + obj + " c", content: buf, keys: "lci" + k, text: buf, want: Position{0, 2}, insert: true, reg: reg},
				emptyRangeCase{name: "i" + obj + " c then typing", content: buf, keys: "lci" + k + "Y<Esc>",
					text: fmt.Sprintf("a%sY%sb %sy%s", o, c, o, c), want: Position{0, 2}, reg: reg},
			)
		}

		k := keyName(o)
		two := fmt.Sprintf("f%s\n%s", o, c)
		col := fmt.Sprintf("a%s%sbcdef\nabcdefgh", o, c)
		tests = append(tests,
			emptyRangeCase{name: "i" + o + " d is repeated", content: buf, keys: "ldi" + k + "4l.",
				text: fmt.Sprintf("a%s%sb %s%s", o, c, o, c), want: Position{0, 6}, reg: anyReg},
			emptyRangeCase{name: "i" + o + " c is repeated", content: buf, keys: "lci" + k + "Y<Esc>4l.",
				text: fmt.Sprintf("a%sY%sb %sY%s", o, c, o, c), want: Position{0, 7}, reg: anyReg},
			emptyRangeCase{name: "i" + o + " d sets the column", content: col, keys: "$hhhhhhdi" + k + "j", text: col, want: Position{1, 2}, reg: reg},

			// The open bracket ends one line and the close bracket starts the next.
			emptyRangeCase{name: "i" + o + " d over two lines", content: two, keys: "ldi" + k, text: two, want: Position{1, 0}, reg: reg},
			emptyRangeCase{name: "i" + o + " y over two lines", content: two, keys: "lyi" + k, text: two, want: Position{1, 0}, reg: ""},
			emptyRangeCase{name: "i" + o + " c over two lines", content: two, keys: "lci" + k + "Y<Esc>",
				text: fmt.Sprintf("f%s\nY%s", o, c), want: Position{1, 0}, reg: reg},
			emptyRangeCase{name: "i" + o + " c over two lines is repeated", content: two + "\n" + fmt.Sprintf("g%s\n%s", o, c),
				keys: "lci" + k + "Y<Esc>jj.", text: fmt.Sprintf("f%s\nY%s\ng%s\nY%s", o, c, o, c), want: Position{3, 0}, reg: anyReg},
		)
	}

	runEmptyRangeCases(t, tests)
}

// TestOperatorOnEmptyLineWord checks d, c and y on iw on an empty line,
// which Vim 9.2 runs on the empty range there. aw and counts take the
// next line in as well (bug 21), so they aren't covered here. Vim's c sets
// the register to "" there, which goeditor's c doesn't model (anyReg).
func TestOperatorOnEmptyLineWord(t *testing.T) {
	const (
		buf      = "abc\n\ndef\nghi"
		indented = "abc\n\n  def"
		col      = "abcdef\n\nabcdef"
		lastLine = "abc\n\n" // an empty last line
		reg      = emptyRangeReg
	)

	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "d", content: buf, keys: "jdiw", text: buf, want: Position{1, 0}, reg: reg},
		{name: "y", content: buf, keys: "jyiw", text: buf, want: Position{1, 0}, reg: ""},
		{name: "c", content: buf, keys: "jciw", text: buf, want: Position{1, 0}, insert: true, reg: anyReg},
		{name: "c then typing", content: buf, keys: "jciwY<Esc>", text: "abc\nY\ndef\nghi", want: Position{1, 0}, reg: anyReg},
		{name: "d is repeated", content: buf, keys: "jdiwk.", text: "\n\ndef\nghi", want: Position{0, 0}, reg: anyReg},
		{name: "c is repeated", content: buf, keys: "jciwY<Esc>j.", text: "abc\nY\nY\nghi", want: Position{2, 0}, reg: anyReg},
		{name: "d before an indented line", content: indented, keys: "jdiw", text: indented, want: Position{1, 0}, reg: reg},
		{name: "d on the only line", content: "", keys: "diw", text: "", want: Position{0, 0}, reg: reg},
		{name: "c on the only line", content: "", keys: "ciwY<Esc>", text: "Y", want: Position{0, 0}, reg: anyReg},
		{name: "d sets the column", content: col, keys: "j$diwj", text: col, want: Position{2, 0}, reg: reg},
		{name: "y sets the column", content: col, keys: "j$yiwj", text: col, want: Position{2, 0}, reg: ""},

		// aw on an empty last line fails: nothing runs and '.' repeats the
		// earlier change.
		{name: "daw on an empty last line fails", content: lastLine, keys: "iQ<Esc>Gdaw.", text: "Qabc\nQ", want: Position{1, 0}, reg: reg},
	})
}

// keyName gives the key notation for a bracket character typed as a key.
func keyName(r string) string {
	if r == "<" {
		return "<lt>"
	}
	return r
}
