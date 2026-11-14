package core

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const historyContent = "foo bar foo"

// TestCmdlineHistoryAdd checks which lines go into the ':' and search
// histories, as Vim 9.2: lines run or abandoned, a repeated line moving to the
// end, the patterns of * and #, but not n, empty lines or "/<CR>".
func TestCmdlineHistoryAdd(t *testing.T) {
	tests := []struct {
		keys    string
		command []string
		search  []string
	}{
		{":echo 1<CR>/bar<CR>:echo 2<CR>/o<CR>", []string{"echo 1", "echo 2"}, []string{"bar", "o"}},
		{":echo 1<CR>:echo 2<CR>:echo 1<CR>", []string{"echo 2", "echo 1"}, nil},
		{"/bar<CR>/foo<CR>/bar<CR>", nil, []string{"foo", "bar"}},
		{"*", nil, []string{`\<foo\>`}},
		{"#", nil, []string{`\<foo\>`}},
		{"/foo<CR>/bar<CR>*", nil, []string{"foo", "bar", `\<bar\>`}},
		{"/bar<CR>n", nil, []string{"bar"}},
		{"/bar<CR>/foo<CR>/<CR>", nil, []string{"bar", "foo"}},
		{"/bar<CR>/foo<CR>n?<CR>", nil, []string{"bar", "foo"}},
		{"/zzz<CR>", nil, []string{"zzz"}},
		// Abandoned lines are kept too, but not empty ones.
		{":echo 1<Esc>", []string{"echo 1"}, nil},
		{"/bar<Esc>", nil, []string{"bar"}},
		{":<Esc>", nil, nil},
		{":<CR>", nil, nil},
		{"/x<BS><BS>", nil, nil},
		// White space is part of the entry.
		{":echo 1 <CR>", []string{"echo 1 "}, nil},
		{": echo 1<CR>", []string{" echo 1"}, nil},
		// Running an entry brought back moves it to the end.
		{":echo 1<CR>:echo 22<CR>:let a=1<CR>:echo 3<CR>:ec<Up><Up><CR>", []string{"echo 1", "let a=1", "echo 3", "echo 22"}, nil},
		{":echo 1<CR>:echo 22<CR>:let a=1<CR>:echo 3<CR>:ec<Up><Up><Esc>", []string{"echo 1", "let a=1", "echo 3", "echo 22"}, nil},
		{"/bar<CR>/foo<CR>/<Up><Up><CR>", nil, []string{"foo", "bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor(historyContent)
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.command, nilIfEmpty(e.CommandLineHistory(CommandHistory)))
			assert.Equal(t, tt.search, nilIfEmpty(e.CommandLineHistory(SearchHistory)))
		})
	}
}

func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

// TestCmdlineHistoryBrowse checks <Up>, <Down> and the unfiltered keys on the
// command line against Vim 9.2: <Up>/<Down> take entries starting with the
// line up to the cursor when the walk started, <Down> past the newest brings
// that line back, <Left>/<Right> keep the walk, and other keys start a new one
// from the same place in the list.
func TestCmdlineHistoryBrowse(t *testing.T) {
	const h = ":echo 1<CR>:echo 22<CR>:let a=1<CR>:echo 3<CR>"
	const e3 = ":ec1<CR>:ex2<CR>:ec3<CR>"
	tests := []struct {
		keys   string
		line   string
		cursor int // in the line, after the prompt
	}{
		{h + ":<Up>", "echo 3", 6},
		{h + ":<Up><Up>", "let a=1", 7},
		{h + ":<Up><Up><Up><Up>", "echo 1", 6},
		{h + ":<Up><Up><Up><Up><Up><Up>", "echo 1", 6},
		{h + ":<Up><Down>", "", 0},
		{h + ":<Down>", "", 0},
		{h + ":<Up><Up><Down><Down>", "", 0},
		{h + ":<Up><Up><Down>", "echo 3", 6},
		{h + ":ec<Up>", "echo 3", 6},
		{h + ":ec<Up><Up>", "echo 22", 7},
		{h + ":ec<Up><Up><Up>", "echo 1", 6},
		{h + ":ec<Up><Up><Up><Up>", "echo 1", 6},
		{h + ":ec<Up><Down>", "ec", 2},
		{h + ":ec<Up><Down><Up>", "echo 3", 6},
		{h + ":ec<Up><Up><Down><Down><Down>", "ec", 2},
		{h + ":ec<Up><Left>", "echo 3", 5},
		{h + ":ec<Left><Up>", "echo 3", 6},
		{h + ":ec<Up><BS><Up>", "echo 22", 7},
		{h + ":ec<Up><BS><Down>", "echo ", 5},
		{h + ":echo 2<Up>", "echo 22", 7},
		{h + ":zz<Up>", "zz", 2},
		{h + ":zz<Up><Down>", "zz", 2},
		{h + ":ec<Up><Up>x<Down>", "echo 22x", 8},
		{h + ":ec<Up><Up><C-U><Down>", "let a=1", 7},
		{h + ":ec<Up><Up><C-U><Up>", "echo 1", 6},
		// Every entry, whatever was typed.
		{h + ":ec<S-Up>", "echo 3", 6},
		{h + ":ec<S-Up><S-Up>", "let a=1", 7},
		{h + ":ec<PageUp>", "echo 3", 6},
		{h + ":ec<PageUp><PageDown>", "ec", 2},
		{h + ":ec<S-Up><S-Down>", "ec", 2},
		{h + ":ec<C-P><C-P>", "let a=1", 7},
		{h + ":ec<C-P><C-N>", "ec", 2},
		{h + ":ec<Up><Up><S-Up>", "echo 1", 6},
		{h + ":ec<Up><S-Down>", "ec", 2},
		{":echo 1<CR>:<Up><Up><Down>", "", 0},
		// Which keys end the walk.
		{e3 + ":ec<Up><Left><Up>", "ec1", 3},
		{e3 + ":ec<Up><Left><Right><Up>", "ec1", 3},
		{e3 + ":ec<Up><Left><Left><Left><Up>", "ec1", 3},
		{e3 + ":ec<Up><End><Up>", "ec3", 3},
		{e3 + ":ec<Up><Home><Up>", "ex2", 3},
		{e3 + ":ec<Up><S-Left><Up>", "ex2", 3},
		{e3 + ":e<Up><Up><BS><Up>", "ex", 2},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor(historyContent)
			typeKeys(t, e, tt.keys)
			require.Equal(t, CommandMode, e.GetMode().Name())
			assert.Equal(t, ":"+tt.line, e.GetState().CommandLine)
			assert.Equal(t, tt.cursor+1, e.GetState().CommandCursor)
		})
	}
}

