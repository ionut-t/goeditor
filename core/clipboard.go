package core

import (
	"errors"
	"fmt"
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
	content, err := e.readClipboard()
	if err != nil {
		return "", err
	}

	cursor := e.buffer.GetCursor()

	if before, ok := strings.CutSuffix(content, "\n"); ok {
		// Linewise paste: insert the content as a new line below the current line,
		// regardless of the cursor column — matching Vim's 'p' behaviour for line-wise yanks.
		// Detected via trailing newline, which all line-wise yanks (yy, Vy) append.
		lineText := before
		lineLen := e.buffer.LineRuneCount(cursor.Position.Row)
		_ = e.buffer.InsertRunesAt(cursor.Position.Row, lineLen, []rune("\n"+lineText))

		// Place cursor at the start of the newly inserted line.
		cursor.Position.Row++
		cursor.Position.Col = 0
		e.buffer.SetCursor(cursor)
	} else {
		// Character-wise paste: insert AFTER the cursor char — matching Vim's 'p' behaviour.
		// On an empty line Col+1 would exceed the line length, so clamp to lineLen.
		lineLen := e.buffer.LineRuneCount(cursor.Position.Row)
		insertCol := min(cursor.Position.Col+1, lineLen)
		_ = e.buffer.InsertRunesAt(cursor.Position.Row, insertCol, []rune(content))
	}

	return content, nil
}

func (e *editor) PasteBefore() (string, error) {
	content, err := e.readClipboard()
	if err != nil {
		return "", err
	}

	cursor := e.buffer.GetCursor()

	if before, ok := strings.CutSuffix(content, "\n"); ok {
		// Linewise paste above: insert the yanked line before the current line.
		// Inserting lineText+"\n" at (row, 0) pushes the current line down; cursor stays at row.
		_ = e.buffer.InsertRunesAt(cursor.Position.Row, 0, []rune(before+"\n"))
		cursor.Position.Col = 0
		e.buffer.SetCursor(cursor)
	} else {
		// Character-wise paste before: insert at the current cursor position (same as 'p' for chars).
		_ = e.buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune(content))
	}

	return content, nil
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
		if state.Mode == "visual-line" || state.YankSelection == SelectionLine {
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

	// Add trailing newline for line-wise single-line operations.
	// This also handles blank-line yanks where content is "" but must become "\n".
	content := contentBuilder.String()
	if isLineWise && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	// Write to the actual clipboard
	if err := e.clipboard.Write(content); err != nil {
		errMsg := fmt.Sprintf("failed to copy to clipboard: %v", err)
		return errors.New(errMsg)
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
