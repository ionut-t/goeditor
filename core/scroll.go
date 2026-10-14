package core

// Scrolling follows Vim's window model. State.TopLine is the first buffer line
// in the window (Vim's 'topline'), and it only changes when a command scrolls
// on purpose (zz, CTRL-D, ...) or when the cursor would otherwise leave the
// window. Line heights are counted in screen rows, so wrapped lines take up as
// much room here as they do on screen.

// LineHeightFunc reports how many screen rows a buffer line takes up when it is
// wrapped to width columns. The UI owns wrapping, so it supplies this. line is
// the buffer's own storage, passed without copying: it must not be modified or
// kept after the call returns.
type LineHeightFunc func(line []rune, width int) int

// ScrollTarget says where z{t,z,b} put the cursor line in the window.
type ScrollTarget int

const (
	ScrollCursorTop    ScrollTarget = iota // zt, z<CR>
	ScrollCursorCenter                     // zz, z.
	ScrollCursorBottom                     // zb, z-
)

// SetLineHeightFunc sets how screen rows are counted per line. Without one,
// every line counts as a single row.
func (e *editor) SetLineHeightFunc(f LineHeightFunc) {
	e.lineHeightFunc = f
	e.ScrollViewport()
}

func (e *editor) lineHeight(row int) int {
	if e.lineHeightFunc == nil {
		return 1
	}
	return max(1, e.lineHeightFunc(e.buffer.GetLineRunes(row), e.state.AvailableWidth))
}

// botline returns the first line that is not shown in full, like Vim's
// w_botline. It is LineCount() when the end of the buffer is in view.
func (e *editor) botline() int {
	room := e.state.ViewportHeight
	row := e.state.TopLine
	for row < e.buffer.LineCount() {
		h := e.lineHeight(row)
		if h > room {
			break
		}
		room -= h
		row++
	}
	return row
}

// ScrollViewport brings the cursor line into the window when it has left it,
// the way Vim's update_topline does: a short way off, it scrolls just enough;
// further off, it puts the cursor line in the middle of the window.
func (e *editor) ScrollViewport() {
	height := e.state.ViewportHeight
	lineCount := e.buffer.LineCount()
	if height <= 0 || lineCount == 0 {
		e.state.TopLine = 0
		return
	}
	e.state.TopLine = max(0, min(e.state.TopLine, lineCount-1))
	row := max(0, min(e.buffer.GetCursor().Position.Row, lineCount-1))

	if row < e.state.TopLine {
		// Vim's threshold above the window is about half its height...
		halfHeight := max(height/2-1, 2)
		if e.state.TopLine-row >= halfHeight {
			e.scrollCursorHalfway(row, false)
		} else {
			e.state.TopLine = row
		}
		return
	}

	if bot := e.botline(); row >= bot {
		// ...and below it, a whole window.
		if row-bot+1 <= height+1 {
			e.scrollCursorBottom(row)
		} else {
			e.scrollCursorHalfway(row, false)
		}
	}
}

// ScrollCursorTo scrolls so the cursor line is at the top, middle or bottom of
// the window (zt, zz, zb). The cursor does not move.
func (e *editor) ScrollCursorTo(target ScrollTarget) {
	if e.state.ViewportHeight <= 0 || e.buffer.LineCount() == 0 {
		return
	}
	row := max(0, min(e.buffer.GetCursor().Position.Row, e.buffer.LineCount()-1))

	switch target {
	case ScrollCursorTop:
		e.state.TopLine = row
	case ScrollCursorCenter:
		e.scrollCursorHalfway(row, true)
	case ScrollCursorBottom:
		e.scrollCursorBottom(row)
	}
}

// scrollCursorBottom puts row on the last screen row of the window, or at the
// top when it is too tall to fit below anything else.
func (e *editor) scrollCursorBottom(row int) {
	used := e.lineHeight(row)
	top := row
	for top > 0 {
		h := e.lineHeight(top - 1)
		if used+h > e.state.ViewportHeight {
			break
		}
		used += h
		top--
	}
	e.state.TopLine = top
}

// scrollCursorHalfway puts row in the middle of the window, a port of Vim's
// scroll_cursor_halfway. Lines are added below and above the cursor line in
// turn until the window is full. Past the end of the buffer there is nothing
// to add below, so the cursor line ends up lower; atEnd (zz) instead counts
// those rows as empty ones, which scrolls past the end of the buffer.
func (e *editor) scrollCursorHalfway(row int, atEnd bool) {
	height := e.state.ViewportHeight
	lineCount := e.buffer.LineCount()

	used := e.lineHeight(row)
	above, below := 0, 0
	top := row
	nextBelow := row + 1
	for top > 0 {
		if below <= above {
			if nextBelow < lineCount {
				h := e.lineHeight(nextBelow)
				used += h
				if used > height {
					break
				}
				below += h
				nextBelow++
			} else {
				below++
				if atEnd {
					used++
				}
			}
		}
		if below > above {
			h := e.lineHeight(top - 1)
			used += h
			if used > height {
				break
			}
			above += h
			top--
		}
	}
	e.state.TopLine = top
}