// TestSearchHistoryBrowse checks that '/' and '?' share the search history,
// apart from the ':' one, and that a pattern brought back searches.
func TestSearchHistoryBrowse(t *testing.T) {
	tests := []struct {
		keys   string
		line   string
		cursor int
	}{
		{"/bar<CR>/foo<CR>:echo 1<CR>/<Up>", "/foo", 4},
		{"/bar<CR>/foo<CR>:echo 1<CR>?<Up><Up>", "?bar", 4},
		{"/bar<CR>/foo<CR>/fo<Up>", "/foo", 4},
		{"/bar<CR>/foo<CR>/fo<Up><Left><Up>", "/foo", 3},
		{"/bar<CR>/foo<CR>/fo<Up><Left><Down>", "/fo", 3},
		{"/bar<CR>/<Up><Up>", "/bar", 4},
		{"*/<Up>", `/\<foo\>`, 8},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e := newTestEditor(historyContent)
			typeKeys(t, e, tt.keys)
			require.True(t, e.IsSearchMode())
			assert.Equal(t, tt.line, e.GetState().CommandLine)
			assert.Equal(t, tt.cursor, e.GetState().CommandCursor)
		})
	}

	t.Run("an entry brought back searches", func(t *testing.T) {
		e := newTestEditor(historyContent)
		typeKeys(t, e, "/bar<CR>/foo<CR>gg/<Up><Up><CR>")
		assert.Equal(t, Position{0, 4}, cursorPos(e))
		assert.Equal(t, "bar", e.GetState().SearchQuery.Pattern)
	})
}

