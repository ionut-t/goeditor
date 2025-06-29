package highlighter

import (
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// Highlighter handles syntax highlighting for the editor
type Highlighter struct {
	lexer      chroma.Lexer
	style      *chroma.Style
	cache      map[int][]chroma.Token // Cache tokens by line number
	styleCache map[chroma.TokenType]lipgloss.Style
	cacheMutex sync.RWMutex
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

// InvalidateCache clears the token cache (call when content changes)
func (sh *Highlighter) InvalidateCache() {
	sh.cacheMutex.Lock()
	defer sh.cacheMutex.Unlock()
	sh.cache = make(map[int][]chroma.Token)
	sh.styleCache = make(map[chroma.TokenType]lipgloss.Style)
}

// InvalidateLine clears the cache for a specific line number.
func (sh *Highlighter) InvalidateLine(lineNum int) {
	sh.cacheMutex.Lock()
	defer sh.cacheMutex.Unlock()
	delete(sh.cache, lineNum)
}

// Tokenize tokenizes the entire content and populates the cache.
// This is necessary for languages like Markdown that have multi-line structures.
/* TODO: Optimize to only tokenize changed lines/current line/selection or to render fewer lines for large files */
func (sh *Highlighter) Tokenize(lines []string) {
	sh.cacheMutex.Lock()
	defer sh.cacheMutex.Unlock()

	// Clear existing cache
	sh.cache = make(map[int][]chroma.Token)

	content := strings.Join(lines, "\n")
	if content == "" {
		return
	}

	iterator, err := sh.lexer.Tokenise(nil, content)
	if err != nil {
		// On error, cache empty tokens to avoid re-tokenizing on every render
		for i := range lines {
			sh.cache[i] = []chroma.Token{}
		}
		return
	}

	tokens := iterator.Tokens()
	lineNum := 0
	sh.cache[lineNum] = []chroma.Token{}

	for _, token := range tokens {
		value := token.Value
		for strings.Contains(value, "\n") {
			before, after, _ := strings.Cut(value, "\n")
			if before != "" {
				sh.cache[lineNum] = append(sh.cache[lineNum], chroma.Token{Type: token.Type, Value: before})
			}
			lineNum++
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
	// If cache is empty, we need to tokenize. We check for line 0 as a heuristic.
	_, cached := sh.cache[0]
	sh.cacheMutex.RUnlock()

	if !cached {
		sh.Tokenize(lines)
	}

	sh.cacheMutex.RLock()
	defer sh.cacheMutex.RUnlock()
	if tokens, ok := sh.cache[lineNum]; ok {
		return tokens
	}

	return nil
}

// GetStyleForToken converts a Chroma token type to a lipgloss style.
func (sh *Highlighter) GetStyleForToken(tokenType chroma.TokenType) lipgloss.Style {
	if style, ok := sh.styleCache[tokenType]; ok {
		return style
	}

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

	sh.styleCache[tokenType] = style

	return style
}

// GetTokenPositions converts tokens to positions in the logical line.
func GetTokenPositions(tokens []chroma.Token) []TokenPosition {
	positions := make([]TokenPosition, 0, len(tokens))
	currentCol := 0

	for _, token := range tokens {
		tokenRunes := []rune(token.Value)
		tokenLen := len(tokenRunes)

		positions = append(positions, TokenPosition{
			Token:    token,
			StartCol: currentCol,
			EndCol:   currentCol + tokenLen,
		})

		currentCol += tokenLen
	}

	return positions
}

// FindTokenAtPosition finds which token contains the given column position.
func FindTokenAtPosition(positions []TokenPosition, col int) (chroma.Token, bool) {
	for _, pos := range positions {
		if col >= pos.StartCol && col < pos.EndCol {
			return pos.Token, true
		}
	}
	return chroma.Token{}, false
}
