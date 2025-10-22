package core

import (
	"errors"
	"fmt"
	"strings"
)

type SearchQuery struct {
	Pattern string
	Term    string
}

// State represents the complete current state of the editor (Refined)
type State struct {
	Mode         Mode   // Current editing mode (Normal, Insert, Visual, Command)
	PreviousMode Mode   // Previous editing mode
	StatusLine   string // Content of the status line (bottom line)
	CommandLine  string // Current command being typed or message to display
	Quit         bool   // Flag indicating if the editor should exit

	// Viewport information
	TopLine        int // First line visible in the viewport (0-indexed)
	ViewportHeight int // Number of lines that can be displayed
	ViewportWidth  int // Number of columns that can be displayed

	// Visual mode
	VisualStart Position // Starting position for visual selection (Use Position{-1,-1} if not active)

	// Command handling
	SearchQuery       SearchQuery // Current search query (for Search command)
	SearchOptions     SearchOptions
	SearchResults     []Position // List of positions for search results
	SearchResultIndex int        // Current index in the search results
	PendingCount      *int       // For handling numeric prefixes to commands (e.g., "5j") - Managed in normalMode

	// Error/Message Display
	Message string // Temporary message to display

	// UI Options
	RelativeNumbers bool // Flag for relative line numbers

	VimMode bool

	AvailableWidth int // Width available for text rendering

	WithCommandMode bool // Whether command mode is enabled

	WithInsertMode bool // Whether insert mode is enabled

	WithVisualMode bool // Whether visual mode is enabled

	WithVisualLineMode bool // Whether visual line mode is enabled

	WithSearchMode bool // Whether search mode is enabled
}

// InitialState creates a default state
func InitialState() State {
	return State{
		Mode:              "normal",
		PreviousMode:      "normal",
		StatusLine:        "-- NORMAL --",
		CommandLine:       "",
		TopLine:           0,
		ViewportHeight:    24,
		ViewportWidth:     80,
		VisualStart:       Position{-1, -1},
		SearchQuery:       SearchQuery{},
		SearchResults:     []Position{},
		SearchResultIndex: -1,
		PendingCount:      nil,
		Message:           "",
		RelativeNumbers:   false, // Default to absolute numbers
		Quit:              false,
		VimMode:           true,

		WithCommandMode:    true,
		WithInsertMode:     true,
		WithVisualMode:     true,
		WithVisualLineMode: true,
	}
}

// Concrete implementation of Editor
type editor struct {
	buffer      Buffer
	currentMode EditorMode
	modes       map[Mode]EditorMode
	state       State

	// IMPROVEMENT: Use a more efficient history mechanism (diffs, ring buffer)
	history       []string // Store snapshots of buffer content as strings
	cursorHistory []Cursor // Store cursor states corresponding to history
	historyPos    int      // Current position in the history (-1 = initial state)
	maxHistory    uint32   // Max number of history entries

	clipboard    Clipboard // Clipboard interface for copy/paste
	updateSignal chan Signal
}

// New creates a new editor instance
func New(clipboard Clipboard) Editor {
	e := &editor{
		buffer:        NewBuffer(),
		modes:         make(map[Mode]EditorMode),
		state:         InitialState(), // Use initial state function
		history:       []string{},     // Initialize history
		cursorHistory: []Cursor{},     // Initialize cursor history
		historyPos:    -1,             // Start before the first save
		maxHistory:    1000,           // Default history size
		clipboard:     clipboard,
		updateSignal:  make(chan Signal, 100), // Buffered channel for updates
	}

	// Register modes (pass editor instance if modes need it during init)
	e.modes[NormalMode] = NewNormalMode()
	e.modes[InsertMode] = NewInsertMode()
	e.modes[VisualMode] = NewVisualMode()
	e.modes[VisualLineMode] = NewVisualLineMode()
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

	// Save the initial state as the first history entry
	e.SaveHistory()

	return e
}

// SetMaxHistory allows setting the maximum number of history entries.
// Default is 1000.
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

	if e.currentMode != nil {
		e.currentMode.Exit(e, e.buffer) // Pass buffer to Exit
	}

	e.state.PreviousMode = e.state.Mode
	e.currentMode = newMode
	e.state.Mode = modeName          // Update state string
	e.currentMode.Enter(e, e.buffer) // Pass buffer to Enter
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

	e.setMode(SearchMode)
}

