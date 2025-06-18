package core

// Position represents a specific location in the text buffer
type Position struct {
	Row int // Zero-indexed row (line number)
	Col int // Zero-indexed column (character position in the line)
}

// SelectionType indicates the selection status of a position
type SelectionType int

const (
	SelectionNone      SelectionType = iota // Position is not selected
	SelectionCharacter                      // Position is part of a character-wise visual selection
	SelectionLine                           // Position is part of a line-wise visual selection
)

// Editor represents the main editor interface
type Editor interface {
	// Buffer manipulation
	GetBuffer() Buffer
	SetBuffer(Buffer)  // Replace the current buffer
	SetContent([]byte) // Set buffer content from byte slice

	// Mode handling
	GetMode() EditorMode
	SetNormalMode()
	SetInsertMode()
	SetVisualMode()
	SetVisualLineMode()
	SetCommandMode()
	DisableVimMode(bool)
	IsVimMode() bool
	DisableCommandMode(bool)
	HasCommandMode() bool

	// Event handling
	HandleKey(key KeyEvent) error // Process a key press

	// State Management
	GetState() State      // Get the current editor state
	SetState(State)       // Update the editor state (used internally)
	UpdateStatus(string)  // Helper to set status line
	UpdateCommand(string) // Helper to set command line

	// Command execution (Called from Command Mode)
	ExecuteCommand(cmd string) error

	// History management
	SaveHistory() // Indicate a state should be saved for undo
	Undo() error
	Redo() error
	Paste() (int, error) // Paste from clipboard
	Copy() error         // Copy to clipboard

	// Viewport scrolling (Could be part of UpdateState or separate)
	ScrollViewport()
	GetUpdateSignalChan() <-chan Signal            // For UI updates
	GetSelectionStatus(pos Position) SelectionType // Get selection status of a position
	Save()                                         // Save the current buffer content
	Quit()                                         // Signal to quit the editor
	DispatchError(id ErrorId, err error)           // Dispatch errors to consumers
	DispatchMessage(args ...string)                // Dispatch (success) messages to consumers
	DispatchSignal(signal Signal)                  // Dispatch signals to consumers
	ResetPendingCount()

	ShowRelativeLineNumbers(bool)
	IsNormalMode() bool
	IsInsertMode() bool
	IsVisualMode() bool
	IsVisualLineMode() bool
	IsCommandMode() bool
}

type Clipboard interface {
	Write(text string) error
	Read() (string, error)
}