// TestCmdlineHistoryMappings checks Vim's rule for lines from mappings: a ':'
// line is kept only when a key of it was typed, a search line always.
func TestCmdlineHistoryMappings(t *testing.T) {
	tests := []struct {
		rhs     string
		keys    string
		command []string
		search  []string
	}{
		{":echo 9<CR>", "X", nil, nil},
		{"/bar<CR>", "X", nil, []string{"bar"}},
		{":", "Xecho 7<CR>", []string{"echo 7"}, nil},
		{":echo 9", "X<CR>", []string{"echo 9"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.rhs, func(t *testing.T) {
			e := newTestEditor(historyContent)
			require.NoError(t, e.Map(MapNormal, "X", tt.rhs, true))
			typeKeys(t, e, tt.keys)
			assert.Equal(t, tt.command, nilIfEmpty(e.CommandLineHistory(CommandHistory)))
			assert.Equal(t, tt.search, nilIfEmpty(e.CommandLineHistory(SearchHistory)))
		})
	}
}

// TestHistoryHostAPI checks the API a host saves and restores the history
// with, and the 'history' size.
func TestHistoryHostAPI(t *testing.T) {
	t.Run("restored entries are browsed", func(t *testing.T) {
		e := newTestEditor(historyContent)
		e.SetCommandLineHistory(CommandHistory, []string{"set rnu", "", "echo 1", "set rnu"})
		e.SetCommandLineHistory(SearchHistory, []string{"bar"})
		assert.Equal(t, []string{"echo 1", "set rnu"}, e.CommandLineHistory(CommandHistory))
		typeKeys(t, e, ":<Up><Up>")
		assert.Equal(t, ":echo 1", e.GetState().CommandLine)
		typeKeys(t, e, "<Esc>/<Up><CR>")
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("CommandLineHistory returns a copy", func(t *testing.T) {
		e := newTestEditor(historyContent)
		e.SetCommandLineHistory(CommandHistory, []string{"a"})
		e.CommandLineHistory(CommandHistory)[0] = "b"
		assert.Equal(t, []string{"a"}, e.CommandLineHistory(CommandHistory))
	})

	t.Run("default size is Vim's", func(t *testing.T) {
		assert.Equal(t, DefaultHistorySize, newTestEditor("").CommandLineHistorySize())
		assert.Equal(t, 200, DefaultHistorySize)
	})

	t.Run("the oldest entries go past the size", func(t *testing.T) {
		e := newTestEditor(historyContent)
		e.SetCommandLineHistorySize(3)
		typeKeys(t, e, ":c0<CR>:c1<CR>:c2<CR>:c3<CR>:c4<CR>:c5<CR>")
		assert.Equal(t, []string{"c3", "c4", "c5"}, e.CommandLineHistory(CommandHistory))
		e.SetCommandLineHistory(SearchHistory, []string{"a", "b", "c", "d"})
		assert.Equal(t, []string{"b", "c", "d"}, e.CommandLineHistory(SearchHistory))
		e.SetCommandLineHistorySize(1)
		assert.Equal(t, []string{"c5"}, e.CommandLineHistory(CommandHistory))
		assert.Equal(t, []string{"d"}, e.CommandLineHistory(SearchHistory))
	})

	t.Run("size 0 keeps none", func(t *testing.T) {
		e := newTestEditor(historyContent)
		require.Nil(t, e.ExecuteCommand("set history=0"))
		typeKeys(t, e, ":echo 1<CR>/bar<CR>:<Up>")
		assert.Empty(t, e.CommandLineHistory(CommandHistory))
		assert.Empty(t, e.CommandLineHistory(SearchHistory))
		assert.Equal(t, ":", e.GetState().CommandLine)
	})

	t.Run(":set history", func(t *testing.T) {
		e := newTestEditor(historyContent)
		require.Nil(t, e.ExecuteCommand("set hi=10"))
		assert.Equal(t, 10, e.CommandLineHistorySize())
		for _, bad := range []string{"history=-1", "history=x", "history=10001"} {
			assert.NotNil(t, e.ExecuteCommand("set "+bad), bad)
		}
		assert.Equal(t, 10, e.CommandLineHistorySize())
	})

	t.Run("unknown kinds are ignored", func(t *testing.T) {
		e := newTestEditor(historyContent)
		e.SetCommandLineHistory(HistoryKind(7), []string{"a"})
		assert.Nil(t, e.CommandLineHistory(HistoryKind(7)))
		assert.Nil(t, e.CommandLineHistory(HistoryKind(-1)))
	})
}

// TestSetCommandLineHistoryMatchesAdd checks that restoring a history gives
// the same list as adding its entries one at a time, as the user typing them
// would.
func TestSetCommandLineHistoryMatchesAdd(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 500 {
		size := r.IntN(8)
		entries := make([]string, r.IntN(20))
		for i := range entries {
			entries[i] = strings.Repeat("x", r.IntN(6)) // "" sometimes, many duplicates
		}

		want := histories{size: size}
		for _, entry := range entries {
			want.add(SearchHistory, entry)
		}

		e := newTestEditor("")
		e.SetCommandLineHistorySize(size)
		e.SetCommandLineHistory(SearchHistory, entries)
		require.Equal(t, nilIfEmpty(want.lists[SearchHistory]), nilIfEmpty(e.CommandLineHistory(SearchHistory)), "size %d, entries %q", size, entries)
	}
}

// BenchmarkSetCommandLineHistory restores a full history at the largest size
// ':set history' allows.
func BenchmarkSetCommandLineHistory(b *testing.B) {
	entries := make([]string, 10000)
	for i := range entries {
		entries[i] = fmt.Sprintf("echo %d", i)
	}
	e := newTestEditor("")
	e.SetCommandLineHistorySize(10000)
	for b.Loop() {
		e.SetCommandLineHistory(CommandHistory, entries)
	}
}
