package core

import (
	"fmt"
)

type Mode string

const (
	NormalMode     Mode = "normal"
	InsertMode     Mode = "insert"
	VisualMode     Mode = "visual"
	VisualLineMode Mode = "visual-line"
	CommandMode    Mode = "command"
	SearchMode     Mode = "search"
)

// EditorMode represents a Vim editing mode
type EditorMode interface {
	Name() Mode
	// HandleKey processes a key press. It can return an error or signal a mode change.
	// It takes the current editor *state* and returns the *new desired state*.
	// This makes state management more explicit.
	HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError
	Enter(editor Editor, buffer Buffer) // Called when entering the mode
	Exit(editor Editor, buffer Buffer)  // Called when exiting the mode

	// AwaitingLiteral reports whether the mode is waiting for a literal
	// character argument — the character after r, f, F, t or T. Key mappings do
	// not apply to it, since it is data rather than a command.
	AwaitingLiteral() bool

	// OperatorPending reports whether an operator (d/y/c) is waiting for a
	// motion, which selects the operator-pending mapping set.
	OperatorPending() bool
}

// baseMode provides the default pending-state answers for modes that have no
// multi-key sequences of their own.
type baseMode struct{}

func (baseMode) AwaitingLiteral() bool { return false }
func (baseMode) OperatorPending() bool { return false }

type VisualModeInterface interface {
	GetCurrentCount() *int
	SetCurrentCount(count *int)
}

// getMoveCount processes numeric key presses to build a command count for visual modes.
// It returns:
// - count: The calculated count (default 1 or accumulated value) if a non-digit key was pressed.
// - processedDigit: true if the key was a digit ('0'-'9') and was consumed, false otherwise.
func getMoveCount(mode VisualModeInterface, editor Editor, key KeyEvent) (count int, processedDigit bool) {
	currentCount := mode.GetCurrentCount()

	// --- Handle Digit Input ---
	if key.Rune >= '1' && key.Rune <= '9' {
		digit := int(key.Rune - '0')
		if currentCount == nil {
			currentCount = new(int) // Allocate memory
		}
		*currentCount = (*currentCount * 10) + digit // Append digit
		mode.SetCurrentCount(currentCount)           //  Set state back
		editor.UpdateCommand(fmt.Sprintf("%d", *currentCount))
		return 0, true
	} else if key.Rune == '0' {
		if currentCount != nil { // Can only append '0' if count already started
			digit := 0
			*currentCount = (*currentCount * 10) + digit // Append digit
			mode.SetCurrentCount(currentCount)
			editor.UpdateCommand(fmt.Sprintf("%d", *currentCount))
			return 0, true
		}
	}

	// --- Key was NOT a digit being accumulated ---
	processedDigit = false // We didn't process a digit here

	// Determine final count to use
	count = 1 // Default count
	if currentCount != nil {
		count = *currentCount
		// Reset the count now that it's being used/finalized
		mode.SetCurrentCount(nil)
		editor.UpdateCommand("") // Clear count display
	}

	// Return the calculated count and indicate no digit was processed *by this part*
	return count, processedDigit
}
