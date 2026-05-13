package core

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
	for i := len(runes) - 1; i >= 0; i-- {
		if isWordChar(runes[i]) {
			length++
		} else {
			break
		}
	}
	return length
}
