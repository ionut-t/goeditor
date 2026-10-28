package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// openSignals returns the targets of the OpenSignals dispatched since the
// last drain.
func openSignals(e Editor) []string {
	var targets []string
	for s := nextSignal(e); s != nil; s = nextSignal(e) {
		if sig, ok := s.(OpenSignal); ok {
			targets = append(targets, sig.Value())
		}
	}
	return targets
}

func setCursorPos(e Editor, row, col int) {
	cursor := e.GetBuffer().GetCursor()
	cursor.Position = Position{Row: row, Col: col}
	cursor.Preferred = col
	e.GetBuffer().SetCursor(cursor)
}

// TestGxNormal checks the target gx sends against Vim 9.2's openPlugin.vim:
// a URL in <cWORD>, else <cfile>.
func TestGxNormal(t *testing.T) {
	tests := []struct {
		name    string
		content string
		row     int
		col     int
		want    string // "" means no OpenSignal
	}{
		// URL in the WORD under the cursor
		{"url, cursor on scheme", "see https://example.com/a?b=1&c=2#frag here", 0, 5, "https://example.com/a?b=1&c=2#frag"},
		{"url, cursor mid", "see https://example.com/a?b=1&c=2#frag here", 0, 29, "https://example.com/a?b=1&c=2#frag"},
		{"url in parens with period", "x (https://example.com/path). y", 0, 9, "https://example.com/path"},
		{"markdown link, cursor on text", "[docs](https://ex.com/a_b)", 0, 1, "https://ex.com/a_b"},
		{"markdown link, cursor on url", "[docs](https://ex.com/a_b)", 0, 14, "https://ex.com/a_b"},
		{"trailing slash kept", "https://ex.com/", 0, 0, "https://ex.com/"},
		{"closing paren dropped", "https://en.wikipedia.org/wiki/Foo_(bar)", 0, 0, "https://en.wikipedia.org/wiki/Foo_(bar"},
		{"quoted href", `<a href="https://x.com/a">`, 0, 11, "https://x.com/a"},
		{"scheme glued to a word", "foohttps://a.com", 0, 0, "https://a.com"},
		{"two urls in one WORD", "https://a.com,https://b.com", 0, 19, "https://a.com,https://b.com"},
		{"irc scheme", "http://a.b ftp://c.d ircs://e.f file:///tmp/x", 0, 24, "ircs://e.f"},
		{"file scheme", "file:///tmp/x;", 0, 2, "file:///tmp/x"},
		{"non-ASCII tail dropped", "https://ex.jp/パス", 0, 2, "https://ex.jp/"},
		{"punctuation tail dropped", "https://ex.com/a-_~", 0, 2, "https://ex.com/a"},
		{"percent and bang kept", "https://x.com/a%20b!c", 0, 0, "https://x.com/a%20b!c"},
		{"blanks before url", "foo    https://x.com", 0, 4, "https://x.com"},
		{"blanks then paren before url", "   (https://x.com)", 0, 0, "https://x.com"},
		{"cursor on paren after url", "(https://x.com)", 0, 14, "https://x.com"},
		{"cursor on ws before WORD", "foo https://x.com", 0, 3, "https://x.com"},
		{"no-break space ends the WORD", "https://a.com\u00a0x", 0, 0, "https://a.com"},

		// No URL: <cfile>
		{"uppercase scheme is a file name", "HTTPS://X.COM/P", 0, 2, "HTTPS://X.COM/P"},
		{"mailto stops at colon", "mailto:a@b.com", 0, 2, "mailto"},
		{"www without scheme", "go www.example.com.", 0, 4, "www.example.com"},
		{"word before a url", "foo https://x.com", 0, 1, "foo"},
		{"blanks before word", "foo    bar.txt", 0, 4, "bar.txt"},
		{"blanks before WORD, url later", "   foo https://x.com", 0, 0, "foo"},
		{"absolute path", "open /tmp/foo.txt now", 0, 7, "/tmp/foo.txt"},
		{"trailing period dropped", "see foo/bar.go.", 0, 5, "foo/bar.go"},
		{"trailing comma dropped", "see foo/bar.go, ok", 0, 5, "foo/bar.go"},
		{"cursor on trailing period", "foo.go.", 0, 6, "foo.go"},
		{"trailing dots kept", "a/b.txt...", 0, 0, "a/b.txt..."},
		{"dot before comma keeps comma", "a/b.txt.,", 0, 0, "a/b.txt.,"},
		{"two-byte name keeps dot", "a.", 0, 0, "a."},
		{"three-byte name drops dot", "ab.", 0, 0, "ab"},
		{"trailing bang", "a/b.txt!", 0, 0, "a/b.txt"},
		{"comma inside name", "a,b.txt", 0, 0, "a,b.txt"},
		{"line and column", "foo.go:12:3", 0, 1, "foo.go"},
		{"path in parens", "(foo/bar.go)", 0, 3, "foo/bar.go"},
		{"cursor on open paren", "(foo/bar.go)", 0, 0, "foo/bar.go"},
		{"cursor on close paren looks forward", "(foo/bar.go) z", 0, 11, "z"},
		{"quoted name stops at space", `"./a b.txt"`, 0, 2, "./a"},
		{"escaped space", `a\ b.txt`, 0, 0, `a\ b.txt`},
		{"tilde", "~/notes.md", 0, 2, "~/notes.md"},
		{"env var", "$HOME/x", 0, 2, "$HOME/x"},
		{"user@host", "user@host:/x", 0, 0, "user"},
		{"backslash path", `a\b\c`, 0, 0, "a"},
		{"non-ASCII name", "naïve/파일.txt", 0, 0, "naïve/파일.txt"},
		{"emoji", "😀/x", 0, 0, "😀/x"},
		{"tab before name", "a\tb.txt", 0, 1, "b.txt"},
		{"no-break and ideographic spaces in a name", "a b.txt　c x", 0, 0, "a b.txt　c"},
		{"second line", "one\ntwo/x", 1, 1, "two/x"},

		// Other schemes: "type://" stays together, and ":?&=" join a URL
		// whose type is all letters.
		{"ssh url with query", "ssh://h/x?a=1&b=2:3", 0, 0, "ssh://h/x?a=1&b=2:3"},
		{"cursor after ://", "ssh://h/x", 0, 7, "ssh://h/x"},
		{"cursor on colon of ://", "ssh://h/x", 0, 3, "ssh://h/x"},
		{"trailing colon dropped", "ssh://h/x:", 0, 0, "ssh://h/x"},
		{"type with plus is not a url", "git+ssh://h/x.", 0, 0, "git+ssh://h/x"},
		{"query outside a url", "a/b?c&d=e", 0, 0, "a/b"},

		// Nothing to open: Vim runs the opener on "", we send nothing.
		{"trailing blanks", "foo.txt    ", 0, 9, ""},
		{"punctuation only", "x  ))) ", 0, 4, ""},
		{"empty line", "", 0, 0, ""},
		{"blank line", "    ", 0, 1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			setCursorPos(e, tt.row, tt.col)
			drainSignals(e)

			keys(e, 'g', 'x')

			if tt.want == "" {
				assert.Empty(t, openSignals(e))
			} else {
				assert.Equal(t, []string{tt.want}, openSignals(e))
			}
			assert.Equal(t, Position{Row: tt.row, Col: tt.col}, cursorPos(e), "gx doesn't move the cursor")
			assert.Equal(t, tt.content, content(e))
		})
	}
}

