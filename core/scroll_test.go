package core

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newScrollEditor returns an editor on lineCount numbered lines in a window
// height rows tall, with every line one row high unless setLineHeight says
// otherwise.
func newScrollEditor(lineCount, height int) Editor {
	var b strings.Builder
	for i := range lineCount {
		fmt.Fprintf(&b, "    line %d\n", i)
	}
	e := newTestEditor(strings.TrimSuffix(b.String(), "\n"))
	s := e.GetState()
	s.ViewportHeight = height
	s.AvailableWidth = 80
	e.SetState(s)
	return e
}

func setCursorRow(e Editor, row int) {
	c := e.GetBuffer().GetCursor()
	c.Position = Position{Row: row}
	e.GetBuffer().SetCursor(c)
}

func setTopLine(e Editor, top int) {
	s := e.GetState()
	s.TopLine = top
	e.SetState(s)
}

func topLine(e Editor) int { return e.GetState().TopLine }

func TestZScrollsCursorLine(t *testing.T) {
	tests := []struct {
		keys    string
		wantTop int
	}{
		{"zt", 50},
		{"zz", 46}, // 4 rows above the cursor line, 5 below
		{"zb", 41},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newScrollEditor(100, 10)
			setCursorRow(e, 50)
			e.ScrollViewport()

			keys(e, []rune(tt.keys)...)

			assert.Equal(t, tt.wantTop, topLine(e))
			assert.Equal(t, Position{Row: 50}, cursorPos(e), "z commands don't move the cursor")
		})
	}
}

func TestZZScrollsPastEndOfBuffer(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 99)
	e.ScrollViewport()
	assert.Equal(t, 90, topLine(e), "following the cursor stops at the end of the buffer")

	keys(e, 'z', 'z')
	assert.Equal(t, 95, topLine(e), "zz centres the last line, leaving empty rows below it")

	keys(e, 'z', 't')
	assert.Equal(t, 99, topLine(e))
}

func TestZFirstNonBlankForms(t *testing.T) {
	for _, tt := range []struct {
		name    string
		send    func(Editor)
		wantTop int
	}{
		{"z<CR>", func(e Editor) { keys(e, 'z'); enter(e) }, 50},
		{"z.", func(e Editor) { keys(e, 'z', '.') }, 46},
		{"z-", func(e Editor) { keys(e, 'z', '-') }, 41},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newScrollEditor(100, 10)
			setCursorRow(e, 50)
			e.ScrollViewport()

			tt.send(e)

			assert.Equal(t, tt.wantTop, topLine(e))
			assert.Equal(t, Position{Row: 50, Col: 4}, cursorPos(e))
		})
	}
}

func TestZWithCountMovesToLine(t *testing.T) {
	e := newScrollEditor(100, 10)

	keys(e, '3', '0', 'z', 'z')

	assert.Equal(t, 29, cursorPos(e).Row)
	assert.Equal(t, 25, topLine(e))
	assert.Nil(t, e.GetState().PendingCount)
}

func TestZWithUnknownKeyIsCancelled(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 50)
	e.ScrollViewport()
	top := topLine(e)

	keys(e, 'z', 'q', 'j')

	assert.Equal(t, top, topLine(e))
	assert.Equal(t, 51, cursorPos(e).Row, "the key after z q is a normal command again")
	assert.Equal(t, "", e.GetState().CommandLine)
}

func TestZWithCountAndUnknownKeyDoesNotMove(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 50)
	e.ScrollViewport()
	top := topLine(e)

	keys(e, '3', '0', 'z', 'q')

	assert.Equal(t, 50, cursorPos(e).Row, "the count belongs to the z command, which was cancelled")
	assert.Equal(t, top, topLine(e))
	assert.Nil(t, e.GetState().PendingCount)
}

func TestZZInVisualModes(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newScrollEditor(100, 10)
			setCursorRow(e, 50)
			e.ScrollViewport()

			keys(e, mode, 'z', 'z')

			assert.Equal(t, 46, topLine(e))
			assert.True(t, e.IsVisualMode() || e.IsVisualLineMode(), "zz keeps the selection")
		})
	}
}

func TestZWithCountInVisualModes(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newScrollEditor(100, 10)

			keys(e, mode, '3', '0', 'z', 'z')

			assert.Equal(t, 29, cursorPos(e).Row)
			assert.Equal(t, 25, topLine(e))
			assert.Nil(t, e.GetState().PendingCount)
			assert.True(t, e.IsVisualMode() || e.IsVisualLineMode(), "zz keeps the selection")
		})
	}
}

