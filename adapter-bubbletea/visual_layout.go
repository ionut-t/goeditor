package adapter_bubbletea

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/goeditor/adapter-bubbletea/highlighter"
	editor "github.com/ionut-t/goeditor/core"
	"github.com/rivo/uniseg"
)

// getVisualWidth calculates the visual width of a string, properly handling
// grapheme clusters (e.g., emojis with variation selectors, combining characters) and tabs.
// Tabs are expanded to the next tab stop (multiples of 4).
func getVisualWidth(s string) int {
	return getVisualWidthAt(s, 0)
}

// getVisualWidthAt calculates the visual width of a string starting at a given column position.
// This is necessary for proper tab width calculation, as tabs expand to the next tab stop.
func getVisualWidthAt(s string, startCol int) int {
	const tabWidth = 4
	width := 0
	currentCol := startCol
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		grapheme := gr.Str()
		if grapheme == "\t" {
			// Calculate spaces needed to reach next tab stop
			spacesToNextTabStop := tabWidth - (currentCol % tabWidth)
			width += spacesToNextTabStop
			currentCol += spacesToNextTabStop
		} else {
			graphemeWidth := uniseg.StringWidth(grapheme)
			width += graphemeWidth
			currentCol += graphemeWidth
		}
	}
	return width
}

// getRuneVisualWidth calculates the visual width of a single rune.
// Variation selectors and other combining marks should return 0 width.
func getRuneVisualWidth(r rune) int {
	// Variation selectors (FE00-FE0F, E0100-E01EF) should have 0 width
	if r >= 0xFE00 && r <= 0xFE0F {
		return 0
	}
	if r >= 0xE0100 && r <= 0xE01EF {
		return 0
	}
	// Zero-width joiner
	if r == 0x200D {
		return 0
	}
	// Combining marks (0300-036F)
	if r >= 0x0300 && r <= 0x036F {
		return 0
	}
	return uniseg.StringWidth(string(r))
}

// nextGrapheme returns the next grapheme cluster starting at the given rune index.
// Returns the grapheme string, its visual width, and the number of runes consumed.
// This centralises grapheme iteration logic to eliminate redundancy across rendering functions.
// The currentCol parameter is used for proper tab width calculation.
func nextGrapheme(runes []rune, startIdx int, currentCol int) (graphemeStr string, visualWidth int, runesConsumed int) {
	const tabWidth = 4

	if startIdx >= len(runes) {
		return "", 0, 0
	}

	// Use uniseg to properly identify the grapheme cluster boundary
	remaining := string(runes[startIdx:])
	gr := uniseg.NewGraphemes(remaining)

	if !gr.Next() {
		// Fallback: treat single rune as grapheme if uniseg fails
		graphemeStr = string(runes[startIdx])
		if graphemeStr == "\t" {
			visualWidth = tabWidth - (currentCol % tabWidth)
		} else {
			visualWidth = getRuneVisualWidth(runes[startIdx])
		}
		return graphemeStr, visualWidth, 1
	}

	graphemeStr = gr.Str()
	if graphemeStr == "\t" {
		// Tab width depends on current column position
		visualWidth = tabWidth - (currentCol % tabWidth)
	} else {
		visualWidth = uniseg.StringWidth(graphemeStr)
	}
	runesConsumed = len([]rune(graphemeStr))

	return graphemeStr, visualWidth, runesConsumed
}

// calculateCursorScreenCol calculates the cursor's screen column position.
// Returns the screen column (including line number width) for the cursor within the given visual line segment.
func (m *Model) calculateCursorScreenCol(vli VisualLineInfo, lineNumWidth int) int {
	visualColInSegmentRuneOffset := max(0, m.clampedCursorLogicalCol-vli.LogicalStartCol)
	segmentRunes := []rune(vli.Content)

	if visualColInSegmentRuneOffset > len(segmentRunes) {
		visualColInSegmentRuneOffset = len(segmentRunes)
	}

	substringToCursor := string(segmentRunes[0:visualColInSegmentRuneOffset])
	visualColInSegmentWidth := getVisualWidth(substringToCursor)
	return lineNumWidth + visualColInSegmentWidth
}

type VisualLineInfo struct {
	Content         string
	LogicalRow      int
	LogicalStartCol int
	IsFirstSegment  bool
}

// calculateLineNumberWidth computes the width needed for line numbers
func (m *Model) calculateLineNumberWidth(totalLines int) int {
	if !m.showLineNumbers {
		return 0
	}

	state := m.editor.GetState()
	maxWidth := len(strconv.Itoa(max(1, totalLines)))

	if state.RelativeNumbers && !m.disableVimMode {
		relWidth := len(strconv.Itoa(max(1, m.viewport.Height)))
		maxWidth = max(maxWidth, relWidth)
	}

	lineNumWidth := max(4, maxWidth) + 1
	return min(lineNumWidth, 10)
}

// isPositionInSearchResult checks if a position is part of a search result
// Uses binary search for O(log n) performance instead of O(n)
func (m *Model) isPositionInSearchResult(pos editor.Position, col int) bool {
	searchTerm := m.editor.GetState().SearchQuery.Term
	if searchTerm == "" {
		return false
	}

	results := m.editor.SearchResults()
	if len(results) == 0 {
		return false
	}

	termLen := len(searchTerm)

	// Binary search to find the first result with row >= pos.Row
	left, right := 0, len(results)
	for left < right {
		mid := (left + right) / 2
		if results[mid].Row < pos.Row {
			left = mid + 1
		} else {
			right = mid
		}
	}

	// Check all results on the same row (usually very few)
	for i := left; i < len(results) && results[i].Row == pos.Row; i++ {
		if col >= results[i].Col && col < results[i].Col+termLen {
			return true
		}
	}

	return false
}

// highlightedWordMatch represents a match for a highlighted word
type highlightedWordMatch struct {
	length int
	style  lipgloss.Style
}

// highlightedWordPattern caches the rune conversion for each highlighted word
type highlightedWordPattern struct {
	runes []rune
	style lipgloss.Style
}

// hashHighlightedWords computes a hash of the highlighted words map
func (m *Model) hashHighlightedWords() uint64 {
	if len(m.highlightedWords) == 0 {
		return 0
	}

	// Hash all words in the map
	hash := uint64(len(m.highlightedWords))
	for word := range m.highlightedWords {
		for _, r := range word {
			hash = hash*31 + uint64(r)
		}
		// Also incorporate word count to ensure different maps hash differently
		hash = hash * 37
	}
	return hash
}

// getCompiledHighlightedWords returns cached compiled patterns, updating cache if needed
func (m *Model) getCompiledHighlightedWords() []highlightedWordPattern {
	if len(m.highlightedWords) == 0 {
		m.compiledHighlightedWords = nil
		m.compiledHighlightedWordsHash = 0
		return nil
	}

	// Check if cache is valid
	currentHash := m.hashHighlightedWords()
	if m.compiledHighlightedWordsHash == currentHash && m.compiledHighlightedWords != nil {
		return m.compiledHighlightedWords
	}

	// Recompile patterns
	patterns := make([]highlightedWordPattern, 0, len(m.highlightedWords))
	for word, style := range m.highlightedWords {
		patterns = append(patterns, highlightedWordPattern{
			runes: []rune(word),
			style: style,
		})
	}

	m.compiledHighlightedWords = patterns
	m.compiledHighlightedWordsHash = currentHash
	return patterns
}

