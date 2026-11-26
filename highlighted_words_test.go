package goeditor

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func highlightedWordsModel() Model {
	m := New(80, 24)
	m.SetHighlightedWords(map[string]lipgloss.Style{
		"SELECT":          lipgloss.NewStyle().Bold(true),
		"SELECT DISTINCT": lipgloss.NewStyle().Italic(true),
		"FROM":            lipgloss.NewStyle().Underline(true),
		"WHERE":           lipgloss.NewStyle().Bold(true),
		"JOIN":            lipgloss.NewStyle().Bold(true),
		"":                lipgloss.NewStyle().Bold(true),
	})
	return m
}

func TestHighlightedWordMatch(t *testing.T) {
	m := highlightedWordsModel()
	tests := []struct {
		name string
		text string
		at   int
		want int
	}{
		{"word", "SELECT a", 0, len("SELECT")},
		{"longest word wins", "SELECT DISTINCT a", 0, len("SELECT DISTINCT")},
		{"after a space", "a FROM b", 2, len("FROM")},
		{"inside a longer word", "xFROM", 1, 0},
		{"followed by a letter", "FROMx", 0, 0},
		{"no match", "nothing", 0, 0},
		{"empty word never matches", "", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.findHighlightedWordMatch([]rune(tt.text), tt.at)
			assert.Equal(t, tt.want, got.length)
		})
	}
}

// The words were compiled again whenever a hash taken in map order, which Go
// randomises, came out different: with two words or more, on most lookups.
func TestHighlightedWordMatchDoesNotAllocate(t *testing.T) {
	m := highlightedWordsModel()
	line := []rune("SELECT a FROM b WHERE c")
	allocs := testing.AllocsPerRun(100, func() {
		for i := range line {
			m.findHighlightedWordMatch(line, i)
		}
	})
	require.Zero(t, allocs)
}

func TestSetHighlightedWordsReplacesTheWords(t *testing.T) {
	m := highlightedWordsModel()
	m.SetHighlightedWords(map[string]lipgloss.Style{"UPDATE": lipgloss.NewStyle()})
	assert.Zero(t, m.findHighlightedWordMatch([]rune("SELECT"), 0).length)
	assert.Equal(t, len("UPDATE"), m.findHighlightedWordMatch([]rune("UPDATE"), 0).length)

	m.SetHighlightedWords(nil)
	assert.Zero(t, m.findHighlightedWordMatch([]rune("UPDATE"), 0).length)
}

// BenchmarkRenderHighlightedWords measures a redraw of a screen of SQL with
// highlighted keywords.
func BenchmarkRenderHighlightedWords(b *testing.B) {
	m := highlightedWordsModel()
	m.Focus()
	m.SetContent(strings.Repeat("SELECT a, b FROM table_name JOIN other ON x WHERE c = 1\n", 200))
	b.ResetTimer()
	for b.Loop() {
		m.renderVisibleSlice()
	}
}
