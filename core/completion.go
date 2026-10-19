package core

import (
	"fmt"
	"slices"
	"time"
)

// CompletionTriggerKind defines how completion was triggered
type CompletionTriggerKind int

const (
	// CompletionTriggerManual indicates completion was triggered manually (e.g., Ctrl+Space)
	CompletionTriggerManual CompletionTriggerKind = iota
	// CompletionTriggerAuto indicates completion was auto-triggered while typing
	CompletionTriggerAuto
)

// CompletionContext provides context for completion requests
type CompletionContext struct {
	// Position is the current cursor position
	Position Position

	// CurrentLine is the text of the current line
	CurrentLine string

	// TextBeforeCursor is the text before the cursor on the current line
	TextBeforeCursor string

	// TextAfterCursor is the text after the cursor on the current line
	TextAfterCursor string

	// LinesBefore contains context lines before the cursor
	LinesBefore []string

	// LinesAfter contains context lines after the cursor
	LinesAfter []string

	// Mode is the current editor mode
	Mode Mode

	// RequestID is a unique identifier for this completion request
	RequestID string

	// TriggerKind indicates how the completion was triggered
	TriggerKind CompletionTriggerKind

	// TriggerCharacter is the character that triggered auto-completion (empty for manual)
	TriggerCharacter string
}

// Completion represents a single completion item
type Completion struct {
	// Text is the text to insert when this completion is selected
	Text string

	// Label is the display label (can differ from Text, e.g., with type annotations)
	Label string

	// Description is longer documentation or details about the completion
	Description string

	// Type is the category of completion (e.g., "keyword", "function", "table", "column")
	Type string

	// Score is used for ranking completions (higher = better match)
	Score float64

	// Meta contains additional extensible metadata for custom use
	Meta map[string]any
}

// findWordLengthBeforeCursor returns the number of word characters immediately
// before the cursor — i.e. the length of the partial word currently being typed.
// This is used by InsertCompletion to determine how much to delete before
// inserting the selected completion.
func findWordLengthBeforeCursor(textBeforeCursor string, isWordChar func(rune) bool) int {
	runes := []rune(textBeforeCursor)
	length := 0
	for _, r := range slices.Backward(runes) {
		if isWordChar(r) {
			length++
		} else {
			break
		}
	}
	return length
}

// TriggerCompletion requests completions at the current cursor position.
// It returns the context that was dispatched so callers can track the RequestID.
func (e *editor) TriggerCompletion(triggerKind CompletionTriggerKind, triggerChar string) CompletionContext {
	ctx := e.buildCompletionContext(triggerKind, triggerChar)
	e.DispatchSignal(CompletionRequestSignal{context: ctx})
	return ctx
}

// InsertCompletion inserts the selected completion into the buffer, replacing
// the entire word under the cursor (both the fragment before and after it).
func (e *editor) InsertCompletion(completion Completion) error {
	cursor := e.buffer.GetCursor()
	pos := cursor.Position

	var lineRunes []rune
	if pos.Row >= 0 && pos.Row < e.buffer.LineCount() {
		lineRunes = e.buffer.GetLineRunes(pos.Row)
	}

	// Count word chars before cursor (the fragment already typed).
	var textBeforeCursor string
	if pos.Col >= 0 && pos.Col <= len(lineRunes) {
		textBeforeCursor = string(lineRunes[:pos.Col])
	}
	wordBefore := findWordLengthBeforeCursor(textBeforeCursor, e.IsWordChar)

	// Count word chars after cursor (the remainder of the same word).
	wordAfter := 0
	for _, r := range lineRunes[pos.Col:] {
		if e.IsWordChar(r) {
			wordAfter++
		} else {
			break
		}
	}

	// Delete the word fragment after the cursor first (col doesn't change).
	if wordAfter > 0 {
		if err := e.buffer.DeleteRunesAt(pos.Row, pos.Col, wordAfter); err != nil {
			return err
		}
	}

	// Delete the word fragment before the cursor.
	if wordBefore > 0 {
		deleteStart := pos.Col - wordBefore
		if deleteStart >= 0 {
			if err := e.buffer.DeleteRunesAt(pos.Row, deleteStart, wordBefore); err != nil {
				return err
			}
			cursor.Position.Col = deleteStart
		}
	}

	// Insert completion text.
	if err := e.buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune(completion.Text)); err != nil {
		return err
	}

	// Move cursor to end of inserted text.
	if err := cursor.MoveRight(e.buffer, len([]rune(completion.Text))); err != nil {
		return err
	}

	e.buffer.SetCursor(cursor)
	e.recordCompletion(wordBefore, completion.Text)
	e.trackInsertCursor()

	return nil
}

// buildCompletionContext creates a CompletionContext from current editor state
func (e *editor) buildCompletionContext(triggerKind CompletionTriggerKind, triggerChar string) CompletionContext {
	cursor := e.buffer.GetCursor()
	pos := cursor.Position

	currentLine := ""
	textBefore := ""
	textAfter := ""

	if pos.Row >= 0 && pos.Row < e.buffer.LineCount() {
		lineRunes := e.buffer.GetLineRunes(pos.Row)
		currentLine = string(lineRunes)

		if pos.Col >= 0 && pos.Col <= len(lineRunes) {
			textBefore = string(lineRunes[:pos.Col])
			after := lineRunes[pos.Col:]
			// Strip trailing newline so TextAfterCursor never includes it
			if len(after) > 0 && after[len(after)-1] == '\n' {
				after = after[:len(after)-1]
			}
			textAfter = string(after)
		}
	}

	// Gather surrounding lines for context (5 lines before/after)
	linesBefore := []string{}
	linesAfter := []string{}
	const contextLines = 5

	for i := max(0, pos.Row-contextLines); i < pos.Row; i++ {
		linesBefore = append(linesBefore, string(e.buffer.GetLineRunes(i)))
	}

	for i := pos.Row + 1; i < min(e.buffer.LineCount(), pos.Row+contextLines+1); i++ {
		linesAfter = append(linesAfter, string(e.buffer.GetLineRunes(i)))
	}

	// Generate unique request ID
	requestID := fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), pos.Row, pos.Col)

	return CompletionContext{
		Position:         pos,
		CurrentLine:      currentLine,
		TextBeforeCursor: textBefore,
		TextAfterCursor:  textAfter,
		LinesBefore:      linesBefore,
		LinesAfter:       linesAfter,
		Mode:             e.state.Mode,
		RequestID:        requestID,
		TriggerKind:      triggerKind,
		TriggerCharacter: triggerChar,
	}
}
