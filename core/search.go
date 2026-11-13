package core

import (
	"fmt"
	"slices"
	"strings"
)

// ExecuteSearch searches for pattern from the cursor and keeps it for n and N.
// An empty pattern reuses the last search, as Vim's "/<CR>" and "?<CR>": its
// pattern and matching options (case, whole word, \< and \>) with the
// direction and Wrap of searchOptions, so n and N then go the new way. With
// no last search it dispatches ErrNoPreviousPattern (Vim's E35) and moves
// nothing.
func (e *editor) ExecuteSearch(pattern string, searchOptions SearchOptions) {
	if pattern != "" {
		e.runSearch(e.parseSearch(pattern, searchOptions))
	} else if e.state.SearchQuery.Pattern != "" {
		options := e.state.SearchOptions
		options.Backwards = searchOptions.Backwards
		options.Wrap = searchOptions.Wrap
		e.runSearch(e.state.SearchQuery, options)
	} else {
		if e.state.Mode == SearchMode {
			e.setMode(e.state.PreviousMode)
		}
		e.DispatchError(ErrNoPreviousPatternId, ErrNoPreviousPattern)
	}
}

// runSearch keeps query and options as the last search, for n, N and an
// empty search, and searches from the cursor. A search typed on the search
// line goes back to the mode it started from. The search is left on the
// command line (see echoSearch).
func (e *editor) runSearch(query SearchQuery, options SearchOptions) {
	e.state.SearchQuery = query
	e.state.SearchOptions = options

	pos, found := e.buffer.Find(query.Term, e.buffer.GetCursor().Position, options)

	if found {
		e.onSearchResultFound(pos)
	} else {
		e.state.SearchResults = []Position{}
		e.state.SearchResultIndex = -1
		e.dispatchNotFound(options)
	}

	if e.state.Mode == SearchMode {
		e.setMode(e.state.PreviousMode)
	}
	// After the mode change, which clears it.
	e.echoSearch(options.Backwards)
	e.DispatchSignal(SearchResultsSignal{positions: e.state.SearchResults})
}

// parseSearch splits pattern's inline modifiers (\c, \C, \<, \>) from the
// text to find. The modifiers take precedence over searchOptions; the other
// fields (Backwards, Wrap, WholeWord, CharClass) are taken from searchOptions,
// and without a CharClass the boundaries use the editor's.
func (e *editor) parseSearch(pattern string, searchOptions SearchOptions) (SearchQuery, SearchOptions) {
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

	charClass := searchOptions.CharClass
	if charClass == nil {
		charClass = e.CharClass
	}
	return SearchQuery{Pattern: pattern, Term: query}, SearchOptions{
		IgnoreCase:        ignoreCase,
		SmartCase:         smartCase,
		Backwards:         searchOptions.Backwards,
		Wrap:              searchOptions.Wrap,
		WholeWord:         searchOptions.WholeWord,
		WordBoundaryStart: wordBoundaryStart,
		WordBoundaryEnd:   wordBoundaryEnd,
		CharClass:         charClass,
	}
}

