package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeys(t *testing.T) {
	tests := []struct {
		name     string
		notation string
		want     []KeyEvent
	}{
		{
			name:     "plain characters",
			notation: "dw",
			want:     []KeyEvent{{Rune: 'd'}, {Rune: 'w'}},
		},
		{
			name:     "ctrl letter is canonical",
			notation: "<C-r>",
			want:     []KeyEvent{{Rune: 'r', Modifiers: ModCtrl}},
		},
		{
			name:     "ctrl is case insensitive and folds to lowercase",
			notation: "<C-R>",
			want:     []KeyEvent{{Rune: 'r', Modifiers: ModCtrl}},
		},
		{
			name:     "alt and meta are the same modifier",
			notation: "<A-x><M-x>",
			want: []KeyEvent{
				{Rune: 'x', Modifiers: ModAlt},
				{Rune: 'x', Modifiers: ModAlt},
			},
		},
		{
			name:     "shifted letter folds to uppercase",
			notation: "<S-a>",
			want:     []KeyEvent{{Rune: 'A'}},
		},
		{
			name:     "stacked modifiers",
			notation: "<C-A-x>",
			want:     []KeyEvent{{Rune: 'x', Modifiers: ModCtrl | ModAlt}},
		},
		{
			name:     "named keys",
			notation: "<Esc><CR><Tab><BS>",
			want: []KeyEvent{
				{Key: KeyEscape},
				{Key: KeyEnter},
				{Key: KeyTab, Rune: '\t'},
				{Key: KeyBackspace},
			},
		},
		{
			name:     "named key aliases",
			notation: "<Escape><Enter><Return><Backspace><Delete>",
			want: []KeyEvent{
				{Key: KeyEscape},
				{Key: KeyEnter},
				{Key: KeyEnter},
				{Key: KeyBackspace},
				{Key: KeyDelete},
			},
		},
		{
			name:     "named keys are case insensitive",
			notation: "<esc><cr>",
			want:     []KeyEvent{{Key: KeyEscape}, {Key: KeyEnter}},
		},
		{
			name:     "space carries its rune",
			notation: "<Space>",
			want:     []KeyEvent{{Key: KeySpace, Rune: ' '}},
		},
		{
			name:     "lt is a literal angle bracket",
			notation: "<lt>",
			want:     []KeyEvent{{Rune: '<'}},
		},
		{
			name:     "Nop yields nothing",
			notation: "<Nop>",
			want:     nil,
		},
		{
			name:     "mixed notation and literals",
			notation: "y$<Esc>",
			want: []KeyEvent{
				{Rune: 'y'},
				{Rune: '$'},
				{Key: KeyEscape},
			},
		},
		{
			name:     "unmatched angle bracket is literal",
			notation: "a<b",
			want:     []KeyEvent{{Rune: 'a'}, {Rune: '<'}, {Rune: 'b'}},
		},
		{
			name:     "unicode passes through",
			notation: "é",
			want:     []KeyEvent{{Rune: 'é'}},
		},
		{
			name:     "empty",
			notation: "",
			want:     nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseKeys(tc.notation, "")
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseKeysLeader(t *testing.T) {
	t.Run("defaults to backslash", func(t *testing.T) {
		got, err := ParseKeys("<leader>w", "")
		require.NoError(t, err)
		assert.Equal(t, []KeyEvent{{Rune: '\\'}, {Rune: 'w'}}, got)
	})

	t.Run("uses the configured leader", func(t *testing.T) {
		got, err := ParseKeys("<leader>w", ",")
		require.NoError(t, err)
		assert.Equal(t, []KeyEvent{{Rune: ','}, {Rune: 'w'}}, got)
	})

	t.Run("multi-character leader expands to a sequence", func(t *testing.T) {
		got, err := ParseKeys("<leader>x", "gs")
		require.NoError(t, err)
		assert.Equal(t, []KeyEvent{{Rune: 'g'}, {Rune: 's'}, {Rune: 'x'}}, got)
	})
}

func TestParseKeysErrors(t *testing.T) {
	for _, notation := range []string{"<Bogus>", "<C-Bogus>", "<>"} {
		t.Run(notation, func(t *testing.T) {
			_, err := ParseKeys(notation, "")
			assert.ErrorIs(t, err, ErrInvalidMapping)
		})
	}
}

func TestFormatKeys(t *testing.T) {
	tests := []struct {
		name string
		keys []KeyEvent
		want string
	}{
		{"plain", []KeyEvent{{Rune: 'd'}, {Rune: 'w'}}, "dw"},
		{"ctrl", []KeyEvent{{Rune: 'r', Modifiers: ModCtrl}}, "<C-r>"},
		{"named", []KeyEvent{{Key: KeyEscape}}, "<Esc>"},
		{"alias collapses to canonical", []KeyEvent{{Key: KeyEnter}}, "<CR>"},
		{"literal angle bracket", []KeyEvent{{Rune: '<'}}, "<lt>"},
		{"empty", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatKeys(tc.keys))
		})
	}
}

// TestKeyNotationRoundTrip checks that formatting a parsed sequence and parsing
// it again is stable, which is what :map listing and Mappings() rely on.
func TestKeyNotationRoundTrip(t *testing.T) {
	for _, notation := range []string{"dw", "<C-r>", "y$", "<Esc>", "<lt>", "gg<C-d>", "<Space>"} {
		t.Run(notation, func(t *testing.T) {
			keys, err := ParseKeys(notation, "")
			require.NoError(t, err)

			again, err := ParseKeys(FormatKeys(keys), "")
			require.NoError(t, err)

			assert.Equal(t, keys, again)
		})
	}
}
