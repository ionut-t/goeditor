package core

import "strings"

func (e *editor) ExecuteSearch(pattern string, searchOptions SearchOptions) {
	e.state.SearchQuery.Pattern = pattern
	query := pattern

	ignoreCase := searchOptions.IgnoreCase
	smartCase := searchOptions.SmartCase

	if before, ok := strings.CutSuffix(pattern, "\\c"); ok {
		pattern = before
		ignoreCase = true
		smartCase = false
		query = strings.TrimRight(pattern, "\\c")
	} else if before0, ok0 := strings.CutSuffix(pattern, "\\C"); ok0 {
		pattern = before0
		ignoreCase = false
		searchOptions.SmartCase = true
		query = strings.TrimRight(pattern, "\\C")
	}

	// Parse \< (word-start) and \> (word-end) boundary anchors.
	wordBoundaryStart := searchOptions.WordBoundaryStart
	wordBoundaryEnd := searchOptions.WordBoundaryEnd
	if after, ok := strings.CutPrefix(query, `\<`); ok {
		query = after
		wordBoundaryStart = true
	}
	if before, ok := strings.CutSuffix(query, `\>`); ok {
		query = before
		wordBoundaryEnd = true
	}

	e.state.SearchQuery.Term = query
	// Inline modifiers (\c, \C, \<, \>) take precedence and override the input searchOptions.
	// Other fields (Backwards, Wrap, WholeWord, IsWordChar) are taken directly from searchOptions.
	e.state.SearchOptions = SearchOptions{
		IgnoreCase:        ignoreCase,
		SmartCase:         smartCase,
		Backwards:         searchOptions.Backwards,
		Wrap:              searchOptions.Wrap,
		WholeWord:         searchOptions.WholeWord,
		WordBoundaryStart: wordBoundaryStart,
		WordBoundaryEnd:   wordBoundaryEnd,
		IsWordChar:        searchOptions.IsWordChar,
	}

	// Find the first result
	pos, found := e.buffer.Find(query, e.buffer.GetCursor().Position, e.state.SearchOptions)

	if !found && e.state.SearchOptions.Wrap {
		pos, found = e.wrapFind(query, e.state.SearchOptions)
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

	if e.state.Mode == SearchMode {
		prefix := "/"
		if e.state.SearchOptions.Backwards {
			prefix = "?"
		}
		e.UpdateCommand(prefix + e.state.SearchQuery.Pattern)
		e.setMode(e.state.PreviousMode)
	}
	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})
}

func (e *editor) CancelSearch() {
	e.state.SearchQuery = SearchQuery{}
	e.state.SearchResults = []Position{}
	e.setMode(e.state.PreviousMode)
}

// wrapFind searches from the appropriate boundary depending on direction:
// forward searches wrap to the start of the buffer, backward searches wrap to the end.
func (e *editor) wrapFind(term string, options SearchOptions) (Position, bool) {
	if options.Backwards {
		lastLine := e.buffer.LineCount() - 1
		lastLineLen := e.buffer.LineRuneCount(lastLine)
		return e.buffer.Find(term, Position{Row: lastLine, Col: lastLineLen}, options)
	}
	// Col: -1 so that Find's "start after position" increment lands on col 0.
	return e.buffer.Find(term, Position{Row: 0, Col: -1}, options)
}

// NextSearchResult repeats the search in the same direction as the original search (vim 'n').
func (e *editor) NextSearchResult() Cursor {
	if len(e.state.SearchResults) == 0 {
		return e.buffer.GetCursor()
	}

	options := e.state.SearchOptions // preserve original direction

	currentPos := e.buffer.GetCursor().Position
	pos, found := e.buffer.Find(e.state.SearchQuery.Term, currentPos, options)

	if !found && options.Wrap {
		pos, found = e.wrapFind(e.state.SearchQuery.Term, options)
	}

	if found {
		e.onSearchResultFound(pos)
		e.ScrollViewport()
	}

	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})

	return e.buffer.GetCursor()
}

// PreviousSearchResult repeats the search in the opposite direction of the original search (vim 'N').
func (e *editor) PreviousSearchResult() Cursor {
	if len(e.state.SearchResults) == 0 {
		return e.buffer.GetCursor()
	}

	options := e.state.SearchOptions
	options.Backwards = !options.Backwards // flip direction

	currentPos := e.buffer.GetCursor().Position
	pos, found := e.buffer.Find(e.state.SearchQuery.Term, currentPos, options)

	if !found && options.Wrap {
		pos, found = e.wrapFind(e.state.SearchQuery.Term, options)
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

// SearchWordUnderCursor implements vim's * (forward) and # (backward):
// it extracts the word under the cursor and searches for it as a whole word.
func (e *editor) SearchWordUnderCursor(backwards bool) Cursor {
	cursor := e.buffer.GetCursor()
	pos := cursor.Position
	lineRunes := e.buffer.GetLineRunes(pos.Row)

	if len(lineRunes) == 0 || pos.Col >= len(lineRunes) || !e.IsWordChar(lineRunes[pos.Col]) {
		return cursor
	}

	// 'i' selects the inner word (without surrounding whitespace).
	startCol, endCol, found := wordTextObjectRange(e.buffer, pos, 'i', 1, e.IsWordChar)
	if !found {
		return cursor
	}

	word := string(lineRunes[startCol : endCol+1])

	e.ExecuteSearch(word, SearchOptions{
		Backwards:  backwards,
		Wrap:       true,
		WholeWord:  true,
		IsWordChar: e.IsWordChar,
	})
	e.ScrollViewport()

	return e.buffer.GetCursor()
}

func (e *editor) onSearchResultFound(pos Position) {
	e.state.SearchResults = []Position{pos}
	e.state.SearchResultIndex = 0
	cursor := e.buffer.GetCursor()
	cursor.Position = pos
	e.buffer.SetCursor(cursor)
}
