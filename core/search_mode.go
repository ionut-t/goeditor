package core

// searchMode is the '/' and '?' search line: edited like the ':' command line
// (cmdline), and searched on <CR>. It opens empty; <Up> and <Down> bring back
// earlier patterns from the search history, which the line adds to when it is
// run or abandoned, as Vim's.
type searchMode struct {
	baseMode
	line      cmdline
	backwards bool
	options   *SearchOptions // the editor's SetSearchOptions
	histories *histories     // the editor's, nil for NewSearchMode
}

// NewSearchMode returns a search mode that runs searches with Vim's default
// options. The editor's own search mode uses the options set with
// SetSearchOptions.
func NewSearchMode() EditorMode { return newSearchMode(&SearchOptions{Wrap: true}, nil) }

func newSearchMode(options *SearchOptions, h *histories) *searchMode {
	m := &searchMode{options: options, histories: h}
	if h != nil {
		m.line.history = &h.lists[SearchHistory]
	}
	return m
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
	m.line.reset()
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
		m.addHistory()
		editor.CancelSearch()
	case cmdlineEnter:
		m.addHistory()
		options := *m.options
		options.Backwards = m.backwards
		editor.ExecuteSearch(string(m.line.text), options)
	case cmdlineEdited:
		m.line.publish(editor, m.prompt())
	}
	return nil
}

// addHistory keeps the line in the search history. Unlike the ':' line, a
// search line from a mapping is kept too, as Vim's.
func (m *searchMode) addHistory() {
	if m.histories != nil {
		m.histories.add(SearchHistory, string(m.line.text))
	}
}
