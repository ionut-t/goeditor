package goeditor

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor/highlighter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pressCtrl(r rune) modelAction {
	return func(m Model) Model {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl})
		return m
	}
}

// newScrollModel opens lineCount numbered lines. Over 100 lines the layout is
// computed lazily, which is the case that has to keep TopLine in its cache.
func newScrollModel(lineCount, width, height int) Model {
	var b strings.Builder
	for i := 1; i <= lineCount; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	m := New(width, height)
	m.Focus()
	m.SetContent(strings.TrimSuffix(b.String(), "\n"))
	return m
}

func screenRows(m Model) []string {
	return strings.Split(ansi.Strip(m.viewport.View()), "\n")
}

// rowLineNumber returns the line number printed at the start of a screen row,
// or 0 for a row that has none (a wrapped continuation or a filler row).
func rowLineNumber(row string) int {
	fields := strings.Fields(row)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return n
}

func cursorScreenRow(m Model) int {
	return m.cursorAbsoluteVisualRow - m.currentVisualTopLine
}

func TestScreenStartsAtTopLine(t *testing.T) {
	m := newScrollModel(300, 60, 42)
	require.NoError(t, m.SetCursorPosition(149, 0))

	for _, keys := range []string{"zt", "zz", "zb"} {
		m = typeText(keys)(m)
		top := m.editor.GetState().TopLine
		rows := screenRows(m)
		assert.Equalf(t, top+1, rowLineNumber(rows[0]), "%s: first row shows TopLine", keys)
		assert.Equalf(t, 150, rowLineNumber(rows[cursorScreenRow(m)]), "%s: cursor row shows the cursor line", keys)
	}

	height := m.viewport.Height()
	m = typeText("zz")(m)
	assert.Equal(t, (height-1)/2, cursorScreenRow(m))
	m = typeText("zt")(m)
	assert.Equal(t, 0, cursorScreenRow(m))
	m = typeText("zb")(m)
	assert.Equal(t, height-1, cursorScreenRow(m))
}

func TestZZAtEndOfBufferShowsEmptyRows(t *testing.T) {
	m := newScrollModel(300, 60, 22)
	m = typeText("Gzz")(m)

	rows := screenRows(m)
	cursorRow := cursorScreenRow(m)
	assert.Equal(t, (m.viewport.Height()-1)/2, cursorRow)
	assert.Equal(t, 300, rowLineNumber(rows[cursorRow]))
	for i := cursorRow + 1; i < len(rows); i++ {
		assert.Emptyf(t, strings.TrimSpace(rows[i]), "row %d is past the end of the buffer", i)
	}
}

func TestCtrlDKeepsCursorOnItsScreenRow(t *testing.T) {
	m := newScrollModel(300, 60, 32)
	m = typeText("jjjj")(m)
	before := cursorScreenRow(m)

	m = pressCtrl('d')(m)
	assert.Equal(t, before, cursorScreenRow(m))
	assert.Equal(t, m.editor.GetState().TopLine+1, rowLineNumber(screenRows(m)[0]))

	m = pressCtrl('u')(m)
	assert.Equal(t, before, cursorScreenRow(m))
	assert.Equal(t, 1, rowLineNumber(screenRows(m)[0]))
}

func TestTopLineIsFirstSegmentOfWrappedLine(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&b, "line %d %s\n", i, strings.Repeat("word ", 12))
	}
	m := New(30, 22)
	m.Focus()
	m.SetContent(b.String())
	require.NoError(t, m.SetCursorPosition(20, 0))

	m = typeText("zt")(m)
	rows := screenRows(m)
	assert.Equal(t, 21, rowLineNumber(rows[0]))
	assert.Zero(t, rowLineNumber(rows[1]), "the second row continues the wrapped line")

	// Each line wraps to the same number of rows. zb fits as many whole lines
	// as it can ending with the cursor line; the leftover rows show the start
	// of the next line, as Neovim's 'display=lastline' does.
	rowsPerLine := len(wrapLine(m.editor.GetBuffer().GetLines()[20], m.editor.GetState().AvailableWidth))
	require.Greater(t, rowsPerLine, 1)
	fit := m.viewport.Height() / rowsPerLine

	m = typeText("zb")(m)
	rows = screenRows(m)
	assert.Equal(t, 21-fit+1, rowLineNumber(rows[0]))
	assert.Equal(t, (fit-1)*rowsPerLine, cursorScreenRow(m))
	assert.Equal(t, 21, rowLineNumber(rows[cursorScreenRow(m)]))
}

func TestResizeKeepsCursorInView(t *testing.T) {
	m := newScrollModel(300, 60, 42)
	require.NoError(t, m.SetCursorPosition(149, 0))
	m = typeText("zt")(m)

	m = resizeTo(60, 12)(m)
	assert.Equal(t, 0, cursorScreenRow(m), "the cursor line was at the top, so it still fits")

	m = typeText("zb")(m)
	m = resizeTo(60, 6)(m)
	row := cursorScreenRow(m)
	assert.True(t, row >= 0 && row < m.viewport.Height(), "cursor row %d is off screen", row)
	assert.Equal(t, 150, rowLineNumber(screenRows(m)[row]))
}

func TestScrollCommandsKeepRenderedContentIntact(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 400; i++ {
		fmt.Fprintf(&b, "line number %d with a bit of filler text so that it wraps now and then\n", i)
	}
	m := New(50, 45)
	m.WithSyntaxHighlighter(highlighter.New("go", "monokai"))
	m.Focus()
	m.SetContent(b.String())

	var actions []modelAction
	for range 12 {
		actions = append(actions, pressCtrl('d'))
	}
	actions = append(actions, typeText("ztzbzz"))
	for range 30 {
		actions = append(actions, typeRune('k'))
	}
	actions = append(actions, typeText("Gzz"))
	for range 12 {
		actions = append(actions, pressCtrl('u'))
	}
	actions = append(actions, typeText("zbggzz"))

	runScenario(t, m, actions)
}
