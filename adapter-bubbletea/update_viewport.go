package bubble_adapter

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/core"
)

// VisualLineInfo holds data about a single line as it appears visually after wrapping.
type VisualLineInfo struct {
	Content         string // The text content of this visual line segment
	LogicalRow      int    // The original row index in the logical buffer
	LogicalStartCol int    // The starting column index in the original logical line for this segment
	IsFirstSegment  bool   // Flag if this is the first visual segment for the logical line
}

/*
// updateViewport refreshes the viewport content based on the editor buffer,
// adding line wrapping and ensuring correct cursor positioning with word wrap
// by pre-calculating the visual layout.
func (m *Model) updateViewport() {
	buffer := m.editor.GetBuffer()
	state := m.editor.GetState()
	cursor := buffer.GetCursor()
	lines := buffer.GetLines() // Original buffer lines as strings

	selectionStyle := m.theme.SelectionStyle
	if m.yanked {
		selectionStyle = m.theme.HighlighYankStyle
	}

	// --- Ensure viewport state is valid ---
	if state.TopLine < 0 {
		state.TopLine = 0
	} else if state.TopLine >= len(lines) {
		if len(lines) == 0 {
			state.TopLine = 0
		} else {
			state.TopLine = max(0, len(lines)-1)
		}
	}
	if m.viewport.Height <= 0 {
		if m.height > 2 {
			m.viewport.Height = m.height - 2
			if m.viewport.Height <= 0 {
				m.viewport.Height = 1
			}
		} else {
			m.viewport.Height = 1
		}
		state.ViewportHeight = m.viewport.Height
	}

	// --- Calculate Layout ---
	lineNumWidth := 0
	if m.showLineNumbers {
		maxLineNum := len(lines)
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
		state.AvailableWidth = availableWidth
		m.editor.SetState(state) // Update state if width changed
	}

	// ========================================================================
	// >>> PRECOMPUTE VISUAL LAYOUT <<<
	// ========================================================================
	visualLayout := make([]VisualLineInfo, 0)
	for bufferRow := range lines {
		lineRunes := []rune(lines[bufferRow])
		lineLen := len(lineRunes)
		wrappedSegments := wrapLine(string(lineRunes), availableWidth)
		currentLogicalCol := 0
		for segmentIdx, segment := range wrappedSegments {
			info := VisualLineInfo{
				Content:         segment,
				LogicalRow:      bufferRow,
				LogicalStartCol: currentLogicalCol,
				IsFirstSegment:  segmentIdx == 0,
			}
			visualLayout = append(visualLayout, info)
			potentialEndLogical := min(currentLogicalCol+availableWidth, lineLen)
			nextLogicalStart := potentialEndLogical
			if potentialEndLogical < lineLen {
				lastSpace := -1
				for i := potentialEndLogical - 1; i >= currentLogicalCol; i-- {
					if unicode.IsSpace(lineRunes[i]) {
						lastSpace = i
						break
					}
				}
				if lastSpace >= currentLogicalCol {
					nextLogicalStart = lastSpace + 1
				}
			}
			currentLogicalCol = nextLogicalStart
			for currentLogicalCol < lineLen && unicode.IsSpace(lineRunes[currentLogicalCol]) {
				currentLogicalCol++
			}
		}
	}
	// ========================================================================

	// ========================================================================
	// >>> Find Cursor Position in Visual Layout <<<
	// ========================================================================
	targetVisualRow := -1         // Relative to viewport *content* (0-based)
	targetVisualCol := -1         // Including gutter
	absoluteTargetVisualRow := -1 // Absolute visual row (0-based index in visualLayout)
	// cursorIsVisible := false      // We don't need this flag anymore within this function

	clampedCursorCol := cursor.Position.Col
	if cursor.Position.Row >= 0 && cursor.Position.Row < len(lines) {
		clampedCursorCol = max(0, min(cursor.Position.Col, len([]rune(lines[cursor.Position.Row]))))
	} else if len(lines) == 0 {
		clampedCursorCol = 0
	}

	// Find the absolute visual row and the relative visual position *if* it's within the current offset
	for absVisRow, vli := range visualLayout {
		if vli.LogicalRow == cursor.Position.Row {
			segmentLen := len([]rune(vli.Content))
			if clampedCursorCol >= vli.LogicalStartCol && clampedCursorCol <= vli.LogicalStartCol+segmentLen {
				absoluteTargetVisualRow = absVisRow // Store the absolute row
				visualColInSegment := clampedCursorCol - vli.LogicalStartCol
				targetVisualCol = lineNumWidth + visualColInSegment // Store the target visual column

				// Calculate the relative row for rendering, based on the *current* viewport offset
				if absoluteTargetVisualRow >= m.viewport.YOffset && absoluteTargetVisualRow < m.viewport.YOffset+m.viewport.Height {
					targetVisualRow = absoluteTargetVisualRow - m.viewport.YOffset
				} else {
					targetVisualRow = -1 // Mark as not renderable in the current frame
				}
				break // Found the cursor's segment
			}
		}
	}
	// Handle edge cases for cursor at end of line / empty buffer
	if absoluteTargetVisualRow == -1 && cursor.Position.Row >= 0 && cursor.Position.Row < len(lines) && clampedCursorCol > 0 && clampedCursorCol == len([]rune(lines[cursor.Position.Row])) {
		for absVisRow := len(visualLayout) - 1; absVisRow >= 0; absVisRow-- {
			vli := visualLayout[absVisRow]
			if vli.LogicalRow == cursor.Position.Row {
				absoluteTargetVisualRow = absVisRow
				targetVisualCol = lineNumWidth + len([]rune(vli.Content))
				if absoluteTargetVisualRow >= m.viewport.YOffset && absoluteTargetVisualRow < m.viewport.YOffset+m.viewport.Height {
					targetVisualRow = absoluteTargetVisualRow - m.viewport.YOffset
				} else {
					targetVisualRow = -1
				}
				break
			}
		}
	}
	if len(visualLayout) == 0 {
		absoluteTargetVisualRow = 0
		targetVisualRow = 0
		targetVisualCol = lineNumWidth
	}
	// ========================================================================

	// --- Build Content String ---
	var contentBuilder strings.Builder
	displayLineCount := 0
	// Use the viewport's current YOffset to determine which part of the visualLayout to render
	startRenderRow := m.viewport.YOffset
	endRenderRow := min(m.viewport.YOffset+m.viewport.Height, len(visualLayout))

	// (Keep the rendering loop logic from the previous version - it correctly uses targetVisualRow/Col)
	for absVisRow := startRenderRow; absVisRow < endRenderRow; absVisRow++ {
		vli := visualLayout[absVisRow]
		currentViewportRow := displayLineCount // This is the row index within the final rendered string (0-based)
		if m.showLineNumbers {
			lineNumStr := ""
			currentLineNumberStyle := m.theme.LineNumberStyle
			if vli.IsFirstSegment {
				if state.RelativeNumbers && !m.disableVimMode && vli.LogicalRow != cursor.Position.Row {
					relNum := vli.LogicalRow - cursor.Position.Row
					if relNum < 0 {
						relNum = -relNum
					}
					lineNumStr = strconv.Itoa(relNum)
				} else {
					lineNumStr = strconv.Itoa(vli.LogicalRow + 1)
				}
				if vli.LogicalRow == cursor.Position.Row {
					currentLineNumberStyle = m.theme.CurrentLineNumberStyle
				}
			}
			gutterStyleWithWidth := currentLineNumberStyle.Width(lineNumWidth - 1)
			contentBuilder.WriteString(gutterStyleWithWidth.Render(lineNumStr) + " ")
		}
		segmentRunes := []rune(vli.Content)
		segmentLen := len(segmentRunes)
		visualColOffset := lineNumWidth
		for charIdx, ch := range segmentRunes {
			bufferCol := vli.LogicalStartCol + charIdx
			bufferPos := editor.Position{Row: vli.LogicalRow, Col: bufferCol}
			selectionStatus := m.editor.GetSelectionStatus(bufferPos)
			charStyle := lipgloss.NewStyle()
			if selectionStatus != editor.SelectionNone {
				charStyle = selectionStyle
			}
			currentVisualCol := visualColOffset + charIdx
			// isCursorOnChar uses targetVisualRow (relative) and targetVisualCol
			isCursorOnChar := (currentViewportRow == targetVisualRow && currentVisualCol == targetVisualCol)
			if isCursorOnChar {
				cursorModeStyle := m.theme.NormalModeStyle
				switch state.Mode {
				case editor.InsertMode:
					cursorModeStyle = m.theme.InsertModeStyle
				case editor.VisualMode, editor.VisualLineMode:
					cursorModeStyle = m.theme.VisualModeStyle
				case editor.CommandMode:
					cursorModeStyle = m.theme.CommandModeStyle
				}
				contentBuilder.WriteString(charStyle.Render(cursorModeStyle.Render(string(ch))))
			} else {
				contentBuilder.WriteString(charStyle.Render(string(ch)))
			}
		}
		// isCursorAfterSegmentEnd uses targetVisualRow (relative) and targetVisualCol
		isCursorAfterSegmentEnd := (currentViewportRow == targetVisualRow && (visualColOffset+segmentLen) == targetVisualCol)
		if isCursorAfterSegmentEnd {
			cursorModeStyle := m.theme.NormalModeStyle
			switch state.Mode {
			case editor.InsertMode:
				cursorModeStyle = m.theme.InsertModeStyle
			case editor.VisualMode, editor.VisualLineMode:
				cursorModeStyle = m.theme.VisualModeStyle
			case editor.CommandMode:
				cursorModeStyle = m.theme.CommandModeStyle
			}
			cursorSelectionStatus := m.editor.GetSelectionStatus(cursor.Position)
			baseStyle := lipgloss.NewStyle()
			if cursorSelectionStatus != editor.SelectionNone {
				baseStyle = selectionStyle
			}
			contentBuilder.WriteString(baseStyle.Render(cursorModeStyle.Render(" ")))
		}
		contentBuilder.WriteString("\n")
		displayLineCount++
	}

	// --- Fill remaining viewport height ---
	// (Keep the tilde filling logic)
	for displayLineCount < m.viewport.Height {
		tildeStyle := m.theme.LineNumberStyle
		if m.showLineNumbers {
			contentBuilder.WriteString(tildeStyle.Width(lineNumWidth-1).Render("~") + " ")
		} else {
			contentBuilder.WriteString(tildeStyle.Render("~"))
		}
		contentBuilder.WriteString("\n")
		displayLineCount++
	}

	// --- Finalize and Set Content ---
	finalContent := strings.TrimSuffix(contentBuilder.String(), "\n")
	// Update the bubbletea viewport's visible content AND its total height awareness
	m.viewport.SetContent(finalContent)
	m.editor.ScrollViewport()

	// ========================================================================
	// >>> START: Viewport Scrolling Logic (REMOVED) <<<
	// ========================================================================
	// NO LONGER forcing scroll offset here.
	// Scrolling is now handled by passing tea.KeyMsg/tea.MouseMsg to m.viewport.Update()
	// in the main Model.Update function.

	// Optional: Final bounds check on YOffset (belt-and-suspenders)
	// Although the viewport should manage its offset internally when updated correctly.
	// maxOffset := max(0, len(visualLayout)-m.viewport.Height)
	// if m.viewport.YOffset < 0 {
	// 	// m.viewport.YOffset = 0
	// }
	// if m.viewport.YOffset > maxOffset {
	// 	// m.viewport.YOffset = maxOffset
	// }
	// ========================================================================
	// >>> END: Viewport Scrolling Logic <<<
	// ========================================================================
}
*/
// wrapLine splits a line into multiple segments based on width, respecting word boundaries.
// It implements Vim-like line wrapping that avoids splitting words across lines when possible.
func wrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line} // Avoid issues with invalid width
	}

	// Handle empty lines
	if line == "" {
		return []string{""}
	}

	var wrappedLines []string
	runes := []rune(line)
	lineLen := len(runes)
	start := 0

	for start < lineLen {
		// If remaining text fits in one line, add it and we're done
		if start+width >= lineLen {
			wrappedLines = append(wrappedLines, string(runes[start:]))
			break
		}

		// Calculate end position - don't exceed line length
		end := min(start+width, lineLen)

		// Find the last space within the width
		lastSpace := -1
		for i := end - 1; i >= start; i-- {
			if runes[i] == ' ' || runes[i] == '\t' {
				lastSpace = i
				break
			}
		}

		if lastSpace >= start {
			// Found a space to break at
			wrappedLines = append(wrappedLines, string(runes[start:lastSpace]))
			// Start next segment after the space
			start = lastSpace + 1

			// Skip any leading spaces on the next line
			for start < lineLen && (runes[start] == ' ' || runes[start] == '\t') {
				start++
			}
		} else {
			// No space found in this segment - we need to decide how to break

			// Case 1: Are we already at the start of a word?
			isWordStart := start == 0 || (start > 0 && (runes[start-1] == ' ' || runes[start-1] == '\t'))

			if isWordStart {
				// We're at the start of a word that's longer than width - must break it
				wrappedLines = append(wrappedLines, string(runes[start:end]))
				start = end
			} else {
				// We're in the middle of a word
				// Find the start of the current word
				wordStart := 0 // Default to beginning of line
				for i := start - 1; i >= 0; i-- {
					if runes[i] == ' ' || runes[i] == '\t' {
						wordStart = i + 1
						break
					}
				}

				// If at this point we're at the start position, we need to break at width
				if wordStart == start {
					wrappedLines = append(wrappedLines, string(runes[start:end]))
					start = end
				} else {
					// Break at the beginning of the current word
					wrappedLines = append(wrappedLines, string(runes[start:wordStart]))
					start = wordStart
				}
			}
		}

		// If we ended up with an empty segment (possible in some edge cases),
		// just move to the next character to avoid infinite loops
		if start < lineLen && len(wrappedLines) > 0 && wrappedLines[len(wrappedLines)-1] == "" {
			start++
		}
	}

	// Handle the case where all processing resulted in no lines
	if len(wrappedLines) == 0 {
		return []string{""}
	}

	return wrappedLines
}

