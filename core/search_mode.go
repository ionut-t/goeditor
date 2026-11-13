package core

// searchMode is the '/' and '?' search line: edited like the ':' command line
// (cmdline), and searched on <CR>. The next '/' or '?' opens with the last
// search's pattern, whichever command searched (/, ?, *, #, or an empty
// <CR> reusing it), and with the cursor where it was when that pattern was
// typed here, at its end otherwise; abandoning the line empties it.
type searchMode struct {
	baseMode
	line      cmdline
	backwards bool
	options   *SearchOptions // the editor's SetSearchOptions
}

// NewSearchMode returns a search mode that runs searches with Vim's default
// options. The editor's own search mode uses the options set with
// SetSearchOptions.
func NewSearchMode() EditorMode { return newSearchMode(&SearchOptions{Wrap: true}) }

func newSearchMode(options *SearchOptions) *searchMode {
	return &searchMode{options: options}
}

func (m *searchMode) Name() Mode { return SearchMode }

func (m *searchMode) prompt() string {
	if m.backwards {
		return "?"
	}
	return "/"
}

func (m *searchMode) Enter(editor Editor, buffer Buffer) {
	state := editor.GetState()
	m.backwards = state.SearchOptions.Backwards
	if pattern := state.SearchQuery.Pattern; pattern != string(m.line.text) {
		m.line.set(pattern)
	}
	editor.DispatchSignal(EnterSearchModeSignal{Backwards: m.backwards})
	m.line.publish(editor, m.prompt())
}

func (m *searchMode) Exit(editor Editor, buffer Buffer) {
	editor.UpdateCommand("")
	editor.DispatchSignal(ExitSearchModeSignal{})
}

// HandleKey edits the search line (see cmdline.handleKey) and searches on
// <CR>. ExecuteSearch and CancelSearch go back to the mode the search started
// from.
func (m *searchMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	switch m.line.handleKey(key) {
	case cmdlineAbandon:
		m.line.reset()
		editor.CancelSearch()
	case cmdlineEnter:
		options := *m.options
		options.Backwards = m.backwards
		editor.ExecuteSearch(string(m.line.text), options)
	case cmdlineEdited:
		m.line.publish(editor, m.prompt())
	}
	return nil
}