func TestGxNormalCount(t *testing.T) {
	e := newTestEditor("a.txt b.txt")
	drainSignals(e)

	keys(e, '3', 'g', 'x')

	assert.Equal(t, []string{"a.txt"}, openSignals(e), "the count is ignored")
	assert.Nil(t, e.GetState().PendingCount)
}

func TestGxAfterOperatorCancels(t *testing.T) {
	e := newTestEditor("https://x.com")
	drainSignals(e)

	keys(e, 'd', 'g', 'x')

	assert.Empty(t, openSignals(e))
	assert.Equal(t, "https://x.com", content(e))
	assert.Equal(t, NormalMode, e.GetState().Mode)
}

// TestGxVisual checks that visual gx sends the selected text as is, lines
// joined with a space, and leaves the selection and cursor alone (Vim 9.2).
func TestGxVisual(t *testing.T) {
	tests := []struct {
		name    string
		content string
		col     int
		keys    []rune
		want    string
		mode    Mode
	}{
		{"charwise, part of a url", "see https://example.com/a here", 4, []rune("v6l"), "https:/", VisualMode},
		{"single char", "abc", 1, []rune("v"), "b", VisualMode},
		{"charwise over lines", "foo bar\nbaz qux", 4, []rune("vj2l"), "bar baz qux", VisualMode},
		{"charwise backwards", "foo bar", 6, []rune("v2h"), "bar", VisualMode},
		{"linewise", "foo bar\nbaz qux", 4, []rune("Vj"), "foo bar baz qux", VisualLineMode},
		{"linewise with an empty line", "a\n\nb", 0, []rune("V2j"), "a  b", VisualLineMode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(tt.content)
			setCursorPos(e, 0, tt.col)
			keys(e, tt.keys...)
			before := cursorPos(e)
			drainSignals(e)

			keys(e, 'g', 'x')

			assert.Equal(t, []string{tt.want}, openSignals(e))
			assert.Equal(t, tt.mode, e.GetState().Mode, "the selection stays")
			assert.Equal(t, before, cursorPos(e))
		})
	}
}

func TestGxVisualEmptyLine(t *testing.T) {
	e := newTestEditor("")
	keys(e, 'v')
	drainSignals(e)

	keys(e, 'g', 'x')

	assert.Empty(t, openSignals(e))
}