func (e *editor) GetBuffer() Buffer {
	return e.buffer
}

func (e *editor) SetBuffer(buffer Buffer) {
	e.buffer = buffer
	// Reset history when buffer changes completely
	e.history = []string{}
	e.cursorHistory = []Cursor{}
	e.historyPos = -1
	e.SaveHistory()                                       // Save the new buffer's initial state
	e.UpdateStatus(fmt.Sprintf("-- %s --", e.state.Mode)) // Update status
	e.ScrollViewport()                                    // Adjust viewport for new buffer
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

	// Let the current mode handle the key
	err := e.currentMode.HandleKey(e, e.buffer, key)

	// Update derived state AFTER handling key
	e.ScrollViewport() // Ensure cursor is visible after potential movement

	return err
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

// ExecuteCommand executes a command string (typically entered in command mode)
func (e *editor) ExecuteCommand(cmd string) *EditorError {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	parts := strings.Fields(cmd)
	command := parts[0]
	args := parts[1:]

	// TODO: Add range parsing (e.g., :%s/foo/bar/g)

	switch command {
	case "q", "quit":
		if e.buffer.IsModified() {
			return &EditorError{
				id:  ErrUnsavedChangesId,
				err: ErrUnsavedChanges,
			}
		}
		e.state.Quit = true
		e.Quit()
		return nil

	case "q!", "quit!":
		e.state.Quit = true
		e.Quit()
		return nil

	case "w", "write":
		// If a path is provided, use it; else nil to indicate current file
		// TODO: Improve file handling
		if len(args) > 0 {
			path := args[0]
			e.Save(&path)
		} else {
			if !e.buffer.IsModified() {
				return &EditorError{
					id:  ErrNoChangesToSaveId,
					err: ErrNoChangesToSave,
				}
			}

			e.Save(nil)
		}

		return nil

	case "wq":
		// Placeholder: write then quit
		err := e.ExecuteCommand("w")
		if err != nil {
			return err // Error during write
		}
		return e.ExecuteCommand("q") // Attempt quit

	case "wq!":
		err := e.ExecuteCommand("w")
		if err != nil {
			return err
		}
		return e.ExecuteCommand("q!") // Force quit

		// Add more commands: e, edit, r, read, s, substitute etc.
		// case "s": return e.executeSubstitute(args)

	case "set": // Handle basic set commands
		if len(args) == 1 {
			switch args[0] {
			case "relativenumber", "rnu":
				e.state.RelativeNumbers = true
				e.DispatchSignal(RelativeNumbersSignal{enabled: true})
				return nil
			case "norelativenumber", "nornu":
				e.state.RelativeNumbers = false
				e.DispatchSignal(RelativeNumbersSignal{enabled: false})
				return nil
				// Add 'number'/'nonu' later if needed
			}
		}
		return &EditorError{
			id:  ErrInvalidCommandId,
			err: ErrInvalidCommand,
		}

	case "rename":
		if len(args) != 1 {
			return &EditorError{
				id:  ErrRenameFailedId,
				err: ErrRenameFailed,
			}
		}

		e.DispatchSignal(RenameSignal{
			fileName: args[0],
		})

		return nil

	case "delete", "del":
		e.DispatchSignal(DeleteFileSignal{})
		return nil

	default:
		// Handle line number navigation (e.g., ":10")
		lineNum := -1
		_, scanErr := fmt.Sscan(command, &lineNum)
		if scanErr == nil && lineNum > 0 {
			targetRow := lineNum - 1 // User enters 1-based, we use 0-based
			cursor := e.buffer.GetCursor()
			// Clamp targetRow
			if targetRow >= e.buffer.LineCount() {
				targetRow = e.buffer.LineCount() - 1
			}
			if targetRow < 0 {
				targetRow = 0
			}

			cursor.Position.Row = targetRow
			cursor.Position.Col = 0 // Move to start of that line
			// Try moving to first non-blank instead?
			// cursor.MoveToFirstNonBlank(e.buffer)
			e.buffer.SetCursor(cursor)
			e.ScrollViewport() // Ensure jumped-to line is visible
			return nil
		}
		return &EditorError{
			id:  ErrInvalidCommandId,
			err: ErrInvalidCommand,
		}
	}
}

func (e *editor) ExecuteSearch(pattern string, searchOptions SearchOptions) {
	e.state.SearchQuery.Pattern = pattern
	query := pattern

	ignoreCase := searchOptions.IgnoreCase
	smartCase := searchOptions.SmartCase

	if strings.HasSuffix(pattern, "\\c") {
		pattern = strings.TrimSuffix(pattern, "\\c")
		ignoreCase = true
		smartCase = false
		query = strings.TrimRight(pattern, "\\c")
	} else if strings.HasSuffix(pattern, "\\C") {
		pattern = strings.TrimSuffix(pattern, "\\C")
		ignoreCase = false
		searchOptions.SmartCase = true
		query = strings.TrimRight(pattern, "\\C")
	}

	e.state.SearchQuery.Term = query
	e.state.SearchOptions = SearchOptions{
		IgnoreCase: ignoreCase,
		SmartCase:  smartCase,
		Backwards:  searchOptions.Backwards,
		Wrap:       searchOptions.Wrap,
	}

	// Find the first result
	pos, found := e.buffer.Find(query, e.buffer.GetCursor().Position, e.state.SearchOptions)

	if !found && e.state.SearchOptions.Wrap {
		pos, found = e.buffer.Find(query, Position{Row: 0, Col: 0}, e.state.SearchOptions)
	}

	if found {
		e.state.SearchResults = []Position{pos}
		e.state.SearchResultIndex = 0
		cursor := e.buffer.GetCursor()
		cursor.Position = pos
		e.buffer.SetCursor(cursor)
	} else {
		e.state.SearchResults = []Position{}
		e.state.SearchResultIndex = -1
	}

	e.UpdateCommand("/" + e.state.SearchQuery.Pattern)
	e.setMode(e.state.PreviousMode)
	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})
}

