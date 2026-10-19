package core

type insertMode struct{ baseMode }

func NewInsertMode() EditorMode { return &insertMode{} }

func (m *insertMode) Name() Mode { return InsertMode }

func (m *insertMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- INSERT --")
	editor.UpdateCommand("")
}

// syncsUndo ends the undo step on cursor keys, as Vim does: text typed after
// moving the cursor undoes separately from text typed before.
func (m *insertMode) syncsUndo(key KeyEvent) bool {
	return breaksInsert(key)
}

// breaksInsert reports whether key is a cursor key, which Vim treats as
// ending one insert and starting another: for undo, for '.', and for the
// count of the command that started the insert.
func breaksInsert(key KeyEvent) bool {
	switch key.Key {
	case KeyLeft, KeyRight, KeyUp, KeyDown:
		return true
	}
	return false
}

func (m *insertMode) Exit(editor Editor, buffer Buffer) {
	if !editor.IsVimMode() {
		return
	}
	// Like Vim, leaving insert steps back onto the last inserted char (no
	// move at col 0), so normal mode never sits past the line end.
	cursor := buffer.GetCursor()
	if cursor.MoveLeft(buffer, 1, editor.GetState().AvailableWidth) == nil {
		buffer.SetCursor(cursor)
	}
}

func (m *insertMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	cursor := buffer.GetCursor()
	row, col := cursor.Position.Row, cursor.Position.Col
	var err *EditorError

	state := editor.GetState()
	availableWidth := state.AvailableWidth

	switch key.Key {
	case KeyEscape:
		if !editor.IsVimMode() {
			return nil
		}
		if r, ok := editor.(insertRepeater); ok {
			err = r.repeatInsert()
		}
		editor.SetNormalMode()
		return err

	case KeyBackspace:
		if col > 0 {
			// Delete character before cursor
			err = buffer.DeleteRunesAt(row, col-1, 1)
			if err == nil {
				_ = cursor.MoveLeft(buffer, 1, availableWidth) // Move cursor back
				buffer.SetCursor(cursor)
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
			}
		} else {
			// Cannot backspace at start of buffer
			err = &EditorError{
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
		} else {
			err = &EditorError{
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
			_ = cursor.MoveRight(buffer, 1, availableWidth) // Tab counts as one "character" position for movement
			buffer.SetCursor(cursor)
		} else {
			err = &EditorError{
				id:  ErrInvalidPositionId,
				err: insertErr,
			}
		}

		return err

		// Handle arrow keys, pgup/dn etc. in Insert mode? (Some terminals allow this)
		// For a basic Vim clone, often these do nothing or exit insert mode.
		// Let's ignore them for now.

	case KeyLeft:
		_ = cursor.MoveLeftOrUp(buffer, 1, availableWidth)
		buffer.SetCursor(cursor)
		return nil

	case KeyRight:
		_ = cursor.MoveRightOrDown(buffer, 1, availableWidth)
		buffer.SetCursor(cursor)
		return nil

	case KeyUp:
		if row > 0 {
			_ = cursor.MoveUp(buffer, 1, availableWidth) // Move cursor up
			buffer.SetCursor(cursor)
		}
		return nil

	case KeyDown:
		if row < buffer.LineCount()-1 {
			_ = cursor.MoveDown(buffer, 1, availableWidth) // Move cursor down
			buffer.SetCursor(cursor)
		}
		return nil

	default: // Handle regular character runes
		// Ctrl+<letter> carries the letter in Rune after normalisation. Insert
		// mode binds no Ctrl keys, so these must be ignored rather than typed.
		if key.Modifiers&ModCtrl != 0 {
			return nil
		}

		if key.Rune != 0 {
			insertErr := buffer.InsertRunesAt(row, col, []rune{key.Rune})
			if insertErr == nil {
				_ = cursor.MoveRight(buffer, 1, availableWidth) // Move cursor forward
				buffer.SetCursor(cursor)
			} else {
				err = &EditorError{
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

// insertRepeater is implemented by the editor so that insert mode can repeat
// what was typed for a count (3ix<Esc>) without it being part of the public
// Editor interface.
type insertRepeater interface {
	repeatInsert() *EditorError
}
