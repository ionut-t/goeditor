package core

import "errors"

// handleVisualGKey handles the second key of a 'g' prefix in any visual mode (gg / ge).
// It resets the waitingForG flag, clears the command display, and applies the motion.
func handleVisualGKey(waitingForG *bool, editor Editor, buffer Buffer, key KeyEvent, count int) {
	*waitingForG = false
	editor.UpdateCommand("")
	if key.Key == KeyEscape {
		return
	}
	cursor := buffer.GetCursor()
	switch key.Rune {
	case 'g':
		cursor.MoveToBufferStart()
		buffer.SetCursor(cursor)
	case 'e':
		moveErr := cursor.MoveWordToEndBackward(buffer, count, editor.GetState().AvailableWidth, editor.IsWordChar)
		if moveErr == nil || errors.Is(moveErr, ErrStartOfBuffer) {
			buffer.SetCursor(cursor)
		}
	}
}

// applyVisualMotion handles motion keys shared by all visual modes.
//
// Covers: j/k, Ctrl-D/U, {/}, 0/$, ^, G, w/e/b, Enter, f/F/t/T, ;/,, /, n/N
// Excludes:
//   - h/l            — count differs: charwise uses the user count, line mode always uses 1
//   - g              — handled by each mode's waitingForG state before reaching here
//   - PageUp/PageDown, arrow keys — line mode only (handled via key.Key in the outer switch)
//
// Returns (movementAttempted, earlyReturn, moveErr).
// earlyReturn=true signals the caller must return nil immediately (charSearch initiated).
func applyVisualMotion(
	cs *charSearchState,
	editor Editor,
	buffer Buffer,
	cursor *Cursor,
	key KeyEvent,
	count int,
) (movementAttempted bool, earlyReturn bool, moveErr error) {
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
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
		movementAttempted = true
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
		movementAttempted = true
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
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
	case key.Rune == ',':
		repeatCharSearch(cs, editor, buffer, count, true)
		*cursor = buffer.GetCursor()
		movementAttempted = true
	case key.Rune == '/':
		editor.SetSearchMode()
		earlyReturn = true
	case key.Rune == '?':
		editor.SetBackwardSearchMode()
		earlyReturn = true
	case key.Rune == 'n':
		*cursor = editor.NextSearchResult()
		movementAttempted = true
	case key.Rune == 'N':
		*cursor = editor.PreviousSearchResult()
		movementAttempted = true
	case key.Rune == '*':
		*cursor = editor.SearchWordUnderCursor(false)
		movementAttempted = true
	case key.Rune == '#':
		*cursor = editor.SearchWordUnderCursor(true)
		movementAttempted = true
	}
	return
}
