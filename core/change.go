package core

import (
	"errors"
	"slices"
)

// changeWordsForward implements cw (cW with bigWordClass). On a word Vim
// changes to its end, as ce, except that from the word's last character it
// changes only that character on the first count. On a space, a tab or an
// empty line it takes w's motion, as dw (Vim without 'w' in 'cpoptions').
func changeWordsForward(editor Editor, buffer Buffer, count int, cls func(rune) int) *EditorError {
	cursor := buffer.GetCursor()
	start := cursor.Position
	line := buffer.GetLineRunes(start.Row)
	if start.Col >= len(line) || isWhiteSpace(line[start.Col]) {
		return applyOperatorToWords(editor, buffer, opChange, count, cls)
	}

	// A blank other than space or tab (a no-break space) takes the ce path
	// too, and changes to the end of the next word.
	tempCursor := cursor
	class := cls(line[start.Col])
	next := charEnd(line, start.Col)
	if class == blankClass || next < len(line) && classAt(line, next, cls) == class {
		_ = tempCursor.MoveWordToEnd(buffer, count, cls)
	} else if count > 1 {
		_ = tempCursor.MoveWordToEnd(buffer, count-1, cls)
	}
	_ = tempCursor.MoveRight(buffer, 1) // the end is inclusive
	return applyOperatorToRange(editor, buffer, opChange, start, tempCursor.Position)
}

// replaceCharUnderCursor is r: it replaces count characters from the cursor
// with ch, and reports whether it did.
func replaceCharUnderCursor(editor Editor, buffer Buffer, ch rune, count int) (replaced bool, err *EditorError) {
	cursor := buffer.GetCursor()

	// Vim replaces nothing when the line has fewer than count characters left.
	// Each character goes with its composing characters.
	endCol, n := charsForward(buffer.GetLineRunes(cursor.Position.Row), cursor.Position.Col, count)
	if n < count {
		return false, nil
	}

	if err := buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, endCol-cursor.Position.Col); err != nil {
		return false, err
	}

	runes := make([]rune, count)
	for i := range runes {
		runes[i] = ch
	}
	if err := buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, runes); err != nil {
		return false, &EditorError{id: ErrInvalidPositionId, err: err}
	}

	// The cursor ends on the last replaced character.
	cursor.Position.Col += count - 1
	buffer.SetCursor(cursor)
	return true, nil
}

// replaceWithLineBreak is r<CR> (and r CTRL-J): it replaces count characters
// from the cursor with one line break, and reports whether it did. As in Vim
// without 'autoindent', no white space around the break is removed. The
// cursor goes to the start of the new line, and undo puts it back where r was.
func replaceWithLineBreak(editor Editor, buffer Buffer, count int) (replaced bool, err *EditorError) {
	cursor := buffer.GetCursor()
	pos := cursor.Position

	endCol, n := charsForward(buffer.GetLineRunes(pos.Row), pos.Col, count)
	if n < count {
		return false, nil
	}

	markChangeStart(editor, pos)
	if err := buffer.DeleteRunesAt(pos.Row, pos.Col, endCol-pos.Col); err != nil {
		return false, err
	}
	if err := buffer.InsertRunesAt(pos.Row, pos.Col, []rune{'\n'}); err != nil {
		return false, &EditorError{id: ErrInvalidPositionId, err: err}
	}

	cursor.Position = Position{Row: pos.Row + 1, Col: 0}
	cursor.Preferred = 0
	buffer.SetCursor(cursor)
	return true, nil
}

// replaceSelection is r{ch} on a visual selection from start to end, both
// inclusive (end on a line break leaves the break alone). Every character
// becomes one ch: a tab, a wide character, or a letter with its composing
// characters alike. Line breaks stay. Linewise, whole lines are replaced. The
// cursor goes to start, and undo puts it back at changeStart. A selection of
// only empty lines changes nothing but is still the change '.' repeats, as in
// Vim.
func replaceSelection(editor Editor, buffer Buffer, start, end Position, linewise bool, changeStart Position, ch rune) *EditorError {
	markChangeStart(editor, changeStart)
	replaced := false
	for row := start.Row; row <= end.Row; row++ {
		line := buffer.GetLineRunes(row)
		from, to := 0, len(line)
		if !linewise && row == start.Row {
			from = min(start.Col, len(line))
		}
		if !linewise && row == end.Row {
			to = min(charEnd(line, end.Col), len(line))
		}
		n := charCount(line, from, to)
		if n == 0 {
			continue
		}
		if err := buffer.DeleteRunesAt(row, from, to-from); err != nil {
			return err
		}
		if err := buffer.InsertRunesAt(row, from, slices.Repeat([]rune{ch}, n)); err != nil {
			return &EditorError{id: ErrInvalidPositionId, err: err}
		}
		replaced = true
	}
	if !replaced {
		countAsChange(editor)
	}
	cursor := buffer.GetCursor()
	cursor.Position = start
	buffer.SetCursor(cursor)
	editor.SetNormalMode()
	endVisualOperator(buffer)
	return nil
}

// visualReplaceChar returns the character the key after a visual r replaces
// the selection with, or false if the key cancels r instead. Esc and
// Backspace cancel, as in Vim. Enter and Ctrl keys cancel too: Vim puts them
// in literally (^M, ^H), but the editor can't show control characters yet.
func visualReplaceChar(key KeyEvent) (rune, bool) {
	if key.Rune == 0 || key.Modifiers&ModCtrl != 0 {
		return 0, false
	}
	switch key.Key {
	case KeyUnknown, KeySpace, KeyTab:
		return key.Rune, true
	}
	return 0, false
}

// changeLineRange is the linewise change (cc, S, cj, cG, Vc): it empties the
// lines [startRow, endRow] down to a single empty line, as Vim does, and
// starts insert mode on it.
func changeLineRange(editor Editor, buffer Buffer, startRow, endRow int) *EditorError {
	if startRow < 0 || endRow >= buffer.LineCount() || startRow > endRow {
		return &EditorError{id: ErrInvalidPositionId, err: errors.New("invalid line range for change")}
	}

	end := Position{Row: endRow, Col: buffer.LineRuneCount(endRow)}
	if err := deleteRange(buffer, Position{Row: startRow, Col: 0}, end); err != nil {
		return err
	}

	cursor := buffer.GetCursor()
	cursor.Position = Position{Row: startRow, Col: 0}
	buffer.SetCursor(cursor)
	editor.SetInsertMode()
	return nil
}

// changeLines changes count lines starting at the cursor row (cc, S).
func changeLines(editor Editor, buffer Buffer, count int) *EditorError {
	startRow := buffer.GetCursor().Position.Row
	endRow := min(startRow+count-1, buffer.LineCount()-1)
	return changeLineRange(editor, buffer, startRow, endRow)
}
