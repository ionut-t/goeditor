package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const searchModeContent = "foo 1\nbar 2\nfoo 3\nbar 4"

// TestSearchLine checks that the search line runs the search on <CR> with the
// text as edited, and goes back to the mode it started from, as Vim 9.2.
func TestSearchLine(t *testing.T) {
	tests := []struct {
		keys   string
		cursor Position
		mode   Mode
		echo   string // State.SearchEcho: the search Vim leaves on the command line
	}{
		{"/bar<CR>", Position{1, 0}, NormalMode, "/bar"},
		{"G?foo<CR>", Position{2, 0}, NormalMode, "?foo"},
		{"/bxr<Left><BS>a<CR>", Position{1, 0}, NormalMode, "/bar"},
		{"/foo 3<C-W><C-W>bar<CR>", Position{1, 0}, NormalMode, "/bar"},
		{"/bar<C-J>", Position{1, 0}, NormalMode, "/bar"},
		{"v/bar<CR>", Position{1, 0}, VisualMode, "/bar"},

		// Abandoning: <Esc>, or <BS> on an empty line.
		{"/bar<Esc>", Position{0, 0}, NormalMode, ""},
		{"j/<BS>", Position{1, 0}, NormalMode, ""},
		{"/b<BS><BS>", Position{0, 0}, NormalMode, ""},
		{"v/bar<Esc>", Position{0, 0}, VisualMode, ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor(searchModeContent)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, tt.mode, e.GetState().Mode)
			assert.Equal(t, "", e.GetState().CommandLine)
			assert.Equal(t, -1, e.GetState().CommandCursor, "no cursor once the search line is gone")
			assert.Equal(t, tt.echo, e.GetState().SearchEcho)
		})
	}
}

// TestSearchEchoStays checks how long the search stays on the command line
// after <CR>, against Vim 9.2 in a terminal: through motions and commands
// that show nothing, under pending keys (Vim's showcmd, which goeditor shows
// in CommandLine), and until a mode change (Vim's -- VISUAL --, -- INSERT --)
// or the next command line replaces it.
func TestSearchEchoStays(t *testing.T) {
	tests := []struct {
		keys        string
		commandLine string
		echo        string
	}{
		{"j", "", "/bar"},
		{"n", "", "/bar"},
		{"x", "", "/bar"},
		{"fo", "", "/bar"},
		{"gg", "", "/bar"},
		{"yy", "", "/bar"},
		{"2", "2", "/bar"},
		{"2d", "2d", "/bar"},
		{"2d<Esc>", "", "/bar"},
		{"v", "", ""},
		{"i", "", ""},
		{":<Esc>", "", ""},
		{"/x<Esc>", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(searchModeContent)
			typeKeys(t, e, "/bar<CR>"+tt.keys)
			assert.Equal(t, tt.commandLine, e.GetState().CommandLine)
			assert.Equal(t, tt.echo, e.GetState().SearchEcho)
		})
	}
}

// TestSearchEchoAfterRepeat checks that n, N, * and # leave the search on the
// command line with the direction they went, as Vim 9.2 in a terminal (also
// after a mode change cleared the last echo).
func TestSearchEchoAfterRepeat(t *testing.T) {
	tests := []struct {
		keys string
		echo string
	}{
		{"/bar<CR>n", "/bar"},
		{"/bar<CR>N", "?bar"},
		{"/bar<CR>nN", "?bar"},
		{"?bar<CR>n", "?bar"},
		{"?bar<CR>N", "/bar"},
		{"/bar<CR>v<Esc>n", "/bar"},
		{"/bar<CR>i<Esc>n", "/bar"},
		{"/bar<CR>:<Esc>n", "/bar"},
		{"/bar<CR>/x<Esc>n", ""},
		{"*", `/\<foo\>`},
		{"*N", `?\<foo\>`},
		{"j#", `?\<bar\>`},
		{"j#n", `?\<bar\>`},
		{"/bar<CR>*", `/\<bar\>`},
		{"v*", `/\<foo\>`},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(searchModeContent)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.echo, e.GetState().SearchEcho)
		})
	}
}

// TestSearchLineKeepsPattern checks that the search line opens with the last
// pattern typed, and its cursor where it was, and that abandoning it empties
// it.
func TestSearchLineKeepsPattern(t *testing.T) {
	tests := []struct {
		keys        string
		commandLine string
		cursor      int
	}{
		{"/bar<CR>/", "/bar", 4},
		{"/bar<CR>?", "?bar", 4},
		{"/bar<Left><CR>/", "/bar", 3},
		{"/bar<CR>/<C-U>foo", "/foo", 4},
		{"/bar<CR>/2<CR>/", "/bar2", 5},
		{"/bar<CR>/<Esc>/", "/", 1},
		{"/bar<Esc>/", "/", 1},
		// The last search's pattern, whichever command searched, with the
		// cursor at its end.
		{"*/", `/\<foo\>`, 8},
		{"j#?", `?\<bar\>`, 8},
		{"/bar<Left><CR>*/", `/\<bar\>`, 8},
		{"/bar<CR>/<C-U><CR>/", "/bar", 4},
		{"*/<C-U><CR>/", `/\<foo\>`, 8},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor(searchModeContent)
			typeKeys(t, e, tt.keys)
			require.True(t, e.IsSearchMode())
			assert.Equal(t, tt.commandLine, e.GetState().CommandLine)
			assert.Equal(t, tt.cursor, e.GetState().CommandCursor)
		})
	}
}

