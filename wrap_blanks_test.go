package goeditor

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/require"
)

// TestCursorOnWrappedBlanks puts the cursor on every column of lines with runs
// of spaces, at widths 1-20. A wrap keeps the blanks (as Vim 9.2 does, with
// and without 'linebreak'): every column has its own screen cell, inside the
// row and inside the viewport.
func TestCursorOnWrappedBlanks(t *testing.T) {
	lines := []string{
		" ",
		strings.Repeat(" ", 5),
		strings.Repeat(" ", 25),
		"ab" + strings.Repeat(" ", 20) + "cd",
		"hello world",
		"aaaaa   bbbbb",
		"abcdefgh ijkl",
		"   leading blanks",
		"trailing blanks   ",
		"a b  c   d    e     f",
		"if x {        return y }",
	}
	const height = 6

	for _, content := range lines {
		for width := 1; width <= 20; width++ {
			t.Run(fmt.Sprintf("%q/width %d", content, width), func(t *testing.T) {
				m := New(width, height)
				theme := DefaultTheme(true)
				theme.NormalModeStyle = lipgloss.NewStyle().Background(lipgloss.Color(tabCursorBackground))
				m.WithTheme(theme)
				m.HideLineNumbers(true)
				m.Focus()
				m.SetContent(content)

				type cell struct{ row, col int }
				seen := make(map[cell]int)
				for col := range len([]rune(content)) {
					require.NoError(t, m.SetCursorPosition(0, col))

					vli := m.visualLayoutCache[m.cursorAbsoluteVisualRow-m.visualLayoutCacheStartVisualRow]
					offset := col - vli.LogicalStartCol
					require.Truef(t, offset >= 0 && offset < len([]rune(vli.Content)),
						"col %d is on the row %q starting at col %d", col, vli.Content, vli.LogicalStartCol)

					screenRow := m.cursorAbsoluteVisualRow - m.currentVisualTopLine
					require.Truef(t, screenRow >= 0 && screenRow < height, "col %d on screen row %d", col, screenRow)
					rows := strings.Split(m.viewport.GetContent(), "\n")
					require.Less(t, screenRow, len(rows))
					drawn := drawnCursorCol(rows[screenRow])
					require.Truef(t, drawn >= 0 && drawn < width, "col %d drawn at screen col %d", col, drawn)

					at := cell{m.cursorAbsoluteVisualRow, drawn}
					if prev, ok := seen[at]; ok {
						t.Fatalf("cols %d and %d both drawn at row %d col %d", prev, col, at.row, at.col)
					}
					seen[at] = col
				}
			})
		}
	}
}