func TestCursorLeavingWindowScrollsLikeVim(t *testing.T) {
	t.Run("one line below scrolls one line", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setCursorRow(e, 9)
		keys(e, 'j')
		assert.Equal(t, 1, topLine(e))
	})

	t.Run("a window or more below centres", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setCursorRow(e, 50)
		e.ScrollViewport()
		assert.Equal(t, 46, topLine(e))
	})

	t.Run("just under a window below scrolls to the bottom", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setCursorRow(e, 20) // botline is 10, so 11 lines away: within height+1
		e.ScrollViewport()
		assert.Equal(t, 11, topLine(e))
	})

	t.Run("a little above scrolls to the top", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setTopLine(e, 46)
		setCursorRow(e, 43)
		e.ScrollViewport()
		assert.Equal(t, 43, topLine(e))
	})

	t.Run("half a window above centres", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setTopLine(e, 46)
		setCursorRow(e, 42)
		e.ScrollViewport()
		assert.Equal(t, 38, topLine(e))
	})

	t.Run("G puts the last line at the bottom", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		keys(e, 'G')
		assert.Equal(t, 90, topLine(e))
	})

	t.Run("cursor inside the window leaves it alone", func(t *testing.T) {
		e := newScrollEditor(100, 10)
		setCursorRow(e, 50)
		keys(e, 'z', 't', 'j', 'j', 'k')
		assert.Equal(t, 50, topLine(e))
	})
}

func TestCtrlDScrollsTextAndCursorTogether(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 2)

	ctrlKey(e, 'd')
	assert.Equal(t, 5, topLine(e))
	assert.Equal(t, 7, cursorPos(e).Row, "the cursor keeps its screen row")

	ctrlKey(e, 'u')
	assert.Equal(t, 0, topLine(e))
	assert.Equal(t, 2, cursorPos(e).Row)
}

func TestCtrlDAtEndOfBufferMovesOnlyTheCursor(t *testing.T) {
	e := newScrollEditor(20, 10)
	setTopLine(e, 8)
	setCursorRow(e, 12)

	ctrlKey(e, 'd')
	assert.Equal(t, 10, topLine(e), "scrolls until the last line is at the bottom")
	assert.Equal(t, 17, cursorPos(e).Row, "then the cursor moves the rest of the way")

	ctrlKey(e, 'd')
	assert.Equal(t, 10, topLine(e))
	assert.Equal(t, 19, cursorPos(e).Row)

	ctrlKey(e, 'd')
	assert.Equal(t, 19, cursorPos(e).Row, "nothing happens on the last line")
}

func TestCtrlUAtStartOfBufferMovesOnlyTheCursor(t *testing.T) {
	e := newScrollEditor(100, 10)
	setTopLine(e, 2)
	setCursorRow(e, 8)

	ctrlKey(e, 'u')
	assert.Equal(t, 0, topLine(e))
	assert.Equal(t, 3, cursorPos(e).Row)

	ctrlKey(e, 'u')
	ctrlKey(e, 'u')
	assert.Equal(t, 0, cursorPos(e).Row)
}

func TestCtrlDCountSetsScrollAmount(t *testing.T) {
	e := newScrollEditor(100, 10)

	keys(e, '3')
	ctrlKey(e, 'd')
	assert.Equal(t, 3, topLine(e))

	ctrlKey(e, 'd')
	assert.Equal(t, 6, topLine(e), "the count is remembered, like Vim's 'scroll'")

	s := e.GetState()
	s.ViewportHeight = 20
	e.SetState(s)
	ctrlKey(e, 'd')
	assert.Equal(t, 16, topLine(e), "a resize resets it to half the window")
}

func TestCtrlDInVisualMode(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 2)

	keys(e, 'v')
	ctrlKey(e, 'd')

	assert.Equal(t, 5, topLine(e))
	assert.Equal(t, 7, cursorPos(e).Row)
	assert.True(t, e.IsVisualMode())
}

func TestCtrlDCountOfOneInVisualMode(t *testing.T) {
	e := newScrollEditor(100, 10)

	keys(e, 'v', '1')
	ctrlKey(e, 'd')

	assert.Equal(t, 1, topLine(e), "an explicit 1 sets 'scroll', unlike no count")
}

func TestScrollingCountsWrappedRows(t *testing.T) {
	e := newScrollEditor(100, 10)
	e.SetLineHeightFunc(func([]rune, int) int { return 3 })

	setCursorRow(e, 10)
	keys(e, 'z', 'z')
	assert.Equal(t, 9, topLine(e), "one 3-row line above the cursor line and one below fill 9 of 10 rows")

	keys(e, 'z', 'b')
	assert.Equal(t, 8, topLine(e), "three 3-row lines fit, ending with the cursor line")

	setCursorRow(e, 11)
	keys(e, 'j')
	assert.Equal(t, 10, topLine(e), "line 12 isn't fully in view until line 10 is at the top")

	ctrlKey(e, 'd')
	assert.Equal(t, 11, topLine(e), "5 rows of scroll takes one 3-row line; the next would overshoot")
	assert.Equal(t, 13, cursorPos(e).Row)
}

func TestNewContentStartsAtTop(t *testing.T) {
	e := newScrollEditor(100, 10)
	keys(e, 'G')
	assert.NotZero(t, topLine(e))

	e.SetContent([]byte("a\nb\nc"))
	assert.Equal(t, 0, topLine(e))
}

func TestScrollInsideMapping(t *testing.T) {
	e := newScrollEditor(100, 10)
	setCursorRow(e, 50)
	e.ScrollViewport()
	assert.NoError(t, e.Map(MapNormal, "Q", "zzj", true))

	keys(e, 'Q')

	assert.Equal(t, 46, topLine(e), "zz takes effect before j, and j stays in view")
	assert.Equal(t, 51, cursorPos(e).Row)
}
