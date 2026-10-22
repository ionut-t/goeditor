package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// dollarCase is a key sequence and what Vim 9.2 leaves after it.
type dollarCase struct {
	name    string
	content string
	keys    []KeyEvent
	want    Position
	text    string // buffer after the keys; empty to skip the check
	yanked  string // clipboard after the keys; empty to skip the check
}

func runDollarCases(t *testing.T, tests []dollarCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, cb := newTestEditorWithClipboard(tt.content)
			for _, k := range tt.keys {
				_ = e.HandleKey(k)
			}
			assert.Equal(t, tt.want, cursorPos(e))
			if tt.text != "" {
				assert.Equal(t, tt.text, content(e))
			}
			if tt.yanked != "" {
				assert.Equal(t, tt.yanked, cb.content)
			}
		})
	}
}

// TestDollarSticksToLineEnd checks that after $ the cursor stays at the end
// of the line through j and k, as Vim's curswant MAXCOL does, until a command
// sets the column again. Expected positions are from Vim 9.2.
func TestDollarSticksToLineEnd(t *testing.T) {
	_, _, space, bs, esc := whichwrapKeys()
	seq := keySeq
	down, end := KeyEvent{Key: KeyDown}, KeyEvent{Key: KeyEnd}
	const t1 = "abcdefghijkl\nab\nabcdefghijkl\nhello world"
	const t3 = "abcdef\nab\nabcdefghij"
	const abc = "abc\ndef\nghi"

	runDollarCases(t, []dollarCase{
		// $ sticks to the end of the line.
		{name: "$k after j onto a shorter line", content: t1, keys: seq("10ljjk$k"), want: Position{0, 11}},
		{name: "$j", content: t1, keys: seq("$j"), want: Position{1, 1}},
		{name: "$jj", content: t3, keys: seq("$jj"), want: Position{2, 9}},
		{name: "$jjkk", content: t3, keys: seq("$jjkk"), want: Position{0, 5}},
		{name: "$ from an empty line", content: "\nabc", keys: seq("$j"), want: Position{1, 2}},
		{name: "<End>", content: t3, keys: seq(end, "jj"), want: Position{2, 9}},
		{name: "<Down>", content: t3, keys: seq("$", down, down), want: Position{2, 9}},
		{name: "count past the last line", content: t3, keys: seq("$9j"), want: Position{2, 9}},
		{name: "count past the first line", content: t3, keys: seq("$jj9k"), want: Position{0, 5}},
		{name: "l that can't move keeps it", content: t3, keys: seq("$jlj"), want: Position{2, 9}},
		{name: "<Space> that can't move keeps it", content: "abcdef\nab", keys: seq("j$", space, "k"), want: Position{0, 5}},
		{name: "failed % keeps it", content: t3, keys: seq("$%jj"), want: Position{2, 9}},
		{name: "failed f keeps it", content: t3, keys: seq("$jfaj"), want: Position{2, 9}},
		{name: "zz keeps it", content: t3, keys: seq("$zzjj"), want: Position{2, 9}},
		{name: "<Esc> in normal mode keeps it", content: t3, keys: seq("$jjk", esc, "k"), want: Position{0, 5}},
		{name: ": then <Esc> keeps it", content: t3, keys: seq("$:", esc, "jj"), want: Position{2, 9}},
		{name: "v toggled off keeps it", content: "abcdef\nab\nabcdef", keys: seq("lv$vjj"), want: Position{2, 5}},

		// [count]$ goes count-1 lines down.
		{name: "2$", content: t1, keys: seq("2$"), want: Position{1, 1}},
		{name: "2$k", content: t1, keys: seq("2$k"), want: Position{0, 11}},
		{name: "3$", content: t1, keys: seq("3$"), want: Position{2, 11}},
		{name: "count past the end goes to the last line", content: t1, keys: seq("9$"), want: Position{3, 10}},
		{name: "2$j", content: t3, keys: seq("2$j"), want: Position{2, 9}},
		{name: "count on the last line fails but sticks", content: abc, keys: seq("jj9$"), want: Position{2, 0}},
		{name: "count on the last line then k", content: abc, keys: seq("jj9$k"), want: Position{1, 2}},

		// Other commands set the column again.
		{name: "h", content: t1, keys: seq("$hjj"), want: Position{2, 10}},
		{name: "0", content: t1, keys: seq("$0j"), want: Position{1, 0}},
		{name: "w", content: t1, keys: seq("$wj"), want: Position{2, 0}},
		{name: "<BS>", content: t3, keys: seq("$j", bs), want: Position{1, 0}},
		{name: "G", content: t1, keys: seq("$G"), want: Position{3, 0}},
		{name: "gg", content: t3, keys: seq("$ggjj"), want: Position{2, 0}},
		{name: "<Enter>", content: t3, keys: seq("$", KeyEvent{Key: KeyEnter}), want: Position{1, 0}},
		{name: "yy", content: t3, keys: seq("$yyjj"), want: Position{2, 5}},
		{name: "y$", content: t3, keys: seq("$y$jj"), want: Position{2, 5}},
		{name: "x", content: t3, keys: seq("$xjj"), want: Position{2, 4}, text: "abcde\nab\nabcdefghij"},
		{name: "d$", content: t1, keys: seq("5ld$jj"), want: Position{2, 4}},
		{name: "d$ goes onto the last character", content: t1, keys: seq("5ld$"), want: Position{0, 4}},
		{name: "D goes onto the last character", content: t1, keys: seq("5lD"), want: Position{0, 4}},
		{name: "D", content: t3, keys: seq("$Djj"), want: Position{2, 4}},
		{name: "~", content: t3, keys: seq("$~jj"), want: Position{2, 5}, text: "abcdeF\nab\nabcdefghij"},
		{name: "r", content: t3, keys: seq("$rxjj"), want: Position{2, 5}, text: "abcdex\nab\nabcdefghij"},
		{name: "J", content: t3, keys: seq("$Jj"), want: Position{1, 6}, text: "abcdef ab\nabcdefghij"},
		{name: "dd", content: t3, keys: seq("$dd"), want: Position{0, 0}},
		{name: "insert <Esc>", content: t3, keys: seq("$ix", esc, "jj"), want: Position{2, 5}, text: "abcdexf\nab\nabcdefghij"},
		{name: "i<Esc> at col 0", content: "\nabc", keys: seq("$i", esc, "j"), want: Position{1, 0}},
		{name: "a<Esc> at col 0", content: "\nabc", keys: seq("$a", esc, "j"), want: Position{1, 0}},
		{name: "C", content: t3, keys: seq("$CX", esc, "jj"), want: Position{2, 5}},
		{name: "s", content: t3, keys: seq("$sX", esc, "jj"), want: Position{2, 5}},
		{name: "o", content: t3, keys: seq("$o", esc), want: Position{1, 0}},
		{name: "visual <Esc>", content: t3, keys: seq("$v", esc, "jj"), want: Position{2, 5}},

		// Insert mode.
		{name: "i resets it", content: t3, keys: seq("$i", down, down), want: Position{2, 5}},
		{name: "A goes after the end", content: t3, keys: seq("A", down, down), want: Position{2, 6}},
		{name: "A<Down> onto a shorter line", content: t3, keys: seq("A", down), want: Position{1, 2}},
		{name: "a", content: t3, keys: seq("$a", down, down), want: Position{2, 6}},
	})
}

