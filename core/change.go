package core

import "errors"

// changeWordsForward implements cw (cW with isBigWordChar). On a word Vim
// changes to its end, as ce, except that from the word's last character it
// changes only that character on the first count. On blanks or an empty line
// it takes w's motion, as dw (Vim without 'w' in 'cpoptions').
func changeWordsForward(editor Editor, buffer Buffer, count int, isWordChar func(rune) bool) *EditorError {
	cursor := buffer.GetCursor()
	start := cursor.Position
	line := buffer.GetLineRunes(start.Row)
	if start.Col >= len(line) || isWhiteSpace(line[start.Col]) {
		return applyOperatorToWords(editor, buffer, "change", count, isWordChar)
	}

	tempCursor := cursor
	if start.Col+1 < len(line) && wordClass(line[start.Col+1], isWordChar) == wordClass(line[start.Col], isWordChar) {
		_ = tempCursor.MoveWordToEnd(buffer, count, isWordChar)
	} else if count > 1 {
		_ = tempCursor.MoveWordToEnd(buffer, count-1, isWordChar)
	}
	_ = tempCursor.MoveRight(buffer, 1) // the end is inclusive
	return applyOperatorToRange(editor, buffer, "change", start, tempCursor.Position)
}

// The classes wordClass returns, with Vim's values.
const (
	blankClass    = 0
	punctClass    = 1 // any other character
	wordCharClass = 2
)

// wordClass is Vim's cls(): blankClass for a blank, wordCharClass for a word
// character and punctClass for any other character.
func wordClass(r rune, isWordChar func(rune) bool) int {
	switch {
	case isWhiteSpace(r):
		return blankClass
	case isWordChar(r):
		return wordCharClass
	default:
		return punctClass
	}
}

func changeToEndOfLine(editor Editor, buffer Buffer) *EditorError {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)
	if cursor.Position.Col < lineLen {
		delCount := lineLen - cursor.Position.Col
		if err := buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, delCount); err != nil {
			return err
		}
	}
	buffer.SetCursor(cursor)
	editor.SetInsertMode()
	return nil
}

func replaceCharUnderCursor(editor Editor, buffer Buffer, ch rune, count int) *EditorError {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)

	// Vim replaces nothing when the line has fewer than count characters left.
	if count > lineLen-cursor.Position.Col {
		return nil
	}

	if err := buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, count); err != nil {
		return err
	}

	runes := make([]rune, count)
	for i := range runes {
		runes[i] = ch
	}
	if err := buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, runes); err != nil {
		return &EditorError{id: ErrInvalidPositionId, err: err}
	}

	// The cursor ends on the last replaced character.
	cursor.Position.Col += count - 1
	buffer.SetCursor(cursor)
	return nil
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
