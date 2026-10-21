package core

import (
	"strings"
	"testing"
)

// BenchmarkFind measures one search over a single 50k-rune line. Find shouldn't
// allocate, and ignore case should cost little more than an exact comparison.
func BenchmarkFind(b *testing.B) {
	ascii := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 50_000/45+1)[:50_000]
	nonASCII := []rune(strings.Repeat("Ζεῦ πάτερ, ἀλλὰ σὺ ῥῦσαι ὑπ᾽ ἠέρος υἷας Ἀχαιῶν. ", 1100))[:50_000]
	tests := []struct {
		name    string
		line    string
		pattern string
		opts    SearchOptions
	}{
		{"exact/no-match", ascii, "xyzzy", SearchOptions{}},
		{"ignorecase/no-match", ascii, "xyzzy", SearchOptions{IgnoreCase: true}},
		{"ignorecase/match-at-end", ascii[:len(ascii)-5] + "XYZZY", "xyzzy", SearchOptions{IgnoreCase: true}},
		{"ignorecase/worst-case", strings.Repeat("a", 50_000), "aaaaaaaaab", SearchOptions{IgnoreCase: true}},
		{"ignorecase/non-ascii/no-match", string(nonASCII), "ωψ", SearchOptions{IgnoreCase: true}},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			buf := NewBuffer()
			buf.SetContent([]byte(tt.line))
			b.ReportAllocs()
			for b.Loop() {
				buf.Find(tt.pattern, Position{}, tt.opts)
			}
		})
	}
}