// TestVisualDollar checks $ in visual mode against Vim 9.2: the cursor goes
// onto the line break, so the selection includes it, and stays at the end of
// each line through j and k.
func TestVisualDollar(t *testing.T) {
	_, right, _, _, esc := whichwrapKeys()
	seq := keySeq
	end := KeyEvent{Key: KeyEnd}
	const abc = "abc\ndef\nghi"
	const short = "abcdef\nab\nabcdef"
	const t3 = "abcdef\nab\nabcdefghij"
	const long = "abcdefghijkl\nab\nabcdef"

	runDollarCases(t, []dollarCase{
		{name: "v$ is on the line break", content: "abc\ndef", keys: seq("lv$"), want: Position{0, 3}},
		{name: "v$d joins the lines", content: "abc\ndef", keys: seq("lv$d"), want: Position{0, 1}, text: "adef"},
		{name: "v<End>d", content: "abc\ndef", keys: seq("lv", end, "d"), text: "adef", want: Position{0, 1}},
		{name: "v$x", content: abc, keys: seq("lv$x"), want: Position{0, 1}, text: "adef\nghi"},
		{name: "v$y yanks the line break", content: abc, keys: seq("lv$y"), want: Position{0, 1}, yanked: "bc\n"},
		{name: "v$y on the last line", content: "abc", keys: seq("lv$y"), want: Position{0, 1}, yanked: "bc"},
		{name: "v$jy to the last line", content: "abc\ndef", keys: seq("lv$jy"), want: Position{0, 1}, yanked: "bc\ndef"},
		{name: "v$y on the second line", content: "abc\ndef", keys: seq("jlv$y"), want: Position{1, 1}, yanked: "ef"},
		{name: "vj onto a shorter line yanks its line break", content: "abcdefghijkl\nab\nabcdefghijkl", keys: seq("10lvjy"), want: Position{0, 10}, yanked: "kl\nab\n"},
		{name: "v$j", content: abc, keys: seq("lv$j"), want: Position{1, 3}},
		{name: "v$jd", content: abc, keys: seq("lv$jd"), text: "aghi", want: Position{0, 1}},
		{name: "vj$d", content: abc, keys: seq("lvj$d"), text: "aghi", want: Position{0, 1}},
		{name: "v$j$", content: abc, keys: seq("lv$j$"), want: Position{1, 3}},
		{name: "v$h", content: abc, keys: seq("lv$h"), want: Position{0, 2}},
		{name: "v$hd", content: abc, keys: seq("lv$hd"), text: "a\ndef\nghi", want: Position{0, 0}},
		{name: "v$jj through a shorter line", content: short, keys: seq("lv$jj"), want: Position{2, 6}},
		{name: "v$jjd", content: short, keys: seq("lv$jjd"), text: "a", want: Position{0, 0}},
		{name: "v2$", content: short, keys: seq("lv2$"), want: Position{1, 2}},
		{name: "v2$d", content: short, keys: seq("lv2$d"), text: "aabcdef", want: Position{0, 1}},
		{name: "v2$ from the middle", content: abc, keys: seq("jv2$"), want: Position{2, 3}},
		{name: "v2$ on the last line fails", content: abc, keys: seq("jjv2$"), want: Position{2, 0}},
		{name: "$vjj", content: t3, keys: seq("$vjj"), want: Position{2, 10}},
		{name: "$Vjj", content: t3, keys: seq("$Vjj"), want: Position{2, 10}},
		{name: "V$", content: abc, keys: seq("lV$"), want: Position{0, 3}},
		{name: "V$j", content: abc, keys: seq("lV$j"), want: Position{1, 3}},
		{name: "V$jd", content: abc, keys: seq("lV$jd"), text: "ghi", want: Position{0, 0}},
		{name: "v$U", content: abc, keys: seq("lv$U"), text: "aBC\ndef\nghi", want: Position{0, 1}},
		{name: "v$Uj", content: abc, keys: seq("lv$Uj"), want: Position{1, 1}},
		{name: "v$c joins the lines", content: abc, keys: seq("lv$cX", esc), text: "aXdef\nghi", want: Position{0, 1}},
		{name: "v$d on the last line", content: "abc", keys: seq("lv$d"), text: "a", want: Position{0, 0}},
		{name: "v$d on a line before an empty one", content: "abc\n", keys: seq("v$d"), text: "", want: Position{0, 0}},
		{name: "v$d on an empty line", content: "\nabc", keys: seq("v$d"), text: "abc", want: Position{0, 0}},
		{name: "v$ on an empty buffer", content: "", keys: seq("v$"), want: Position{0, 0}},
		{name: "v$d on the third line", content: abc, keys: seq("jjlv$d"), text: "abc\ndef\ng", want: Position{2, 0}},
		{name: "v$j$d to the last line", content: abc, keys: seq("jlv$j$d"), text: "abc\nd", want: Position{1, 0}},

		// l on the line break can't move, so it keeps the column.
		{name: "v$l", content: t3, keys: seq("lv$ljj"), want: Position{2, 10}},
		{name: "V$l", content: t3, keys: seq("lV$ljj"), want: Position{2, 10}},
		{name: "v$<Right>", content: t3, keys: seq("lv$", right, "jj"), want: Position{2, 10}},
		{name: "v$2l", content: t3, keys: seq("lv$2ljj"), want: Position{2, 10}},
		{name: "v$l on the last line", content: long, keys: seq("Glv$lkk"), want: Position{0, 12}},
		{name: "v$l on an empty line", content: "\nabc", keys: seq("v$lj"), want: Position{1, 3}},

		// Leaving visual mode.
		{name: "<Esc> goes onto the last character", content: abc, keys: seq("lv$", esc), want: Position{0, 2}},
		{name: "<Esc> resets the column", content: short, keys: seq("lv$", esc, "jj"), want: Position{2, 5}},
		{name: "V$<Esc>", content: t3, keys: seq("lV$", esc), want: Position{0, 5}},
		{name: "y resets the column", content: t3, keys: seq("$vjjy"), want: Position{0, 5}},
		{name: "y from the second line", content: t3, keys: seq("jv$jjy"), want: Position{1, 0}},
		{name: "d resets the column", content: t3, keys: seq("$vjjd"), text: "abcde", want: Position{0, 4}},
	})
}

