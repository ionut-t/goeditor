package core

import "errors"

// handleVisualPrefix runs the key that completes a g or z prefix in any
// visual mode.
func handleVisualPrefix(prefix rune, editor Editor, buffer Buffer, key KeyEvent) {
	switch prefix {
	case 'g':
		handleVisualGKey(editor, buffer, key)
	case 'z':
		handleZKey(editor, buffer, key, takeCount(editor))
	}
}

// handleVisualGKey handles the second key of a 'g' prefix in any visual mode
// (gg / ge). [count]gg goes to line count, as in normal mode.
func handleVisualGKey(editor Editor, buffer Buffer, key KeyEvent) {
	count := takeCount(editor)
	editor.UpdateCommand("")
	if key.Key == KeyEscape {
		return
	}
	cursor := buffer.GetCursor()
	switch key.Rune {
	case 'g':
		if count > 0 && buffer.LineCount() > 0 {
			cursor.Position.Row = min(count, buffer.LineCount()) - 1
			cursor.Position.Col = 0
		} else {
			cursor.MoveToBufferStart()
		}
		buffer.SetCursor(cursor)
	case 'e':
		moveErr := cursor.MoveWordToEndBackward(buffer, max(count, 1), editor.GetState().AvailableWidth, editor.IsWordChar)
		if moveErr == nil || errors.Is(moveErr, ErrStartOfBuffer) {
			buffer.SetCursor(cursor)
		}
	}
}

// applyVisualMotion handles motion keys shared by all visual modes.
//
// Covers: j/k, Ctrl-D/U, {/}, 0/$, ^, G, w/e/b, Enter, ;/,, /, n/N
// Excludes:
//   - h/l            — count differs: charwise uses the user count, line mode always uses 1
//   - g, z           — prefixes, completed by handleVisualPrefix before reaching here
//   - f/F/t/T        — started by each mode before its count is taken, so 3fx keeps the 3
//   - PageUp/PageDown, arrow keys — line mode only (handled via key.Key in the outer switch)
//
// count is 0 when none was typed; motions treat that as 1.
//
// Returns (movementAttempted, earlyReturn, moveErr).
// earlyReturn=true signals the caller must return nil immediately (search mode entered).
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
	scrollCount := count // 0 keeps 'scroll'; an explicit count, even 1, sets it
	lineCount := count   // 0 sends G to the last line; a count sends it to that line
	count = max(count, 1)
	switch {
	// Ctrl-modified keys first: normalisation puts the letter in Rune, so they
	// would otherwise match the plain-letter motions below.
	case key.IsCtrl('d'), key.IsCtrl('u'):
		*cursor, moveErr = editor.ScrollHalfPage(key.IsCtrl('d'), scrollCount)
		movementAttempted = true
	case key.Modifiers&ModCtrl != 0:
		// Unbound Ctrl combination — ignore rather than fall through.
	case key.Rune == 'j' || key.Key == KeyDown:
		moveErr = cursor.MoveDown(buffer, count, availableWidth)
		movementAttempted = true
	case key.Rune == 'k' || key.Key == KeyUp:
		moveErr = cursor.MoveUp(buffer, count, availableWidth)
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
		cursor.moveToLine(buffer, lineCount, availableWidth)
		movementAttempted = true
	case key.Key == KeyEnter:
		if count > 0 {
			cursor.Position.Row = count - 1
			buffer.SetCursor(*cursor)
			editor.UpdateCommand("")
			editor.ResetPendingCount()
		}
		movementAttempted = true
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
