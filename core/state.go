package core

import (
	"slices"
	"sync"
	"unicode"
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

	isWordCharFunc func(rune) bool // Pre-computed classifier for word characters

	WithCommandMode bool // Whether command mode is enabled

	WithInsertMode bool // Whether insert mode is enabled

	WithVisualMode bool // Whether visual mode is enabled

	WithVisualLineMode bool // Whether visual line mode is enabled

	WithSearchMode bool // Whether search mode is enabled
}

// defaultIsWordCharFunc is the singleton default classifier (letters, digits, '_').
// Initialized once via sync.Once to avoid repeated allocations.
var (
	defaultIsWordCharFunc     func(rune) bool
	defaultIsWordCharFuncOnce sync.Once
)

func getDefaultIsWordCharFunc() func(rune) bool {
	defaultIsWordCharFuncOnce.Do(func() {
		defaultIsWordCharFunc = func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_'
		}
	})
	return defaultIsWordCharFunc
}

// createIsWordCharClassifier builds a func(rune) bool that returns true for
// standard word characters (letters, digits, '_') plus any extra runes.
// For small sets of extra chars a linear scan is used (better cache locality);
// for larger sets a map provides O(1) lookup.
func createIsWordCharClassifier(extraChars []rune) func(rune) bool {
	if len(extraChars) == 0 {
		return getDefaultIsWordCharFunc()
	}
	if len(extraChars) < 8 {
		extra := make([]rune, len(extraChars))
		copy(extra, extraChars)
		return func(r rune) bool {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
				return true
			}
			return slices.Contains(extra, r)
		}
	}
	extraMap := make(map[rune]struct{}, len(extraChars))
	for _, ch := range extraChars {
		extraMap[ch] = struct{}{}
	}
	return func(r rune) bool {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			return true
		}
		_, ok := extraMap[r]
		return ok
	}
}

// IsWordChar reports whether r is considered a word character in this state.
// isWordCharFunc is guaranteed to be non-nil by InitialState and SetExtraWordChars.
func (s State) IsWordChar(r rune) bool {
	return s.isWordCharFunc(r)
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
		isWordCharFunc:    getDefaultIsWordCharFunc(),

		WithCommandMode:    true,
		WithInsertMode:     true,
		WithVisualMode:     true,
		WithVisualLineMode: true,
		WithSearchMode:     true,
	}
}