func (e *editor) CancelSearch() {
	e.state.SearchQuery = SearchQuery{}
	e.state.SearchResults = []Position{}
	e.setMode(e.state.PreviousMode)
}

func (e *editor) NextSearchResult() Cursor {
	if len(e.state.SearchResults) == 0 {
		return e.buffer.GetCursor()
	}

	options := e.state.SearchOptions
	options.Backwards = false

	currentPos := e.buffer.GetCursor().Position
	pos, found := e.buffer.Find(e.state.SearchQuery.Term, currentPos, options)

	// If not found and wrap is enabled, search from beginning
	if !found && options.Wrap {
		pos, found = e.buffer.Find(e.state.SearchQuery.Term, Position{Row: 0, Col: 0}, options)
	}

	if found {
		e.onSearchResultFound(pos)
		e.ScrollViewport()
	}

	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})

	return e.buffer.GetCursor()
}

func (e *editor) PreviousSearchResult() Cursor {
	if len(e.state.SearchResults) == 0 {
		return e.buffer.GetCursor()
	}

	// Create backward search options
	options := e.state.SearchOptions
	options.Backwards = true

	currentPos := e.buffer.GetCursor().Position
	pos, found := e.buffer.Find(e.state.SearchQuery.Term, currentPos, options)

	// If not found and wrap is enabled, search from end
	if !found && e.state.SearchOptions.Wrap && e.buffer.LineCount() > 0 {
		lastLine := e.buffer.LineCount() - 1
		lastLineLen := e.buffer.LineRuneCount(lastLine)
		pos, found = e.buffer.Find(e.state.SearchQuery.Term, Position{Row: lastLine, Col: lastLineLen}, options)
	}

	if found {
		e.onSearchResultFound(pos)
		e.ScrollViewport()
	}

	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})

	return e.buffer.GetCursor()
}

func (e *editor) SearchResults() []Position {
	return e.state.SearchResults
}

func (e *editor) onSearchResultFound(pos Position) {
	e.state.SearchResults = []Position{pos}
	e.state.SearchResultIndex = 0
	cursor := e.buffer.GetCursor()
	cursor.Position = pos
	e.buffer.SetCursor(cursor)
}

// ScrollViewport ensures the cursor is within the visible area
func (e *editor) ScrollViewport() {
	cursor := e.buffer.GetCursor()
	row := cursor.Position.Row

	if row < e.state.TopLine {
		e.state.TopLine = row
	} else if row >= e.state.TopLine+e.state.ViewportHeight {
		// Scroll down so cursor is on the last line of the viewport
		e.state.TopLine = row - e.state.ViewportHeight + 1
	}

	// Ensure TopLine doesn't go below 0
	if e.state.TopLine < 0 {
		e.state.TopLine = 0
	}
}

