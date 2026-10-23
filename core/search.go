package core

import "strings"

func (e *editor) ExecuteSearch(pattern string, searchOptions SearchOptions) {
	e.state.SearchQuery.Pattern = pattern
	query := pattern

	ignoreCase := searchOptions.IgnoreCase
	smartCase := searchOptions.SmartCase

	if before, ok := strings.CutSuffix(pattern, "\\c"); ok {
		ignoreCase = true
		smartCase = false
		query = before
	} else if before, ok := strings.CutSuffix(pattern, "\\C"); ok {
		ignoreCase = false
		smartCase = false
		query = before
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

	if found {
		e.onSearchResultFound(pos)
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
	e.state.SearchOptions = SearchOptions{}
	e.state.SearchResults = []Position{}
	e.state.SearchResultIndex = -1
	e.setMode(e.state.PreviousMode)
}

// NextSearchResult repeats the search in the same direction as the original search (vim 'n').
func (e *editor) NextSearchResult() Cursor {
	return e.repeatSearch(false)
}

// PreviousSearchResult repeats the search in the opposite direction of the original search (vim 'N').
func (e *editor) PreviousSearchResult() Cursor {
	return e.repeatSearch(true)
}

// repeatSearch repeats the last search from the cursor, in the original direction
// or, with flip, the opposite one.
func (e *editor) repeatSearch(flip bool) Cursor {
	if len(e.state.SearchResults) == 0 {
		return e.buffer.GetCursor()
	}

	options := e.state.SearchOptions
	if flip {
		options.Backwards = !options.Backwards
	}

	pos, found := e.buffer.Find(e.state.SearchQuery.Term, e.buffer.GetCursor().Position, options)
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

	// iw selects the word without the blanks around it; on a word character
	// it stays on the line.
	start, end, _, _ := currentWord(e.buffer, pos, pos, false, 1, false, e.IsWordChar)
	word := string(lineRunes[start.Col : end.Col+1])

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
