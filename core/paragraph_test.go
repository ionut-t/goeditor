package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Buffers for the paragraph motion tests. goeditor's SetContent drops one
// trailing line break, so a buffer whose last line is empty ends in two.
const (
	paraNoBlank    = "abc\nx\ndef"
	paraOneBlank   = "abc\n\ndef"
	paraThree      = "p1a\np1b\n\np2a\np2b\n\np3a\np3b"
	paraBlankEdges = "\nabc\n\ndef\n\n"
	paraBlankRuns  = "abc\n\n\ndef\n\n\nghi"
	paraEmptyLast  = "abc\ndef\n\n"
	paraBlanksLine = "abc\n  \ndef"
	paraIndented   = "  abc\ndef\n\n  ghi"
	paraLong       = "abcdef\nx\nabcdef"
)

// paragraphCase is a key sequence and what Vim 9.2 leaves after it.
type paragraphCase struct {
	name    string
	content string
	keys    []KeyEvent
	want    Position
	text    string // buffer after the keys; empty when unchanged
	empty   bool   // the buffer is empty after the keys
	yanked  string // clipboard after the keys; empty to skip the check
}

// runParagraphCases runs each case with the clipboard set to "REG", so a
// failed y can check that it is unchanged.
func runParagraphCases(t *testing.T, tests []paragraphCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			cb.content = "REG"
			want := tt.text
			if want == "" && !tt.empty {
				want = content(e)
			}
			for _, k := range tt.keys {
				_ = e.HandleKey(k)
			}
			assert.Equal(t, tt.want, cursorPos(e))
			assert.Equal(t, want, content(e))
			if tt.yanked != "" {
				assert.Equal(t, tt.yanked, cb.content)
			}
		})
	}
}

