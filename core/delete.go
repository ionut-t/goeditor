package core

import (
	"errors"
	"strings"
)

// deleteRange deletes text from start (inclusive) to end (exclusive).
// Handles multi-line deletions.
func deleteRange(buffer Buffer, start, end Position) *EditorError {
	// Ensure start is before end
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}

	if start.Row == end.Row {
		// Single line deletion
		count := end.Col - start.Col
		if count > 0 {
			return buffer.DeleteRunesAt(start.Row, start.Col, count)
		}
		return nil
	}

	// Multi-line deletion
	// 1. Delete from start.Col to end of start line
	startLineLen := buffer.LineRuneCount(start.Row)
	if startLineLen > start.Col {
		if err := buffer.DeleteRunesAt(start.Row, start.Col, startLineLen-start.Col); err != nil {
			return err
		}
	}

	// 2. Delete full lines between start.Row and end.Row
	// We delete from bottom-up to keep indices stable
	for r := end.Row - 1; r > start.Row; r-- {
		lineLen := buffer.LineRuneCount(r)
		// Delete line + newline
		if err := buffer.DeleteRunesAt(r, 0, lineLen+1); err != nil {
			return err
		}
	}

	// 3. Delete from start of end line to end.Col, then merge with start line
	// The end line is now at start.Row + 1
	if err := buffer.DeleteRunesAt(start.Row+1, 0, end.Col); err != nil {
		return err
	}

	// Merge start.Row and the remains of end line (which is now at start.Row+1)
	// Deleting the newline at the end of start.Row merges them.
	return buffer.DeleteRunesAt(start.Row, buffer.LineRuneCount(start.Row), 1)
}

// deleteLineRange deletes an inclusive range of lines [startRow, endRow].
// It handles single-line buffers correctly and returns content in top-to-bottom order.
func deleteLineRange(editor Editor, buffer Buffer, startRow, endRow int) (string, *EditorError) {
	if startRow < 0 || endRow >= buffer.LineCount() || startRow > endRow {
		return "", &EditorError{
			id:  ErrInvalidPositionId,
			err: errors.New("invalid line range for deletion"),
		}
	}

	availableWidth := editor.GetState().AvailableWidth

	// Collect content in top-to-bottom order before deleting
	var deletedContent strings.Builder

	// Delete lines from bottom up to keep indices valid, collecting content top-to-bottom
	lineContents := make([]string, endRow-startRow+1)
	for i := startRow; i <= endRow; i++ {
		lineContents[i-startRow] = string(buffer.GetLineRunes(i)) + "\n"
	}
	for i := range lineContents {
		deletedContent.WriteString(lineContents[i])
	}

	var firstErr *EditorError

	for i := endRow; i >= startRow; i-- {
		lineRunes := buffer.GetLineRunes(i)

		var err *EditorError
		if buffer.LineCount() == 1 {
			// Only line left — clear it but keep the row
			err = buffer.DeleteRunesAt(i, 0, len(lineRunes))
		} else if i == buffer.LineCount()-1 {
			// Last line in a multi-line buffer: clear content then remove the row
			// by deleting the newline at the end of the previous line.
			if len(lineRunes) > 0 {
				err = buffer.DeleteRunesAt(i, 0, len(lineRunes))
			}
			if err == nil {
				err = buffer.DeleteRunesAt(i-1, buffer.LineRuneCount(i-1), 1)
			}
		} else {
			err = buffer.DeleteRunesAt(i, 0, len(lineRunes)+1)
		}

		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Adjust cursor after deletion
	cursor := buffer.GetCursor()
	newRow := startRow
	if newRow >= buffer.LineCount() {
		newRow = max(buffer.LineCount()-1, 0)
	}
	cursor.Position.Row = newRow
	buffer.SetCursor(cursor)
	cursor = buffer.GetCursor()
	cursor.MoveToFirstNonBlank(buffer, availableWidth)
	buffer.SetCursor(cursor)

	if firstErr == nil {
		editor.SaveHistory()
	}

	return deletedContent.String(), firstErr
}

// deleteLines deletes `count` lines starting from the cursor row.
func deleteLines(editor Editor, buffer Buffer, count int) (string, *EditorError) {
	cursor := buffer.GetCursor()
	startRow := cursor.Position.Row
	endRow := startRow + count - 1
	if endRow >= buffer.LineCount() {
		endRow = buffer.LineCount() - 1
	}
	return deleteLineRange(editor, buffer, startRow, endRow)
}

func deleteWords(editor Editor, buffer Buffer, count int) (err *EditorError) {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	_ = tempCursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
	endPos := tempCursor.Position

	if startPos != endPos {
		err = deleteRange(buffer, startPos, endPos)
		if err == nil {
			editor.SaveHistory()
			buffer.SetCursor(cursor) // Cursor stays at startPos
		}
	}

	return err
}

func deleteWordsBackward(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	originalPos := cursor.Position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	_ = tempCursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
	startPos := tempCursor.Position

	if startPos != originalPos {
		err := deleteRange(buffer, startPos, originalPos)
		if err == nil {
			editor.SaveHistory()
			cursor.Position = startPos // Move cursor to the beginning of the deleted range
			buffer.SetCursor(cursor)
		}
		return err
	}
	return nil
}

func deleteWordToEnd(editor Editor, buffer Buffer, count int) *EditorError {
	cursor := buffer.GetCursor()
	startPos := cursor.Position
	tempCursor := cursor
	availableWidth := editor.GetState().AvailableWidth

	_ = tempCursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
	// MoveWordToEnd lands on the last char of the word (inclusive), so move one right
	// to get the exclusive end for deleteRange.
	tempCursor.MoveRight(buffer, 1, availableWidth)
	exclusiveEndPos := tempCursor.Position

	if startPos != exclusiveEndPos {
		err := deleteRange(buffer, startPos, exclusiveEndPos)
		if err == nil {
			editor.SaveHistory()
			buffer.SetCursor(cursor)
		}
		return err
	}
	return nil
}

func deleteToEndOfLine(editor Editor, buffer Buffer) (string, *EditorError) {
	cursor := buffer.GetCursor()
	lineLen := buffer.LineRuneCount(cursor.Position.Row)
	var deletedContent string
	var err *EditorError

	if cursor.Position.Col < lineLen { // Only delete if not already at/past end
		lineRunes := buffer.GetLineRunes(cursor.Position.Row)
		deletedContent = string(lineRunes[cursor.Position.Col:])

		deleteCount := lineLen - cursor.Position.Col
		err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, deleteCount)
		if err == nil {
			editor.SaveHistory()
		}
	}

	return deletedContent, err
}

