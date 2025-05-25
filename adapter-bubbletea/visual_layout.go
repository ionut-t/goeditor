package bubble_adapter

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/core"
)

type VisualLineInfo struct {
	Content         string
	LogicalRow      int
	LogicalStartCol int
	IsFirstSegment  bool
}

// calculateVisualMetrics computes the full visual layout and cursor's position within it.
func (m *Model) calculateVisualMetrics() {
	buffer := m.editor.GetBuffer()
	state := m.editor.GetState()
	cursor := buffer.GetCursor()
	allLogicalLines := buffer.GetLines()

	// --- Calculate Layout Widths ---
	lineNumWidth := 0
	if m.showLineNumbers {
		maxLineNum := len(allLogicalLines)
		maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
		if state.RelativeNumbers && !m.disableVimMode {
			relWidth := len(strconv.Itoa(max(1, m.viewport.Height)))
			maxWidth = max(maxWidth, relWidth)
		}
		lineNumWidth = max(4, maxWidth) + 1
		lineNumWidth = min(lineNumWidth, 10)
	}

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
	// >>> 1. PRECOMPUTE FULL VISUAL LAYOUT <<<
	// ========================================================================
	visualLayout := make([]VisualLineInfo, 0)
	for bufferRowIdx, logicalLineContent := range allLogicalLines {
		originalLineRunes := []rune(logicalLineContent)
		originalLineLen := len(originalLineRunes)
		currentLogicalColToReport := 0

		if originalLineLen == 0 && logicalLineContent == "" {
			visualLayout = append(visualLayout, VisualLineInfo{
				Content:         "",
				LogicalRow:      bufferRowIdx,
				LogicalStartCol: 0,
				IsFirstSegment:  true,
			})
			continue
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
			visualLayout = append(visualLayout, info)

			currentLogicalColToReport += segmentRunesLen
			if segIdx < len(wrappedSegmentStrings)-1 {
				for currentLogicalColToReport < originalLineLen && unicode.IsSpace(originalLineRunes[currentLogicalColToReport]) {
					currentLogicalColToReport++
				}
			}
		}
	}
	m.visualLayoutCache = visualLayout
	m.fullVisualLayoutHeight = len(visualLayout)

	// ========================================================================
	// >>> 2. Find Cursor's Absolute Visual Row and Clamped Logical Column <<<
	// ========================================================================
	absoluteTargetVisualRow := -1
	m.clampedCursorLogicalCol = cursor.Position.Col

	clampedCursorRow := cursor.Position.Row
	if clampedCursorRow < 0 {
		clampedCursorRow = 0
	} else if clampedCursorRow >= len(allLogicalLines) && len(allLogicalLines) > 0 {
		clampedCursorRow = len(allLogicalLines) - 1
	} else if len(allLogicalLines) == 0 {
		clampedCursorRow = 0
	}

	if clampedCursorRow >= 0 && clampedCursorRow < len(allLogicalLines) {
		lineContentRunes := []rune(allLogicalLines[clampedCursorRow])
		m.clampedCursorLogicalCol = max(0, min(cursor.Position.Col, len(lineContentRunes)))
	} else {
		m.clampedCursorLogicalCol = 0
	}

	if m.fullVisualLayoutHeight == 0 {
		absoluteTargetVisualRow = 0
	} else {
		for absVisRowIdx, vli := range m.visualLayoutCache {
			if vli.LogicalRow == clampedCursorRow {
				segmentRuneLen := len([]rune(vli.Content))
				if m.clampedCursorLogicalCol >= vli.LogicalStartCol {
					if (segmentRuneLen > 0 && m.clampedCursorLogicalCol <= vli.LogicalStartCol+segmentRuneLen) ||
						(segmentRuneLen == 0 && m.clampedCursorLogicalCol == vli.LogicalStartCol) {
						absoluteTargetVisualRow = absVisRowIdx
						break
					}
				}
			}
		}

		if absoluteTargetVisualRow == -1 {
			foundFirstSegment := false
			for absVisRowIdx, vli := range m.visualLayoutCache { // Use cached layout
				if vli.LogicalRow == clampedCursorRow && vli.IsFirstSegment {
					if m.clampedCursorLogicalCol == vli.LogicalStartCol {
						absoluteTargetVisualRow = absVisRowIdx
						foundFirstSegment = true
						break
					}
					if !foundFirstSegment {
						absoluteTargetVisualRow = absVisRowIdx
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

// renderVisibleSlice renders the calculated slice of the visual layout to the viewport.
func (m *Model) renderVisibleSlice() {
	state := m.editor.GetState()
	allLogicalLines := m.editor.GetBuffer().GetLines()

	selectionStyle := m.theme.SelectionStyle
	if m.yanked {
		selectionStyle = m.theme.HighlighYankStyle
	}

	lineNumWidth := 0
	if m.showLineNumbers {
		maxLineNum := len(allLogicalLines)
		maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
		if state.RelativeNumbers && !m.disableVimMode {
			relWidth := len(strconv.Itoa(max(1, m.viewport.Height)))
			maxWidth = max(maxWidth, relWidth)
		}
		lineNumWidth = max(4, maxWidth) + 1
		lineNumWidth = min(lineNumWidth, 10)
	}

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
		if len(m.visualLayoutCache) > m.cursorAbsoluteVisualRow {
			vliAtCursor := m.visualLayoutCache[m.cursorAbsoluteVisualRow]
			visualColInSegment := max(0, m.clampedCursorLogicalCol-vliAtCursor.LogicalStartCol)
			targetScreenColForCursor = lineNumWidth + visualColInSegment
		} else if m.fullVisualLayoutHeight > 0 {
			targetScreenColForCursor = lineNumWidth
		}
	} else if m.fullVisualLayoutHeight == 0 {
		targetScreenColForCursor = lineNumWidth
	}

	clampedCursorRowForLineNumbers := m.editor.GetBuffer().GetCursor().Position.Row
	if len(allLogicalLines) == 0 {
		clampedCursorRowForLineNumbers = 0
	} else {
		clampedCursorRowForLineNumbers = max(0, min(clampedCursorRowForLineNumbers, len(allLogicalLines)-1))
	}

	for absVisRowIdxToRender := startRenderVisualRow; absVisRowIdxToRender < endRenderVisualRow; absVisRowIdxToRender++ {
		if absVisRowIdxToRender < 0 || absVisRowIdxToRender >= len(m.visualLayoutCache) {
			break
		}
		vli := m.visualLayoutCache[absVisRowIdxToRender]
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

			baseCharStyle := lipgloss.NewStyle()
			charsToAdvance := 1

			var bestMatchStyle lipgloss.Style
			bestMatchLen := 0

			if len(m.highlightedWords) > 0 {
				for wordToHighlight, style := range m.highlightedWords {
					wordRunes := []rune(wordToHighlight)
					currentWordLen := len(wordRunes)
					if currentWordLen == 0 {
						continue
					}

					if charIdx+currentWordLen <= segmentLen {
						match := true
						for k := range currentWordLen {
							if segmentRunes[charIdx+k] != wordRunes[k] {
								match = false
								break
							}
						}
						if match {
							// Whole word check
							isWholeWord := true
							// Check character before the match (within segmentRunes)
							if charIdx > 0 {
								prevChar := segmentRunes[charIdx-1]
								if unicode.IsLetter(prevChar) || unicode.IsDigit(prevChar) {
									isWholeWord = false
								}
							}
							// Check character after the match (within segmentRunes)
							if charIdx+currentWordLen < segmentLen {
								nextChar := segmentRunes[charIdx+currentWordLen]
								if unicode.IsLetter(nextChar) || unicode.IsDigit(nextChar) {
									isWholeWord = false
								}
							}

							if isWholeWord && currentWordLen > bestMatchLen {
								bestMatchLen = currentWordLen
								bestMatchStyle = style
							}
						}
					}
				}
			}

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

		lineNumWidth := 0
		if m.showLineNumbers {
			maxLineNum := 1
			maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
			state := m.editor.GetState()
			if state.RelativeNumbers && !m.disableVimMode {
				relWidth := len(strconv.Itoa(max(1, m.viewport.Height)))
				maxWidth = max(maxWidth, relWidth)
			}
			lineNumWidth = max(4, maxWidth) + 1
			lineNumWidth = min(lineNumWidth, 10)
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