// TestVisualRepeatAfterDollar checks Vim's ':h visual-repeat': when $ was
// used to extend the selection, '.' goes to the end of the line, whatever its
// length, on as many lines. Expected text is from Vim 9.2.
func TestVisualRepeatAfterDollar(t *testing.T) {
	_, _, _, _, esc := whichwrapKeys()
	seq := keySeq
	const four = "abcdef\nabcdef\nabcdef\nabcdef"

	runDollarCases(t, []dollarCase{
		{name: "v$d", content: four, keys: seq("lv$dj."), text: "aabcdef\naabcdef", want: Position{1, 1}},
		{name: "vj$d", content: four, keys: seq("llvj$dj."), text: "ababcdef\nab", want: Position{1, 1}},
		{name: "v$jhd is not to the end", content: four, keys: seq("llv$jhdj."), text: "ab\na", want: Position{1, 0}},
		{name: "v$ld", content: "abcdef\nab\nabcdefghij", keys: seq("lv$ldj."), text: "aab\na", want: Position{1, 0}},
		{name: "v$hd is not to the end", content: four, keys: seq("lv$hdj."), text: "a\nf\nabcdef\nabcdef", want: Position{1, 0}},
		{name: "v$U", content: four, keys: seq("lv$Uj."), text: "aBCDEF\naBCDEF\nabcdef\nabcdef", want: Position{1, 1}},
		{name: "longer line", content: "abcdef\nab\nabcdefgh\nabcdefghij", keys: seq("lv$djj."), text: "aab\nabcdefgh\na", want: Position{2, 0}},
		{name: "two lines, then shorter ones", content: "abcdef\nabcdef\nab\nabcdefgh\nx", keys: seq("lvj$dj."), text: "aab\na", want: Position{1, 0}},
		{name: "cursor at the start of the selection", content: four, keys: seq("j$vkdj."), text: "abcdef\nabcde", want: Position{1, 4}},
		{name: "cursor at the start, then h", content: four, keys: seq("j$vkhdj."), text: "abcde\nabcd", want: Position{1, 3}},
		{name: "a new selection after <Esc> doesn't stick", content: four, keys: seq("lv$", esc, "jlvhd"), text: "abcdef\nabcd\nabcdef\nabcdef", want: Position{1, 3}},
	})
}
