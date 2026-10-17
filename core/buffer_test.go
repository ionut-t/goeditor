package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteRunesAtAcrossLines pins down deletions that cross line breaks,
// which undo relies on to remove exactly the text it inserted.
func TestDeleteRunesAtAcrossLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		row, col int
		count    int
		want     string
	}{
		{"line break only", "a\nb\nc", 0, 1, 1, "ab\nc"},
		{"ends at the end of the next line", "a\nb\nc", 0, 1, 2, "a\nc"},
		{"ends inside the next line", "a\nbc\nd", 0, 1, 2, "ac\nd"},
		{"spans whole lines", "a\nb\nc\nd", 0, 1, 4, "a\nd"},
		{"ends at the end of the buffer", "a\nb", 0, 1, 2, "a"},
		{"runs past the end of the buffer", "a\nb", 0, 1, 10, "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBufferFromBytes([]byte(tt.content))
			require.Nil(t, b.DeleteRunesAt(tt.row, tt.col, tt.count))
			assert.Equal(t, tt.want, b.GetCurrentContent())
		})
	}
}