// TestParagraphMotion checks { and } against Vim 9.2's findpar: count times,
// move past at least one non-empty line and stop on the next empty one (a
// line of blanks isn't empty). With the count used up at the buffer edge, {
// stops at (0, 0) and } on the last character of the last line; with count
// left over the motion fails and doesn't move. Either way it sets the column
// j and k go back to (j$2{k, $}k).
func TestParagraphMotion(t *testing.T) {
	seq := keySeq
	runParagraphCases(t, []paragraphCase{
		{name: "noBlank ll{", content: paraNoBlank, keys: seq("ll{"), want: Position{0, 0}},
		{name: "noBlank }", content: paraNoBlank, keys: seq("}"), want: Position{2, 2}},
		{name: "noBlank G}", content: paraNoBlank, keys: seq("G}"), want: Position{2, 2}},
		{name: "noBlank G2{", content: paraNoBlank, keys: seq("G2{"), want: Position{2, 0}},
		{name: "noBlank 2}", content: paraNoBlank, keys: seq("2}"), want: Position{0, 0}},
		{name: "noBlank j3}", content: paraNoBlank, keys: seq("j3}"), want: Position{1, 0}},
		{name: "noBlank 3{", content: paraNoBlank, keys: seq("3{"), want: Position{0, 0}},
		{name: "oneBlank }", content: paraOneBlank, keys: seq("}"), want: Position{1, 0}},
		{name: "oneBlank 2}", content: paraOneBlank, keys: seq("2}"), want: Position{2, 2}},
		{name: "oneBlank 3}", content: paraOneBlank, keys: seq("3}"), want: Position{0, 0}},
		{name: "oneBlank j}", content: paraOneBlank, keys: seq("j}"), want: Position{2, 2}},
		{name: "oneBlank j{", content: paraOneBlank, keys: seq("j{"), want: Position{0, 0}},
		{name: "oneBlank G{", content: paraOneBlank, keys: seq("G{"), want: Position{1, 0}},
		{name: "oneBlank G2{", content: paraOneBlank, keys: seq("G2{"), want: Position{0, 0}},
		{name: "oneBlank G3{", content: paraOneBlank, keys: seq("G3{"), want: Position{2, 0}},
		{name: "three }", content: paraThree, keys: seq("}"), want: Position{2, 0}},
		{name: "three 2}", content: paraThree, keys: seq("2}"), want: Position{5, 0}},
		{name: "three 3}", content: paraThree, keys: seq("3}"), want: Position{7, 2}},
		{name: "three l3}", content: paraThree, keys: seq("l3}"), want: Position{7, 2}},
		{name: "three G{", content: paraThree, keys: seq("G{"), want: Position{5, 0}},
		{name: "three G2{", content: paraThree, keys: seq("G2{"), want: Position{2, 0}},
		{name: "three G3{", content: paraThree, keys: seq("G3{"), want: Position{0, 0}},
		{name: "three Gk3{", content: paraThree, keys: seq("Gk3{"), want: Position{0, 0}},
		{name: "three 3jl{", content: paraThree, keys: seq("3jl{"), want: Position{2, 0}},
		{name: "three 5j}", content: paraThree, keys: seq("5j}"), want: Position{7, 2}},
		{name: "three 2j{", content: paraThree, keys: seq("2j{"), want: Position{0, 0}},
		{name: "three 2j2{", content: paraThree, keys: seq("2j2{"), want: Position{2, 0}},
		{name: "blankEdges }", content: paraBlankEdges, keys: seq("}"), want: Position{2, 0}},
		{name: "blankEdges 3}", content: paraBlankEdges, keys: seq("3}"), want: Position{4, 0}},
		{name: "blankEdges G}", content: paraBlankEdges, keys: seq("G}"), want: Position{4, 0}},
		{name: "blankEdges {", content: paraBlankEdges, keys: seq("{"), want: Position{0, 0}},
		{name: "blankEdges j{", content: paraBlankEdges, keys: seq("j{"), want: Position{0, 0}},
		{name: "blankEdges 2}", content: paraBlankEdges, keys: seq("2}"), want: Position{4, 0}},
		{name: "blankEdges 4}", content: paraBlankEdges, keys: seq("4}"), want: Position{0, 0}},
		{name: "blankEdges G2{", content: paraBlankEdges, keys: seq("G2{"), want: Position{0, 0}},
		{name: "blankEdges G3{", content: paraBlankEdges, keys: seq("G3{"), want: Position{0, 0}},
		{name: "blankRuns }", content: paraBlankRuns, keys: seq("}"), want: Position{1, 0}},
		{name: "blankRuns 2}", content: paraBlankRuns, keys: seq("2}"), want: Position{4, 0}},
		{name: "blankRuns 3}", content: paraBlankRuns, keys: seq("3}"), want: Position{6, 2}},
		{name: "blankRuns j}", content: paraBlankRuns, keys: seq("j}"), want: Position{4, 0}},
		{name: "blankRuns G{", content: paraBlankRuns, keys: seq("G{"), want: Position{5, 0}},
		{name: "blankRuns G2{", content: paraBlankRuns, keys: seq("G2{"), want: Position{2, 0}},
		{name: "blankRuns 2j{", content: paraBlankRuns, keys: seq("2j{"), want: Position{0, 0}},
		{name: "emptyLast }", content: paraEmptyLast, keys: seq("}"), want: Position{2, 0}},
		{name: "emptyLast G{", content: paraEmptyLast, keys: seq("G{"), want: Position{0, 0}},
		{name: "emptyLast G}", content: paraEmptyLast, keys: seq("G}"), want: Position{2, 0}},
		{name: "emptyLast j$}", content: paraEmptyLast, keys: seq("j$}"), want: Position{2, 0}},
		{name: "emptyLast 2}", content: paraEmptyLast, keys: seq("2}"), want: Position{2, 0}},
		{name: "blanksLine }", content: paraBlanksLine, keys: seq("}"), want: Position{2, 2}},
		{name: "blanksLine G{", content: paraBlanksLine, keys: seq("G{"), want: Position{0, 0}},
		{name: "indented w2}", content: paraIndented, keys: seq("w2}"), want: Position{3, 4}},
		{name: "long j$2{k", content: paraLong, keys: seq("j$2{k"), want: Position{0, 0}},
		{name: "long j$2}j", content: paraLong, keys: seq("j$2}j"), want: Position{2, 0}},
		{name: "long G$}k", content: paraLong, keys: seq("G$}k"), want: Position{1, 0}},
		{name: "long ll}k", content: paraLong, keys: seq("ll}k"), want: Position{1, 0}},
		{name: "long Gll}k", content: paraLong, keys: seq("Gll}k"), want: Position{1, 0}},
		{name: "long G$2{k", content: paraLong, keys: seq("G$2{k"), want: Position{1, 0}},
		{name: "long j${k", content: paraLong, keys: seq("j${k"), want: Position{0, 0}},
		{name: "long $}k", content: paraLong, keys: seq("$}k"), want: Position{1, 0}},
		{name: "long j$}k", content: paraLong, keys: seq("j$}k"), want: Position{1, 0}},
	})
}

