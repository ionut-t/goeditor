package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// readClipboard returns the clipboard's content, or an error if there is no
// clipboard or it can't be read.
func (e *editor) readClipboard() (string, error) {
	if e.clipboard == nil {
		return "", errors.New("clipboard handler not set")
	}
	content, err := e.clipboard.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}
	return content, nil
}

func (e *editor) PasteAfter() (string, error) {
	content, err := e.paste(false, 1)
	if err != nil {
		return "", err.err
	}
	return content, nil
}

func (e *editor) PasteBefore() (string, error) {
	content, err := e.paste(true, 1)
	if err != nil {
		return "", err.err
	}
	return content, nil
}

// paste puts count copies of the clipboard after the cursor, or before it,
// with one edit, and returns the clipboard's content. Content ending in a
// newline is linewise (all line-wise yanks, yy and Vy, append one): it goes
// below the cursor line, or above it, whatever the column. Otherwise it goes
// after the cursor char, or at the cursor. The cursor goes where Vim's p puts
// it: the first non-blank of the first pasted line for a linewise paste, the
// last pasted char for text within a line, and the first pasted char for text
// spanning lines. A count over maxRepeatedInsert runes in all fails with
// ErrTextTooLong and pastes nothing; a single copy has no limit, as a single
// insert has none.
func (e *editor) paste(before bool, count int) (string, *EditorError) {
	content, err := e.readClipboard()
	if err != nil {
		return "", &EditorError{id: ErrFailedToPasteId, err: err}
	}
	return content, e.put([]rune(content), before, count)
}

// pasteLines is paste for p and P over a linewise selection: text within a
// line, or spanning lines, goes in as whole lines of its own, as Vim puts a
// charwise register with PUT_LINE. Linewise content is pasted as it is.
func (e *editor) pasteLines(before bool, count int) (string, *EditorError) {
	content, err := e.readClipboard()
	if err != nil {
		return "", &EditorError{id: ErrFailedToPasteId, err: err}
	}
	text := []rune(content)
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	return content, e.put(text, before, count)
}

// put inserts count copies of text as paste describes.
func (e *editor) put(text []rune, before bool, count int) *EditorError {
	if len(text) == 0 {
		return nil
	}
	count = max(count, 1)
	if count > 1 && len(text) > maxRepeatedInsert/count {
		return textTooLong(count, len(text), "rune")
	}
	all := slices.Repeat(text, count)

	cursor := e.buffer.GetCursor()
	row := cursor.Position.Row
	lineLen := e.buffer.LineRuneCount(row)

	if all[len(all)-1] == '\n' {
		at := 0
		if !before {
			// The line break moves from the end of the pasted lines to
			// before them, so they start on the next line. Shifted in
			// place: a long paste isn't copied a second time.
			copy(all[1:], all)
			all[0] = '\n'
			at = lineLen
		}
		// Before: inserting the lines at (row, 0) pushes the current line
		// down, so the first pasted line is at row.
		if err := e.buffer.InsertRunesAt(row, at, all); err != nil {
			return &EditorError{id: ErrInvalidPositionId, err: err}
		}
		if !before {
			cursor.Position.Row++
		}
		cursor.MoveToFirstNonBlank(e.buffer)
		e.buffer.SetCursor(cursor)
		return nil
	}

	col := cursor.Position.Col
	if !before {
		// After the cursor char; on an empty line Col+1 would be past the
		// end, so the text goes at col 0.
		col = min(col+1, lineLen)
	}
	if err := e.buffer.InsertRunesAt(row, col, all); err != nil {
		return &EditorError{id: ErrInvalidPositionId, err: err}
	}
	cursor.Position.Col = col
	if !slices.Contains(text, '\n') {
		cursor.Position.Col += len(all) - 1
	}
	cursor.Preferred = cursor.Position.Col
	e.buffer.SetCursor(cursor)
	return nil
}

// yankNothing empties the clipboard, as Vim's yank over an empty range
// (y0 in column 0) empties the register.
func (e *editor) yankNothing() error {
	if e.clipboard == nil {
		return errors.New("clipboard handler not set")
	}
	if err := e.clipboard.Write(""); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return nil
}

