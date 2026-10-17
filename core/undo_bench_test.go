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