// updateViewport refreshes the viewport content based on the editor buffer, adding line wrapping.
func (m *Model) updateViewport() {
	buffer := m.editor.GetBuffer()
	state := m.editor.GetState()
	cursor := buffer.GetCursor()
	lines := buffer.GetLines() // Original buffer lines as strings

	selectionStyle := m.theme.SelectionStyle

	if m.yanked {
		selectionStyle = m.theme.HighlighYankStyle
	}

	// --- Ensure viewport state is valid ---
	// Ensure TopLine index is valid
	if state.TopLine < 0 {
		state.TopLine = 0
	} else if state.TopLine >= len(lines) {
		// Handle case where TopLine might be invalid after buffer changes (e.g., deletions)
		if len(lines) == 0 {
			state.TopLine = 0
		} else {
			// Adjust TopLine to ensure the last line is potentially visible
			// This might need more sophisticated logic depending on desired scroll behavior
			state.TopLine = max(0, len(lines)-1) // Simple fallback: go to last line index
			// A better approach might involve clamping based on viewport height:
			// state.TopLine = max(0, len(lines)-state.ViewportHeight)
		}
		// Update the editor's state if we corrected TopLine (optional, depends if state should persist)
		// m.editor.SetState(state)
	}
	// Ensure ViewportHeight is positive
	if state.ViewportHeight <= 0 {
		state.ViewportHeight = 1 // Avoid division by zero or negative loops
	}

	// --- Calculate Layout ---
	lineNumWidth := 0
	if m.showLineNumbers {
		maxLineNum := len(lines)
		// Calculate width needed for largest absolute or relative number shown
		maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
		if state.RelativeNumbers {
			// Max relative number could be ViewportHeight or distance to start/end
			relWidth := len(strconv.Itoa(max(1, state.ViewportHeight))) // Approximate
			maxWidth = max(maxWidth, relWidth)
		}
		lineNumWidth = max(4, maxWidth) + 1  // Minimum width 4, plus 1 for padding space
		lineNumWidth = min(lineNumWidth, 10) // Clamp max width to 10
	}

	// Calculate available width for text content
	availableWidth := m.viewport.Width - lineNumWidth
	if availableWidth <= 0 {
		availableWidth = 1 // Ensure at least 1 character width for content
	}

	// --- Build Content String ---
	var contentBuilder strings.Builder
	displayLineCount := 0 // Track how many visual lines (including wraps) we've added

	// Iterate through buffer lines visible in the viewport
	for bufferRow := state.TopLine; bufferRow < len(lines) && displayLineCount < state.ViewportHeight; bufferRow++ {
		originalLine := lines[bufferRow]
		lineRunes := []rune(originalLine) // Work with runes for indexing
		lineLen := len(lineRunes)
		wrappedSegments := wrapLine(string(lineRunes), availableWidth) // Wrap based on runes

		// Render each segment of the (potentially wrapped) line
		for segmentIdx, segment := range wrappedSegments {
			if displayLineCount >= state.ViewportHeight {
				break
			} // Stop if viewport is full

			segmentRunes := []rune(segment)
			segmentLen := len(segmentRunes)

			// --- Render Line Numbers ---
			if m.showLineNumbers {
				lineNumStr := ""
				currentLineNumberStyle := m.theme.LineNumberStyle
				if segmentIdx == 0 { // Only show number on the first segment of a wrapped line
					if state.RelativeNumbers && bufferRow != cursor.Position.Row {
						relNum := bufferRow - cursor.Position.Row
						if relNum < 0 {
							relNum = -relNum
						} // Absolute difference
						lineNumStr = strconv.Itoa(relNum)
					} else { // Absolute number or the current line (0 in relative mode)
						lineNumStr = strconv.Itoa(bufferRow + 1)
					}
					// Highlight the current line number differently
					if bufferRow == cursor.Position.Row {
						currentLineNumberStyle = m.theme.CurrentLineNumberStyle
					}
				}
				// Apply fixed width alignment to the chosen style
				gutterStyleWithWidth := currentLineNumberStyle.Width(lineNumWidth - 1) // -1 for trailing space
				gutterContent := gutterStyleWithWidth.Render(lineNumStr) + " "

				contentBuilder.WriteString(gutterContent) // Render with default background
			}
			// --- End Line Number ---

			// --- Render Segment Content ---
			styledSegment := strings.Builder{}
			for charIdx, ch := range segmentRunes {
				// Calculate the column index within the original, unwrapped line
				bufferCol := (segmentIdx * availableWidth) + charIdx
				bufferPos := editor.Position{Row: bufferRow, Col: bufferCol}

				// Get selection status from the editor core
				selectionStatus := m.editor.GetSelectionStatus(bufferPos)

				// Determine base background style based on selection
				charStyle := lipgloss.NewStyle() // Default: transparent background

				if selectionStatus != editor.SelectionNone {
					charStyle = selectionStyle // Apply selection style background
				}

				// Check if the cursor is exactly on this character
				isCursor := (bufferPos.Row == cursor.Position.Row && bufferPos.Col == cursor.Position.Col)
				if isCursor {
					cursorModeStyle := m.theme.NormalModeStyle // Default cursor style
					switch state.Mode {
					case "insert":
						cursorModeStyle = m.theme.InsertModeStyle
					case "visual", "visual-line":
						cursorModeStyle = m.theme.VisualModeStyle
					case "command":
						cursorModeStyle = m.theme.CommandModeStyle
					}
					// Render the character *with* the cursor mode style (overrides selection style for this char)
					styledSegment.WriteString(cursorModeStyle.Render(string(ch)))
				} else {
					// Render character with its determined style (default or selection)
					styledSegment.WriteString(charStyle.Render(string(ch)))
				}
			} // End character loop for segment

			// Write the fully styled segment to the main content builder
			contentBuilder.WriteString(styledSegment.String())
			// --- End Render Segment Content ---

			// --- Handle Cursor Rendering if it's *after* the last char of this segment ---
			// (Handles cursor at end-of-line or end-of-wrapped-segment)
			isCursorAfterSegment := cursor.Position.Row == bufferRow &&
				cursor.Position.Col == (segmentIdx*availableWidth)+segmentLen
			// Check if this segment is the *last* segment for the line AND the cursor is exactly at the end
			isCursorAtEndOfOriginalLine := (segmentIdx == len(wrappedSegments)-1) &&
				(cursor.Position.Row == bufferRow && cursor.Position.Col == lineLen)

			// Render cursor block if cursor is logically after the content on this visual line
			if isCursorAfterSegment || isCursorAtEndOfOriginalLine {
				// Determine the cursor block's style based on mode
				cursorModeStyle := m.theme.NormalModeStyle // Default cursor style
				switch state.Mode {
				case editor.InsertMode:
					cursorModeStyle = m.theme.InsertModeStyle
				case editor.VisualMode, editor.VisualLineMode:
					cursorModeStyle = m.theme.VisualModeStyle
				case editor.CommandMode:
					cursorModeStyle = m.theme.CommandModeStyle
				}

				// Determine the background style for the cursor block position
				cursorRenderPos := cursor.Position // Use the actual cursor position
				cursorSelectionStatus := m.editor.GetSelectionStatus(cursorRenderPos)

				baseStyle := lipgloss.NewStyle() // Default background for cursor space
				switch cursorSelectionStatus {
				case editor.SelectionLine, editor.SelectionCharacter:
					baseStyle = selectionStyle // Apply selection style background
					// If cursor is at Col 0 of a line selected line-wise, apply line style
					// (The GetSelectionStatus should handle this based on row)
				}
				// Render cursor block (space) with cursor style, potentially on a selection background
				contentBuilder.WriteString(baseStyle.Render(cursorModeStyle.Render(" ")))
			}
			// --- End Cursor Handling ---

			// Add newline for the next visual line
			contentBuilder.WriteString("\n")
			displayLineCount++
		} // End loop over segments (wrapped parts of a line)
	} // End loop over buffer lines

	// --- Fill remaining viewport height with empty lines (optional) ---
	// You might want to add '~' characters for lines below the end of the buffer
	// for displayLineCount < state.ViewportHeight {
	// 	if m.showLineNums {
	// 		contentBuilder.WriteString(lineNumberStyle.Width(lineNumWidth-1).Render("~") + " ")
	// 	}
	// 	contentBuilder.WriteString("\n")
	// 	displayLineCount++
	// }

	// --- Finalize and Set Content ---
	// Remove trailing newline which might be added unnecessarily by the loops
	finalContent := strings.TrimSuffix(contentBuilder.String(), "\n")
	m.viewport.SetContent(finalContent)

	// --- Ensure Cursor is Visible ---
	// This basic implementation sets the content string. For perfect scrolling
	// that keeps the cursor visible within the viewport, especially with wrapping,
	// more complex logic interacting with viewport.SetYOffset or viewport.GotoX/Y
	// might be needed, potentially calculated *after* the content string is built.
	// The current m.editor.ScrollViewport() likely only adjusts state.TopLine based on
	// buffer rows, not visual wrapped lines.
	// For now, we rely on ScrollViewport being called elsewhere to adjust TopLine.
}
