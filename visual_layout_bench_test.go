package goeditor

import (
	"strings"
	"testing"
)

func benchmarkWrapLine(b *testing.B, line string, width int) {
	b.ReportAllocs()
	for b.Loop() {
		wrapLine(line, width)
	}
}

func BenchmarkWrapLineShortCode(b *testing.B) {
	benchmarkWrapLine(b, "\treturn len(wrapLine(string(line), width)) // a typical source line that fits", 40)
}

func BenchmarkWrapLineLongProse(b *testing.B) {
	benchmarkWrapLine(b, strings.Repeat("the quick brown fox jumps over the lazy dog ", 50), 80)
}

func BenchmarkWrapLineLongMixed(b *testing.B) {
	benchmarkWrapLine(b, strings.Repeat("漢字 👍🏽 é 🇷🇴 tab\tend ", 100), 80)
}
