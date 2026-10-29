package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUtfClass spot-checks utfClass against Vim 9.2's charclass().
func TestUtfClass(t *testing.T) {
	for r, want := range map[rune]int{
		0: 1, '\n': 1, ' ': 0, '.': 1, 'a': 2, '_': 2, '0': 2,
		0xaa: 1, 0xb2: 1, 0xb5: 2, 0xbf: 1, 0xd7: 2, 0xf7: 2,
		0x301: 2, 0x2102: 1, 0x17d7: 1, 0xfeff: 2, 0x3000: 0, 0x3001: 1,
		0x2070: 0x2070, 0x2080: 0x2080, 0x2800: 0x2800, 0x3042: 0x3040, 0x30ab: 0x30a0,
		0x3300: 0x4e00, 0x4dc0: 0x4e00, 0x6f22: 0x4e00, 0xac00: 0xac00, 0xd7a3: 0xac00, 0xd7a4: 2,
		0x1f600: 3, 0x20000: 0x4e00, 0x10ffff: 2,
	} {
		assert.Equal(t, want, utfClass(r), "%U", r)
	}
}

// TestCharClassWords checks the rows below against Vim 9.2: a word is a run
// of one utf_class class, so combining marks, U+FEFF, × and ÷ are word
// characters, ² and ª are punctuation, emoji are words of their own, and
// kanji, kana, hangul, braille and super- and subscripts are separate words.
func TestCharClassWords(t *testing.T) {
	runEmptyRangeCases(t, []emptyRangeCase{
		{name: "e\u0301te ab diw", content: "e\u0301te ab", keys: "diw", text: " ab", want: Position{0, 0}, reg: anyReg},
		{name: "e\u0301te ab w", content: "e\u0301te ab", keys: "w", text: "e\u0301te ab", want: Position{0, 5}, reg: anyReg},
		{name: "ab\ufeffcd ef diw", content: "ab\ufeffcd ef", keys: "diw", text: " ef", want: Position{0, 0}, reg: anyReg},
		{name: "ab\ufeffcd ef w", content: "ab\ufeffcd ef", keys: "w", text: "ab\ufeffcd ef", want: Position{0, 6}, reg: anyReg},
		{name: "a\u00d7b c diw", content: "a\u00d7b c", keys: "diw", text: " c", want: Position{0, 0}, reg: anyReg},
		{name: "a\u00f7b c w", content: "a\u00f7b c", keys: "w", text: "a\u00f7b c", want: Position{0, 4}, reg: anyReg},
		{name: "x\u00b2y z diw", content: "x\u00b2y z", keys: "diw", text: "\u00b2y z", want: Position{0, 0}, reg: anyReg},
		{name: "x\u00b2y z w", content: "x\u00b2y z", keys: "w", text: "x\u00b2y z", want: Position{0, 1}, reg: anyReg},
		{name: "x\u00aay z w", content: "x\u00aay z", keys: "w", text: "x\u00aay z", want: Position{0, 1}, reg: anyReg},
		{name: "ab\u2102cd w", content: "ab\u2102cd", keys: "w", text: "ab\u2102cd", want: Position{0, 2}, reg: anyReg},
		{name: "ab\u17d7cd w", content: "ab\u17d7cd", keys: "w", text: "ab\u17d7cd", want: Position{0, 2}, reg: anyReg},
		{name: "\U0001f600! x w", content: "\U0001f600! x", keys: "w", text: "\U0001f600! x", want: Position{0, 1}, reg: anyReg},
		{name: "\U0001f600! x diw", content: "\U0001f600! x", keys: "diw", text: "! x", want: Position{0, 0}, reg: anyReg},
		{name: "a\U0001f600b c w", content: "a\U0001f600b c", keys: "w", text: "a\U0001f600b c", want: Position{0, 1}, reg: anyReg},
		{name: "a\U0001f600\U0001f601b ldiw", content: "a\U0001f600\U0001f601b", keys: "ldiw", text: "ab", want: Position{0, 1}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x w", content: "\u6f22\u5b57\u304b\u306a x", keys: "w", text: "\u6f22\u5b57\u304b\u306a x", want: Position{0, 2}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x diw", content: "\u6f22\u5b57\u304b\u306a x", keys: "diw", text: "\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x e", content: "\u6f22\u5b57\u304b\u306a x", keys: "e", text: "\u6f22\u5b57\u304b\u306a x", want: Position{0, 1}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x $b", content: "\u6f22\u5b57\u304b\u306a x", keys: "$b", text: "\u6f22\u5b57\u304b\u306a x", want: Position{0, 2}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x $ge", content: "\u6f22\u5b57\u304b\u306a x", keys: "$ge", text: "\u6f22\u5b57\u304b\u306a x", want: Position{0, 3}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x 3ldaw", content: "\u6f22\u5b57\u304b\u306a x", keys: "3ldaw", text: "\u6f22\u5b57x", want: Position{0, 2}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x cwZ<Esc>", content: "\u6f22\u5b57\u304b\u306a x", keys: "cwZ<Esc>", text: "Z\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x dw", content: "\u6f22\u5b57\u304b\u306a x", keys: "dw", text: "\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x de", content: "\u6f22\u5b57\u304b\u306a x", keys: "de", text: "\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "\u30ab\u30bf\u6f22\u5b57 w", content: "\u30ab\u30bf\u6f22\u5b57", keys: "w", text: "\u30ab\u30bf\u6f22\u5b57", want: Position{0, 2}, reg: anyReg},
		{name: "\ud55c\uad6dabc w", content: "\ud55c\uad6dabc", keys: "w", text: "\ud55c\uad6dabc", want: Position{0, 2}, reg: anyReg},
		{name: "\ud55c\uad6dabc diw", content: "\ud55c\uad6dabc", keys: "diw", text: "abc", want: Position{0, 0}, reg: anyReg},
		{name: "x\u2070\u2071y w", content: "x\u2070\u2071y", keys: "w", text: "x\u2070\u2071y", want: Position{0, 1}, reg: anyReg},
		{name: "x\u2080\u2081y w", content: "x\u2080\u2081y", keys: "w", text: "x\u2080\u2081y", want: Position{0, 1}, reg: anyReg},
		{name: "\u2801\u2802a w", content: "\u2801\u2802a", keys: "w", text: "\u2801\u2802a", want: Position{0, 2}, reg: anyReg},
		{name: "\u3300\u4e00\u4dc0 a diw", content: "\u3300\u4e00\u4dc0 a", keys: "diw", text: " a", want: Position{0, 0}, reg: anyReg},
		{name: "\u3042\u3044 \u3046 W", content: "\u3042\u3044 \u3046", keys: "W", text: "\u3042\u3044 \u3046", want: Position{0, 3}, reg: anyReg},
		{name: "\u6f22\u304b x diW", content: "\u6f22\u304b x", keys: "diW", text: " x", want: Position{0, 0}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x gUiw", content: "\u6f22\u5b57\u304b\u306a x", keys: "gUiw", text: "\u6f22\u5b57\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "\u6f22\u5b57\u304b\u306a x viwd", content: "\u6f22\u5b57\u304b\u306a x", keys: "viwd", text: "\u304b\u306a x", want: Position{0, 0}, reg: anyReg},
		{name: "ab\u00b5c d diw", content: "ab\u00b5c d", keys: "diw", text: " d", want: Position{0, 0}, reg: anyReg},
		{name: "a\u00bfb w", content: "a\u00bfb", keys: "w", text: "a\u00bfb", want: Position{0, 1}, reg: anyReg},
	})
}

// TestExtraWordCharsClass checks that extra word characters join the words
// on either side of them, of any class.
func TestExtraWordCharsClass(t *testing.T) {
	e := newTestEditor("ab-cd ef")
	e.SetExtraWordChars('-')
	typeKeys(t, e, "diw")
	assert.Equal(t, " ef", content(e))
}

// TestSearchWordBoundaryClasses checks \<, \> and * against Vim 9.2: a
// boundary is a change of class, and the character inside it must be a word
// character.
func TestSearchWordBoundaryClasses(t *testing.T) {
	tests := []struct {
		name, content, pattern string
		want                   Position
	}{
		{"\\< before kana after a blank", "ab \u304b\u306a\u6f22 \u6f22\u304b\u306a", "\\<\u304b\u306a", Position{0, 3}},
		{"\\> after kana before kanji", "ab \u304b\u306a\u6f22 \u6f22\u304b\u306a", "\u304b\u306a\\>", Position{0, 3}},
		{"\\< needs a word character", "a -b -c", "\\<-", Position{0, 0}},
		{"\\> needs a word character", "a b- c-", "-\\>", Position{0, 0}},
		{"emoji is a word", "x \U0001f600a \U0001f600", "\\<\U0001f600\\>", Position{0, 2}},
		{"² ends a word", "x a\u00b2 a", "\\<a\\>", Position{0, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			defaultSearch(e, tt.pattern)
			assert.Equal(t, tt.want, cursorPos(e))
		})
	}

	t.Run("* on kanji before kana", func(t *testing.T) {
		e := newTestEditor("\u6f22\u5b57\u304b\u306a \u6f22\u5b57x \u6f22\u5b57")
		e.SearchWordUnderCursor(false)
		assert.Equal(t, Position{0, 5}, cursorPos(e))
	})

	t.Run("* skips kanji inside a longer run", func(t *testing.T) {
		e := newTestEditor("\u6f22\u5b57\u304b\u306a \u6f22\u5b57\u6f22 \u6f22\u5b57")
		e.SearchWordUnderCursor(false)
		assert.Equal(t, Position{0, 9}, cursorPos(e))
	})
}
