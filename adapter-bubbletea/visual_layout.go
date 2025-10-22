package adapter_bubbletea

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/goeditor/adapter-bubbletea/highlighter"
	editor "github.com/ionut-t/goeditor/core"
)

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
	state := m.editor.GetState()

	// Determine visible range based on BOTH cursor position AND current scroll position
	// Use the editor's TopLine to understand what's actually visible
	topLine := state.TopLine
	viewportHeight := m.viewport.Height

	// Calculate the range we need to cache
	// Must include: what's visible + buffer above/below + cursor position
	visibleStart := max(0, topLine-viewportBuffer/2)
	visibleEnd := min(totalLines, topLine+viewportHeight+viewportBuffer/2)

	// Also ensure cursor is in range
	cursorStart := max(0, cursor.Position.Row-viewportBuffer/4)
	cursorEnd := min(totalLines, cursor.Position.Row+viewportBuffer/4)

	// Take the union of both ranges
	startLine := min(visibleStart, cursorStart)
	endLine := max(visibleEnd, cursorEnd)

	// Clamp to reasonable bounds
	startLine = max(0, startLine)
	endLine = min(totalLines, endLine)

	// Compute visual layout only for the cached range
	visualLayout := make([]VisualLineInfo, 0, (endLine-startLine)*2)

	for bufferRowIdx := startLine; bufferRowIdx < endLine; bufferRowIdx++ {
		m.appendVisualLayoutForLine(bufferRowIdx, allLogicalLines[bufferRowIdx], availableWidth, &visualLayout)
	}

	m.visualLayoutCache = visualLayout
	m.visualLayoutCacheStartRow = startLine // Track where this cache starts (logical row)

	// Estimate the visual row offset (approximate but fast)
	avgVisualLinesPerLogical := float64(len(visualLayout)) / float64(max(1, endLine-startLine))
	m.visualLayoutCacheStartVisualRow = int(avgVisualLinesPerLogical * float64(startLine))

	// Estimate full visual height
	m.fullVisualLayoutHeight = int(avgVisualLinesPerLogical * float64(totalLines))
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
	// Threshold chosen based on viewport buffer size: when buffer has >1000 lines,
	// lazy mode (computing ~400 lines) is significantly faster than full layout
	const largeFileThreshold = 1000
	const viewportBuffer = 200 // Lines to cache above/below visible area

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

	if m.yanked {
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
			visualColInSegment := max(0, m.clampedCursorLogicalCol-vliAtCursor.LogicalStartCol)
			targetScreenColForCursor = lineNumWidth + visualColInSegment
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

		charIdx := 0
		segmentLen := len(segmentRunes)

		for charIdx < segmentLen {
			currentLogicalCharCol := vli.LogicalStartCol + charIdx
			currentBufferPos := editor.Position{Row: vli.LogicalRow, Col: currentLogicalCharCol}

			isSearchResult := m.isPositionInSearchResult(currentBufferPos, currentLogicalCharCol)

			baseCharStyle := lipgloss.NewStyle()
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

					selectionStatus := m.editor.GetSelectionStatus(posForStyledChar)
					if selectionStatus != editor.SelectionNone {
						charSpecificRenderStyle = charSpecificRenderStyle.Background(selectionStyle.GetBackground())
					}

					currentScreenColForChar := lineNumWidth + idxInSegment
					isCursorOnThisChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

					if isCursorOnThisChar && m.isFocused && m.cursorVisible {
						styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
					} else {
						styledSegment.WriteString(charSpecificRenderStyle.Render(string(chRuneToStyle)))
					}
				}
				charsToAdvance = bestMatchLen
			} else {
				chRuneToStyle := segmentRunes[charIdx]

				selectionStatus := m.editor.GetSelectionStatus(currentBufferPos)
				if selectionStatus != editor.SelectionNone {
					baseCharStyle = selectionStyle
				}

				if isSearchResult {
					baseCharStyle = searchHighlightStyle
				}

				currentScreenColForChar := lineNumWidth + charIdx
				isCursorOnChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

				if isCursorOnChar && m.isFocused && m.cursorVisible {
					styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
				} else {
					styledSegment.WriteString(baseCharStyle.Render(string(chRuneToStyle)))
				}
			}
			charIdx += charsToAdvance
		}
		contentBuilder.WriteString(styledSegment.String())

		isCursorAfterSegmentEnd := (currentSliceRow == targetVisualRowInSlice && (lineNumWidth+len(segmentRunes)) == targetScreenColForCursor)
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

		if m.isFocused && (isCursorAfterSegmentEnd || isCursorAtLogicalEndOfLineAndThisIsLastSegment) {
			cursorBlockPos := editor.Position{Row: clampedCursorRowForLineNumbers, Col: m.clampedCursorLogicalCol}
			cursorBlockSelectionStatus := m.editor.GetSelectionStatus(cursorBlockPos)

			baseStyleForCursorBlock := lipgloss.NewStyle()
			if cursorBlockSelectionStatus != editor.SelectionNone {
				baseStyleForCursorBlock = selectionStyle
			}

			if m.cursorVisible {
				contentBuilder.WriteString(baseStyleForCursorBlock.Render(m.getCursorStyles().Render(" ")))
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

// wrapLine function wraps a single line of text to fit within the specified width.
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

	var wrappedLines []string
	runes := []rune(line)
	lineLen := len(runes)
	start := 0

	for start < lineLen {
		if start+width >= lineLen {
			wrappedLines = append(wrappedLines, string(runes[start:]))
			break
		}
		end := min(start+width, lineLen)
		lastSpace := -1
		for i := end - 1; i >= start; i-- {
			if unicode.IsSpace(runes[i]) {
				lastSpace = i
				break
			}
		}
		if lastSpace >= start {
			wrappedLines = append(wrappedLines, string(runes[start:lastSpace]))
			start = lastSpace + 1
			for start < lineLen && unicode.IsSpace(runes[start]) {
				start++
			}
		} else {
			wrappedLines = append(wrappedLines, string(runes[start:end]))
			start = end
		}
	}
	if len(wrappedLines) == 0 && lineLen > 0 {
		return []string{line}
	}
	if len(wrappedLines) == 0 {
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
	if m.yanked {
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
			visualColInSegment := max(0, m.clampedCursorLogicalCol-vliAtCursor.LogicalStartCol)
			targetScreenColForCursor = lineNumWidth + visualColInSegment
		} else if m.fullVisualLayoutHeight > 0 {
			targetScreenColForCursor = lineNumWidth
		}
	} else if m.fullVisualLayoutHeight == 0 {
		targetScreenColForCursor = lineNumWidth
	}

	clampedCursorRowForLineNumbers := m.clampCursorRow(m.editor.GetBuffer().GetCursor().Position.Row, len(allLogicalLines))

	// Cache token positions for each logical line we're rendering
	lineTokenCache := make(map[int][]highlighter.TokenPosition)
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
				}
			}
		}

		for absVisRowIdx := startRenderVisualRow; absVisRowIdx < endRenderVisualRow; absVisRowIdx++ {
			// Convert absolute visual row to cache index
			cacheIdx := absVisRowIdx - m.visualLayoutCacheStartVisualRow
			if cacheIdx < 0 || cacheIdx >= len(m.visualLayoutCache) {
				continue
			}
			vli := m.visualLayoutCache[cacheIdx]
			if vli.IsFirstSegment && vli.LogicalRow < len(allLogicalLines) {
				if _, cached := lineTokenCache[vli.LogicalRow]; !cached {
					tokens := m.highlighter.GetTokensForLine(vli.LogicalRow, allLogicalLines)
					lineTokenCache[vli.LogicalRow] = highlighter.GetTokenPositions(tokens)
				}
			}
		}
	}

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
		isCursorAfterSegmentEnd := (currentSliceRow == targetVisualRowInSlice && (lineNumWidth+len([]rune(vli.Content))) == targetScreenColForCursor)
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

		if m.isFocused && (isCursorAfterSegmentEnd || isCursorAtLogicalEndOfLineAndThisIsLastSegment) {
			cursorBlockPos := editor.Position{Row: clampedCursorRowForLineNumbers, Col: m.clampedCursorLogicalCol}
			cursorBlockSelectionStatus := m.editor.GetSelectionStatus(cursorBlockPos)

			baseStyleForCursorBlock := lipgloss.NewStyle()
			if cursorBlockSelectionStatus != editor.SelectionNone {
				baseStyleForCursorBlock = selectionStyle
			}

			if m.cursorVisible {
				contentBuilder.WriteString(baseStyleForCursorBlock.Render(m.getCursorStyles().Render(" ")))
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
	searchHighlightStyle lipgloss.Style) {
	segmentRunes := []rune(vli.Content)
	styledSegment := strings.Builder{}

	charIdx := 0
	segmentLen := len(segmentRunes)

	for charIdx < segmentLen {
		currentLogicalCharCol := vli.LogicalStartCol + charIdx
		currentBufferPos := editor.Position{Row: vli.LogicalRow, Col: currentLogicalCharCol}

		// Find the token for this position
		token, hasToken := highlighter.FindTokenAtPosition(tokenPositions, currentLogicalCharCol)

		isSearchResult := m.isPositionInSearchResult(currentBufferPos, currentLogicalCharCol)

		// Get base style from syntax highlighting
		baseCharStyle := lipgloss.NewStyle()
		if hasToken && m.highlighter != nil {
			baseCharStyle = m.highlighter.GetStyleForToken(token.Type)
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

				// Apply selection style if needed
				selectionStatus := m.editor.GetSelectionStatus(posForStyledChar)
				if selectionStatus != editor.SelectionNone {
					charSpecificRenderStyle = charSpecificRenderStyle.Background(selectionStyle.GetBackground())
				}

				currentScreenColForChar := lineNumWidth + idxInSegment
				isCursorOnThisChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

				if isCursorOnThisChar && m.isFocused && m.cursorVisible {
					styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
				} else {
					styledSegment.WriteString(charSpecificRenderStyle.Render(string(chRuneToStyle)))
				}
			}
			charsToAdvance = bestMatchLen
		} else {
			// Normal character rendering with syntax highlighting
			chRuneToStyle := segmentRunes[charIdx]

			// Apply selection style on top of syntax highlighting
			selectionStatus := m.editor.GetSelectionStatus(currentBufferPos)
			if selectionStatus != editor.SelectionNone {
				if isSearchResult {
					baseCharStyle = baseCharStyle.Background(searchHighlightStyle.GetBackground())
				} else {
					baseCharStyle = baseCharStyle.Background(selectionStyle.GetBackground())
				}
			}

			currentScreenColForChar := lineNumWidth + charIdx
			isCursorOnChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

			if isCursorOnChar && m.isFocused && m.cursorVisible {
				styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
			} else {
				styledSegment.WriteString(baseCharStyle.Render(string(chRuneToStyle)))
			}
		}

		charIdx += charsToAdvance
	}

	contentBuilder.WriteString(styledSegment.String())
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
	segmentRunes := []rune(vli.Content)
	styledSegment := strings.Builder{}

	charIdx := 0
	segmentLen := len(segmentRunes)

	for charIdx < segmentLen {
		currentLogicalCharCol := vli.LogicalStartCol + charIdx
		currentBufferPos := editor.Position{Row: vli.LogicalRow, Col: currentLogicalCharCol}

		isSearchResult := m.isPositionInSearchResult(currentBufferPos, currentLogicalCharCol)

		baseCharStyle := lipgloss.NewStyle()
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

				selectionStatus := m.editor.GetSelectionStatus(posForStyledChar)
				if selectionStatus != editor.SelectionNone {
					charSpecificRenderStyle = charSpecificRenderStyle.Background(selectionStyle.GetBackground())
				}

				currentScreenColForChar := lineNumWidth + idxInSegment
				isCursorOnThisChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

				if isCursorOnThisChar && m.isFocused && m.cursorVisible {
					styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
				} else {
					styledSegment.WriteString(charSpecificRenderStyle.Render(string(chRuneToStyle)))
				}
			}
			charsToAdvance = bestMatchLen
		} else {
			chRuneToStyle := segmentRunes[charIdx]

			selectionStatus := m.editor.GetSelectionStatus(currentBufferPos)
			if selectionStatus != editor.SelectionNone {
				baseCharStyle = selectionStyle
			}

			if isSearchResult {
				baseCharStyle = searchHighlightStyle
			}

			currentScreenColForChar := lineNumWidth + charIdx
			isCursorOnChar := (currentSliceRow == targetVisualRowInSlice && currentScreenColForChar == targetScreenColForCursor)

			if isCursorOnChar && m.isFocused && m.cursorVisible {
				styledSegment.WriteString(m.getCursorStyles().Render(string(chRuneToStyle)))
			} else {
				styledSegment.WriteString(baseCharStyle.Render(string(chRuneToStyle)))
			}
		}
		charIdx += charsToAdvance
	}

	contentBuilder.WriteString(styledSegment.String())
}

// handleContentChange is called when the content of the editor changes.
func (m *Model) handleContentChange() {
	if m.highlighter != nil {
		currentLine := m.editor.GetBuffer().GetCursor().Position.Row
		m.highlighter.InvalidateLine(currentLine)
	}
	m.calculateVisualMetrics()
	m.updateVisualTopLine()
}
