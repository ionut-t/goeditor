package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsBlank checks isBlank against class 0 of Vim 9.2's utf_class, as
// charclass() reports it over every code point.
func TestIsBlank(t *testing.T) {
	for _, r := range []rune{' ', '\t', 0xa0, 0x1680, 0x2000, 0x2002, 0x200b, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000} {
		assert.True(t, isBlank(r), "%U", r)
	}
	for _, r := range []rune{'a', '.', 0, '\n', '\v', '\f', '\r', 0x85, 0x9f, 0xa1, 0x180e, 0x1fff, 0x200c, 0x2027, 0x202a, 0x202e, 0x2030, 0x205e, 0x2060, 0x2fff, 0x3001, 0xfeff} {
		assert.False(t, isBlank(r), "%U", r)
	}
}

// TestUnicodeBlanks checks the rows below against Vim 9.2. Word motions and
// objects treat the Unicode spaces of utf_class class 0 (no-break space,
// U+3000, U+2002) as blanks, but not U+000B. The first non-blank, I, J and
// the blank check at the start of cw use VIM_ISWHITE (space and tab), so
// they keep a no-break space as text: ^ stops on it and cw on it changes to
// the end of the next word, as ce.
func TestUnicodeBlanks(t *testing.T) {
	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "NBSP: ab_cd ef diW", content: "ab\u00a0cd ef", keys: "diW", text: "\u00a0cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef diw", content: "ab\u00a0cd ef", keys: "diw", text: "\u00a0cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef daw", content: "ab\u00a0cd ef", keys: "daw", text: "cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef daW", content: "ab\u00a0cd ef", keys: "daW", text: "cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef Wx", content: "ab\u00a0cd ef", keys: "Wx", text: "ab\u00a0d ef", want: Position{0, 3}, reg: anyReg},
		{name: "NBSP: ab_cd ef wx", content: "ab\u00a0cd ef", keys: "wx", text: "ab\u00a0d ef", want: Position{0, 3}, reg: anyReg},
		{name: "NBSP: ab_cd ef ex", content: "ab\u00a0cd ef", keys: "ex", text: "a\u00a0cd ef", want: Position{0, 1}, reg: anyReg},
		{name: "NBSP: ab_cd ef Ex", content: "ab\u00a0cd ef", keys: "Ex", text: "a\u00a0cd ef", want: Position{0, 1}, reg: anyReg},
		{name: "NBSP: ab_cd ef $bx", content: "ab\u00a0cd ef", keys: "$bx", text: "ab\u00a0cd f", want: Position{0, 6}, reg: anyReg},
		{name: "NBSP: ab_cd ef $Bx", content: "ab\u00a0cd ef", keys: "$Bx", text: "ab\u00a0cd f", want: Position{0, 6}, reg: anyReg},
		{name: "NBSP: ab_cd ef $gex", content: "ab\u00a0cd ef", keys: "$gex", text: "ab\u00a0c ef", want: Position{0, 4}, reg: anyReg},
		{name: "NBSP: ab_cd ef $gEx", content: "ab\u00a0cd ef", keys: "$gEx", text: "ab\u00a0c ef", want: Position{0, 4}, reg: anyReg},
		{name: "NBSP: ab_cd ef dw", content: "ab\u00a0cd ef", keys: "dw", text: "cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef dW", content: "ab\u00a0cd ef", keys: "dW", text: "cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef de", content: "ab\u00a0cd ef", keys: "de", text: "\u00a0cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef 3ldb", content: "ab\u00a0cd ef", keys: "3ldb", text: "cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab_cd ef 3ldge", content: "ab\u00a0cd ef", keys: "3ldge", text: "ad ef", want: Position{0, 1}, reg: anyReg},
		{name: "NBSP: __ab cd lcwZ<Esc>", content: "\u00a0\u00a0ab cd", keys: "lcwZ<Esc>", text: "\u00a0Z cd", want: Position{0, 1}, reg: anyReg},
		{name: "NBSP: __ab cd 2lcWZ<Esc>", content: "\u00a0\u00a0ab cd", keys: "2lcWZ<Esc>", text: "\u00a0\u00a0Z cd", want: Position{0, 2}, reg: anyReg},
		{name: "NBSP: __ab cd ^x", content: "\u00a0\u00a0ab cd", keys: "^x", text: "\u00a0ab cd", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: __ab cd Ix<Esc>", content: "\u00a0\u00a0ab cd", keys: "Ix<Esc>", text: "x\u00a0\u00a0ab cd", want: Position{0, 0}, reg: anyReg},
		{name: "NBSP: ab/_cd wx", content: "ab\n\u00a0cd", keys: "wx", text: "ab\n\u00a0d", want: Position{1, 1}, reg: anyReg},
		{name: "NBSP: ab/_cd J", content: "ab\n\u00a0cd", keys: "J", text: "ab \u00a0cd", want: Position{0, 2}, reg: anyReg},
		{name: "NBSP: ab_/cd J", content: "ab\u00a0\ncd", keys: "J", text: "ab\u00a0 cd", want: Position{0, 3}, reg: anyReg},
		{name: "NBSP: ab/__/cd d2w", content: "ab\n\u00a0\u00a0\ncd", keys: "d2w", text: "", want: Position{0, 0}, reg: anyReg},
		{name: "U+3000: ab_cd ef diW", content: "ab\u3000cd ef", keys: "diW", text: "\u3000cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+3000: ab_cd ef diw", content: "ab\u3000cd ef", keys: "diw", text: "\u3000cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+3000: ab_cd ef Wx", content: "ab\u3000cd ef", keys: "Wx", text: "ab\u3000d ef", want: Position{0, 3}, reg: anyReg},
		{name: "U+3000: ab_cd ef ex", content: "ab\u3000cd ef", keys: "ex", text: "a\u3000cd ef", want: Position{0, 1}, reg: anyReg},
		{name: "U+2002: ab_cd ef diW", content: "ab\u2002cd ef", keys: "diW", text: "\u2002cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+2002: ab_cd ef diw", content: "ab\u2002cd ef", keys: "diw", text: "\u2002cd ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+2002: ab_cd ef Wx", content: "ab\u2002cd ef", keys: "Wx", text: "ab\u2002d ef", want: Position{0, 3}, reg: anyReg},
		{name: "U+2002: ab_cd ef ex", content: "ab\u2002cd ef", keys: "ex", text: "a\u2002cd ef", want: Position{0, 1}, reg: anyReg},
		{name: "U+000B: ab_cd ef diW", content: "ab\u000bcd ef", keys: "diW", text: " ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+000B: ab_cd ef diw", content: "ab\u000bcd ef", keys: "diw", text: "\u000bcd ef", want: Position{0, 0}, reg: anyReg},
		{name: "U+000B: ab_cd ef Wx", content: "ab\u000bcd ef", keys: "Wx", text: "ab\u000bcd f", want: Position{0, 6}, reg: anyReg},
	})
}
