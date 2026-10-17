package core

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkUndoableEdit measures one undoable edit (ix<Esc>) at several buffer
// sizes. Undo history records only what each edit changed, so the cost should
// not grow with the buffer.
func BenchmarkUndoableEdit(b *testing.B) {
	for _, lines := range []int{100, 10_000, 200_000} {
		b.Run(fmt.Sprintf("lines=%d", lines), func(b *testing.B) {
			line := strings.Repeat("x", 40)
			e := newTestEditor(strings.TrimSuffix(strings.Repeat(line+"\n", lines), "\n"))
			b.ReportAllocs()
			for b.Loop() {
				keys(e, 'i', 'x')
				escape(e)
			}
		})
	}
}

// BenchmarkInsertSession measures typing into one long insert session, where
// each key extends the same edit. Lines are kept short so the buffer's own
// per-line cost stays fixed; the cost per key should not grow with the session.
func BenchmarkInsertSession(b *testing.B) {
	e := newTestEditor("x")
	keys(e, 'i')
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		if i++; i%80 == 0 {
			enter(e)
		} else {
			keys(e, 'x')
		}
	}
}