// findHighlightedWordMatch finds the longest highlighted word match at the current position
// Returns a highlightedWordMatch with length 0 if no match is found
func (m *Model) findHighlightedWordMatch(segmentRunes []rune, charIdx int) highlightedWordMatch {
	if len(m.highlightedWords) == 0 {
		return highlightedWordMatch{}
	}

	segmentLen := len(segmentRunes)
	bestMatch := highlightedWordMatch{}

	// Get cached compiled patterns (avoids repeated rune conversions)
	patterns := m.getCompiledHighlightedWords()

	for _, pattern := range patterns {
		wordLen := len(pattern.runes)

		if wordLen == 0 || charIdx+wordLen > segmentLen {
			continue
		}

		// Check if runes match
		match := true
		for k := range wordLen {
			if segmentRunes[charIdx+k] != pattern.runes[k] {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		// Whole word boundary check
		isWholeWord := true

		// Check character before the match
		if charIdx > 0 {
			prevChar := segmentRunes[charIdx-1]
			if unicode.IsLetter(prevChar) || unicode.IsDigit(prevChar) {
				isWholeWord = false
			}
		}

		// Check character after the match
		if charIdx+wordLen < segmentLen {
			nextChar := segmentRunes[charIdx+wordLen]
			if unicode.IsLetter(nextChar) || unicode.IsDigit(nextChar) {
				isWholeWord = false
			}
		}

		if isWholeWord && wordLen > bestMatch.length {
			bestMatch = highlightedWordMatch{
				length: wordLen,
				style:  pattern.style,
			}
		}
	}

	return bestMatch
}

// clampCursorRow clamps the cursor row to valid buffer bounds
func (m *Model) clampCursorRow(cursorRow int, totalLines int) int {
	if cursorRow < 0 {
		return 0
	}
	if totalLines == 0 {
		return 0
	}
	if cursorRow >= totalLines {
		return totalLines - 1
	}
	return cursorRow
}

// calculateFullVisualLayout computes layout for entire buffer (small files)
func (m *Model) calculateFullVisualLayout(allLogicalLines []string, availableWidth int) {
	visualLayout := make([]VisualLineInfo, 0, len(allLogicalLines)*2)

	for bufferRowIdx, logicalLineContent := range allLogicalLines {
		m.appendVisualLayoutForLine(bufferRowIdx, logicalLineContent, availableWidth, &visualLayout)
	}

	m.visualLayoutCache = visualLayout
	m.visualLayoutCacheStartRow = 0       // Full layout starts at row 0
	m.visualLayoutCacheStartVisualRow = 0 // Full layout starts at visual row 0
	m.fullVisualLayoutHeight = len(visualLayout)
}

// calculateLazyVisualLayout computes layout only for visible region (large files)
func (m *Model) calculateLazyVisualLayout(allLogicalLines []string, cursor editor.Cursor, availableWidth int, viewportBuffer int) {
	totalLines := len(allLogicalLines)
	cursorLogicalRow := max(0, min(cursor.Position.Row, totalLines-1))

	// Initialise anchors map if needed
	if m.visualRowAnchors == nil {
		m.visualRowAnchors = make(map[int]int)
	}

	// Clear anchors if content changed (line count different)
	if m.lastKnownLineCount != totalLines {
		m.visualRowAnchors = make(map[int]int)
		m.lastKnownLineCount = totalLines
		// Invalidate cache validity range
		m.cacheValidStartRow = 0
		m.cacheValidEndRow = 0
	}

	// Cache validity check: only recalculate if cursor is approaching cache boundaries
	// This prevents unnecessary recalculation on every keystroke during scrolling
	const cacheHysteresis = 20 // Don't recalculate unless within 20 lines of edge
	if len(m.visualLayoutCache) > 0 &&
		cursorLogicalRow >= m.cacheValidStartRow+cacheHysteresis &&
		cursorLogicalRow <= m.cacheValidEndRow-cacheHysteresis {
		// Cursor is well within valid range - no need to recalculate
		return
	}

	// Calculate wrapping factor from previous cache if available
	avgVisualLinesPerLogical := 1.5 // Default: assume some wrapping
	if len(m.visualLayoutCache) > 0 {
		// Count unique logical lines in current cache
		uniqueLogicalLines := 0
		lastLogicalRow := -1
		for _, vli := range m.visualLayoutCache {
			if vli.LogicalRow != lastLogicalRow {
				uniqueLogicalLines++
				lastLogicalRow = vli.LogicalRow
			}
		}
		if uniqueLogicalLines > 0 {
			avgVisualLinesPerLogical = float64(len(m.visualLayoutCache)) / float64(uniqueLogicalLines)
		}
	}

	// Use a larger buffer for better accuracy
	viewportHeight := m.viewport.Height
	largerBuffer := viewportBuffer * 2

	// Estimate logical lines needed to fill visual viewport + buffer
	estimatedLogicalLinesNeeded := max(int(float64(viewportHeight+largerBuffer)/avgVisualLinesPerLogical), viewportHeight*2)

	// Center cache around cursor position
	halfRange := estimatedLogicalLinesNeeded / 2
	startLine := max(0, cursorLogicalRow-halfRange)
	endLine := min(totalLines, cursorLogicalRow+halfRange)

	// Ensure we have the full estimated range
	if endLine-startLine < estimatedLogicalLinesNeeded {
		if startLine > 0 {
			startLine = max(0, endLine-estimatedLogicalLinesNeeded)
		}
		if endLine < totalLines {
			endLine = min(totalLines, startLine+estimatedLogicalLinesNeeded)
		}
	}

	// Calculate visual row offset using anchors for better accuracy
	var visualRowOffset int
	if anchorVisualRow, exists := m.visualRowAnchors[startLine]; exists {
		// We have an exact anchor for this start line
		visualRowOffset = anchorVisualRow
	} else {
		// Find the closest anchor before startLine
		closestLogical := -1
		closestVisual := -1
		for logicalRow, visualRow := range m.visualRowAnchors {
			if logicalRow < startLine && logicalRow > closestLogical {
				closestLogical = logicalRow
				closestVisual = visualRow
			}
		}

		if closestLogical >= 0 {
			// Interpolate from closest anchor
			gap := startLine - closestLogical
			visualRowOffset = closestVisual + int(avgVisualLinesPerLogical*float64(gap))
		} else {
			// No anchors before, use simple estimation
			visualRowOffset = int(avgVisualLinesPerLogical * float64(startLine))
		}
	}

	// Build visual layout for the cached range and update anchors
	visualLayout := make([]VisualLineInfo, 0, (endLine-startLine)*2)
	currentVisualRow := visualRowOffset

	// Adaptive anchor interval: more anchors for smaller files = better accuracy
	anchorInterval := 50
	if totalLines < 500 {
		anchorInterval = 20 // Dense anchors for medium files reduce estimation errors
	}

	for bufferRowIdx := startLine; bufferRowIdx < endLine; bufferRowIdx++ {
		// Store anchors periodically for progressive accuracy
		if bufferRowIdx%anchorInterval == 0 {
			m.visualRowAnchors[bufferRowIdx] = currentVisualRow
		}

		layoutBefore := len(visualLayout)
		m.appendVisualLayoutForLine(bufferRowIdx, allLogicalLines[bufferRowIdx], availableWidth, &visualLayout)
		layoutAfter := len(visualLayout)

		currentVisualRow += (layoutAfter - layoutBefore)
	}

	m.visualLayoutCache = visualLayout
	m.visualLayoutCacheStartRow = startLine
	m.visualLayoutCacheStartVisualRow = visualRowOffset

	// Update cache validity range for hysteresis check
	// Cache is valid for cursor positions within [startLine, endLine]
	m.cacheValidStartRow = startLine
	m.cacheValidEndRow = endLine

	// Estimate full visual height using best available anchor
	lastAnchorLogical := -1
	lastAnchorVisual := -1
	for logicalRow, visualRow := range m.visualRowAnchors {
		if logicalRow > lastAnchorLogical {
			lastAnchorLogical = logicalRow
			lastAnchorVisual = visualRow
		}
	}

	if lastAnchorLogical >= 0 && lastAnchorLogical < totalLines {
		remaining := totalLines - lastAnchorLogical
		m.fullVisualLayoutHeight = lastAnchorVisual + int(avgVisualLinesPerLogical*float64(remaining))
	} else {
		m.fullVisualLayoutHeight = int(avgVisualLinesPerLogical * float64(totalLines))
	}
}

// appendVisualLayoutForLine wraps a single logical line and appends to visual layout
func (m *Model) appendVisualLayoutForLine(bufferRowIdx int, logicalLineContent string, availableWidth int, visualLayout *[]VisualLineInfo) {
	originalLineRunes := []rune(logicalLineContent)
	originalLineLen := len(originalLineRunes)
	currentLogicalColToReport := 0

	if originalLineLen == 0 && logicalLineContent == "" {
		*visualLayout = append(*visualLayout, VisualLineInfo{
			Content:         "",
			LogicalRow:      bufferRowIdx,
			LogicalStartCol: 0,
			IsFirstSegment:  true,
		})
		return
	}

	wrappedSegmentStrings := wrapLine(logicalLineContent, availableWidth)

	for segIdx, segmentStr := range wrappedSegmentStrings {
		segmentRunes := []rune(segmentStr)
		segmentRunesLen := len(segmentRunes)

		info := VisualLineInfo{
			Content:         segmentStr,
			LogicalRow:      bufferRowIdx,
			LogicalStartCol: currentLogicalColToReport,
			IsFirstSegment:  segIdx == 0,
		}
		*visualLayout = append(*visualLayout, info)

		currentLogicalColToReport += segmentRunesLen
		if segIdx < len(wrappedSegmentStrings)-1 {
			for currentLogicalColToReport < originalLineLen && unicode.IsSpace(originalLineRunes[currentLogicalColToReport]) {
				currentLogicalColToReport++
			}
		}
	}
}

// calculateVisualMetrics computes visual layout for visible lines only (lazy evaluation).
func (m *Model) calculateVisualMetrics() {
	buffer := m.editor.GetBuffer()
	state := m.editor.GetState()
	cursor := buffer.GetCursor()
	allLogicalLines := buffer.GetLines()
	totalLogicalLines := len(allLogicalLines)

	// --- Calculate Layout Widths ---
	lineNumWidth := m.calculateLineNumberWidth(totalLogicalLines)
	availableWidth := m.viewport.Width - lineNumWidth
	if availableWidth <= 0 {
		availableWidth = 1
	}

	if state.AvailableWidth != availableWidth {
		newState := m.editor.GetState()
		newState.AvailableWidth = availableWidth
		m.editor.SetState(newState)
	}

	// ========================================================================
	// >>> 1. LAZY VISUAL LAYOUT - Only compute viewport + buffer <<<
	// ========================================================================

	// For large files, only compute visible region instead of entire buffer
	// Lowered threshold to 100 lines to enable lazy mode for medium files
	// This allows cache hysteresis and anchoring to work for files 100-1000 lines
	const largeFileThreshold = 100

	// Adaptive buffer size: larger buffer for medium files reduces cache thrashing
	viewportBuffer := 100 // Default for large files (>500 lines)
	if totalLogicalLines >= largeFileThreshold && totalLogicalLines < 500 {
		viewportBuffer = 150 // Larger buffer for medium files (100-500 lines)
	}

	if totalLogicalLines > largeFileThreshold {
		// Lazy mode: only compute what we need
		m.calculateLazyVisualLayout(allLogicalLines, cursor, availableWidth, viewportBuffer)
	} else {
		// Small files: compute full layout (original behavior)
		m.calculateFullVisualLayout(allLogicalLines, availableWidth)
	}

	// ========================================================================
	// >>> 2. Find Cursor's Absolute Visual Row and Clamped Logical Column <<<
	// ========================================================================
	absoluteTargetVisualRow := -1
	m.clampedCursorLogicalCol = cursor.Position.Col

	clampedCursorRow := m.clampCursorRow(cursor.Position.Row, len(allLogicalLines))

	if clampedCursorRow >= 0 && clampedCursorRow < len(allLogicalLines) {
		lineContentRunes := []rune(allLogicalLines[clampedCursorRow])
		m.clampedCursorLogicalCol = max(0, min(cursor.Position.Col, len(lineContentRunes)))
	} else {
		m.clampedCursorLogicalCol = 0
	}

	if m.fullVisualLayoutHeight == 0 {
		absoluteTargetVisualRow = 0
	} else {
		// Use the pre-computed visual row offset from lazy layout
		visualRowOffset := m.visualLayoutCacheStartVisualRow

		for cacheIdx, vli := range m.visualLayoutCache {
			if vli.LogicalRow == clampedCursorRow {
				segmentRuneLen := len([]rune(vli.Content))
				if m.clampedCursorLogicalCol >= vli.LogicalStartCol {
					if (segmentRuneLen > 0 && m.clampedCursorLogicalCol <= vli.LogicalStartCol+segmentRuneLen) ||
						(segmentRuneLen == 0 && m.clampedCursorLogicalCol == vli.LogicalStartCol) {
						absoluteTargetVisualRow = visualRowOffset + cacheIdx
						break
					}
				}
			}
		}

		if absoluteTargetVisualRow == -1 {
			foundFirstSegment := false
			for cacheIdx, vli := range m.visualLayoutCache { // Use cached layout
				if vli.LogicalRow == clampedCursorRow && vli.IsFirstSegment {
					if m.clampedCursorLogicalCol == vli.LogicalStartCol {
						absoluteTargetVisualRow = visualRowOffset + cacheIdx
						foundFirstSegment = true
						break
					}
					if !foundFirstSegment {
						absoluteTargetVisualRow = visualRowOffset + cacheIdx
						foundFirstSegment = true
					}
				}
			}
			if !foundFirstSegment {
				if clampedCursorRow == 0 {
					absoluteTargetVisualRow = 0
				} else if m.fullVisualLayoutHeight > 0 {
					absoluteTargetVisualRow = m.fullVisualLayoutHeight - 1
				} else {
					absoluteTargetVisualRow = 0
				}
			}
		}
	}

	if m.fullVisualLayoutHeight > 0 {
		absoluteTargetVisualRow = max(0, min(absoluteTargetVisualRow, m.fullVisualLayoutHeight-1))
	} else {
		absoluteTargetVisualRow = 0
	}
	m.cursorAbsoluteVisualRow = absoluteTargetVisualRow
}

// renderVisibleSliceDefault renders the calculated slice of the visual layout to the viewport.
func (m *Model) renderVisibleSliceDefault() {
	state := m.editor.GetState()
	allLogicalLines := m.editor.GetBuffer().GetLines()

	selectionStyle := m.theme.SelectionStyle
	searchHighlightStyle := m.theme.SearchHighlightStyle

	// Check if we're highlighting a yank operation
	// Either from normal mode (YankSelection) or from visual mode (m.yanked flag)
	if state.YankSelection != editor.SelectionNone || m.yanked {
		selectionStyle = m.theme.HighlightYankStyle
	}

	lineNumWidth := m.calculateLineNumberWidth(len(allLogicalLines))

	var contentBuilder strings.Builder
	renderedDisplayLineCount := 0

	startRenderVisualRow := m.currentVisualTopLine
	if m.fullVisualLayoutHeight == 0 {
		startRenderVisualRow = 0
	} else {
		if startRenderVisualRow < 0 {
			startRenderVisualRow = 0
		}
		maxTop := max(0, m.fullVisualLayoutHeight-m.viewport.Height)
		if startRenderVisualRow > maxTop {
			startRenderVisualRow = maxTop
		}
	}

	endRenderVisualRow := min(startRenderVisualRow+m.viewport.Height, m.fullVisualLayoutHeight)

	targetVisualRowInSlice := -1
	if m.cursorAbsoluteVisualRow >= startRenderVisualRow && m.cursorAbsoluteVisualRow < endRenderVisualRow {
		targetVisualRowInSlice = m.cursorAbsoluteVisualRow - startRenderVisualRow
	}

	targetScreenColForCursor := -1
	if m.fullVisualLayoutHeight > 0 && m.cursorAbsoluteVisualRow >= 0 && m.cursorAbsoluteVisualRow < m.fullVisualLayoutHeight {
		// Convert absolute visual row to cache-relative index for cursor lookup
		cursorCacheIdx := m.cursorAbsoluteVisualRow - m.visualLayoutCacheStartVisualRow
		if cursorCacheIdx >= 0 && cursorCacheIdx < len(m.visualLayoutCache) {
			vliAtCursor := m.visualLayoutCache[cursorCacheIdx]
			targetScreenColForCursor = m.calculateCursorScreenCol(vliAtCursor, lineNumWidth)
		} else if m.fullVisualLayoutHeight > 0 {
			targetScreenColForCursor = lineNumWidth
		}
	} else if m.fullVisualLayoutHeight == 0 {
		targetScreenColForCursor = lineNumWidth
	}

	clampedCursorRowForLineNumbers := m.clampCursorRow(m.editor.GetBuffer().GetCursor().Position.Row, len(allLogicalLines))

	for absVisRowIdxToRender := startRenderVisualRow; absVisRowIdxToRender < endRenderVisualRow; absVisRowIdxToRender++ {
		// Convert absolute visual row to cache-relative index
		cacheIdx := absVisRowIdxToRender - m.visualLayoutCacheStartVisualRow
		if cacheIdx < 0 || cacheIdx >= len(m.visualLayoutCache) {
			break
		}
		vli := m.visualLayoutCache[cacheIdx]
		currentSliceRow := renderedDisplayLineCount

		if m.showLineNumbers {
			lineNumStr := ""
			currentLineNumberStyle := m.theme.LineNumberStyle
			if vli.IsFirstSegment {
				if state.RelativeNumbers && !m.disableVimMode && vli.LogicalRow != clampedCursorRowForLineNumbers {
					relNum := vli.LogicalRow - clampedCursorRowForLineNumbers
					if relNum < 0 {
						relNum = -relNum
					}
					lineNumStr = strconv.Itoa(relNum)
				} else {
					lineNumStr = strconv.Itoa(vli.LogicalRow + 1)
				}
				if vli.LogicalRow == clampedCursorRowForLineNumbers {
					currentLineNumberStyle = m.theme.CurrentLineNumberStyle
				}
			}
			contentBuilder.WriteString(currentLineNumberStyle.Width(lineNumWidth-1).Render(lineNumStr) + " ")
		}

		segmentRunes := []rune(vli.Content)
		styledSegment := strings.Builder{}
		currentVisualCol := 0

		charIdx := 0
		segmentLen := len(segmentRunes)

		// Check if this is the current line for background highlighting
		isCurrentLine := vli.LogicalRow == clampedCursorRowForLineNumbers
		var currentLineBackground lipgloss.TerminalColor
		if isCurrentLine {
			currentLineBackground = m.theme.CurrentLineStyle.GetBackground()
		}

		for charIdx < segmentLen {
			currentLogicalCharCol := vli.LogicalStartCol + charIdx
			currentBufferPos := editor.Position{Row: vli.LogicalRow, Col: currentLogicalCharCol}

			isSearchResult := m.isPositionInSearchResult(currentBufferPos, currentLogicalCharCol)

			baseCharStyle := lipgloss.NewStyle()

			// Apply current line background if this is the cursor line
			if isCurrentLine {
				baseCharStyle = baseCharStyle.Background(currentLineBackground)
			}

			charsToAdvance := 1

			bestMatch := m.findHighlightedWordMatch(segmentRunes, charIdx)
			bestMatchLen := bestMatch.length
			bestMatchStyle := bestMatch.style

			if bestMatchLen > 0 {
				for k := range bestMatchLen {
					idxInSegment := charIdx + k
					chRuneToStyle := segmentRunes[idxInSegment]
					logicalColForStyledChar := vli.LogicalStartCol + idxInSegment
					posForStyledChar := editor.Position{Row: vli.LogicalRow, Col: logicalColForStyledChar}

					charSpecificRenderStyle := bestMatchStyle

					// Apply current line background to highlighted words
					if isCurrentLine {
						charSpecificRenderStyle = charSpecificRenderStyle.Background(currentLineBackground)
					}

					selectionStatus := m.editor.GetSelectionStatus(posForStyledChar)
					if selectionStatus != editor.SelectionNone {
						charSpecificRenderStyle = charSpecificRenderStyle.Background(selectionStyle.GetBackground())
					}

					currentScreenColForChar := lineNumWidth + currentVisualCol
					isCursorOnThisChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

					if isCursorOnThisChar && m.isFocused && m.cursorVisible {
						styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
					} else {
						styledSegment.WriteString(charSpecificRenderStyle.Render(string(chRuneToStyle)))
					}
					currentVisualCol += getRuneVisualWidth(chRuneToStyle)
				}
				charsToAdvance = bestMatchLen
			} else {
				// Get the next grapheme cluster using centralised helper
				graphemeStr, graphemeWidth, runesConsumed := nextGrapheme(segmentRunes, charIdx, currentVisualCol)
				charsToAdvance = runesConsumed

				selectionStatus := m.editor.GetSelectionStatus(currentBufferPos)
				if selectionStatus != editor.SelectionNone {
					baseCharStyle = selectionStyle
				}

				if isSearchResult {
					baseCharStyle = searchHighlightStyle
				}

				currentScreenColForChar := lineNumWidth + currentVisualCol
				isCursorOnChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

				if isCursorOnChar && m.isFocused && m.cursorVisible {
					styledSegment.WriteString(m.getCursorStyles().Render(graphemeStr))
				} else {
					styledSegment.WriteString(baseCharStyle.Render(graphemeStr))
				}
				currentVisualCol += graphemeWidth
			}
			charIdx += charsToAdvance
		}
		contentBuilder.WriteString(styledSegment.String())

		isCursorAfterSegmentEnd := (currentSliceRow == targetVisualRowInSlice && (lineNumWidth+currentVisualCol) == targetScreenColForCursor)
		isCursorAtLogicalEndOfLineAndThisIsLastSegment := false
		if currentSliceRow == targetVisualRowInSlice && vli.LogicalRow == clampedCursorRowForLineNumbers {
			logicalLineLen := 0
			if vli.LogicalRow >= 0 && vli.LogicalRow < len(allLogicalLines) {
				logicalLineLen = len([]rune(allLogicalLines[vli.LogicalRow]))
			}

			if m.clampedCursorLogicalCol == logicalLineLen && (vli.LogicalStartCol+len(segmentRunes) == logicalLineLen) {
				isCursorAtLogicalEndOfLineAndThisIsLastSegment = true
			}
		}

		cursorWidth := 0
		if m.isFocused && (isCursorAfterSegmentEnd || isCursorAtLogicalEndOfLineAndThisIsLastSegment) {
			cursorBlockPos := editor.Position{Row: clampedCursorRowForLineNumbers, Col: m.clampedCursorLogicalCol}
			cursorBlockSelectionStatus := m.editor.GetSelectionStatus(cursorBlockPos)

			baseStyleForCursorBlock := lipgloss.NewStyle()

			// Apply current line style if this is the cursor line
			if vli.LogicalRow == clampedCursorRowForLineNumbers {
				baseStyleForCursorBlock = m.theme.CurrentLineStyle
			}

			if cursorBlockSelectionStatus != editor.SelectionNone {
				baseStyleForCursorBlock = selectionStyle
			}

			if m.cursorVisible {
				contentBuilder.WriteString(baseStyleForCursorBlock.Render(m.getCursorStyles().Render(" ")))
				cursorWidth = 1
			}

		}

		// Fill remaining width with current line style if this is the cursor line
		if vli.LogicalRow == clampedCursorRowForLineNumbers {
			segmentWidth := getVisualWidth(vli.Content)
			usedWidth := lineNumWidth + segmentWidth + cursorWidth
			remainingWidth := m.viewport.Width - usedWidth
			if remainingWidth > 0 {
				contentBuilder.WriteString(m.theme.CurrentLineStyle.Render(strings.Repeat(" ", remainingWidth)))
			}
		}

		contentBuilder.WriteString("\n")
		renderedDisplayLineCount++
	}

	for renderedDisplayLineCount < m.viewport.Height {
		tildeStyle := m.theme.LineNumberStyle
		if m.showLineNumbers && m.showTildeIndicator {
			contentBuilder.WriteString(tildeStyle.Width(lineNumWidth-1).Render("~") + " ")
		}

		contentBuilder.WriteString("\n")
		renderedDisplayLineCount++
	}

	finalContentSlice := strings.TrimSuffix(contentBuilder.String(), "\n")

	if m.placeholder != "" && m.IsEmpty() {
		placeholderRunes := []rune(m.placeholder)
		styledPlaceholder := strings.Builder{}

		lineNumWidth := m.calculateLineNumberWidth(1)
		if m.showLineNumbers {
			lineNumStr := "1"
			lineNumStyle := m.theme.LineNumberStyle
			if m.theme.CurrentLineNumberStyle.String() != "" {
				lineNumStyle = m.theme.CurrentLineNumberStyle
			}
			styledPlaceholder.WriteString(lineNumStyle.Width(lineNumWidth-1).Render(lineNumStr) + " ")
		}

		for i, r := range placeholderRunes {
			if i == 0 && m.isFocused && m.cursorVisible {
				styledPlaceholder.WriteString(m.getCursorStyles().Foreground(m.theme.PlaceholderStyle.GetForeground()).Render(string(r)))
			} else {
				styledPlaceholder.WriteString(m.theme.PlaceholderStyle.Render(string(r)))
			}
		}

		finalContentSlice = styledPlaceholder.String()
	}

	m.viewport.SetContent(finalContentSlice)
}

// renderVisibleSlice renders the visible slice of the visual layout.
func (m *Model) renderVisibleSlice() {
	if m.highlighter != nil {
		m.renderVisibleSliceWithSyntax()
	} else {
		m.renderVisibleSliceDefault()
	}
}

// updateVisualTopLine adjusts the current visual top line based on the cursor's position.
// It ensures that the cursor is always visible within the viewport.
// If the cursor is above the current top line, it moves the top line up.
// If the cursor is below the current top line, it moves the top line down.
func (m *Model) updateVisualTopLine() {
	if m.fullVisualLayoutHeight > 0 {
		if m.cursorAbsoluteVisualRow < m.currentVisualTopLine {
			m.currentVisualTopLine = m.cursorAbsoluteVisualRow
		} else if m.cursorAbsoluteVisualRow >= m.currentVisualTopLine+m.viewport.Height {
			m.currentVisualTopLine = m.cursorAbsoluteVisualRow - m.viewport.Height + 1
		}

		maxPossibleTopLine := 0
		if m.fullVisualLayoutHeight > m.viewport.Height {
			maxPossibleTopLine = m.fullVisualLayoutHeight - m.viewport.Height
		}
		if m.currentVisualTopLine > maxPossibleTopLine {
			m.currentVisualTopLine = maxPossibleTopLine
		}
		if m.currentVisualTopLine < 0 {
			m.currentVisualTopLine = 0
		}
	} else {
		m.currentVisualTopLine = 0
	}

	m.viewport.YOffset = 0
}

// wrapLine wraps a line to fit within the specified width.
// It operates on grapheme clusters (not runes) to correctly handle multi-rune characters
// like flag emojis (🇷🇴), skin tone modifiers (👍🏽), and ZWJ sequences (👨‍👩‍👧‍👦).
func wrapLine(line string, width int) []string {
	if width <= 0 {
		if line == "" {
			return []string{""}
		}
		return []string{line}
	}
	if line == "" {
		return []string{""}
	}

	runes := []rune(line)
	var wrappedLines []string
	currentRuneIdx := 0

	for currentRuneIdx < len(runes) {
		// Early exit optimization: Quick check if remaining runes might fit
		// Most characters are width 1, so if rune count <= width, text likely fits
		remainingRuneCount := len(runes) - currentRuneIdx
		if remainingRuneCount <= width {
			// Only now do the expensive visual width calculation
			remainingText := string(runes[currentRuneIdx:])
			remainingWidth := getVisualWidth(remainingText)
			if remainingWidth <= width {
				wrappedLines = append(wrappedLines, remainingText)
				break
			}
		}

		lineStartRuneIdx := currentRuneIdx
		currentVisualWidth := 0
		lastSpaceGraphemeStartRuneIdx := -1 // Start rune index of space grapheme

		// Find the longest segment that fits within width, breaking at grapheme boundaries
		tempRuneIdx := currentRuneIdx
		for tempRuneIdx < len(runes) {
			graphemeStr, graphemeWidth, runesConsumed := nextGrapheme(runes, tempRuneIdx, currentVisualWidth)

			// If adding this grapheme would exceed width, break here
			if currentVisualWidth+graphemeWidth > width {
				break
			}

			currentVisualWidth += graphemeWidth

			// Check if this grapheme starts with whitespace
			graphemeRunes := []rune(graphemeStr)
			if len(graphemeRunes) > 0 && unicode.IsSpace(graphemeRunes[0]) {
				lastSpaceGraphemeStartRuneIdx = tempRuneIdx
			}

			tempRuneIdx += runesConsumed
		}

		// Determine where to break the line
		var breakEndRuneIdx int
		if tempRuneIdx == lineStartRuneIdx {
			// First grapheme is wider than width - must include it anyway to make progress
			_, _, runesConsumed := nextGrapheme(runes, lineStartRuneIdx, 0)
			breakEndRuneIdx = lineStartRuneIdx + runesConsumed
		} else if lastSpaceGraphemeStartRuneIdx >= lineStartRuneIdx {
			// Break before the space
			breakEndRuneIdx = lastSpaceGraphemeStartRuneIdx
		} else {
			// Hard break at grapheme boundary
			breakEndRuneIdx = tempRuneIdx
		}

		// Ensure progress to prevent infinite loops
		if breakEndRuneIdx <= lineStartRuneIdx {
			if lineStartRuneIdx < len(runes) {
				_, _, runesConsumed := nextGrapheme(runes, lineStartRuneIdx, 0)
				breakEndRuneIdx = lineStartRuneIdx + runesConsumed
			} else {
				break
			}
		}

		// Append the wrapped segment
		segment := string(runes[lineStartRuneIdx:breakEndRuneIdx])
		wrappedLines = append(wrappedLines, segment)

		// Advance, skipping leading spaces on the next line
		currentRuneIdx = breakEndRuneIdx
		for currentRuneIdx < len(runes) {
			graphemeStr, _, runesConsumed := nextGrapheme(runes, currentRuneIdx, 0)
			graphemeRunes := []rune(graphemeStr)
			if len(graphemeRunes) == 0 || !unicode.IsSpace(graphemeRunes[0]) {
				break
			}
			currentRuneIdx += runesConsumed
		}
	}

	if len(wrappedLines) == 0 {
		// If wrapping failed but we had non-empty input, return the original line
		if len(runes) > 0 {
			return []string{line}
		}
		return []string{""}
	}
	return wrappedLines
}

func (m *Model) getCursorStyles() lipgloss.Style {
	state := m.editor.GetState()
	switch state.Mode {
	case editor.InsertMode:
		return m.theme.InsertModeStyle
	case editor.VisualMode, editor.VisualLineMode:
		return m.theme.VisualModeStyle
	case editor.CommandMode:
		return m.theme.CommandModeStyle
	default:
		return m.theme.NormalModeStyle
	}
}

// renderVisibleSliceWithSyntax is the modified version of renderVisibleSlice with syntax highlighting support
func (m *Model) renderVisibleSliceWithSyntax() {
	state := m.editor.GetState()
	allLogicalLines := m.editor.GetBuffer().GetLines()

	selectionStyle := m.theme.SelectionStyle
	searchHighlightStyle := m.theme.SearchHighlightStyle

	// Check if we're highlighting a yank operation
	// Either from normal mode (YankSelection) or from visual mode (m.yanked flag)
	if state.YankSelection != editor.SelectionNone || m.yanked {
		selectionStyle = m.theme.HighlightYankStyle
	}

	lineNumWidth := m.calculateLineNumberWidth(len(allLogicalLines))

	var contentBuilder strings.Builder
	renderedDisplayLineCount := 0

	startRenderVisualRow := m.currentVisualTopLine
	if m.fullVisualLayoutHeight == 0 {
		startRenderVisualRow = 0
	} else {
		if startRenderVisualRow < 0 {
			startRenderVisualRow = 0
		}
		maxTop := max(0, m.fullVisualLayoutHeight-m.viewport.Height)
		if startRenderVisualRow > maxTop {
			startRenderVisualRow = maxTop
		}
	}

	endRenderVisualRow := min(startRenderVisualRow+m.viewport.Height, m.fullVisualLayoutHeight)

	targetVisualRowInSlice := -1
	if m.cursorAbsoluteVisualRow >= startRenderVisualRow && m.cursorAbsoluteVisualRow < endRenderVisualRow {
		targetVisualRowInSlice = m.cursorAbsoluteVisualRow - startRenderVisualRow
	}

	targetScreenColForCursor := -1
	if m.fullVisualLayoutHeight > 0 && m.cursorAbsoluteVisualRow >= 0 && m.cursorAbsoluteVisualRow < m.fullVisualLayoutHeight {
		// Convert absolute visual row to cache-relative index for cursor lookup
		cursorCacheIdx := m.cursorAbsoluteVisualRow - m.visualLayoutCacheStartVisualRow
		if cursorCacheIdx >= 0 && cursorCacheIdx < len(m.visualLayoutCache) {
			vliAtCursor := m.visualLayoutCache[cursorCacheIdx]
			targetScreenColForCursor = m.calculateCursorScreenCol(vliAtCursor, lineNumWidth)
		} else if m.fullVisualLayoutHeight > 0 {
			targetScreenColForCursor = lineNumWidth
		}
	} else if m.fullVisualLayoutHeight == 0 {
		targetScreenColForCursor = lineNumWidth
	}

	clampedCursorRowForLineNumbers := m.clampCursorRow(m.editor.GetBuffer().GetCursor().Position.Row, len(allLogicalLines))

	// Initialise persistent token cache if needed
	if m.persistentTokenCache == nil {
		m.persistentTokenCache = make(map[int][]highlighter.TokenPosition)
	}

	// Populate persistent cache with tokenised lines
	if m.highlighter != nil {
		extraHighlightedContextLines := int(m.extraHighlightedContextLines)

		// Pre-tokenise all visible logical lines, with context
		startLogicalLine := -1
		endLogicalLine := -1
		if len(m.visualLayoutCache) > 0 && m.fullVisualLayoutHeight > 0 {
			// Convert absolute visual rows to cache indices
			startCacheIdx := max(0, startRenderVisualRow-m.visualLayoutCacheStartVisualRow)
			endCacheIdx := min(len(m.visualLayoutCache)-1, endRenderVisualRow-m.visualLayoutCacheStartVisualRow)

			if startCacheIdx >= 0 && startCacheIdx < len(m.visualLayoutCache) && endCacheIdx >= 0 && endCacheIdx < len(m.visualLayoutCache) {
				startLogicalLine = m.visualLayoutCache[startCacheIdx].LogicalRow
				endLogicalLine = m.visualLayoutCache[endCacheIdx].LogicalRow + 1

				// Expand the range for better syntax highlighting context
				// For markdown, we need extra context to properly tokenise code blocks
				// The incremental tokeniser will skip already-cached lines, so this is efficient
				expandedStartLine := max(0, startLogicalLine-extraHighlightedContextLines)
				expandedEndLine := min(len(allLogicalLines), endLogicalLine+extraHighlightedContextLines)

				if expandedStartLine < expandedEndLine {
					m.highlighter.Tokenise(allLogicalLines, expandedStartLine, expandedEndLine)

					// Populate persistent cache for the expanded range
					// This ensures large code blocks have tokens available even when scrolled
					// Always check highlighter first - it knows which lines are invalidated
					for logicalLine := expandedStartLine; logicalLine < expandedEndLine; logicalLine++ {
						tokens := m.highlighter.GetTokensForLine(logicalLine, allLogicalLines)
						if tokens != nil {
							// Highlighter has valid tokens, cache them (may overwrite stale cache)
							m.persistentTokenCache[logicalLine] = highlighter.GetTokenPositions(tokens)
						} else {
							// Line was invalidated in highlighter, remove from persistent cache
							delete(m.persistentTokenCache, logicalLine)
						}
					}
				}
			}
		}
	}

	// Use persistent cache for rendering (reference, not copy)
	lineTokenCache := m.persistentTokenCache

	for absVisRowIdxToRender := startRenderVisualRow; absVisRowIdxToRender < endRenderVisualRow; absVisRowIdxToRender++ {
		// Convert absolute visual row to cache-relative index
		cacheIdx := absVisRowIdxToRender - m.visualLayoutCacheStartVisualRow
		if cacheIdx < 0 || cacheIdx >= len(m.visualLayoutCache) {
			break
		}
		vli := m.visualLayoutCache[cacheIdx]
		currentSliceRow := renderedDisplayLineCount

		// Render line number
		if m.showLineNumbers {
			lineNumStr := ""
			currentLineNumberStyle := m.theme.LineNumberStyle
			if vli.IsFirstSegment {
				if state.RelativeNumbers && !m.disableVimMode && vli.LogicalRow != clampedCursorRowForLineNumbers {
					relNum := vli.LogicalRow - clampedCursorRowForLineNumbers
					if relNum < 0 {
						relNum = -relNum
					}
					lineNumStr = strconv.Itoa(relNum)
				} else {
					lineNumStr = strconv.Itoa(vli.LogicalRow + 1)
				}
				if vli.LogicalRow == clampedCursorRowForLineNumbers {
					currentLineNumberStyle = m.theme.CurrentLineNumberStyle
				}
			}
			contentBuilder.WriteString(currentLineNumberStyle.Width(lineNumWidth-1).Render(lineNumStr) + " ")
		}

		// Get token positions for this line
		var tokenPositions []highlighter.TokenPosition
		if m.highlighter != nil {
			if positions, ok := lineTokenCache[vli.LogicalRow]; ok {
				tokenPositions = positions
			}
		}

		// Render the segment
		if len(tokenPositions) > 0 {
			m.renderSegmentWithSyntax(
				vli,
				tokenPositions,
				&contentBuilder,
				currentSliceRow,
				targetVisualRowInSlice,
				targetScreenColForCursor,
				lineNumWidth,
				selectionStyle,
				searchHighlightStyle,
			)
		} else {
			// Fall back to original rendering logic (without syntax highlighting)
			m.renderSegmentPlain(
				vli,
				&contentBuilder,
				currentSliceRow,
				targetVisualRowInSlice,
				targetScreenColForCursor,
				lineNumWidth,
				selectionStyle,
				searchHighlightStyle,
			)
		}

		// Handle cursor at end of line
		segmentVisualWidth := getVisualWidth(vli.Content)
		isCursorAfterSegmentEnd := (currentSliceRow == targetVisualRowInSlice && (lineNumWidth+segmentVisualWidth) == targetScreenColForCursor)
		isCursorAtLogicalEndOfLineAndThisIsLastSegment := false
		if currentSliceRow == targetVisualRowInSlice && vli.LogicalRow == clampedCursorRowForLineNumbers {
			logicalLineLen := 0
			if vli.LogicalRow >= 0 && vli.LogicalRow < len(allLogicalLines) {
				logicalLineLen = len([]rune(allLogicalLines[vli.LogicalRow]))
			}

			if m.clampedCursorLogicalCol == logicalLineLen && (vli.LogicalStartCol+len([]rune(vli.Content)) == logicalLineLen) {
				isCursorAtLogicalEndOfLineAndThisIsLastSegment = true
			}
		}

		cursorWidth := 0
		if m.isFocused && (isCursorAfterSegmentEnd || isCursorAtLogicalEndOfLineAndThisIsLastSegment) {
			cursorBlockPos := editor.Position{Row: clampedCursorRowForLineNumbers, Col: m.clampedCursorLogicalCol}
			cursorBlockSelectionStatus := m.editor.GetSelectionStatus(cursorBlockPos)

			baseStyleForCursorBlock := lipgloss.NewStyle()

			// Apply current line style if this is the cursor line
			if vli.LogicalRow == clampedCursorRowForLineNumbers {
				baseStyleForCursorBlock = m.theme.CurrentLineStyle
			}

			if cursorBlockSelectionStatus != editor.SelectionNone {
				baseStyleForCursorBlock = selectionStyle
			}

			if m.cursorVisible {
				contentBuilder.WriteString(baseStyleForCursorBlock.Render(m.getCursorStyles().Render(" ")))
				cursorWidth = 1
			}
		}

		// Fill remaining width with current line style if this is the cursor line
		if vli.LogicalRow == clampedCursorRowForLineNumbers {
			segmentWidth := getVisualWidth(vli.Content)
			usedWidth := lineNumWidth + segmentWidth + cursorWidth
			remainingWidth := m.viewport.Width - usedWidth
			if remainingWidth > 0 {
				contentBuilder.WriteString(m.theme.CurrentLineStyle.Render(strings.Repeat(" ", remainingWidth)))
			}
		}

		contentBuilder.WriteString("\n")
		renderedDisplayLineCount++
	}

	// Render empty lines with tildes
	for renderedDisplayLineCount < m.viewport.Height {
		tildeStyle := m.theme.LineNumberStyle
		if m.showLineNumbers && m.showTildeIndicator {
			contentBuilder.WriteString(tildeStyle.Width(lineNumWidth-1).Render("~") + " ")
		}
		contentBuilder.WriteString("\n")
		renderedDisplayLineCount++
	}

	finalContentSlice := strings.TrimSuffix(contentBuilder.String(), "\n")

	// Handle placeholder
	if m.placeholder != "" && m.IsEmpty() {
		placeholderRunes := []rune(m.placeholder)
		styledPlaceholder := strings.Builder{}

		if m.showLineNumbers {
			lineNumStr := "1"
			lineNumStyle := m.theme.LineNumberStyle
			if m.theme.CurrentLineNumberStyle.String() != "" {
				lineNumStyle = m.theme.CurrentLineNumberStyle
			}
			styledPlaceholder.WriteString(lineNumStyle.Width(lineNumWidth-1).Render(lineNumStr) + " ")
		}

		for i, r := range placeholderRunes {
			if i == 0 && m.isFocused && m.cursorVisible {
				styledPlaceholder.WriteString(m.getCursorStyles().Foreground(m.theme.PlaceholderStyle.GetForeground()).Render(string(r)))
			} else {
				styledPlaceholder.WriteString(m.theme.PlaceholderStyle.Render(string(r)))
			}
		}

		finalContentSlice = styledPlaceholder.String()
	}

	m.viewport.SetContent(finalContentSlice)
}

// renderSegment renders a segment with an optional base style provider
func (m *Model) renderSegment(
	vli VisualLineInfo,
	contentBuilder *strings.Builder,
	currentSliceRow int,
	targetVisualRowInSlice int,
	targetScreenColForCursor int,
	lineNumWidth int,
	selectionStyle lipgloss.Style,
	searchHighlightStyle lipgloss.Style,
	getBaseStyle func(col int) lipgloss.Style,
) {
	segmentRunes := []rune(vli.Content)
	styledSegment := strings.Builder{}
	currentVisualCol := 0

	charIdx := 0
	segmentLen := len(segmentRunes)

	clampedCursorRow := m.clampCursorRow(m.editor.GetBuffer().GetCursor().Position.Row, m.editor.GetBuffer().LineCount())
	isCurrentLine := vli.LogicalRow == clampedCursorRow

	// Pre-calculate current line background once per segment for performance
	var currentLineBackground lipgloss.TerminalColor
	if isCurrentLine {
		currentLineBackground = m.theme.CurrentLineStyle.GetBackground()
	}

	for charIdx < segmentLen {
		currentLogicalCharCol := vli.LogicalStartCol + charIdx
		currentBufferPos := editor.Position{Row: vli.LogicalRow, Col: currentLogicalCharCol}

		isSearchResult := m.isPositionInSearchResult(currentBufferPos, currentLogicalCharCol)

		// Get base style from provider function
		baseCharStyle := getBaseStyle(currentLogicalCharCol)

		// Apply current line background if this is the cursor line
		if isCurrentLine {
			baseCharStyle = baseCharStyle.Background(currentLineBackground)
		}

		if isSearchResult {
			baseCharStyle = searchHighlightStyle
		}

		// Check for highlighted words (this takes precedence over syntax highlighting)
		charsToAdvance := 1
		bestMatch := m.findHighlightedWordMatch(segmentRunes, charIdx)
		bestMatchLen := bestMatch.length
		bestMatchStyle := bestMatch.style

		if bestMatchLen > 0 {
			// Render highlighted word
			for k := range bestMatchLen {
				idxInSegment := charIdx + k
				chRuneToStyle := segmentRunes[idxInSegment]
				logicalColForStyledChar := vli.LogicalStartCol + idxInSegment
				posForStyledChar := editor.Position{Row: vli.LogicalRow, Col: logicalColForStyledChar}

				charSpecificRenderStyle := bestMatchStyle

				// Apply current line background to highlighted words
				if isCurrentLine {
					charSpecificRenderStyle = charSpecificRenderStyle.Background(currentLineBackground)
				}

				// Apply selection style if needed
				selectionStatus := m.editor.GetSelectionStatus(posForStyledChar)
				if selectionStatus != editor.SelectionNone {
					charSpecificRenderStyle = charSpecificRenderStyle.Background(selectionStyle.GetBackground())
				}

				currentScreenColForChar := lineNumWidth + currentVisualCol // <-- MUST USE currentVisualCol
				isCursorOnThisChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

				if isCursorOnThisChar && m.isFocused && m.cursorVisible {
					styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
				} else {
					styledSegment.WriteString(charSpecificRenderStyle.Render(string(chRuneToStyle)))
				}
				currentVisualCol += getRuneVisualWidth(chRuneToStyle) // <-- MUST INCREMENT BY WIDTH
			}
			charsToAdvance = bestMatchLen
		} else {
			// Get the next grapheme cluster using centralised helper
			graphemeStr, graphemeWidth, runesConsumed := nextGrapheme(segmentRunes, charIdx, currentVisualCol)
			charsToAdvance = runesConsumed

			// Apply selection style on top of syntax highlighting
			selectionStatus := m.editor.GetSelectionStatus(currentBufferPos)
			if selectionStatus != editor.SelectionNone {
				if isSearchResult {
					baseCharStyle = baseCharStyle.Background(searchHighlightStyle.GetBackground())
				} else {
					baseCharStyle = baseCharStyle.Background(selectionStyle.GetBackground())
				}
			}

			currentScreenColForChar := lineNumWidth + currentVisualCol
			isCursorOnChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

			if isCursorOnChar && m.isFocused && m.cursorVisible {
				styledSegment.WriteString(m.getCursorStyles().Render(graphemeStr))
			} else {
				styledSegment.WriteString(baseCharStyle.Render(graphemeStr))
			}
			currentVisualCol += graphemeWidth
		}

		charIdx += charsToAdvance
	}

	contentBuilder.WriteString(styledSegment.String())
}

// renderSegmentWithSyntax renders a segment with syntax highlighting
func (m *Model) renderSegmentWithSyntax(
	vli VisualLineInfo,
	tokenPositions []highlighter.TokenPosition,
	contentBuilder *strings.Builder,
	currentSliceRow int,
	targetVisualRowInSlice int,
	targetScreenColForCursor int,
	lineNumWidth int,
	selectionStyle lipgloss.Style,
	searchHighlightStyle lipgloss.Style,
) {
	getBaseStyle := func(col int) lipgloss.Style {
		token, hasToken := highlighter.FindTokenAtPosition(tokenPositions, col)
		if hasToken && m.highlighter != nil {
			return m.highlighter.GetStyleForToken(token.Type)
		}
		return lipgloss.NewStyle()
	}

	m.renderSegment(vli, contentBuilder, currentSliceRow, targetVisualRowInSlice,
		targetScreenColForCursor, lineNumWidth, selectionStyle, searchHighlightStyle, getBaseStyle)
}

// renderSegmentPlain renders a segment without syntax highlighting (fallback)
func (m *Model) renderSegmentPlain(
	vli VisualLineInfo,
	contentBuilder *strings.Builder,
	currentSliceRow int,
	targetVisualRowInSlice int,
	targetScreenColForCursor int,
	lineNumWidth int,
	selectionStyle lipgloss.Style,
	searchHighlightStyle lipgloss.Style,
) {
	getBaseStyle := func(col int) lipgloss.Style {
		return lipgloss.NewStyle()
	}

	m.renderSegment(vli, contentBuilder, currentSliceRow, targetVisualRowInSlice,
		targetScreenColForCursor, lineNumWidth, selectionStyle, searchHighlightStyle, getBaseStyle)
}

// handleContentChange is called when the content of the editor changes.
func (m *Model) handleContentChange() {
	if m.highlighter != nil {
		currentLine := m.editor.GetBuffer().GetCursor().Position.Row
		m.highlighter.InvalidateLine(currentLine)
	}
	// Clear persistent token cache on content changes
	m.persistentTokenCache = make(map[int][]highlighter.TokenPosition)

	// Force cache recalculation by invalidating the cache validity range
	// This ensures the visual layout cache is updated with the new content
	m.cacheValidStartRow = 0
	m.cacheValidEndRow = 0

	m.calculateVisualMetrics()
	m.updateVisualTopLine()
}
