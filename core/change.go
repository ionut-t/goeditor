package core

func changeWords(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	// For 'cw', Vim deletes to the end of the current word (like 'ce').
	_ = tempCursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)

	// In 'cw', we delete INCLUDING the character at the end of the word.
	// But deleteRange is exclusive of endPos, so we move one right.
	tempCursor.MoveRight(buffer, 1, availableWidth)
	exclusiveEndPos := tempCursor.Position

	if startPos != exclusiveEndPos {
		err := deleteRange(buffer, startPos, exclusiveEndPos)
		if err == nil {
			editor.SaveHistory()
			buffer.SetCursor(cursor)
			editor.SetInsertMode()
		}
		return err
	}

	return nil
}

func changeWordsBackward(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	endPos := cursor.Position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	_ = tempCursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
	startPos := tempCursor.Position

	if startPos != endPos {
		err := deleteRange(buffer, startPos, endPos)
		if err == nil {
			editor.SaveHistory()
			cursor.Position = startPos
			buffer.SetCursor(cursor)
			editor.SetInsertMode()
		}
		return err
	}
	return nil
}

func changeToEndOfLine(editor Editor, buffer Buffer) *EditorError {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)
	if cursor.Position.Col < lineLen {
		delCount := lineLen - cursor.Position.Col
		if err := buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, delCount); err != nil {
			return err
		}
		editor.SaveHistory()
	}
	buffer.SetCursor(cursor)
	editor.SetInsertMode()
	return nil
}

func replaceCharUnderCursor(editor Editor, buffer Buffer, ch rune) *EditorError {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)

	if lineLen == 0 || cursor.Position.Col >= lineLen {
		return nil
	}

	if err := buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, 1); err != nil {
		return err
	}

	if err := buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune{ch}); err != nil {
		return &EditorError{id: ErrInvalidPositionId, err: err}
	}

	editor.SaveHistory()
	return nil
}
