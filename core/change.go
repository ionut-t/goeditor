package core

import "errors"

// changeWordsForward implements cw (cW with bigWordClass). On a word Vim
// changes to its end, as ce, except that from the word's last character it
// changes only that character on the first count. On a space, a tab or an
// empty line it takes w's motion, as dw (Vim without 'w' in 'cpoptions').
func changeWordsForward(editor Editor, buffer Buffer, count int, cls func(rune) int) *EditorError {
	cursor := buffer.GetCursor()
	start := cursor.Position
	line := buffer.GetLineRunes(start.Row)
	if start.Col >= len(line) || isWhiteSpace(line[start.Col]) {
		return applyOperatorToWords(editor, buffer, "change", count, cls)
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
	return applyOperatorToRange(editor, buffer, "change", start, tempCursor.Position)
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