// TestParagraphMotionWithOperator checks d, y and c with { and }. A } that
// ends on the last character of the last line is inclusive; a { or } that
// fails runs no operator (c doesn't insert, y leaves the clipboard). An end
// in column 0 of a later line follows :h exclusive-linewise.
func TestParagraphMotionWithOperator(t *testing.T) {
	seq := keySeq
	esc := KeyEvent{Key: KeyEscape}
	runParagraphCases(t, []paragraphCase{
		{name: "noBlank lld{", content: paraNoBlank, keys: seq("lld{"), want: Position{0, 0}, text: "c\nx\ndef"},
		{name: "noBlank Gd}", content: paraNoBlank, keys: seq("Gd}"), want: Position{2, 0}, text: "abc\nx\n"},
		{name: "noBlank jd2{", content: paraNoBlank, keys: seq("jd2{"), want: Position{1, 0}},
		{name: "noBlank j$d2}", content: paraNoBlank, keys: seq("j$d2}"), want: Position{1, 0}},
		{name: "noBlank Gc}Y<Esc>", content: paraNoBlank, keys: seq("Gc}Y", esc), want: Position{2, 0}, text: "abc\nx\nY"},
		{name: "noBlank lc{Y<Esc>", content: paraNoBlank, keys: seq("lc{Y", esc), want: Position{0, 0}, text: "Ybc\nx\ndef"},
		{name: "noBlank d}", content: paraNoBlank, keys: seq("d}"), want: Position{0, 0}, empty: true},
		{name: "noBlank y}", content: paraNoBlank, keys: seq("y}"), want: Position{0, 0}, yanked: "abc\nx\ndef"},
		{name: "noBlank c}Y<Esc>", content: paraNoBlank, keys: seq("c}Y", esc), want: Position{0, 0}, text: "Y"},
		{name: "noBlank Gy2{", content: paraNoBlank, keys: seq("Gy2{"), want: Position{2, 0}, yanked: "REG"},
		{name: "noBlank G$y}", content: paraNoBlank, keys: seq("G$y}"), want: Position{2, 2}, yanked: "f"},
		{name: "oneBlank d2}", content: paraOneBlank, keys: seq("d2}"), want: Position{0, 0}, empty: true},
		{name: "oneBlank y2}", content: paraOneBlank, keys: seq("y2}"), want: Position{0, 0}, yanked: "abc\n\ndef"},
		{name: "oneBlank jd}", content: paraOneBlank, keys: seq("jd}"), want: Position{0, 0}, text: "abc"},
		{name: "oneBlank jy}", content: paraOneBlank, keys: seq("jy}"), want: Position{1, 0}, yanked: "\ndef"},
		{name: "oneBlank Gd{", content: paraOneBlank, keys: seq("Gd{"), want: Position{1, 0}, text: "abc\ndef"},
		{name: "oneBlank Gy{", content: paraOneBlank, keys: seq("Gy{"), want: Position{1, 0}, yanked: "\n"},
		{name: "oneBlank jd{", content: paraOneBlank, keys: seq("jd{"), want: Position{0, 0}, text: "\ndef"},
		{name: "three 3jd}", content: paraThree, keys: seq("3jd}"), want: Position{3, 0}, text: "p1a\np1b\n\n\np3a\np3b"},
		{name: "three 3jd{", content: paraThree, keys: seq("3jd{"), want: Position{2, 0}, text: "p1a\np1b\np2a\np2b\n\np3a\np3b"},
		{name: "three 3jy}", content: paraThree, keys: seq("3jy}"), want: Position{3, 0}, yanked: "p2a\np2b\n"},
		{name: "three 3jc}Y<Esc>", content: paraThree, keys: seq("3jc}Y", esc), want: Position{3, 0}, text: "p1a\np1b\n\nY\n\np3a\np3b"},
		{name: "three 3jd2}", content: paraThree, keys: seq("3jd2}"), want: Position{2, 0}, text: "p1a\np1b\n"},
		{name: "three 3jy2}", content: paraThree, keys: seq("3jy2}"), want: Position{3, 0}, yanked: "p2a\np2b\n\np3a\np3b"},
		{name: "three 3jd3}", content: paraThree, keys: seq("3jd3}"), want: Position{3, 0}},
		{name: "three Gkd{", content: paraThree, keys: seq("Gkd{"), want: Position{5, 0}, text: "p1a\np1b\n\np2a\np2b\np3a\np3b"},
		{name: "three Gd2{", content: paraThree, keys: seq("Gd2{"), want: Position{2, 0}, text: "p1a\np1b\np3b"},
		{name: "three G$d{", content: paraThree, keys: seq("G$d{"), want: Position{5, 0}, text: "p1a\np1b\n\np2a\np2b\nb"},
		{name: "three G$y3{", content: paraThree, keys: seq("G$y3{"), want: Position{0, 0}, yanked: "p1a\np1b\n\np2a\np2b\n\np3a\np3"},
		{name: "three 5jd}", content: paraThree, keys: seq("5jd}"), want: Position{4, 0}, text: "p1a\np1b\n\np2a\np2b"},
		{name: "three d3}", content: paraThree, keys: seq("d3}"), want: Position{0, 0}, empty: true},
		{name: "three c3}Y<Esc>", content: paraThree, keys: seq("c3}Y", esc), want: Position{0, 0}, text: "Y"},
		{name: "three G$d3{", content: paraThree, keys: seq("G$d3{"), want: Position{0, 0}, text: "b"},
		{name: "blankEdges Gd}", content: paraBlankEdges, keys: seq("Gd}"), want: Position{4, 0}},
		{name: "blankEdges Gd{", content: paraBlankEdges, keys: seq("Gd{"), want: Position{2, 0}, text: "\nabc\n"},
		{name: "blankEdges d3}", content: paraBlankEdges, keys: seq("d3}"), want: Position{0, 0}, empty: true},
		{name: "blankEdges 3j$d}", content: paraBlankEdges, keys: seq("3j$d}"), want: Position{3, 1}, text: "\nabc\n\nde\n"},
		{name: "blankRuns jd}", content: paraBlankRuns, keys: seq("jd}"), want: Position{1, 0}, text: "abc\n\n\nghi"},
		{name: "blankRuns 2jd2}", content: paraBlankRuns, keys: seq("2jd2}"), want: Position{1, 0}, text: "abc\n"},
		{name: "blankRuns Gd2{", content: paraBlankRuns, keys: seq("Gd2{"), want: Position{2, 0}, text: "abc\n\nghi"},
		{name: "emptyLast d}", content: paraEmptyLast, keys: seq("d}"), want: Position{0, 0}, empty: true},
		{name: "emptyLast y}", content: paraEmptyLast, keys: seq("y}"), want: Position{0, 0}, yanked: "abc\ndef\n"},
		{name: "emptyLast Gd}", content: paraEmptyLast, keys: seq("Gd}"), want: Position{2, 0}},
		{name: "emptyLast jd}", content: paraEmptyLast, keys: seq("jd}"), want: Position{1, 0}, text: "abc\n"},
		{name: "blanksLine d}", content: paraBlanksLine, keys: seq("d}"), want: Position{0, 0}, empty: true},
		{name: "blanksLine y}", content: paraBlanksLine, keys: seq("y}"), want: Position{0, 0}, yanked: "abc\n  \ndef"},
		{name: "blanksLine ld}", content: paraBlanksLine, keys: seq("ld}"), want: Position{0, 0}, text: "a"},
		{name: "indented wd}", content: paraIndented, keys: seq("wd}"), want: Position{0, 0}, text: "\n  ghi"},
		{name: "indented wd2}", content: paraIndented, keys: seq("wd2}"), want: Position{0, 0}, empty: true},
		{name: "indented wy2}", content: paraIndented, keys: seq("wy2}"), want: Position{0, 2}, yanked: "abc\ndef\n\n  ghi"},
		{name: "indented Gwd{", content: paraIndented, keys: seq("Gwd{"), want: Position{2, 0}, text: "  abc\ndef\ni"},
		{name: "indented G$y{", content: paraIndented, keys: seq("G$y{"), want: Position{2, 0}, yanked: "\n  gh"},
		{name: "indented Gwc{Y<Esc>", content: paraIndented, keys: seq("Gwc{Y", esc), want: Position{2, 0}, text: "  abc\ndef\nYi"},
		{name: "long j$d2{k", content: paraLong, keys: seq("j$d2{k"), want: Position{0, 0}},
		{name: "long j$y2{k", content: paraLong, keys: seq("j$y2{k"), want: Position{0, 0}, yanked: "REG"},
		{name: "long j$c2}j", content: paraLong, keys: seq("j$c2}j"), want: Position{2, 0}},
		// . repeats the count; a failed { or } isn't recorded.
		{name: "three xGd3{.", content: paraThree, keys: seq("xGd3{."), want: Position{0, 0}, text: "p3b"},
		{name: "three xjc9}.", content: paraThree, keys: seq("xjc9}."), want: Position{1, 0}, text: "1a\n1b\n\np2a\np2b\n\np3a\np3b"},
		{name: "noBlank xjd2{.", content: paraNoBlank, keys: seq("xjd2{."), want: Position{1, 0}, text: "bc\n\ndef"},
		{name: "noBlank Gc}Y<Esc>k.", content: paraNoBlank, keys: seq("Gc}Y", esc, "k."), want: Position{1, 0}, text: "abc\nY"},
	})
}

