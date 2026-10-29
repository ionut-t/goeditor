package core

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
	TopLine        int // First buffer line in the window (0-indexed); see scroll.go
	ViewportHeight int // Number of lines that can be displayed
	ViewportWidth  int // Number of columns that can be displayed

	// Visual mode
	VisualStart   Position      // Starting position for visual selection (Use Position{-1,-1} if not active)
	YankSelection SelectionType // Type of selection for yank highlighting from normal mode (None, Character, or Line)

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

	charClassFunc func(rune) int // Pre-computed classifier for word motions and objects

	WithCommandMode bool // Whether command mode is enabled

	WithInsertMode bool // Whether insert mode is enabled

	WithVisualMode bool // Whether visual mode is enabled

	WithVisualLineMode bool // Whether visual line mode is enabled

	WithSearchMode bool // Whether search mode is enabled
}

// createCharClassifier returns utfClass with the extra runes counted as
// word characters.
func createCharClassifier(extraChars []rune) func(rune) int {
	if len(extraChars) == 0 {
		return utfClass
	}
	extra := make(map[rune]struct{}, len(extraChars))
	for _, ch := range extraChars {
		extra[ch] = struct{}{}
	}
	return func(r rune) int {
		if _, ok := extra[r]; ok {
			return wordCharClass
		}
		return utfClass(r)
	}
}

// CharClass is the class of r for word motions and objects (see utfClass).
// charClassFunc is guaranteed to be non-nil by InitialState and SetExtraWordChars.
func (s State) CharClass(r rune) int {
	return s.charClassFunc(r)
}

// IsWordChar reports whether r is considered a word character in this state:
// Vim's vim_iswordc, which counts emoji and the script classes.
func (s State) IsWordChar(r rune) bool {
	return s.charClassFunc(r) >= wordCharClass
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
		charClassFunc:     utfClass,

		WithCommandMode:    true,
		WithInsertMode:     true,
		WithVisualMode:     true,
		WithVisualLineMode: true,
		WithSearchMode:     true,
	}
}