// --- History Management (Simple Snapshot Implementation) ---
func (e *editor) SaveHistory() {
	currentState := e.buffer.GetCurrentContent()
	currentCursor := e.buffer.GetCursor()

	// If we used Undo, truncate the future history
	if e.historyPos < len(e.history)-1 {
		e.history = e.history[:e.historyPos+1]
		e.cursorHistory = e.cursorHistory[:e.historyPos+1]
	}

	// Avoid saving duplicate state if no changes occurred
	if len(e.history) > 0 && e.historyPos >= 0 && e.historyPos < len(e.history) {
		if e.history[e.historyPos] == currentState {
			// Even if content is the same, update cursor position if it changed
			if e.historyPos < len(e.cursorHistory) {
				savedCursor := e.cursorHistory[e.historyPos]
				if savedCursor.Position.Row != currentCursor.Position.Row ||
					savedCursor.Position.Col != currentCursor.Position.Col {
					e.cursorHistory[e.historyPos] = currentCursor
				}
			}
			return
		}
	}

	// Add the new state
	e.history = append(e.history, currentState)
	e.cursorHistory = append(e.cursorHistory, currentCursor)
	e.historyPos = len(e.history) - 1

	maxHistory := int(e.maxHistory)

	// Limit history size
	if len(e.history) > maxHistory {
		// Remove the oldest entry
		e.history = e.history[len(e.history)-maxHistory:]
		e.cursorHistory = e.cursorHistory[len(e.cursorHistory)-maxHistory:]
		e.historyPos = len(e.history) - 1
	}
}

func (e *editor) Undo() (string, error) {
	if e.historyPos <= 0 {
		return "", errors.New("already at oldest change")
	}

	e.historyPos--
	prevStateContent := e.history[e.historyPos]
	prevCursor := e.cursorHistory[e.historyPos]

	currentStateContent := e.buffer.GetCurrentContent()

	if prevStateContent == "" {
		prevStateContent = "\n"
	}

	e.buffer.SetContent([]byte(prevStateContent))
	e.buffer.SetCursor(prevCursor)

	e.ScrollViewport()

	return currentStateContent, nil
}

func (e *editor) Redo() (string, error) {
	if e.historyPos >= len(e.history)-1 {
		return "", errors.New("already at newest change")
	}

	currentContent := e.buffer.GetCurrentContent()

	e.historyPos++
	nextStateContent := e.history[e.historyPos]
	nextCursor := e.cursorHistory[e.historyPos]

	e.buffer.SetContent([]byte(nextStateContent))
	e.buffer.SetCursor(nextCursor)

	e.ScrollViewport()

	return currentContent, nil
}

func (e *editor) Paste() (string, error) {
	content, err := e.clipboard.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	// Insert content at the current cursor position
	cursor := e.buffer.GetCursor()
	e.buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune(content))

	// Update the state to reflect the new content
	e.SaveHistory() // Save the new state after pasting

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
		// Visual mode: use the selected range, normalize it
		start, end = NormalizeSelection(state.VisualStart, cursor.Position)
		// Note: Vim's visual line mode ('V') would force isLineWise = true here.
		// We don't have that mode explicitly yet, so visual is character-wise.

		// Check if the selection is line-wise

		if state.Mode == "visual-line" {
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
	}

	// Add trailing newline for line-wise operations
	content := contentBuilder.String()
	if isLineWise && !strings.HasSuffix(content, "\n") && content != "" { // Ensure we add newline for line-wise if needed
		// Check if content is just "" (e.g. empty line copied line-wise)
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

func (e *editor) IsSelected() bool {
	return e.state.VisualStart.Row != -1
}

func (e *editor) GetSelectionStatus(pos Position) SelectionType {
	state := e.GetState() // Get current state
	buffer := e.GetBuffer()
	cursor := buffer.GetCursor()

	if state.VisualStart.Row == -1 { // Not in visual mode
		return SelectionNone
	}

	// Normalize selection range using the accessible function
	selStart, selEnd := NormalizeSelection(state.VisualStart, cursor.Position)

	// Check Visual Line Mode first
	if state.Mode == "visual-line" {
		if pos.Row >= selStart.Row && pos.Row <= selEnd.Row {
			return SelectionLine
		}
		// Note: In visual-line mode, even if the column is outside the typical character range,
		// the *entire line* is considered selected for styling purposes.
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
