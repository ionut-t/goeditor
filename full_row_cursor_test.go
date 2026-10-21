package goeditor

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor/highlighter"
	"github.com/stretchr/testify/require"
)

// newCursorTestModel returns a focused Model whose text area is width columns
// wide, with the cursor drawn in tabCursorBackground in every mode.
func newCursorTestModel(width, height int, lineNumbers, syntax bool) (Model, int) {
	lineNumWidth := 0
	if lineNumbers {
		lineNumWidth = 5
	}
	m := New(width+lineNumWidth, height)
	theme := DefaultTheme(true)
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color(tabCursorBackground))
	theme.NormalModeStyle = cursorStyle
	theme.InsertModeStyle = cursorStyle
	theme.VisualModeStyle = cursorStyle
	m.WithTheme(theme)
	m.HideLineNumbers(!lineNumbers)
	if syntax {
		m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
	}
	m.Focus()
	return m, lineNumWidth
}

// drawnCursor returns the screen row and column the cursor is drawn at, or
// fails the test if it isn't drawn inside the viewport.
func drawnCursor(t *testing.T, m Model) (row, col int, rows []string) {
	t.Helper()
	row = m.cursorAbsoluteVisualRow - m.currentVisualTopLine
	require.Truef(t, row >= 0 && row < m.viewport.Height(), "cursor on screen row %d of %d", row, m.viewport.Height())
	rows = strings.Split(m.viewport.GetContent(), "\n")
	require.Less(t, row, len(rows))
	col = drawnCursorCol(rows[row])
	require.GreaterOrEqualf(t, col, 0, "no cursor drawn on screen row %d: %q", row, ansi.Strip(rows[row]))
	return row, col, rows
}

// TestCursorAfterFullLastRow appends (A) to lines whose last row is exactly
// full and to lines whose last row isn't. Vim 9.2 and Neovim (with and without
// 'linebreak' and 'number') draw the cursor after a full last row on the first
// cell of the next screen row, over whatever is there: the next line's first
// character, or an empty row past the end of the buffer. No row is added, so
// the lines below don't move and lineRows stays as it is. When the line is the
// last one in the window, Vim draws the cursor outside it (on the status line);
// goeditor shifts the view by a row instead, so the cursor stays visible.
func TestCursorAfterFullLastRow(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	others := []string{
		"hello world",
		"aaaaa   bbbbb",
		"trailing blanks   ",
		"ab\tc",
		"abc中",
		"中文字",
	}

	variants := []struct {
		name                string
		lineNumbers, syntax bool
	}{
		{"plain", false, false},
		{"plain with line numbers", true, false},
		{"syntax", false, true},
		{"syntax with line numbers", true, true},
	}

	scenarios := []string{"alone", "followed by a line", "last visible"}

	for width := 1; width <= 20; width++ {
		var lines []string
		seen := make(map[string]bool)
		add := func(line string) {
			if !seen[line] {
				seen[line] = true
				lines = append(lines, line)
			}
		}
		for rows := 1; rows <= 3; rows++ {
			add(letters[:rows*width])   // exactly full
			add(letters[:rows*width+1]) // one past full
			if rows*width > 1 {
				add(letters[:rows*width-1]) // one short
			}
		}
		for _, line := range others {
			add(line)
		}

		for _, line := range lines {
			segs := wrapLine(line, width)
			last := segs[len(segs)-1]
			full := getVisualWidth(last) >= width
			wantRow, wantCol := len(segs)-1, getVisualWidth(last)
			if full {
				wantRow, wantCol = len(segs), 0
			}

			for _, v := range variants {
				for _, scenario := range scenarios {
					name := fmt.Sprintf("%q/width %d/%s/%s", line, width, v.name, scenario)
					t.Run(name, func(t *testing.T) {
						// The status and command lines take two rows.
						m, lineNumWidth := newCursorTestModel(width, len(segs)+5, v.lineNumbers, v.syntax)
						visible := m.viewport.Height()

						var content []string
						if scenario == "last visible" {
							// Fill the window above, so the line's last row is
							// the window's last row.
							for range visible - len(segs) {
								content = append(content, "a")
							}
						}
						lineRow := len(content)
						content = append(content, line)
						next := "xyz"[:min(width, 3)] // one row
						if scenario != "alone" {
							content = append(content, next)
						}

						m.SetContent(strings.Join(content, "\n"))
						// SetContent leaves the cursor on the last line, which
						// may have scrolled the view; start from the top.
						require.NoError(t, m.SetCursorPosition(0, 0))
						require.NoError(t, m.SetCursorPosition(lineRow, 0))
						m = typeRune('A')(m)
						require.Equal(t, lineNumWidth, m.calculateLineNumberWidth(len(content)))

						require.Equal(t, lineRow+wantRow, m.cursorAbsoluteVisualRow, "cursor's row in the layout")
						row, col, rows := drawnCursor(t, m)
						require.Equal(t, lineNumWidth+wantCol, col, "drawn cursor column")
						require.Less(t, col, lineNumWidth+width, "cursor drawn past the row")

						if full && scenario != "alone" {
							// Drawn over the next line, which hasn't moved.
							text := ansi.Strip(rows[row])
							require.Equalf(t, next, strings.TrimRight(text[lineNumWidth:], " "), "screen row %d", row)
						}
						if full && scenario == "last visible" {
							// Shifted by one row, so the cursor is on the last.
							require.Equal(t, 1, m.currentVisualTopLine, "view not shifted by one row")
							require.Equal(t, visible-1, row, "drawn cursor row")
						} else {
							require.Equal(t, 0, m.currentVisualTopLine, "view scrolled")
						}
					})
				}
			}
		}
	}
}

