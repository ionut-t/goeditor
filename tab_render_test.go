package goeditor

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor/core"
	"github.com/ionut-t/goeditor/highlighter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tabCursorBackground is a background no other style in the test theme uses,
// so the drawn cursor cell can be found in the rendered row.
const tabCursorBackground = "#abcdef"

// drawnCursorCol returns the screen column of the first cell drawn with
// tabCursorBackground in a rendered row, or -1 if there is none.
func drawnCursorCol(row string) int {
	idx := strings.Index(row, "48;2;171;205;239")
	if idx < 0 {
		return -1
	}
	start := strings.LastIndex(row[:idx], "\x1b[")
	return ansi.StringWidth(row[:start])
}

// TestTabsRenderAtTabStops checks that a tab is drawn as wide as the layout
// and cursor maths make it (up to the next tab stop), not the 4 spaces
// lipgloss turns every tab into. The cursor on a tab is drawn as Vim 9.2 draws
// it (ts=4): on the last cell in normal mode, on the first in insert mode, and
// in visual mode on the last cell only when the cursor is after the anchor.
func TestTabsRenderAtTabStops(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
		words   []string // highlighted words
		row     int
		col     int
		keys    string // typed after the cursor is placed
		wantCol int    // drawn cursor column, counted from the start of the text
	}{
		{name: "normal on c after an off-stop tab", content: "ab\tc", row: 0, col: 3, wantCol: 4},
		{name: "normal on an off-stop tab draws on its last cell", content: "ab\tc", row: 0, col: 2, wantCol: 3},
		{name: "normal on a full-width tab draws on its last cell", content: "\tab", row: 0, col: 0, wantCol: 3},
		{name: "insert on an off-stop tab draws on its first cell", content: "ab\tc", row: 0, col: 2, keys: "i", wantCol: 2},
		{name: "insert after an off-stop tab", content: "ab\tc", row: 0, col: 3, keys: "i", wantCol: 4},
		{name: "visual onto a tab after the anchor draws on its last cell", content: "ab\tc", row: 0, col: 0, keys: "vll", wantCol: 3},
		{name: "visual started on a tab draws on its first cell", content: "ab\tc", row: 0, col: 2, keys: "v", wantCol: 2},
		{name: "visual onto a tab before the anchor draws on its first cell", content: "ab\tc", row: 0, col: 3, keys: "vh", wantCol: 2},
		{name: "spaces and tabs", content: "  \t  \tx", row: 0, col: 6, wantCol: 8},
		{name: "normal on the second off-stop tab", content: "  \t  \tx", row: 0, col: 5, wantCol: 7},
		{name: "many off-stop tabs", content: "a\tb\tc\td\te\tf", width: 30, row: 0, col: 10, wantCol: 20},
		{name: "off-stop tab on a wrapped row", content: "abcdefgh ab\tcd", width: 10, row: 0, col: 13, wantCol: 5},
		{name: "highlighted word after an off-stop tab", content: "ab\tfoo", words: []string{"foo"}, row: 0, col: 4, wantCol: 5},
		{name: "highlighted word with an off-stop tab", content: "a x\ty", words: []string{"x\ty"}, row: 0, col: 4, wantCol: 4},
		{name: "normal on a tab inside a highlighted word", content: "ab x\tyz", words: []string{"x\tyz"}, row: 0, col: 4, wantCol: 7},
		{name: "insert on a tab inside a highlighted word", content: "ab x\tyz", words: []string{"x\tyz"}, row: 0, col: 4, keys: "i", wantCol: 4},
	}

	variants := []struct {
		name        string
		syntax      bool
		lineNumbers bool
	}{
		{"plain", false, false},
		{"plain with line numbers", false, true},
		{"syntax", true, false},
		{"syntax with line numbers", true, true},
	}

	for _, tt := range tests {
		for _, v := range variants {
			t.Run(tt.name+"/"+v.name, func(t *testing.T) {
				width := tt.width
				if width == 0 {
					width = 20
				}
				lineNumWidth := 0
				if v.lineNumbers {
					lineNumWidth = 5
				}

				m := New(width+lineNumWidth, 8)
				theme := DefaultTheme(true)
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(tabCursorBackground))
				theme.NormalModeStyle = cursorStyle
				theme.InsertModeStyle = cursorStyle
				theme.VisualModeStyle = cursorStyle
				m.WithTheme(theme)
				m.HideLineNumbers(!v.lineNumbers)
				if v.syntax {
					m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
				}
				if len(tt.words) > 0 {
					words := make(map[string]lipgloss.Style)
					for _, w := range tt.words {
						words[w] = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
					}
					m.SetHighlightedWords(words)
				}
				m.Focus()
				m.SetContent(tt.content)
				require.NoError(t, m.SetCursorPosition(tt.row, tt.col))
				m = typeText(tt.keys)(m)
				require.Equal(t, lineNumWidth, m.calculateLineNumberWidth(m.editor.GetBuffer().LineCount()))

				// The content handed to the viewport, before the viewport
				// cuts it at its width.
				rows := strings.Split(m.viewport.GetContent(), "\n")
				for i, row := range rows {
					assert.LessOrEqualf(t, ansi.StringWidth(row), m.viewport.Width(), "row %d is wider than the viewport: %q", i, ansi.Strip(row))
				}

				for i := range rows {
					cacheIdx := m.currentVisualTopLine + i - m.visualLayoutCacheStartVisualRow
					if cacheIdx < 0 || cacheIdx >= len(m.visualLayoutCache) {
						continue
					}
					vli := m.visualLayoutCache[cacheIdx]
					text := ansi.Strip(rows[i])
					require.GreaterOrEqualf(t, len(text), lineNumWidth, "row %d: %q", i, text)
					assert.Equalf(t, strings.TrimRight(expandTabsForCompare(vli.Content), " "), strings.TrimRight(text[lineNumWidth:], " "), "row %d", i)
				}

				cursorRow := m.cursorAbsoluteVisualRow - m.currentVisualTopLine
				require.Less(t, cursorRow, len(rows))
				drawn := drawnCursorCol(rows[cursorRow])
				assert.Equal(t, lineNumWidth+tt.wantCol, drawn, "drawn cursor column")

				// The cursor is drawn inside the cell the layout gives it.
				vli := m.visualLayoutCache[m.cursorAbsoluteVisualRow-m.visualLayoutCacheStartVisualRow]
				screenCol := m.calculateCursorScreenCol(vli, lineNumWidth)
				offset := m.clampedCursorLogicalCol - vli.LogicalStartCol
				cellWidth := max(1, getVisualWidth(expandTabsForCompare(string([]rune(vli.Content)[:offset+1])))-(screenCol-lineNumWidth))
				assert.GreaterOrEqual(t, drawn, screenCol, "cursor drawn before its cell")
				assert.Less(t, drawn, screenCol+cellWidth, "cursor drawn after its cell")
			})
		}
	}
}

