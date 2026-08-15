package core

import (
	"fmt"
	"testing"
)

// benchTable builds a mapping table with n distinct two-key mappings.
func benchTable(n int) *mappingTable {
	t := newMappingTable()
	for i := range n {
		lhs, err := ParseKeys(fmt.Sprintf("<leader>%c%c", 'a'+byte(i/26), 'a'+byte(i%26)), ",")
		if err != nil {
			panic(err)
		}
		t.set(MapNormal, Mapping{LHS: lhs, RHS: []KeyEvent{{Rune: 'x'}}, NoRemap: true})
	}
	return t
}

func BenchmarkMappingMatch(b *testing.B) {
	for _, n := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("%d_mappings", n), func(b *testing.B) {
			table := benchTable(n)
			// Worst case: a prefix every mapping shares, so no candidate is
			// rejected by the cheap length check.
			pending := []KeyEvent{{Rune: ','}}

			b.ResetTimer()
			for b.Loop() {
				table.match(MapNormal, pending)
			}
		})
	}
}

// BenchmarkHandleKeyWithMappings measures the whole per-keystroke path, which
// is what actually has to fit inside a keypress.
func BenchmarkHandleKeyWithMappings(b *testing.B) {
	for _, n := range []int{0, 100, 500} {
		b.Run(fmt.Sprintf("%d_mappings", n), func(b *testing.B) {
			e := newTestEditor("hello world")
			e.SetMapLeader(",")
			for i := range n {
				lhs := fmt.Sprintf("<leader>%c%c", 'a'+byte(i/26), 'a'+byte(i%26))
				if err := e.Map(MapNormal, lhs, "x", true); err != nil {
					b.Fatal(err)
				}
			}

			key := KeyEvent{Rune: 'l'} // an ordinary unmapped motion

			b.ResetTimer()
			for b.Loop() {
				e.HandleKey(key)
			}
		})
	}
}
