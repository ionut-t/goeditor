package core

// applyVisualMotion handles motion keys shared by all visual modes.
//
// Covers: j/k, Ctrl-D/U, {/}, 0/$, ^, g, G, Enter, w/e/b, f/F/t/T, ;/,
// Excludes:
//   - h/l  — count differs between charwise (user count) and line (always 1)
//   - PageUp/PageDown, arrow keys — line mode only (handled via key.Key in the outer switch)
//
// Note: charwise visual mode handles w with an additional exclusive-motion adjustment
// in its own switch before delegating here, so the w case here only activates for
// visual line mode (where the adjustment is not needed).
//
// Returns (moveErr, movementAttempted, earlyReturn).
// earlyReturn=true signals the caller must return nil immediately (charSearch initiated).
func applyVisualMotion(
	cs *charSearchState,
	editor Editor,
	buffer Buffer,
	cursor *Cursor,
	key KeyEvent,
	count int,
) (moveErr error, movementAttempted bool, earlyReturn bool) {
	state := editor.GetState()
	availableWidth := state.AvailableWidth
	viewportHeight := state.ViewportHeight
	switch {
	case key.Rune == 'j' || key.Key == KeyDown:
		moveErr = cursor.MoveDown(buffer, count, availableWidth)
		movementAttempted = true
	case key.Rune == 'k' || key.Key == KeyUp:
		moveErr = cursor.MoveUp(buffer, count, availableWidth)
		movementAttempted = true
	case key.Key == KeyCtrlD:
		moveErr = cursor.ScrollDown(buffer, viewportHeight, availableWidth)
		movementAttempted = true
	case key.Key == KeyCtrlU:
		moveErr = cursor.ScrollUp(buffer, viewportHeight, availableWidth)
		movementAttempted = true
	case key.Rune == '{':
		moveErr = cursor.MoveBlockBackward(buffer, count)
		movementAttempted = true
	case key.Rune == '}':
		moveErr = cursor.MoveBlockForward(buffer, count)
		movementAttempted = true
	case key.Rune == '0' || key.Key == KeyHome:
		cursor.MoveToLineStart()
		movementAttempted = true
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth)
		movementAttempted = true
	case key.Rune == '^':
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
		movementAttempted = true
	case key.Rune == 'g':
		cursor.MoveToBufferStart()
		movementAttempted = true
	case key.Rune == 'G':
		cursor.MoveToBufferEnd(buffer, availableWidth)
		movementAttempted = true
	case key.Key == KeyEnter:
		if count > 0 {
			cursor.Position.Row = count - 1
			buffer.SetCursor(*cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}
		movementAttempted = true
	case key.Rune == 'f':
		cs.searchType = 'f'
		cs.waitingForChar = true
		editor.UpdateCommand("f")
		earlyReturn = true
	case key.Rune == 'F':
		cs.searchType = 'F'
		cs.waitingForChar = true
		editor.UpdateCommand("F")
		earlyReturn = true
	case key.Rune == 't':
		cs.searchType = 't'
		cs.waitingForChar = true
		editor.UpdateCommand("t")
		earlyReturn = true
	case key.Rune == 'T':
		cs.searchType = 'T'
		cs.waitingForChar = true
		editor.UpdateCommand("T")
		earlyReturn = true
	case key.Rune == ';':
		repeatCharSearch(cs, editor, buffer, count, false)
		*cursor = buffer.GetCursor()
		movementAttempted = true
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
		movementAttempted = true
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
		movementAttempted = true
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
		movementAttempted = true
	case key.Rune == ',':
		repeatCharSearch(cs, editor, buffer, count, true)
		*cursor = buffer.GetCursor()
		movementAttempted = true
	}
	return
}
