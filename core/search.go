package core

import (
	"slices"
	"strings"
)

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
	// Other fields (Backwards, Wrap, WholeWord, CharClass) are taken directly from searchOptions;
	// without a CharClass the boundaries use the editor's.
	charClass := searchOptions.CharClass
	if charClass == nil {
		charClass = e.CharClass
	}
	e.state.SearchOptions = SearchOptions{
		IgnoreCase:        ignoreCase,
		SmartCase:         smartCase,
		Backwards:         searchOptions.Backwards,
		Wrap:              searchOptions.Wrap,
		WholeWord:         searchOptions.WholeWord,
		WordBoundaryStart: wordBoundaryStart,
		WordBoundaryEnd:   wordBoundaryEnd,
		CharClass:         charClass,
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

// SearchResults returns the positions of the last search's matches that the
// pattern still matches: an edit since the search drops the ones it changed
// (Vim's 'hlsearch' re-matches on every redraw), while n and N still search for
// the pattern.
func (e *editor) SearchResults() []Position {
	results := e.state.SearchResults
	for i, pos := range results {
		if !e.searchMatchesAt(pos) {
			kept := slices.Clone(results[:i])
			for _, p := range results[i+1:] {
				if e.searchMatchesAt(p) {
					kept = append(kept, p)
				}
			}
			return kept
		}
	}
	return results
}

// searchMatchesAt reports whether the last search's term still matches at pos.
func (e *editor) searchMatchesAt(pos Position) bool {
	if pos.Row < 0 || pos.Row >= e.buffer.LineCount() {
		return false
	}
	return matchesAt(e.buffer.GetLineRunes(pos.Row), e.state.SearchQuery.Term, pos.Col, e.state.SearchOptions)
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
	start, end, _, _ := currentWord(e.buffer, pos, pos, false, 1, false, e.CharClass)
	word := string(lineRunes[start.Col:charEnd(lineRunes, end.Col)])

	e.ExecuteSearch(word, SearchOptions{
		Backwards: backwards,
		Wrap:      true,
		WholeWord: true,
		CharClass: e.CharClass,
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