// TestCursorAfterFullLastRowModes covers the other ways the cursor gets after
// the end of a line in goeditor. Expected positions are from Vim 9.2 (a
// vsplit resized to the width, 'ts=4'; winline()/wincol() and the screen).
func TestCursorAfterFullLastRowModes(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		width    int
		row, col int
		keys     string
		noVim    bool
		wantRow  int // screen row
		wantCol  int // screen column, counted from the start of the text
		wantText string
	}{
		{name: "visual j onto a full line", content: "abcdefgh\nhello", width: 5, keys: "7lvj", wantRow: 3},
		{name: "visual-line j onto a full line", content: "abcdefgh\nhello", width: 5, keys: "7lVj", wantRow: 3},
		{name: "visual over a tab after the anchor", content: "abcdefgh\nhello\n\tx", width: 5, keys: "7lvj", wantRow: 3, wantText: "    x"},
		{name: "insert over a tab", content: "abcdefgh\nhello\n\tx", width: 5, row: 1, keys: "A", wantRow: 3, wantText: "    x"},
		{name: "insert over an empty line", content: "hello\n\nx", width: 5, keys: "A", wantRow: 1},
		{name: "typing past the full row", content: "hell", width: 5, keys: "Aox", wantRow: 1, wantCol: 1},
		{name: "typing up to the full row", content: "hell", width: 5, keys: "Ao", wantRow: 1},
		{name: "escape goes back onto the last character", content: "hello", width: 5, keys: "A\x1b", wantRow: 0, wantCol: 4},
		{name: "last line of a large file", content: strings.Repeat("a\n", 150) + "hello", width: 5, row: 150, keys: "A", wantRow: 3},
		{name: "non-Vim mode", content: "hello", width: 5, col: 5, noVim: true, wantRow: 1},
		{name: "non-Vim mode typing to the full row", content: "abcdefghi", width: 5, col: 9, keys: "j", noVim: true, wantRow: 2},
	}

	for _, tt := range tests {
		for _, lineNumbers := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/line numbers %v", tt.name, lineNumbers), func(t *testing.T) {
				m, lineNumWidth := newCursorTestModel(tt.width, 6, lineNumbers, false)
				m.DisableVimMode(tt.noVim)
				m.SetContent(tt.content)
				require.NoError(t, m.SetCursorPosition(tt.row, tt.col))
				for _, r := range tt.keys {
					if r == '\x1b' {
						m = pressSpecial(tea.KeyEscape)(m)
					} else {
						m = typeRune(r)(m)
					}
				}

				row, col, rows := drawnCursor(t, m)
				require.Equal(t, tt.wantRow, row, "drawn cursor row")
				require.Equal(t, lineNumWidth+tt.wantCol, col, "drawn cursor column")
				if tt.wantText != "" {
					require.Equal(t, tt.wantText, strings.TrimRight(ansi.Strip(rows[row])[lineNumWidth:], " "))
				}
			})
		}
	}
}