// TestParagraphMotionInVisualMode checks { and } in visual mode: } onto the
// last line selects through its last character, and a failed motion leaves
// the selection as it was (after $ it still ends on the line break).
func TestParagraphMotionInVisualMode(t *testing.T) {
	seq := keySeq
	runParagraphCases(t, []paragraphCase{
		{name: "noBlank v}d", content: paraNoBlank, keys: seq("v}d"), want: Position{0, 0}, empty: true},
		{name: "noBlank G$v{d", content: paraNoBlank, keys: seq("G$v{d"), want: Position{0, 0}, empty: true},
		{name: "noBlank jv2}d", content: paraNoBlank, keys: seq("jv2}d"), want: Position{1, 0}, text: "abc\n\ndef"},
		{name: "oneBlank Gv2{d", content: paraOneBlank, keys: seq("Gv2{d"), want: Position{0, 0}, text: "ef"},
		{name: "oneBlank V2}d", content: paraOneBlank, keys: seq("V2}d"), want: Position{0, 0}, empty: true},
		{name: "three 3jv}d", content: paraThree, keys: seq("3jv}d"), want: Position{3, 0}, text: "p1a\np1b\n\np3a\np3b"},
		{name: "three 3jV}d", content: paraThree, keys: seq("3jV}d"), want: Position{3, 0}, text: "p1a\np1b\n\np3a\np3b"},
		{name: "three v3}d", content: paraThree, keys: seq("v3}d"), want: Position{0, 0}, empty: true},
		{name: "long j$v2{", content: paraLong, keys: seq("j$v2{"), want: Position{1, 0}},
		{name: "long jv$2{d", content: paraLong, keys: seq("jv$2{d"), want: Position{1, 0}, text: "abcdef\nabcdef"},
		{name: "long v$3}d", content: paraLong, keys: seq("v$3}d"), want: Position{0, 0}, text: "x\nabcdef"},
		{name: "long Gv$}d", content: paraLong, keys: seq("Gv$}d"), want: Position{2, 0}, text: "abcdef\nx\n"},
	})
}
