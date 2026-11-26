package highlighter

import (
	"strings"
	"sync"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter handles syntax highlighting for the editor
type Highlighter struct {
	lexer           chroma.Lexer
	style           *chroma.Style
	cache           map[int][]chroma.Token // Cache tokens by line number
	styleCache      map[chroma.TokenType]lipgloss.Style
	cacheMutex      sync.RWMutex
	styleCacheMutex sync.RWMutex
}

// TokenPosition represents a token's position in the original line
type TokenPosition struct {
	Token    chroma.Token
	StartCol int
	EndCol   int
}

// New creates a new syntax highlighter
func New(language string, theme string) *Highlighter {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = chroma.Coalesce(lexer)

	style := styles.Get(theme)

	return &Highlighter{
		lexer:      lexer,
		style:      style,
		cache:      make(map[int][]chroma.Token),
		styleCache: make(map[chroma.TokenType]lipgloss.Style),
	}
}

// InvalidateCache clears the token cache (call when content changes).
// The style cache depends only on the theme, which is fixed for the
// lifetime of a Highlighter, so it is left intact.
func (sh *Highlighter) InvalidateCache() {
	sh.cacheMutex.Lock()
	defer sh.cacheMutex.Unlock()
	sh.cache = make(map[int][]chroma.Token)
}

// Tokenise tokenises only the visible range of lines.
// Optimised to skip re-tokenisation if all lines are already cached.
// The range is widened so it doesn't start or end inside a Markdown code
// block, which the lexer can only read from its opening fence.
func (sh *Highlighter) Tokenise(lines []string, startLine, endLine int) {
	sh.cacheMutex.Lock()
	defer sh.cacheMutex.Unlock()

	if startLine < 0 || endLine > len(lines) || startLine >= endLine {
		return
	}
	startLine, endLine = sh.syncRange(lines, startLine, endLine)

	// Check if all lines are already cached
	allCached := true
	for i := startLine; i < endLine; i++ {
		if _, exists := sh.cache[i]; !exists {
			allCached = false
			break
		}
	}

	// If everything is cached, skip tokenisation
	if allCached {
		return
	}

	// Clear cache for the range we're about to tokenise
	for i := startLine; i < endLine; i++ {
		delete(sh.cache, i)
	}

	sh.tokeniseRange(lines, startLine, endLine)
}

// syncRange widens the range of lines [start, end) to lex so that it starts
// and ends outside the constructs the lexer can only read from their first
// line, as Vim's :syntax sync. The lexer starts in its root state at start,
// so in a Markdown code block it would read the closing fence as an opening
// one and colour everything after it as the wrong side of a fence. Only
// Markdown's fenced code blocks are handled; other languages get the range
// back as it is.
func (sh *Highlighter) syncRange(lines []string, start, end int) (int, int) {
	if sh.lexer.Config().Name != "markdown" {
		return start, end
	}
	return markdownSyncRange(lines, start, end)
}

// markdownSyncRange moves start to the opening fence of the code block it is
// in, and end past the closing fence of the block it cuts, if any. Fences
// are read as chroma's Markdown lexer reads them: an opening line is ```
// followed by nothing or by a language name, and the block ends at the next
// line that is ``` alone; an opening line with no such line after it starts
// no block.
func markdownSyncRange(lines []string, start, end int) (int, int) {
	newStart, newEnd := start, end
	for i := 0; i < end && i < len(lines); i++ {
		if !isFenceOpen(lines[i]) {
			continue
		}
		j := i + 1
		for j < len(lines) && lines[j] != "```" {
			j++
		}
		if j == len(lines) {
			break // no closing fence below, so no later opening line has one
		}
		if i < start && j >= start {
			newStart = i
		}
		if j >= end {
			newEnd = j + 1
		}
		i = j
	}
	return newStart, newEnd
}

// isFenceOpen reports whether line opens a code block in chroma's Markdown
// lexer: ``` followed by word characters only.
func isFenceOpen(line string) bool {
	rest, ok := strings.CutPrefix(line, "```")
	if !ok {
		return false
	}
	for _, r := range rest {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// tokeniseRange tokenises a specific range of lines and updates the cache
func (sh *Highlighter) tokeniseRange(lines []string, startLine, endLine int) {
	// Join only the lines in this range
	content := strings.Join(lines[startLine:endLine], "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content == "" {
		return
	}

	iterator, err := sh.lexer.Tokenise(nil, content)
	if err != nil {
		for i := startLine; i < endLine; i++ {
			sh.cache[i] = []chroma.Token{}
		}
		return
	}

	tokens := iterator.Tokens()
	lineNum := startLine
	sh.cache[lineNum] = []chroma.Token{}

	for _, token := range tokens {
		value := token.Value
		for strings.Contains(value, "\n") {
			before, after, _ := strings.Cut(value, "\n")
			if before != "" {
				sh.cache[lineNum] = append(sh.cache[lineNum], chroma.Token{Type: token.Type, Value: before})
			}
			lineNum++
			// The newline after the last line ends the range. Writing an
			// entry for endLine would mark it as tokenised (or wipe its
			// tokens), so a later Tokenise would skip it.
			if lineNum >= endLine {
				return
			}
			sh.cache[lineNum] = []chroma.Token{}
			value = after
		}
		if value != "" {
			sh.cache[lineNum] = append(sh.cache[lineNum], chroma.Token{Type: token.Type, Value: value})
		}
	}
}

// GetTokensForLine returns syntax tokens for a specific line.
func (sh *Highlighter) GetTokensForLine(lineNum int, lines []string) []chroma.Token {
	sh.cacheMutex.RLock()
	defer sh.cacheMutex.RUnlock()
	if tokens, ok := sh.cache[lineNum]; ok {
		return tokens
	}

	return nil
}

// GetStyleForToken converts a Chroma token type to a lipgloss style.
// Thread-safe with read-write lock for cache access.
func (sh *Highlighter) GetStyleForToken(tokenType chroma.TokenType) lipgloss.Style {
	// Try read lock first (fast path for cached styles)
	sh.styleCacheMutex.RLock()
	if style, ok := sh.styleCache[tokenType]; ok {
		sh.styleCacheMutex.RUnlock()
		return style
	}
	sh.styleCacheMutex.RUnlock()

	// Compute style (slow path)
	entry := sh.style.Get(tokenType)

	style := lipgloss.NewStyle()
	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	}

	if entry.Bold == chroma.Yes {
		style = style.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		style = style.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		style = style.Underline(true)
	}

	// Write lock to update cache
	sh.styleCacheMutex.Lock()
	sh.styleCache[tokenType] = style
	sh.styleCacheMutex.Unlock()

	return style
}

// GetTokenPositions converts tokens to positions in the logical line.
// Token positions are rune-based (not visual-width based) to match cursor positions.
func GetTokenPositions(tokens []chroma.Token) []TokenPosition {
	positions := make([]TokenPosition, 0, len(tokens))
	runeCol := 0 // Keep track of rune column for FindTokenAtPosition

	for _, token := range tokens {
		tokenRuneLen := len([]rune(token.Value))

		positions = append(positions, TokenPosition{
			Token:    token,
			StartCol: runeCol,
			EndCol:   runeCol + tokenRuneLen,
		})

		runeCol += tokenRuneLen
	}

	return positions
}

// FindTokenAtPosition finds which token contains the given column position.
// This function MUST use rune-based columns to match renderSegment's `currentLogicalCharCol`.
func FindTokenAtPosition(positions []TokenPosition, col int) (chroma.Token, bool) {
	for _, pos := range positions {
		if col >= pos.StartCol && col < pos.EndCol {
			return pos.Token, true
		}
	}
	return chroma.Token{}, false
}