// deleteVisualSelection deletes the text covered by a charwise visual selection.
// Returns the deleted content, the cursor position after deletion, and any error.
func deleteVisualSelection(buffer Buffer, startPos, endPos Position) (string, Position, *EditorError) {
	var err *EditorError

	var deletedContent string
	startSel, endSel := NormalizeSelection(startPos, endPos)
	finalCursorPos := startSel // Default final position is the start of selection

	// Simple case: Single line selection
	if startSel.Row == endSel.Row {
		lineRunes := buffer.GetLineRunes(startSel.Row)
		count := endSel.Col - startSel.Col + 1 // Inclusive delete
		if count > 0 {
			endCol := min(startSel.Col+count, len(lineRunes))
			deletedContent = string(lineRunes[startSel.Col:endCol])
			err = buffer.DeleteRunesAt(startSel.Row, startSel.Col, count)
		}
	} else {
		// Multi-line selection.
		// First, gather all the content that will be deleted.
		var contentBuilder strings.Builder
		// Part of the first line
		startLineRunes := buffer.GetLineRunes(startSel.Row)
		if startSel.Col < len(startLineRunes) {
			contentBuilder.WriteString(string(startLineRunes[startSel.Col:]))
		}
		contentBuilder.WriteString("\n")

		// Intermediate lines
		for i := startSel.Row + 1; i < endSel.Row; i++ {
			lineRunes := buffer.GetLineRunes(i)
			contentBuilder.WriteString(string(lineRunes))
			contentBuilder.WriteString("\n")
		}

		// Part of the last line
		endLineRunes := buffer.GetLineRunes(endSel.Row)
		if endSel.Col+1 <= len(endLineRunes) {
			contentBuilder.WriteString(string(endLineRunes[:endSel.Col+1]))
		} else {
			contentBuilder.WriteString(string(endLineRunes))
		}
		deletedContent = contentBuilder.String()

		// 1. Delete from startCol to end of startLine
		startLine := buffer.GetLineRunes(startSel.Row)
		startLineLen := len(startLine)
		delCount1 := startLineLen - startSel.Col
		if delCount1 > 0 {
			if err := buffer.DeleteRunesAt(startSel.Row, startSel.Col, delCount1); err != nil {
				return "", finalCursorPos, err
			}
		}

		// 2. Delete intermediate full lines
		linesToDelete := endSel.Row - startSel.Row - 1
		for range linesToDelete {
			targetRow := startSel.Row + 1
			lineLen := buffer.LineRuneCount(targetRow)
			err = buffer.DeleteRunesAt(targetRow, 0, lineLen+1)
			if err != nil {
				return "", finalCursorPos, err
			}
		}

		// 3. Delete from beginning of the original endLine up to endCol
		currentEndRow := startSel.Row + 1
		if currentEndRow < buffer.LineCount() {
			delCount2 := endSel.Col + 1
			if delCount2 > 0 {
				err = buffer.DeleteRunesAt(currentEndRow, 0, delCount2)
				if err != nil {
					return "", finalCursorPos, err
				}
			}

			// 4. Merge lines
			startLineLenAfterDel := buffer.LineRuneCount(startSel.Row)
			if startLineLenAfterDel >= 0 && startSel.Row+1 < buffer.LineCount() {
				err = buffer.DeleteRunesAt(startSel.Row, startLineLenAfterDel, 1)
				if err != nil {
					return "", finalCursorPos, err
				}
			}
		} else {
			if startLineLen == delCount1 && buffer.LineRuneCount(startSel.Row) == 0 && startSel.Row+1 < buffer.LineCount() {
				err = buffer.DeleteRunesAt(startSel.Row, 0, 1)
				if err != nil {
					return "", finalCursorPos, err
				}
			}
		}
	}

	return deletedContent, finalCursorPos, err
}