func (e *editor) SetSearchOptions(options SearchOptions) {
	e.searchOptions = SearchOptions{
		IgnoreCase: options.IgnoreCase,
		SmartCase:  options.SmartCase,
		Wrap:       options.Wrap,
	}
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
// or, with flip, the opposite one. As Vim's, it searches again after a search
// that found nothing, since an edit may have added a match. With no last
// search it dispatches ErrNoPreviousPattern and moves nothing.
func (e *editor) repeatSearch(flip bool) Cursor {
	if e.state.SearchQuery.Pattern == "" {
		e.DispatchError(ErrNoPreviousPatternId, ErrNoPreviousPattern)
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
	} else {
		e.dispatchNotFound(options)
	}
	e.echoSearch(options.Backwards)

	// The matches the pattern still matches: a failed n keeps the last match
	// while it is there (n past the last one with Wrap off), as Vim's
	// highlight, and drops it once an edit changed it.
	e.DispatchSignal(SearchResultsSignal{positions: e.SearchResults()})

	return e.buffer.GetCursor()
}

// dispatchNotFound reports a search that found no match, as Vim: "pattern
// not found: bar" (ErrPatternNotFound), or with Wrap off "search hit BOTTOM
// without match for: bar" (ErrSearchHitBottom) going forward and
// ErrSearchHitTop going backward.
func (e *editor) dispatchNotFound(options SearchOptions) {
	pattern := e.state.SearchQuery.Pattern
	switch {
	case options.Wrap:
		e.DispatchError(ErrPatternNotFoundId, fmt.Errorf("%w: %s", ErrPatternNotFound, pattern))
	case options.Backwards:
		e.DispatchError(ErrSearchHitTopId, fmt.Errorf("%w for: %s", ErrSearchHitTop, pattern))
	default:
		e.DispatchError(ErrSearchHitBottomId, fmt.Errorf("%w for: %s", ErrSearchHitBottom, pattern))
	}
}

// echoSearch leaves the last search on the command line (State.SearchEcho)
// with the direction it just went, as Vim shows it after every search, n,
// N, * and #: "/bar", "?bar" for N after "/bar", "/\<foo\>" for *.
func (e *editor) echoSearch(backwards bool) {
	prefix := "/"
	if backwards {
		prefix = "?"
	}
	e.state.SearchEcho = prefix + e.state.SearchQuery.Pattern
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

// SearchWordUnderCursor implements vim's * (forward) and # (backward): it
// searches for the keyword under or after the cursor as a whole word or, with
// none on the rest of the line, for the non-blank text there as it is (see
// starTarget). As Vim's, the search starts from the start of that text, and
// the last search's pattern is the one Vim keeps ("\<foo\>", or "++"). With
// only blanks from the cursor to the end of the line it dispatches
// ErrNoStringUnderCursor (Vim's E348) and moves nothing.
func (e *editor) SearchWordUnderCursor(backwards bool) Cursor {
	cursor := e.buffer.GetCursor()
	line := e.buffer.GetLineRunes(cursor.Position.Row)
	start, end, keyword, ok := starTarget(line, cursor.Position.Col, e.CharClass)
	if !ok {
		e.DispatchError(ErrNoStringUnderCursorId, ErrNoStringUnderCursor)
		return cursor
	}
	text := string(line[start:end])
	query := SearchQuery{Pattern: text, Term: text}
	if keyword {
		query.Pattern = `\<` + text + `\>`
	}

	from := cursor
	from.Position.Col = start
	e.buffer.SetCursor(from)
	e.runSearch(query, SearchOptions{
		Backwards: backwards,
		Wrap:      true,
		WholeWord: keyword,
		CharClass: e.CharClass,
	})
	if len(e.state.SearchResults) == 0 {
		e.buffer.SetCursor(cursor)
	}
	e.ScrollViewport()

	return e.buffer.GetCursor()
}

// starTarget returns the columns of the text * and # search for on line, from
// col, as Vim's find_ident_under_cursor: the keyword under the cursor or the
// first one after it on the line (a run of one word class, keyword true);
// with none, the non-blank text under or after the cursor, from the start of
// the punctuation it is in up to the next blank. ok is false when the line
// from col on is blank.
func starTarget(line []rune, col int, cls func(rune) int) (start, end int, keyword, ok bool) {
	col = charStart(line, col)
	i := col
	for i < len(line) && cls(line[i]) < wordCharClass {
		i = charEnd(line, i)
	}
	keyword = i < len(line)
	if !keyword {
		for i = col; i < len(line) && cls(line[i]) == blankClass; i = charEnd(line, i) {
		}
		if i >= len(line) {
			return 0, 0, false, false
		}
	}
	class := cls(line[i])
	start = i
	for start > 0 {
		prev := charStart(line, start-1)
		if cls(line[prev]) != class {
			break
		}
		start = prev
	}
	end = i
	for end < len(line) && (keyword && cls(line[end]) == class || !keyword && cls(line[end]) != blankClass) {
		end = charEnd(line, end)
	}
	return start, end, keyword, true
}

func (e *editor) onSearchResultFound(pos Position) {
	e.state.SearchResults = []Position{pos}
	e.state.SearchResultIndex = 0
	cursor := e.buffer.GetCursor()
	cursor.Position = pos
	e.buffer.SetCursor(cursor)
}