// TestSearchEmptyPattern checks that an empty "/<CR>" or "?<CR>" reuses the
// last search in the new direction, which n and N then follow, as Vim 9.2.
// The line opens with the last pattern, so the cases empty it with <C-U>.
func TestSearchEmptyPattern(t *testing.T) {
	tests := []struct {
		content string
		keys    string
		cursor  Position
		echo    string
	}{
		{searchModeContent, "/bar<CR>gg/<C-U><CR>", Position{1, 0}, "/bar"},
		{searchModeContent, "/bar<CR>/<C-U><CR>", Position{3, 0}, "/bar"},
		{searchModeContent, "/bar<CR>/<C-U><CR>N", Position{1, 0}, "?bar"},
		{searchModeContent, "/bar<CR>G?<C-U><CR>", Position{1, 0}, "?bar"},
		{searchModeContent, "/bar<CR>G?<C-U><CR>n", Position{3, 0}, "?bar"},
		{searchModeContent, "?bar<CR>gg/<C-U><CR>n", Position{3, 0}, "/bar"},
		{searchModeContent, "/BAR\\c<CR>gg/<C-U><CR>", Position{1, 0}, "/BAR\\c"},
		// * keeps its whole-word match: foobar and xfoo are skipped.
		{"foo foobar\nxfoo foo", "*gg/<C-U><CR>", Position{1, 5}, `/\<foo\>`},
		{"foo foobar\nxfoo foo", "*gg/<C-U><CR>n", Position{0, 0}, `/\<foo\>`},
		{"foo foobar\nxfoo foo", "*gg?<C-U><CR>n", Position{0, 0}, `?\<foo\>`},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(tt.content)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.cursor, cursorPos(e))
			assert.Equal(t, NormalMode, e.GetState().Mode)
			assert.Equal(t, tt.echo, e.GetState().SearchEcho)
		})
	}
}

// TestSearchEmptyPatternWithoutLastSearch checks Vim's E35: an empty search
// with no last search moves nothing and reports ErrNoPreviousPattern. Unlike
// Vim, abandoning the search line clears the last search (a kept divergence),
// so "/bar<CR>/<Esc>/<CR>" fails too, where Vim searches for bar.
func TestSearchEmptyPatternWithoutLastSearch(t *testing.T) {
	for _, keys := range []string{"/<CR>", "?<CR>", "v/<CR>", "/bar<CR>gg/<Esc>/<CR>"} {
		t.Run(keys, func(t *testing.T) {
			e := newTestEditor(searchModeContent)
			typeKeys(t, e, keys)
			assert.Equal(t, Position{0, 0}, cursorPos(e))
			assert.False(t, e.IsSearchMode())
			assert.Equal(t, "", e.GetState().SearchEcho)

			assert.Equal(t, ErrNoPreviousPatternId, nextErrorID(t, e))

			typeKeys(t, e, "n")
			assert.Equal(t, Position{0, 0}, cursorPos(e), "n has nothing to repeat")
		})
	}
}

// TestSearchLineShowsPrompt checks the prompt and cursor published while the
// search line is edited.
func TestSearchLineShowsPrompt(t *testing.T) {
	e := newTestEditor(searchModeContent)
	keys(e, '?')
	assert.Equal(t, "?", e.GetState().CommandLine)
	assert.Equal(t, 1, e.GetState().CommandCursor)

	typeKeys(t, e, "ab<Left>")
	assert.Equal(t, "?ab", e.GetState().CommandLine)
	assert.Equal(t, 2, e.GetState().CommandCursor, "on the b, after the prompt and the a")
}

// TestSearchLineUsesSearchOptions checks that a typed search runs with the
// options set with SetSearchOptions, and that the default is Vim's: case
// sensitive.
func TestSearchLineUsesSearchOptions(t *testing.T) {
	e := newTestEditor(searchModeContent)
	typeKeys(t, e, "/BAR<CR>")
	assert.Equal(t, Position{0, 0}, cursorPos(e), "case sensitive by default")

	e = newTestEditor(searchModeContent)
	e.SetSearchOptions(SearchOptions{IgnoreCase: true, Wrap: true})
	typeKeys(t, e, "/BAR<CR>")
	assert.Equal(t, Position{1, 0}, cursorPos(e))

	e = newTestEditor(searchModeContent)
	e.SetSearchOptions(SearchOptions{Wrap: false})
	typeKeys(t, e, "G?bar<CR>")
	assert.Equal(t, Position{1, 0}, cursorPos(e), "backwards from the last line")
	typeKeys(t, e, "gg?bar<CR>")
	assert.Equal(t, Position{0, 0}, cursorPos(e), "no wrap: not found")
}

// TestMappingIntoSearch checks that a mapping can type a search and run it
// (open bug 60: the mapped keys used to reach a search mode that ignored them).
func TestMappingIntoSearch(t *testing.T) {
	e := newTestEditor(searchModeContent)
	require.NoError(t, e.Map(MapNormal, "s", "/bar<CR>", true))
	keys(e, 's')
	assert.Equal(t, Position{1, 0}, cursorPos(e))
	assert.Equal(t, NormalMode, e.GetState().Mode)
	assert.Equal(t, "bar", e.GetState().SearchQuery.Pattern)
}
