package core

type insertMode struct{} // Can hold state if needed (e.g., for abbreviations)

func NewInsertMode() EditorMode { return &insertMode{} }

func (m *insertMode) Name() Mode { return InsertMode }

func (m *insertMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- INSERT --")
	editor.UpdateCommand("")
	// Save state for undo *before* the first insertion
	editor.SaveHistory()
}

func (m *insertMode) Exit(editor Editor, buffer Buffer) {}

func (m *insertMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *Error {
	cursor := buffer.GetCursor()
	row, col := cursor.Position.Row, cursor.Position.Col
	var err *Error

	state := editor.GetState()
	availableWidth := state.AvailableWidth

	switch key.Key {
	case KeyEscape:
		if !editor.IsVimMode() {
			return nil
		}
		editor.SetNormalMode()
		return nil

	case KeyBackspace:
		if col > 0 {
			// Delete character before cursor
			err = buffer.DeleteRunesAt(row, col-1, 1)
			if err == nil {
				cursor.MoveLeft(buffer, 1, availableWidth) // Move cursor back
				buffer.SetCursor(cursor)
				editor.SaveHistory() // Save after modification
			}
		} else if row > 0 {
			// At beginning of line, merge with previous line
			prevLineLen := buffer.LineRuneCount(row - 1)
			// Delete the newline character between row-1 and row
			// This effectively merges the lines. DeleteRunesAt handles merging.
			err := buffer.DeleteRunesAt(row-1, prevLineLen, 1) // Delete the newline char
			if err == nil {
				// Move cursor to the join point
				cursor.Position.Row--
				cursor.Position.Col = prevLineLen
				buffer.SetCursor(cursor)
				editor.SaveHistory()
			}
		} else {
			// Cannot backspace at start of buffer
			err = &Error{
				err: ErrStartOfBuffer,
				id:  ErrStartOfBufferId,
			}
		}
		return err

	case KeyEnter:
		// Insert newline character
		insertErr := buffer.InsertRunesAt(row, col, []rune{'\n'})
		if insertErr == nil {
			// Move cursor to beginning of new line (which is row+1, col 0)
			cursor.Position.Row++
			cursor.Position.Col = 0
			cursor.Preferred = 0 // Reset preferred col
			buffer.SetCursor(cursor)
			editor.SaveHistory()
		} else {
			err = &Error{
				id:  ErrInvalidPositionId,
				err: insertErr,
			}
		}
		return err

	case KeyTab:
		// Insert tab character (or spaces if configured)
		// For simplicity, insert literal tab rune
		insertErr := buffer.InsertRunesAt(row, col, []rune{'\t'})
		if insertErr == nil {
			cursor.MoveRight(buffer, 1, availableWidth) // Tab counts as one "character" position for movement
			buffer.SetCursor(cursor)
			editor.SaveHistory()
		} else {
			err = &Error{
				id:  ErrInvalidPositionId,
				err: insertErr,
			}
		}

		return err

		// Handle arrow keys, pgup/dn etc. in Insert mode? (Some terminals allow this)
		// For a basic Vim clone, often these do nothing or exit insert mode.
		// Let's ignore them for now.

	case KeyLeft:
		cursor.MoveLeftOrUp(buffer, 1, col)
		buffer.SetCursor(cursor)
		editor.SaveHistory() // Save after modification
		return nil

	case KeyRight:
		cursor.MoveRightOrDown(buffer, 1, col)
		buffer.SetCursor(cursor)
		editor.SaveHistory() // Save after modification
		return nil

	case KeyUp:
		if row > 0 {
			cursor.MoveUp(buffer, 1, availableWidth) // Move cursor up
			buffer.SetCursor(cursor)
			editor.SaveHistory() // Save after modification
		}
		return nil

	case KeyDown:
		if row < buffer.LineCount()-1 {
			cursor.MoveDown(buffer, 1, availableWidth) // Move cursor down
			buffer.SetCursor(cursor)
			editor.SaveHistory() // Save after modification
		}
		return nil

	default: // Handle regular character runes
		if key.Rune != 0 {
			insertErr := buffer.InsertRunesAt(row, col, []rune{key.Rune})
			if insertErr == nil {
				cursor.MoveRight(buffer, 1, availableWidth) // Move cursor forward
				buffer.SetCursor(cursor)
				editor.SaveHistory() // Save after modification
			} else {
				err = &Error{
					id:  ErrInvalidPositionId,
					err: insertErr,
				}
			}
			return err
		}
		// Ignore unknown special keys or modifiers without runes in insert mode
		return nil
	}
}
