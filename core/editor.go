package core

import (
	"errors"
	"fmt"
	"time"
)

// Position represents a specific location in the text buffer
type Position struct {
	Row int // Zero-indexed row (line number)
	Col int // Zero-indexed column (character position in the line)
}

// TextEdit replaces the text from Start up to End with Text. As in Position,
// Row is the line and Col counts runes within it. Undo, redo and 'U' report
// what they changed as a list of TextEdits: apply them in order, each one's
// positions referring to the text left by the edit before it. That ordering
// matches LSP's didChange, but LSP counts columns in UTF-16 code units by
// default, so convert the columns before sending edits to a language server.
type TextEdit struct {
	Start, End Position
	Text       string
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

	// Whether h, l and the arrow keys cross line boundaries, as Vim's
	// ':set whichwrap+=<,>,[,],h,l'. Off by default, matching Vim, where only
	// <BS> and <Space> do.
	SetMoveAcrossLines(enabled bool)
	MoveAcrossLines() bool

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

	// History management. Undo, Redo and UndoLine return the edits they applied.
	Undo() ([]TextEdit, error)
	Redo() ([]TextEdit, error)
	UndoLine() ([]TextEdit, error) // Vim's 'U' — undo all recent changes on one line
	PasteAfter() (string, error)   // Paste from clipboard after/below cursor
	PasteBefore() (string, error)  // Paste from clipboard before/above cursor
	Copy(op copyType) error        // Copy to clipboard

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

// Concrete implementation of Editor
type editor struct {
	buffer      *recordingBuffer // records every edit into the open undo step
	currentMode EditorMode
	modes       map[Mode]EditorMode
	state       State

	// Undo history (see undo.go). undoSteps[:undoPos] are applied, the rest
	// can be redone. openStep collects the edits of the command in progress
	// and is nil between commands. maxHistory is Vim's 'undolevels'.
	undoSteps  []undoStep
	undoPos    int
	openStep   *undoStep
	maxHistory uint32

	// Line-undo ('U') tracking. Vim's U restores the line where the latest
	// changes were made, undoing *all* of them at once. A "run" is the trailing
	// sequence of undo steps whose changes all happened on the same row;
	// undoLineStart is the undoPos immediately before that run began.
	// undoLineRow is -1 when no run is active.
	undoLineRow   int
	undoLineStart int

	clipboard    Clipboard // Clipboard interface for copy/paste
	updateSignal chan Signal

	// Key mappings (:map and friends).
	mappings       *mappingTable
	mapLeader      string
	pendingMapKeys []KeyEvent // keys held while a longer mapping might still match
	mapDepth       int        // expansion depth, guarding against recursive mappings

	// 'timeout' and 'timeoutlen'. mapPendingGen identifies the current run of
	// held keys so a timer that fires after the run is over can be discarded.
	mapTimeout    bool
	mapTimeoutLen time.Duration
	mapPendingGen MapToken

	// moveAcrossLines lets h, l and the arrow keys cross line boundaries.
	moveAcrossLines bool

	// Scrolling (see scroll.go). scrollLines is Vim's 'scroll' option as set by
	// a count on CTRL-D/CTRL-U, valid while the window is scrollLinesHeight tall.
	lineHeightFunc    LineHeightFunc
	scrollLines       int
	scrollLinesHeight int

	// The last f/F/t/T search, shared by normal, visual and visual-line mode
	// so ; and , still repeat it after a mode switch, as in Vim.
	charSearch lastCharSearch

	// Repeat ('.', see repeat.go). editCount counts recorded edits, so a
	// command can tell whether it changed the text.
	repeat    repeatRecord
	recording commandRecording
	replaying bool
	editCount int
	insert    insertSession
}

// New creates a new editor instance
func New(clipboard Clipboard) Editor {
	e := &editor{
		modes:         make(map[Mode]EditorMode),
		state:         InitialState(), // Use initial state function
		maxHistory:    1000,           // Vim's default 'undolevels'
		clipboard:     clipboard,
		updateSignal:  make(chan Signal, 100), // Buffered channel for updates
		undoLineRow:   -1,
		mappings:      newMappingTable(),
		mapLeader:     DefaultMapLeader,
		mapTimeout:    true,
		mapTimeoutLen: DefaultMapTimeoutLen,
	}
	e.buffer = &recordingBuffer{Buffer: NewBuffer(), e: e}

	// Register modes (pass editor instance if modes need it during init)
	e.modes[NormalMode] = newNormalMode(&e.charSearch)
	e.modes[InsertMode] = NewInsertMode()
	e.modes[VisualMode] = newVisualMode(&e.charSearch)
	e.modes[VisualLineMode] = newVisualLineMode(&e.charSearch)
	e.modes[CommandMode] = NewCommandMode()
	e.modes[SearchMode] = NewSearchMode()

	// Set initial mode
	initialModeName := e.state.Mode
	e.currentMode = e.modes[initialModeName]
	if e.currentMode != nil {
		e.currentMode.Enter(e, e.buffer) // Pass buffer to Enter
	} else {
		// Fallback or error if initial mode not found
		e.currentMode = e.modes[NormalMode]
		e.state.Mode = NormalMode
		e.currentMode.Enter(e, e.buffer)
	}

	return e
}

// SetMaxHistory sets how many changes can be undone, like Vim's
// 'undolevels'. Default is 1000.
func (e *editor) SetMaxHistory(max uint32) {
	e.maxHistory = max
}

func (e *editor) DisableVimMode(disable bool) {
	e.state.VimMode = !disable
	if disable {
		e.SetInsertMode()
		e.ShowRelativeLineNumbers(false) // Disable relative numbers in non-Vim mode
	} else {
		e.SetNormalMode()
	}
}

func (e *editor) IsVimMode() bool {
	return e.state.VimMode
}

func (e *editor) DisableCommandMode(disable bool) {
	e.state.WithCommandMode = !disable
}

func (e *editor) DisableInsertMode(disable bool) {
	e.state.WithInsertMode = !disable
}

func (e *editor) DisableVisualMode(disable bool) {
	e.state.WithVisualMode = !disable
}

func (e *editor) DisableVisualLineMode(disable bool) {
	e.state.WithVisualLineMode = !disable
}

func (e *editor) DisableSearchMode(disable bool) {
	e.state.WithSearchMode = !disable
}

func (e *editor) ShowRelativeLineNumbers(show bool) {
	e.state.RelativeNumbers = show
}

func (e *editor) setMode(modeName Mode) {
	newMode := e.modes[modeName]

	// Switching to the current mode is a no-op: re-running Exit/Enter would
	// e.g. step the cursor back in insert or reset the visual anchor.
	if newMode == e.currentMode {
		return
	}

	if e.currentMode != nil {
		e.currentMode.Exit(e, e.buffer) // Pass buffer to Exit
	}

	e.state.PreviousMode = e.state.Mode
	e.currentMode = newMode
	e.state.Mode = modeName          // Update state string
	e.currentMode.Enter(e, e.buffer) // Pass buffer to Enter
	if modeName == InsertMode {
		e.resetInsert()
	}
}

func (e *editor) SetNormalMode() {
	e.setMode(NormalMode)
}

func (e *editor) SetInsertMode() {
	if !e.state.WithInsertMode {
		return
	}

	e.setMode(InsertMode)
}

func (e *editor) SetVisualMode() {
	if !e.state.WithVisualMode {
		return
	}

	e.setMode(VisualMode)
}

func (e *editor) SetVisualLineMode() {
	if !e.state.WithVisualLineMode {
		return
	}

	e.setMode(VisualLineMode)
}

func (e *editor) SetCommandMode() {
	if !e.state.WithCommandMode {
		return
	}

	e.setMode(CommandMode)
}

func (e *editor) SetSearchMode() {
	if !e.state.WithSearchMode {
		return
	}

	e.state.SearchOptions.Backwards = false
	e.setMode(SearchMode)
}

func (e *editor) SetBackwardSearchMode() {
	if !e.state.WithSearchMode {
		return
	}

	e.state.SearchOptions.Backwards = true
	e.setMode(SearchMode)
}

func (e *editor) GetBuffer() Buffer {
	return e.buffer
}

func (e *editor) SetBuffer(buffer Buffer) {
	// GetBuffer hands out the recording wrapper; never wrap it twice.
	if rb, ok := buffer.(*recordingBuffer); ok {
		buffer = rb.Buffer
	}
	e.buffer = &recordingBuffer{Buffer: buffer, e: e}
	e.resetHistory()                                      // A new buffer starts a new undo history
	e.resetRepeat()                                       // and forgets the change '.' repeats
	e.UpdateStatus(fmt.Sprintf("-- %s --", e.state.Mode)) // Update status
	e.state.TopLine = 0                                   // A new buffer starts at the top
	e.ScrollViewport()
}

func (e *editor) SetContent(content []byte) {
	e.SetBuffer(NewBufferFromBytes(content))
}

func (e *editor) GetMode() EditorMode {
	return e.currentMode
}

func (e *editor) GetUpdateSignalChan() <-chan Signal {
	return e.updateSignal // Return the read-only channel
}

func (e *editor) HandleKey(key KeyEvent) *EditorError {
	if e.currentMode == nil {
		return &EditorError{
			id:  ErrInvalidModeId,
			err: errors.New("no current mode set"),
		}
	}

	// Resolve key mappings before the mode sees anything. Mappings substitute
	// key sequences and re-feed the result through this same path, so they
	// compose with operators and motions without the mapping layer needing to
	// know anything about them.
	return e.handleKeyMapped(normalizeKey(key), true)
}

func (e *editor) GetState() State {
	// Ensure cursor pos in state reflects buffer (optional, state doesn't store it now)
	return e.state
}

// SetState allows internal updates (e.g., from modes)
func (e *editor) SetState(state State) {
	e.state = state
}

// UpdateStatus is a helper for modes to update the status line
func (e *editor) UpdateStatus(status string) {
	e.state.StatusLine = status
}

// UpdateCommand is a helper for modes to update the command line
func (e *editor) UpdateCommand(cmd string) {
	if cmd != "" {
		e.DispatchSignal(CommandSignal{})
	}
	e.state.CommandLine = cmd
}

func (e *editor) IsSelected() bool {
	return e.state.VisualStart.Row != -1
}

func (e *editor) GetSelectionStatus(pos Position) SelectionType {
	state := e.GetState() // Get current state
	buffer := e.GetBuffer()
	cursor := buffer.GetCursor()

	if state.VisualStart.Row == -1 { // No selection active
		return SelectionNone
	}

	// Normalize selection range using the accessible function
	selStart, selEnd := NormalizeSelection(state.VisualStart, cursor.Position)

	// Check if this is line-wise selection (either visual-line mode or yank line selection)
	isLineWise := state.Mode == VisualLineMode || state.YankSelection == SelectionLine
	if isLineWise {
		if pos.Row >= selStart.Row && pos.Row <= selEnd.Row {
			return SelectionLine
		}
		// Note: In line-wise mode, the *entire line* is considered selected for styling purposes.
		return SelectionNone // Position's row is not within the selected lines
	}

	// Check Character Mode (if not visual line)
	// This is the detailed logic from main.go's original inCharSelection check
	inCharSelection := (pos.Row > selStart.Row && pos.Row < selEnd.Row) || // Full intermediate lines
		(pos.Row == selStart.Row && pos.Row == selEnd.Row && pos.Col >= selStart.Col && pos.Col <= selEnd.Col) || // Single line selection range
		(pos.Row == selStart.Row && pos.Row != selEnd.Row && pos.Col >= selStart.Col) || // First line partial selection
		(pos.Row == selEnd.Row && pos.Row != selStart.Row && pos.Col <= selEnd.Col) // Last line partial selection

	if inCharSelection {
		return SelectionCharacter
	}

	return SelectionNone
}

func (e *editor) Save(path *string) {
	e.buffer.SaveContent()
	signal := SaveSignal{path: path, content: e.buffer.GetSavedContent()}
	e.DispatchSignal(signal)
}

func (e *editor) Quit() {
	e.state.Quit = true
	e.DispatchSignal(QuitSignal{})
}

func (e *editor) SetExtraWordChars(chars ...rune) {
	e.state.isWordCharFunc = createIsWordCharClassifier(chars)
}

func (e *editor) IsWordChar(r rune) bool {
	return e.state.IsWordChar(r)
}

func (e *editor) ResetPendingCount() {
	if e.state.PendingCount != nil {
		e.state.PendingCount = nil
		e.UpdateCommand("")
	}
}

func (e *editor) IsNormalMode() bool {
	return e.state.Mode == NormalMode
}

func (e *editor) IsInsertMode() bool {
	return e.state.Mode == InsertMode
}

func (e *editor) IsVisualMode() bool {
	return e.state.Mode == VisualMode
}

func (e *editor) IsVisualLineMode() bool {
	return e.state.Mode == VisualLineMode
}

func (e *editor) IsCommandMode() bool {
	return e.state.Mode == CommandMode
}

func (e *editor) IsSearchMode() bool {
	return e.state.Mode == SearchMode
}

func (e *editor) ResetSelection() {
	state := e.GetState()
	state.VisualStart = Position{Row: -1, Col: -1}
	state.YankSelection = SelectionNone
	e.SetState(state)
}
