package core

import "time"

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

type copyType int

const (
	yankType copyType = iota
	cutType
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
	SetSearchMode()
	SetBackwardSearchMode()
	DisableVimMode(bool)
	IsVimMode() bool
	DisableCommandMode(bool)
	DisableInsertMode(bool)
	DisableVisualMode(bool)
	DisableVisualLineMode(bool)
	DisableSearchMode(bool)

	// Event handling
	HandleKey(key KeyEvent) *EditorError // Process a key press

	// Key mappings (:map and friends)
	Map(modes MapMode, lhs, rhs string, noremap bool) error
	Unmap(modes MapMode, lhs string) error
	ClearMappings(modes MapMode)
	Mappings(mode MapMode) []Mapping
	SetMapLeader(leader string)
	MapLeader() string
	FlushPendingMapping() *EditorError // Deliver keys held while resolving a mapping, unmapped

	// Mapping timeout ('timeout' and 'timeoutlen'). The core has no timers, so
	// the UI layer drives this: after each key, PendingMapTimeout says whether
	// one should be running, and TimeoutPendingMapping resolves it when it fires.
	PendingMapTimeout() (d time.Duration, token MapToken, ok bool)
	TimeoutPendingMapping(token MapToken) *EditorError
	SetMapTimeout(enabled bool)
	MapTimeoutEnabled() bool
	SetMapTimeoutLen(d time.Duration)
	MapTimeoutLen() time.Duration

	TriggerCompletion(triggerKind CompletionTriggerKind, triggerChar string) CompletionContext
	InsertCompletion(completion Completion) error

	// State Management
	GetState() State      // Get the current editor state
	SetState(State)       // Update the editor state (used internally)
	UpdateStatus(string)  // Helper to set status line
	UpdateCommand(string) // Helper to set command line

	// Command execution (Called from Command Mode)
	ExecuteCommand(cmd string) *EditorError
	ExecuteSearch(query string, searchOptions SearchOptions)
	CancelSearch()

	// History management
	SaveHistory() // End the current undo step; the editor already does after each command
	Undo() (string, error)
	Redo() (string, error)
	UndoLine() (string, error)    // Vim's 'U' — undo all recent changes on one line
	PasteAfter() (string, error)  // Paste from clipboard after/below cursor
	PasteBefore() (string, error) // Paste from clipboard before/above cursor
	Copy(op copyType) error       // Copy to clipboard

	// Viewport scrolling. State.TopLine is the first line in the window; the UI
	// draws from it and supplies line heights so wrapped lines count in rows.
	ScrollViewport()                                     // Bring the cursor line into the window if it left
	ScrollCursorTo(target ScrollTarget)                  // zt / zz / zb
	ScrollHalfPage(down bool, count int) (Cursor, error) // CTRL-D / CTRL-U; returns the moved cursor
	SetLineHeightFunc(f LineHeightFunc)
	GetUpdateSignalChan() <-chan Signal            // For UI updates
	GetSelectionStatus(pos Position) SelectionType // Get selection status of a position
	Save(*string)                                  // Save the current buffer content
	Quit()                                         // Signal to quit the editor
	DispatchError(id ErrorId, err error)           // Dispatch errors to consumers
	DispatchSignal(signal Signal)                  // Dispatch signals to consumers
	ResetPendingCount()

	ShowRelativeLineNumbers(bool)
	IsNormalMode() bool
	IsInsertMode() bool
	IsVisualMode() bool
	IsVisualLineMode() bool
	IsCommandMode() bool
	IsSearchMode() bool

	SearchResults() []Position
	NextSearchResult() Cursor
	PreviousSearchResult() Cursor
	SearchWordUnderCursor(backwards bool) Cursor

	SetMaxHistory(max uint32) // Set how many changes can be undone ('undolevels')

	SetExtraWordChars(chars ...rune) // Set additional characters to be considered part of words for navigation and selection
	IsWordChar(r rune) bool          // Reports whether r is considered a word character in this editor's context

	ResetSelection()
}

type Clipboard interface {
	Write(text string) error
	Read() (string, error)
}