// scrollAmount is Vim's 'scroll' option: how far CTRL-D and CTRL-U go. It is
// half the window unless a count set it, and resets when the window resizes.
func (e *editor) scrollAmount() int {
	height := e.state.ViewportHeight
	if e.scrollLines > 0 && e.scrollLinesHeight == height {
		return e.scrollLines
	}
	return max(height/2, 1)
}

// ScrollHalfPage is CTRL-D (down) and CTRL-U, a port of Vim's halfpage. The
// text scrolls by the 'scroll' amount and the cursor moves the same number of
// lines, so it keeps its place on screen. Once the window can't scroll any
// further, the cursor moves on its own. A count sets 'scroll' for later uses.
// It returns the moved cursor, which the caller stores.
func (e *editor) ScrollHalfPage(down bool, count int) (Cursor, error) {
	cursor := e.buffer.GetCursor()
	height := e.state.ViewportHeight
	lineCount := e.buffer.LineCount()
	if height <= 0 || lineCount == 0 {
		return cursor, nil
	}
	if down && cursor.Position.Row >= lineCount-1 {
		return cursor, ErrEndOfBuffer
	}
	if !down && cursor.Position.Row <= 0 {
		return cursor, ErrStartOfBuffer
	}

	if count > 0 {
		e.scrollLines = min(count, height)
		e.scrollLinesHeight = height
	}
	n := e.scrollAmount()

	e.ScrollViewport()
	row := cursor.Position.Row
	scrolled := 0

	if down {
		bot := e.botline()
		room := height
		for r := e.state.TopLine; r < bot; r++ {
			room -= e.lineHeight(r)
		}
		for n > 0 && bot < lineCount {
			h := e.lineHeight(e.state.TopLine)
			n -= h
			if n < 0 && scrolled > 0 {
				break
			}
			e.state.TopLine++
			if row < lineCount-1 {
				row++
			}
			scrolled += h

			room += h
			for bot < lineCount {
				bh := e.lineHeight(bot)
				if bh > room {
					break
				}
				room -= bh
				bot++
			}
		}
		if n > 0 {
			row = min(row+n, lineCount-1)
		}
		// Lines that wrap scroll further than the cursor moved; keep it in view.
		row = max(row, e.state.TopLine)
	} else {
		for n > 0 && e.state.TopLine > 0 {
			h := e.lineHeight(e.state.TopLine - 1)
			n -= h
			if n < 0 && scrolled > 0 {
				break
			}
			e.state.TopLine--
			if row > 0 {
				row--
			}
			scrolled += h
		}
		if n > 0 {
			row = max(row-n, 0)
		}
		if bot := e.botline(); row >= bot {
			row = max(bot-1, e.state.TopLine)
		}
	}

	// row is within the buffer, so neither move can hit its end and fail.
	availableWidth := e.state.AvailableWidth
	if delta := row - cursor.Position.Row; delta > 0 {
		_ = cursor.MoveDown(e.buffer, delta, availableWidth)
	} else if delta < 0 {
		_ = cursor.MoveUp(e.buffer, -delta, availableWidth)
	}
	return cursor, nil
}

// zScrollTarget reads the key after z: where it puts the cursor line, and
// whether the cursor also moves to the first non-blank (z<CR>, z., z-). ok is
// false for keys that aren't z scroll commands.
func zScrollTarget(key KeyEvent) (target ScrollTarget, firstNonBlank bool, ok bool) {
	if key.Modifiers&ModCtrl != 0 {
		return 0, false, false
	}
	switch {
	case key.Rune == 't':
		return ScrollCursorTop, false, true
	case key.Key == KeyEnter:
		return ScrollCursorTop, true, true
	case key.Rune == 'z':
		return ScrollCursorCenter, false, true
	case key.Rune == '.':
		return ScrollCursorCenter, true, true
	case key.Rune == 'b':
		return ScrollCursorBottom, false, true
	case key.Rune == '-':
		return ScrollCursorBottom, true, true
	}
	return 0, false, false
}

// handleZKey runs the second key of a z scroll command in any mode. A count
// (0 for none) first moves the cursor to that line, as in Vim. A key that
// isn't a z command cancels the z without moving anything.
func handleZKey(editor Editor, buffer Buffer, key KeyEvent, count int) {
	editor.UpdateCommand("")

	target, firstNonBlank, ok := zScrollTarget(key)
	if !ok {
		return
	}

	cursor := buffer.GetCursor()
	if count > 0 && buffer.LineCount() > 0 {
		cursor.Position.Row = min(count, buffer.LineCount()) - 1
		cursor.Position.Col = min(cursor.Position.Col, max(buffer.LineRuneCount(cursor.Position.Row)-1, 0))
	}
	if firstNonBlank {
		cursor.MoveToFirstNonBlank(buffer, editor.GetState().AvailableWidth)
	}
	buffer.SetCursor(cursor)
	editor.ScrollCursorTo(target)
}