// Copy extracts text based on visual selection or current line and writes to clipboard.
func (e *editor) Copy(op copyType) error {
	if e.clipboard == nil {
		return errors.New("clipboard handler not set")
	}

	state := e.GetState() // Use local variable for state
	buffer := e.GetBuffer()
	cursor := buffer.GetCursor()

	var start, end Position
	isVisual := state.VisualStart.Row != -1
	isLineWise := false // Flag to indicate if the copy includes trailing newline(s)

	if isVisual {
		// Visual mode or yank from normal mode: use the selected range, normalize it
		start, end = NormalizeSelection(state.VisualStart, cursor.Position)

		// Check if the selection is line-wise
		// Either from visual-line mode OR from YankSelection being set to SelectionLine
		if state.Mode == VisualLineMode || state.YankSelection == SelectionLine {
			isLineWise = true
			start = Position{Row: start.Row, Col: 0}                             // Start of the line
			end = Position{Row: end.Row, Col: buffer.LineRuneCount(end.Row) - 1} // End of the line
		}

	} else {
		// Not in visual mode: Assume line-wise copy of the current line.
		// Commands like 'yy' would set this up before calling a generalized copy/yank function.
		// For a standalone Copy, let's copy the current line.
		start = Position{Row: cursor.Position.Row, Col: 0}
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		end = Position{Row: cursor.Position.Row, Col: lineLen - 1} // End is inclusive for calculation below
		if lineLen == 0 {
			end.Col = 0 // Handle empty line
		}
		isLineWise = true
	}

	// Extract the content based on the calculated range
	var contentBuilder strings.Builder

	if start.Row == end.Row {
		// Single line copy/selection
		lineRunes := buffer.GetLineRunes(start.Row)
		lineLen := len(lineRunes)

		// Adjust end column for slicing (make it exclusive upper bound)
		sliceEndCol := min(end.Col+1, lineLen)
		// Adjust start column too
		sliceStartCol := min(max(start.Col, 0), lineLen) // Can't start past end

		if sliceStartCol < sliceEndCol { // Ensure valid slice range
			contentBuilder.WriteString(string(lineRunes[sliceStartCol:sliceEndCol]))
		}

	} else {
		// Multi-line selection
		// 1. First line part
		firstLineRunes := buffer.GetLineRunes(start.Row)
		firstLineLen := len(firstLineRunes)
		sliceStartCol := max(start.Col, 0)
		if sliceStartCol < firstLineLen { // Only copy if start is within bounds
			contentBuilder.WriteString(string(firstLineRunes[sliceStartCol:]))
		}
		contentBuilder.WriteRune('\n') // Newline after first partial line

		// 2. Intermediate full lines
		for r := start.Row + 1; r < end.Row; r++ {
			contentBuilder.WriteString(string(buffer.GetLineRunes(r)))
			contentBuilder.WriteRune('\n')
		}

		// 3. Last line part
		lastLineRunes := buffer.GetLineRunes(end.Row)
		lastLineLen := len(lastLineRunes)
		// Adjust end column for slicing (make it exclusive upper bound)
		sliceEndCol := min(end.Col+1, lastLineLen)
		if sliceEndCol > 0 { // Only copy if end is not before beginning
			contentBuilder.WriteString(string(lastLineRunes[:sliceEndCol]))
		}
		// For linewise selections every line — including a blank last line — must end with "\n".
		if isLineWise {
			contentBuilder.WriteRune('\n')
		}
	}

	// A characterwise selection ending on a line break ($, or j onto a
	// shorter line) includes it, except on the last line, which has none.
	if isVisual && !isLineWise && end.Col >= buffer.LineRuneCount(end.Row) && end.Row < buffer.LineCount()-1 {
		contentBuilder.WriteRune('\n')
	}

	// Add trailing newline for line-wise single-line operations.
	// This also handles blank-line yanks where content is "" but must become "\n".
	content := contentBuilder.String()
	if isLineWise && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Write to the actual clipboard
	if err := e.clipboard.Write(content); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	if op == cutType {
		return nil
	}

	signal := YankSignal{
		content: content,
	}

	e.DispatchSignal(signal)

	return nil
}