// TestCursorAtWrapPoint walks the cursor back with h over lines that wrap
// with nothing dropped at the break (hard breaks, breaks before a tab). The
// column just past a row is the first column of the next row, so the cursor
// must be drawn there, not after the end of the row above. The lines have no
// spaces: a space at a wrap point is dropped, which is a separate bug.
func TestCursorAtWrapPoint(t *testing.T) {
	lines := []string{
		"abcdefghij",
		"ab\tc",
		"\t\t\tx",
		"abc\tdefg\thij\tk",
		"a\tb\tc\td\te\tf\tg\th",
		"if(x){\n\t\treturn\tnil\n\t}",
	}

	for _, content := range lines {
		for width := 4; width <= 14; width++ {
			t.Run(fmt.Sprintf("%q/width %d", content, width), func(t *testing.T) {
				m := New(width, 10)
				theme := DefaultTheme(true)
				cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(tabCursorBackground))
				theme.NormalModeStyle = cursorStyle
				m.WithTheme(theme)
				m.HideLineNumbers(true)
				m.Focus()
				m.SetContent(content)
				require.NoError(t, m.SetCursorPosition(strings.Count(content, "\n"), 0))
				m = typeRune('$')(m)

				for {
					pos := m.editor.GetBuffer().GetCursor().Position
					vli := m.visualLayoutCache[m.cursorAbsoluteVisualRow-m.visualLayoutCacheStartVisualRow]
					offset := pos.Col - vli.LogicalStartCol
					require.Truef(t, offset >= 0 && offset < len([]rune(vli.Content)),
						"cursor %v is on the row %q starting at col %d", pos, vli.Content, vli.LogicalStartCol)

					rows := strings.Split(m.viewport.GetContent(), "\n")
					drawn := drawnCursorCol(rows[m.cursorAbsoluteVisualRow-m.currentVisualTopLine])
					require.Truef(t, drawn >= 0 && drawn < width, "cursor %v drawn at col %d", pos, drawn)

					if pos == (core.Position{}) {
						break
					}
					m = typeRune('h')(m)
					if m.editor.GetBuffer().GetCursor().Position == pos {
						m = typeText("k$")(m)
					}
				}
			})
		}
	}
}
