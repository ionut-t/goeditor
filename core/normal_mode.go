package core

import (
	"errors"
	"fmt"
	"strings"
)

type normalMode struct {
	cmd        pendingCmd      // The partly typed command, if any
	charSearch charSearchState // Character search state (f/F/t/T)
}

// pendingCmd is the normal-mode command typed so far, before the key that
// completes it. The zero value is no command. The count typed after the
// operator (the 3 in 2d3w) stays in State.PendingCount.
type pendingCmd struct {
	count1   int           // Count typed before the operator (the 2 in 2d3w), 0 if none
	op       rune          // 'd', 'c', 'y', or 'u'/'U'/'~' for gu/gU/g~; 0 if none
	prefix   pendingPrefix // g or z waiting for its second key, also after an operator (dgg, gUgg)
	modifier rune          // 'i' or 'a' after an operator, waiting for the text object key
	literal  literalKind   // Waiting for the character after r or f/F/t/T
}

// literalKind is the command whose character argument the next key is.
type literalKind uint8

const (
	literalNone       literalKind = iota
	literalReplace                // r
	literalCharSearch             // f/F/t/T; charSearch.searchType says which
)

// hasOperator reports whether d, c or y is waiting for a motion.
func (c *pendingCmd) hasOperator() bool { return c.op != 0 && !c.hasCaseOp() }

// hasCaseOp reports whether gu, gU or g~ is waiting for a motion.
func (c *pendingCmd) hasCaseOp() bool { return c.op == 'u' || c.op == 'U' || c.op == '~' }

func NewNormalMode() EditorMode {
	return newNormalMode(&lastCharSearch{})
}

// newNormalMode builds normal mode sharing last (the last f/F/t/T search)
// with the other modes.
func newNormalMode(last *lastCharSearch) *normalMode {
	return &normalMode{charSearch: newCharSearchState(last)}
}

func (m *normalMode) Name() Mode { return NormalMode }

// AwaitingLiteral reports whether the next key is the character argument of
// 'r' or of a character search (f/F/t/T), which mappings must not rewrite.
func (m *normalMode) AwaitingLiteral() bool {
	return m.cmd.literal != literalNone
}

// OperatorPending reports whether d, y, c, gu, gU or g~ is waiting for a motion.
func (m *normalMode) OperatorPending() bool {
	return m.cmd.op != 0
}

// Idle reports whether no command is part way typed: no operator, prefix,
// text object modifier or literal argument is pending. A count typed before
// any command is not part of it; it lives in State.PendingCount.
func (m *normalMode) Idle() bool {
	return m.cmd == pendingCmd{}
}

func (m *normalMode) Enter(editor Editor, buffer Buffer) {
	editor.UpdateStatus("-- NORMAL --")
	editor.UpdateCommand("")

	m.clearPendingState(editor)
	// Clear visual selection when entering normal mode
	state := editor.GetState()
	state.VisualStart = Position{-1, -1}
	editor.SetState(state)
}

func (m *normalMode) Exit(editor Editor, buffer Buffer) {
	m.resetPending()
}

// HandleKey dispatches each incoming key to the appropriate handler.
func (m *normalMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	switch m.cmd.literal {
	case literalCharSearch:
		return m.handleCharSearchKey(editor, buffer, key)
	case literalReplace:
		return m.handleReplaceKey(editor, buffer, key)
	}
	if prefix := m.cmd.prefix.take(); prefix != 0 {
		if m.cmd.hasCaseOp() {
			return m.handleCaseGKey(editor, buffer, key)
		}
		return m.handlePrefix(prefix, editor, buffer, key)
	}
	if m.cmd.hasCaseOp() {
		return m.handleCaseMotionKey(editor, buffer, key)
	}
	if m.cmd.hasOperator() {
		return m.handlePendingOperator(editor, buffer, key)
	}
	return m.handleBaseKey(editor, buffer, key)
}

// handleCharSearchKey handles character input while waiting for the target of f/F/t/T.
func (m *normalMode) handleCharSearchKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	m.cmd.literal = literalNone
	editor.UpdateCommand("") // Clear the command display

	state := editor.GetState()
	pendingCount := state.PendingCount

	// Handle escape to cancel
	if key.Key == KeyEscape {
		m.clearPendingState(editor)
		return nil
	}

	// Get the character to search for
	if key.Rune == 0 {
		// Not a valid character
		m.clearPendingState(editor)
		return nil
	}

	count := 1
	if pendingCount != nil {
		count = *pendingCount
		editor.ResetPendingCount()
	}

	// Check if there's a pending operator (d/y/c)
	if m.cmd.hasOperator() {
		// Operator + character search motion (e.g., df,  yt;)
		op := operatorName(m.cmd.op)
		m.cmd.op = 0

		if op != "" {
			// Record the search so ;/, (and d;/d,) can repeat it later.
			m.charSearch.record(key.Rune)
			err := handleCharSearchOperator(editor, buffer, op, m.charSearch.searchType, key.Rune, count, false)
			if err != nil {
				m.clearPendingState(editor)
			}
			return err
		}
	}

	// No pending operator - just perform the character search
	m.charSearch.record(key.Rune)
	if err := performCharSearch(buffer, m.charSearch.searchType, key.Rune, count); err != nil {
		m.clearPendingState(editor)
		editor.DispatchError(ErrCharNotFoundId, err)
	}
	return nil
}

// handleReplaceKey handles character input while waiting for the replacement character after 'r'.
func (m *normalMode) handleReplaceKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	m.cmd.literal = literalNone
	editor.UpdateCommand("")

	count := max(takeCount(editor), 1)
	if key.Key == KeyEscape || key.Rune == 0 {
		return nil
	}

	return replaceCharUnderCursor(editor, buffer, key.Rune, count)
}

// handlePrefix runs the key that completes a g or z prefix.
func (m *normalMode) handlePrefix(prefix rune, editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	switch prefix {
	case 'g':
		return m.handleGKey(editor, buffer, key)
	case 'z':
		handleZKey(editor, buffer, key, takeCount(editor))
	}
	return nil
}

// handleGKey handles the second key of a 'g' prefix sequence (gg, ge, dgg, ygg, cgg,
// and the case-op prefixes gu, gU, g~).
func (m *normalMode) handleGKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	state := editor.GetState()
	pendingCount := state.PendingCount
	cursor := buffer.GetCursor()
	availableWidth := state.AvailableWidth

	if key.Key == KeyEscape {
		m.cmd.op = 0
		editor.ResetPendingCount()
		editor.UpdateCommand("")
		return nil
	}

	// Operator + gg: dgg, ygg, cgg
	if m.cmd.hasOperator() && key.Rune == 'g' {
		editor.UpdateCommand("")
		return m.handleOperatorGG(editor, buffer, cursor, state)
	}

	// Operator + ge/gE: dge, yge, cge, dgE, … — back to end of previous word
	// (inclusive motion; 'E' uses WORD semantics)
	if m.cmd.hasOperator() && (key.Rune == 'e' || key.Rune == 'E') {
		op := operatorName(m.cmd.op)
		m.cmd.op = 0
		editor.UpdateCommand("")

		geCount := 1
		if pendingCount != nil {
			geCount = *pendingCount
			editor.ResetPendingCount()
		}

		isWordChar := editor.IsWordChar
		if key.Rune == 'E' {
			isWordChar = isBigWordChar
		}

		tempCursor := cursor
		_ = tempCursor.MoveWordToEndBackward(buffer, geCount, availableWidth, isWordChar)

		// ge is inclusive: the character under the cursor is part of the range,
		// so move the exclusive end one character right.
		endCursor := cursor
		_ = endCursor.MoveRight(buffer, 1, availableWidth)

		return applyOperatorToRange(editor, buffer, op, tempCursor.Position, endCursor.Position)
	}

	// Any other key after an operator + g isn't a motion (dgw, ygu): Vim
	// cancels the whole command rather than leaving the operator pending.
	if m.cmd.hasOperator() {
		m.cmd.op = 0
		m.cmd.modifier = 0
		editor.ResetPendingCount()
		editor.UpdateCommand("")
		return nil
	}

	// Case operations: gu / gU / g~ — record the op and wait for the motion key.
	// The count typed so far is set aside to multiply the motion's count.
	if key.Rune == 'u' || key.Rune == 'U' || key.Rune == '~' {
		m.cmd.op = key.Rune
		m.startOperatorCount(editor)
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil
	}

	editor.UpdateCommand("")
	gCount := 1
	if pendingCount != nil {
		gCount = *pendingCount
		editor.ResetPendingCount()
	}
	switch key.Rune {
	case 'g': // gg — go to top; with count N go to line N
		if gCount > 1 {
			row := gCount - 1
			if row >= buffer.LineCount() {
				row = buffer.LineCount() - 1
			}
			cursor.Position.Row = row
			cursor.Position.Col = 0
		} else {
			cursor.MoveToBufferStart()
		}
		buffer.SetCursor(cursor)
	case 'e': // ge — end of previous word
		moveErr := cursor.MoveWordToEndBackward(buffer, gCount, availableWidth, editor.IsWordChar)
		if moveErr == nil || errors.Is(moveErr, ErrStartOfBuffer) {
			buffer.SetCursor(cursor)
		}
	case 'E': // gE — end of previous WORD
		moveErr := cursor.MoveWordToEndBackward(buffer, gCount, availableWidth, isBigWordChar)
		if moveErr == nil || errors.Is(moveErr, ErrStartOfBuffer) {
			buffer.SetCursor(cursor)
		}
	}
	return nil
}

// handleCaseGKey handles the key after gu / gU / g~ + g: 'g' → gugg/gUgg/g~gg
// (buffer start); the op again → gugu/gUgU/g~g~ (current line).
func (m *normalMode) handleCaseGKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	op := m.cmd.op
	m.cmd.op = 0
	editor.UpdateCommand("")

	if key.Key == KeyEscape {
		editor.ResetPendingCount()
		return nil
	}
	state := editor.GetState()
	if !state.WithInsertMode {
		editor.ResetPendingCount()
		return nil
	}
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
		editor.ResetPendingCount()
	}
	switch key.Rune {
	case 'g':
		return applyCaseToBufferStart(editor, buffer, op)
	case op: // gugu / gUgU / g~g~ — same as guu/gUU/g~~
		return applyCaseToLines(editor, buffer, op, count)
	}
	return nil
}

// handleCaseMotionKey handles the motion key after gu / gU / g~, and also the
// text object key after gu{i,a} / gU{i,a} / g~{i,a}.
func (m *normalMode) handleCaseMotionKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	op := m.cmd.op

	// ── Step 2: we already have the i/a modifier; this key is the text object ──
	if m.cmd.modifier != 0 {
		modifier := m.cmd.modifier
		m.cmd.op = 0
		m.cmd.modifier = 0
		editor.UpdateCommand("")

		if key.Key == KeyEscape {
			editor.ResetPendingCount()
			return nil
		}
		state := editor.GetState()
		if !state.WithInsertMode {
			editor.ResetPendingCount()
			return nil
		}
		editor.ResetPendingCount()
		return applyCaseToTextObject(editor, buffer, op, modifier, key.Rune)
	}

	// ── Step 1: waiting for the motion or i/a modifier ──

	// Digit keys accumulate a count for the upcoming motion (e.g., 5 in gu5G).
	// Must be checked before clearing the op so we stay in this handler.
	if accumulateCount(editor, key) {
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil
	}
	state := editor.GetState()

	m.cmd.op = 0

	if key.Key == KeyEscape || !state.WithInsertMode {
		m.clearPendingState(editor)
		editor.UpdateCommand("")
		return nil
	}

	// Compose before the i/a check: the text object key that follows reads
	// State.PendingCount directly (2gUaw).
	m.composeOperatorCount(editor)
	state = editor.GetState()

	// i/a modifier: gUiw, g~aw, gui(, … — record modifier and wait for object key.
	if key.Rune == 'i' || key.Rune == 'a' {
		m.cmd.op = op
		m.cmd.modifier = key.Rune
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil
	}

	editor.UpdateCommand("")
	hasPendingCount := state.PendingCount != nil
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
	}
	editor.ResetPendingCount()

	switch key.Rune {
	case op: // guu / gUU / g~~ — apply to current line (and count more lines)
		return applyCaseToLines(editor, buffer, op, count)
	case 'w':
		return applyCaseWord(editor, buffer, op, count)
	case 'e':
		return applyCaseWordEnd(editor, buffer, op, count)
	case 'b':
		return applyCaseWordBackward(editor, buffer, op, count)
	case '0':
		return applyCaseToLineStart(editor, buffer, op)
	case '^':
		return applyCaseToFirstNonBlank(editor, buffer, op)
	case '$':
		return applyCaseToEndOfLine(editor, buffer, op)
	case 'g': // gugg / gugu — wait for the second key
		m.cmd.op = op
		m.cmd.prefix.start('g')
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil
	case 'G':
		if hasPendingCount {
			return applyCaseToTargetLine(editor, buffer, op, count-1) // 1-indexed → 0-indexed
		}
		return applyCaseToBufferEnd(editor, buffer, op)
	}

	return nil
}

// handleOperatorGG handles dgg / ygg / cgg — operator applied from current line to buffer top.
func (m *normalMode) handleOperatorGG(editor Editor, buffer Buffer, cursor Cursor, state State) *EditorError {
	var err *EditorError

	op := m.cmd.op
	m.cmd.op = 0
	editor.ResetPendingCount()

	originalRow := cursor.Position.Row
	cursor.MoveToBufferStart()
	buffer.SetCursor(cursor)
	lineCount := originalRow + 1

	switch op {
	case 'd':
		if !state.WithInsertMode {
			return nil
		}
		var deletedContent string
		deletedContent, err = deleteLines(editor, buffer, lineCount)
		editor.DispatchSignal(DeleteSignal{content: deletedContent})
	case 'y':
		err = yankLines(editor, buffer, lineCount)
	case 'c':
		if !state.WithInsertMode {
			return nil
		}
		err = changeLines(editor, buffer, lineCount)
	}
	return err
}

// handlePendingOperator handles the second key of a d/y/c operator sequence.
func (m *normalMode) handlePendingOperator(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	opKey := m.cmd.op

	op := operatorName(opKey)
	if op == "" {
		m.cmd.op = 0
		m.cmd.modifier = 0
		return &EditorError{
			id:  ErrNoPendingOperationId,
			err: ErrNoPendingOperation,
		}
	}

	if key.Key == KeyEscape {
		m.clearPendingState(editor)
		editor.UpdateCommand("")
		return nil
	}

	state := editor.GetState()

	// Check if we're waiting for a text object after modifier (i/a)
	if m.cmd.modifier != 0 {
		count := 1
		if state.PendingCount != nil {
			count = *state.PendingCount
		}
		editor.ResetPendingCount()
		return m.handleTextObject(editor, buffer, key, op, count)
	}

	// Digit keys accumulate a count for the upcoming motion (e.g., the 2 in d2w).
	// '0' is only a digit when a count is already started; otherwise it's a motion.
	if accumulateCount(editor, key) {
		editor.UpdateCommand(fmt.Sprintf("%s%c", state.CommandLine, key.Rune))
		return nil
	}
	m.composeOperatorCount(editor)

	// Check for text object modifiers (i/a)
	if key.Rune == 'i' || key.Rune == 'a' {
		m.cmd.modifier = key.Rune
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the text object key
	}

	// Check for character search motions (f/F/t/T)
	// The pending count is preserved for handleCharSearchKey to consume.
	if key.Rune == 'f' || key.Rune == 'F' || key.Rune == 't' || key.Rune == 'T' {
		m.charSearch.searchType = key.Rune
		m.cmd.literal = literalCharSearch
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		// Keep the op - we'll process the operator after getting the character
		return nil
	}

	// Check for 'g' prefix motion (dgg, ygg, cgg, dge, ...)
	// The pending count is preserved for handleGKey to consume.
	if key.Rune == 'g' {
		m.cmd.prefix.start('g')
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Keep the op, wait for the second key
	}

	count := takeCount(editor)

	// Consume the op now if not waiting for text object
	m.cmd.op = 0

	return m.dispatchOperatorMotion(editor, buffer, key, op, opKey, count)
}

// handleTextObject handles a text object key (w, p, …) after a pending modifier (i/a).
// count extends word objects over additional words (e.g. d2aw); other objects ignore it.
func (m *normalMode) handleTextObject(editor Editor, buffer Buffer, key KeyEvent, op string, count int) *EditorError {
	var err *EditorError
	actionTaken := false

	modifier := m.cmd.modifier
	m.cmd.modifier = 0
	m.cmd.op = 0

	// Handle text objects after modifier
	switch key.Rune {
	case 'w': // iw or aw = inside/around word
		err = applyWordTextObject(editor, buffer, op, modifier, count)
		actionTaken = true
	case 'p': // ip or ap = inside/around paragraph
		switch op {
		case "yank":
			err = yankParagraphTextObject(editor, buffer, modifier)
			actionTaken = true
		case "delete":
			err = deleteParagraphTextObject(editor, buffer, modifier)
			actionTaken = true
		case "change":
			err = changeParagraphTextObject(editor, buffer, modifier)
			actionTaken = true
		}
	case '"', '\'', '`': // i"/a", i'/a', i`/a` = inside/around quote
		err = applyQuoteOp(editor, buffer, modifier, key.Rune, op)
		actionTaken = true
	case 'q': // iq/aq = inside/around any quote (", ', `)
		err = applyAnyQuoteOp(editor, buffer, modifier, op)
		actionTaken = true
	case '(', ')': // i(/a(, i)/a) = inside/around parentheses
		err = applyBracketOp(editor, buffer, modifier, '(', ')', op)
		actionTaken = true
	case 'b': // ib/ab = inside/around nearest bracket of any type (()/[]/{}/)
		err = applyAnyBracketOp(editor, buffer, modifier, op)
		actionTaken = true
	case '[', ']': // i[/a[, i]/a] = inside/around square brackets
		err = applyBracketOp(editor, buffer, modifier, '[', ']', op)
		actionTaken = true
	case '{', '}', 'B': // i{/a{, i}/a}, iB/aB = inside/around curly braces
		err = applyBracketOp(editor, buffer, modifier, '{', '}', op)
		actionTaken = true
	case '<', '>': // i</a<, i>/a> = inside/around angle brackets
		err = applyBracketOp(editor, buffer, modifier, '<', '>', op)
		actionTaken = true
	default:
		editor.DispatchError(ErrInvalidMotionId, fmt.Errorf("invalid text object '%c' after '%c'", key.Rune, modifier))
		actionTaken = true
	}

	if err != nil {
		return err
	}

	if actionTaken {
		editor.UpdateCommand("")
		return nil
	}

	return nil
}

// dispatchOperatorMotion executes the appropriate buffer operation for an operator+motion pair.
// count is 0 when none was typed; motions treat that as 1, except G, which then
// goes to the last line.
func (m *normalMode) dispatchOperatorMotion(editor Editor, buffer Buffer, key KeyEvent, op string, firstKeyRune rune, count int) *EditorError {
	var err *EditorError
	actionTaken := false
	lineCount := count
	count = max(count, 1)

	cursor := buffer.GetCursor()
	availableWidth := editor.GetState().AvailableWidth

	// Handle motion keys after the operator (op is delete/yank/change):
	//
	// Line-wise:      dd yy cc, dj dk, dG dgg
	// Character-wise: w b e W B E ge gE $ 0 ^ h l { } % f F t T ; , and i/a text objects
	switch key.Rune {
	case 'd': // dd = delete line
		if op == "delete" {
			var deletedContent string
			deletedContent, err = deleteLines(editor, buffer, count)
			editor.DispatchSignal(DeleteSignal{content: deletedContent})
			actionTaken = true
		}
	case 'y': // yy = yank line
		if op == "yank" {
			err = yankLines(editor, buffer, count)
			actionTaken = true
		}
	case 'c': // cc = change line
		if op == "change" {
			err = changeLines(editor, buffer, count)
			actionTaken = true
		}
	case 'w': // dw = delete word, yw = yank word forward, cw = change word
		switch op {
		case "delete":
			err = deleteWords(editor, buffer, count)
			actionTaken = true
		case "yank":
			err = yankWords(editor, buffer, count, true) // forward
			actionTaken = true
		case "change":
			err = changeWords(editor, buffer, count)
			actionTaken = true
		}
	case 'b': // yb = yank word backward, cb = change word backward, db = delete word backward
		switch op {
		case "delete":
			err = deleteWordsBackward(editor, buffer, count)
			actionTaken = true
		case "yank":
			err = yankWords(editor, buffer, count, false) // backward
			actionTaken = true
		case "change":
			err = changeWordsBackward(editor, buffer, count)
			actionTaken = true
		}
	case 'e': // de = delete to word end, ye = yank to word end, ce = change to word end
		switch op {
		case "delete":
			err = deleteWordToEnd(editor, buffer, count)
			actionTaken = true
		case "yank":
			err = yankWordToEnd(editor, buffer, count)
			actionTaken = true
		case "change":
			err = changeWords(editor, buffer, count) // ce and cw behave the same
			actionTaken = true
		}
	case '$': // d$ = delete to end of line, y$ = yank to end of line, c$ = change to end of line
		switch op {
		case "delete":
			var deletedContent string
			deletedContent, err = deleteToEndOfLine(editor, buffer)
			editor.DispatchSignal(DeleteSignal{content: deletedContent})
			actionTaken = true
		case "yank":
			err = yankToEndOfLine(editor, buffer)
			actionTaken = true
		case "change":
			err = changeToEndOfLine(editor, buffer)
			actionTaken = true
		}
	case 'G': // dG, yG, cG — cursor line to line count (last line if none), line-wise
		targetRow := gotoLineRow(buffer, lineCount)
		startRow := min(cursor.Position.Row, targetRow)
		endRow := max(cursor.Position.Row, targetRow)
		err = applyOperatorToLineRange(editor, buffer, op, startRow, endRow)
		actionTaken = true

	case 'j': // dj, yj, cj — current line and count lines below (line-wise)
		if cursor.Position.Row < buffer.LineCount()-1 {
			endRow := min(cursor.Position.Row+count, buffer.LineCount()-1)
			err = applyOperatorToLineRange(editor, buffer, op, cursor.Position.Row, endRow)
		}
		actionTaken = true

	case 'k': // dk, yk, ck — current line and count lines above (line-wise)
		if cursor.Position.Row > 0 {
			startRow := max(cursor.Position.Row-count, 0)
			err = applyOperatorToLineRange(editor, buffer, op, startRow, cursor.Position.Row)
		}
		actionTaken = true

	case 'h': // dh, yh, ch — count characters left of the cursor
		tempCursor := cursor
		_ = tempCursor.MoveLeft(buffer, count, availableWidth)
		err = applyOperatorToRange(editor, buffer, op, tempCursor.Position, cursor.Position)
		actionTaken = true

	case 'l': // dl, yl, cl — count characters under/after the cursor
		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		endCol := min(cursor.Position.Col+count, lineLen)
		err = applyOperatorToRange(editor, buffer, op, cursor.Position, Position{Row: cursor.Position.Row, Col: endCol})
		actionTaken = true

	case '0': // d0, y0, c0 — back to start of line
		err = applyOperatorToRange(editor, buffer, op, Position{Row: cursor.Position.Row, Col: 0}, cursor.Position)
		actionTaken = true

	case '^': // d^, y^, c^ — to first non-blank of line
		tempCursor := cursor
		tempCursor.MoveToFirstNonBlank(buffer, availableWidth)
		err = applyOperatorToRange(editor, buffer, op, tempCursor.Position, cursor.Position)
		actionTaken = true

	case '{': // d{, y{, c{ — back to previous paragraph boundary
		tempCursor := cursor
		_ = tempCursor.MoveBlockBackward(buffer, count)
		err = applyOperatorToRange(editor, buffer, op, tempCursor.Position, cursor.Position)
		actionTaken = true

	case '}': // d}, y}, c} — forward to next paragraph boundary
		tempCursor := cursor
		_ = tempCursor.MoveBlockForward(buffer, count)
		err = applyOperatorToRange(editor, buffer, op, cursor.Position, tempCursor.Position)
		actionTaken = true

	case 'W': // dW, yW, cW — WORD forward (cW changes to end of WORD, like cE)
		tempCursor := cursor
		if op == "change" {
			_ = tempCursor.MoveWordToEnd(buffer, count, availableWidth, isBigWordChar)
			_ = tempCursor.MoveRight(buffer, 1, availableWidth)
		} else {
			_ = tempCursor.MoveWordForward(buffer, count, availableWidth, isBigWordChar)
		}
		err = applyOperatorToRange(editor, buffer, op, cursor.Position, tempCursor.Position)
		actionTaken = true

	case 'E': // dE, yE, cE — to end of WORD (inclusive)
		tempCursor := cursor
		_ = tempCursor.MoveWordToEnd(buffer, count, availableWidth, isBigWordChar)
		_ = tempCursor.MoveRight(buffer, 1, availableWidth)
		err = applyOperatorToRange(editor, buffer, op, cursor.Position, tempCursor.Position)
		actionTaken = true

	case 'B': // dB, yB, cB — WORD backward
		tempCursor := cursor
		_ = tempCursor.MoveWordBackward(buffer, count, availableWidth, isBigWordChar)
		err = applyOperatorToRange(editor, buffer, op, tempCursor.Position, cursor.Position)
		actionTaken = true

	case '%': // d%, y%, c% — through the matching bracket (inclusive both ends)
		if target, ok := findMatchingBracket(buffer, cursor.Position); ok {
			start, end := NormalizeSelection(cursor.Position, target)
			err = applyOperatorToRange(editor, buffer, op, start, Position{Row: end.Row, Col: end.Col + 1})
		}
		actionTaken = true

	case ';', ',': // d;, d, — repeat the last character search as the motion
		err = repeatCharSearchOperator(&m.charSearch, editor, buffer, op, count, key.Rune == ',')
		actionTaken = true

	default:
		// Invalid motion key after operator
		editor.DispatchError(ErrInvalidMotionId, fmt.Errorf("invalid motion after '%c'", firstKeyRune))
		actionTaken = true         // Consumed the keys, even if invalid combo
		editor.ResetPendingCount() // Reset count if combo was invalid
	}

	if err != nil {
		return err // Return error from buffer operation
	}

	if actionTaken {
		editor.UpdateCommand("")
		return nil
	} // Sequence handled

	// If we fall through here, it means the second key wasn't a recognized motion
	// for the pending operator. We just discard the pending op.

	return nil
}

// handleBaseKey handles count accumulation and single-key normal-mode commands.
func (m *normalMode) handleBaseKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	var err *EditorError
	state := editor.GetState()
	pendingCount := state.PendingCount
	availableWidth := state.AvailableWidth
	skipCursorUpdate := false
	cursor := buffer.GetCursor()

	// --- Handle Numeric Input for Counts ---
	if accumulateCount(editor, key) {
		editor.UpdateCommand(fmt.Sprintf("%d", *editor.GetState().PendingCount))
		return nil // Just consuming digits, wait for command
	}
	if key.Rune == '0' {
		// '0' with no count started is the move-to-start-of-line motion
		cursor.MoveToLineStart()
		buffer.SetCursor(cursor)
		// Don't return yet, let subsequent logic handle potential errors/updates
	}

	// --- Get Count or Default to 1 ---
	// This count applies to the command/motion executed *now*
	count := 1
	countWasPending := false
	if pendingCount != nil {
		count = *pendingCount
		countWasPending = true
		// Reset count state *after* using it for this command/motion
		// We do this reset *after* the switch statement below
	}

	// --- Handle Single-Key Commands or Start of Sequences ---
	col := cursor.Position.Col // Use Column from cursor
	var moveErr error

	switch {
	// Ctrl-modified keys must be matched before the plain-letter commands:
	// normalisation puts the letter in Rune, so Ctrl+D would otherwise fall into
	// the 'd' operator, Ctrl+U into undo, Ctrl+R into replace-character, and so on.
	case key.IsCtrl('d'), key.IsCtrl('u'):
		scrollCount := 0
		if countWasPending {
			scrollCount = count
		}
		cursor, moveErr = editor.ScrollHalfPage(key.IsCtrl('d'), scrollCount)

	case key.IsCtrl('r'): // Redo
		edits, redoErr := editor.Redo()
		if len(edits) > 0 {
			// Sent even on failure: a partial redo still changed the text.
			editor.DispatchSignal(RedoSignal{edits: edits})
		}
		if redoErr != nil {
			err = &EditorError{
				id:  ErrRedoFailedId,
				err: redoErr,
			}
		}
		skipCursorUpdate = true

	case key.Modifiers&ModCtrl != 0:
		// Unbound Ctrl combination — swallow it rather than letting it reach the
		// plain-letter commands below.
		return nil

	// Movement keys
	case key.Rune == 'h' || key.Key == KeyLeft:
		moveErr = cursor.MoveLeftOrUp(buffer, count, col)
	case key.Rune == 'j' || key.Key == KeyDown:
		moveErr = cursor.MoveDown(buffer, count, availableWidth)
	case key.Rune == 'k' || key.Key == KeyUp:
		moveErr = cursor.MoveUp(buffer, count, availableWidth)
	case key.Rune == 'l' || key.Key == KeyRight || key.Key == KeySpace:
		moveErr = cursor.MoveRightOrDown(buffer, count, col)
	case key.Rune == '{':
		moveErr = cursor.MoveBlockBackward(buffer, count)
	case key.Rune == '}':
		moveErr = cursor.MoveBlockForward(buffer, count)
	case key.Rune == 'w':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'e':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'b':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, editor.IsWordChar)
	case key.Rune == 'W':
		moveErr = cursor.MoveWordForward(buffer, count, availableWidth, isBigWordChar)
	case key.Rune == 'E':
		moveErr = cursor.MoveWordToEnd(buffer, count, availableWidth, isBigWordChar)
	case key.Rune == 'B':
		moveErr = cursor.MoveWordBackward(buffer, count, availableWidth, isBigWordChar)
	case key.Rune == '%': // Jump to the matching bracket
		if target, ok := findMatchingBracket(buffer, cursor.Position); ok {
			cursor.Position = target
		}
	case key.Rune == '0':
		cursor.MoveToLineStart()
	case key.Rune == '$' || key.Key == KeyEnd:
		cursor.MoveToLineEnd(buffer, availableWidth) // Move to last char
	case key.Rune == '^' || key.Key == KeyHome:
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
	case key.Rune == 'g':
		m.cmd.prefix.start('g')
		showPrefix(editor, 'g')
		return nil // wait for second key; preserve pendingCount for 5gg etc.
	case key.Rune == 'z':
		m.cmd.prefix.start('z')
		showPrefix(editor, 'z')
		return nil // wait for second key; preserve pendingCount for 10zz etc.
	case key.Rune == 'G': // [count]G — line count, or the last line with no count
		lineCount := 0
		if countWasPending {
			lineCount = count
		}
		cursor.moveToLine(buffer, lineCount, availableWidth)
	case key.Key == KeyEnter: // Move down count lines to first non-blank
		moveErr = cursor.MoveDownToFirstNonBlank(buffer, count, availableWidth)

	// Mode changes
	case key.Rune == 'i': // Insert before cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'I': // Insert at first non-blank
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToFirstNonBlank(buffer, availableWidth)
		buffer.SetCursor(cursor) // Update buffer's cursor
		editor.SetInsertMode()
		editor.ResetPendingCount() // Clear count if entering insert mode

	case key.Rune == 'a': // Insert after cursor
		if !state.WithInsertMode {
			return nil
		}
		_ = cursor.MoveRight(buffer, 1, availableWidth) // Move one right (allows append at end of line)
		buffer.SetCursor(cursor)                        // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'A': // Insert at end of line
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Move *after* last char
		buffer.SetCursor(cursor)                          // Update buffer's cursor
		editor.SetInsertMode()

	case key.Rune == 'o': // Open line below
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToAfterLineEnd(buffer, availableWidth) // Go to end of current line
		buffer.SetCursor(cursor)
		_ = buffer.InsertRunesAt(cursor.Position.Row, cursor.Position.Col, []rune("\n")) // Insert newline
		_ = cursor.MoveDown(buffer, 1, availableWidth)                                   // Move cursor down
		cursor.MoveToFirstNonBlank(buffer, availableWidth)                               // Go to start of new line
		buffer.SetCursor(cursor)
		editor.SetInsertMode()

	case key.Rune == 'O': // Open line above
		if !state.WithInsertMode {
			return nil
		}
		cursor.MoveToLineStart()                                       // Go to start of current linavailableWidthe
		_ = buffer.InsertRunesAt(cursor.Position.Row, 0, []rune("\n")) // Insert newline (pushes current line down)
		// Cursor stays on original line index, which is now the new blank line
		cursor.MoveToFirstNonBlank(buffer, availableWidth) // Ensure col=0 on the new line
		buffer.SetCursor(cursor)
		editor.SetInsertMode()

	case key.Rune == 'v': // Enter visual mode
		editor.SetVisualMode()

	case key.Rune == 'V': // Enter visual line mode
		editor.SetVisualLineMode()

	case key.Key == KeyEscape:
		// If pending count or op, clear them
		m.cmd.op = 0
		editor.UpdateCommand("") // Clear count display
		editor.SetNormalMode()

	case key.Rune == ':': // Enter command mode
		editor.SetCommandMode()

	case key.Rune == '/': // Enter forward search mode
		editor.SetSearchMode()

	case key.Rune == '?': // Enter backward search mode
		editor.SetBackwardSearchMode()

	case key.Rune == 'n': // Go to next search result
		cursor = editor.NextSearchResult()

	case key.Rune == 'N': // Go to previous search result
		cursor = editor.PreviousSearchResult()

	case key.Rune == '*': // Search word under cursor forward
		cursor = editor.SearchWordUnderCursor(false)

	case key.Rune == '#': // Search word under cursor backward
		cursor = editor.SearchWordUnderCursor(true)

	// Character search motions
	case isCharSearchKey(key): // f / F / t / T
		m.charSearch.searchType = key.Rune
		m.cmd.literal = literalCharSearch
		showPrefix(editor, key.Rune)
		return nil // the count stays pending for the target character

	case key.Rune == ';': // Repeat last character search
		cursor = m.handleCharSearchRepeat(editor, buffer, false)

	case key.Rune == ',': // Repeat last character search in opposite direction
		cursor = m.handleCharSearchRepeat(editor, buffer, true)

	// Editing commands (single key or start of sequence)
	case key.Rune == 'x': // Delete character under cursor
		if !state.WithInsertMode {
			return nil
		}

		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		if cursor.Position.Col < lineLen { // Only delete if cursor is on a char
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col, count)

			// Ensure cursor doesn't go past end of line after deletion
			cursor = buffer.GetCursor() // Refresh cursor state
			finalCol := cursor.Position.Col
			newLineLen := buffer.LineRuneCount(cursor.Position.Row)
			if finalCol >= newLineLen && newLineLen > 0 {
				cursor.Position.Col = newLineLen - 1
			} else if newLineLen == 0 {
				cursor.Position.Col = 0
			}
			buffer.SetCursor(cursor)
		}
	case key.Rune == 'X': // Delete character before cursor
		if !state.WithInsertMode {
			return nil
		}

		if cursor.Position.Col > 0 {
			err = buffer.DeleteRunesAt(cursor.Position.Row, cursor.Position.Col-1, count)
			if err == nil {
				_ = cursor.MoveLeft(buffer, count, availableWidth) // Move cursor back
				buffer.SetCursor(cursor)
			}
		} else {
			err = &EditorError{
				id:  ErrStartOfLineId,
				err: ErrStartOfLine,
			}
		}

	case key.Rune == 'D': // Delete to end of line (equivalent to d$)
		if !state.WithInsertMode {
			return nil
		}

		var deletedContent string
		deletedContent, err = deleteToEndOfLine(editor, buffer)
		editor.DispatchSignal(DeleteSignal{content: deletedContent})

	case key.Rune == 'r': // Replace character under cursor
		if !state.WithInsertMode {
			return nil
		}

		m.cmd.literal = literalReplace
		editor.UpdateCommand("r")
		return nil

	case key.Rune == '~': // Toggle case of count characters under/after cursor
		if !state.WithInsertMode {
			return nil
		}
		err = toggleCaseChar(editor, buffer, count)
		skipCursorUpdate = true

	case key.Rune == 's': // Substitute character(s) — like 'cl'
		if !state.WithInsertMode {
			return nil
		}

		lineLen := buffer.LineRuneCount(cursor.Position.Row)
		endCol := min(cursor.Position.Col+count, lineLen)
		err = applyOperatorToRange(editor, buffer, "change", cursor.Position, Position{Row: cursor.Position.Row, Col: endCol})
		if err == nil && !editor.IsInsertMode() {
			editor.SetInsertMode() // empty line: 's' still enters insert mode
		}
		skipCursorUpdate = true

	case key.Rune == 'S': // Substitute line(s) — like 'cc'
		if !state.WithInsertMode {
			return nil
		}

		err = changeLines(editor, buffer, count)
		skipCursorUpdate = true

	case key.Rune == 'J': // Join lines
		if !state.WithInsertMode {
			return nil
		}

		err = joinLines(editor, buffer, count)
		skipCursorUpdate = true

	case key.Rune == 'C': // Change to end of line (equivalent to c$)
		if !state.WithInsertMode {
			return nil
		}

		err = changeToEndOfLine(editor, buffer)
	case key.Rune == 'd': // Start 'delete' operation
		if !state.WithInsertMode {
			return nil
		}

		m.cmd.op = key.Rune
		m.startOperatorCount(editor)
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))

		return nil // Wait for the next key (motion)

	case key.Rune == 'c': // Start 'change' operation
		if !state.WithInsertMode {
			return nil
		}

		m.cmd.op = key.Rune
		m.startOperatorCount(editor)
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'y': // Start 'yank' operation
		m.cmd.op = key.Rune
		m.startOperatorCount(editor)
		editor.UpdateCommand(fmt.Sprintf("%s%c", editor.GetState().CommandLine, key.Rune))
		return nil // Wait for the next key (motion)

	case key.Rune == 'p':
		if !state.WithInsertMode {
			return nil
		}

		content, pasteErr := editor.PasteAfter()

		if strings.HasSuffix(content, "\n") {
			// Linewise paste: PasteAfter() already inserted below and set cursor to (row+1, 0).
			// Repeat for any additional count (e.g. 3p pastes 3 lines).
			for i := 1; i < count; i++ {
				if _, loopErr := editor.PasteAfter(); loopErr != nil {
					pasteErr = loopErr
					break
				}
			}
			// Cursor was set inside PasteAfter(); refresh and skip the normal MoveRight.
			cursor = buffer.GetCursor()
			skipCursorUpdate = true
		} else {
			count = len(content)
			_ = cursor.MoveRight(buffer, count, availableWidth)
		}

		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}

	case key.Rune == 'P':
		if !state.WithInsertMode {
			return nil
		}

		content, pasteErr := editor.PasteBefore()

		if strings.HasSuffix(content, "\n") {
			// Linewise paste above: PasteBefore() inserted above and cursor stays at the same row.
			// Repeat for any additional count (e.g. 3P pastes 3 lines above).
			for i := 1; i < count; i++ {
				if _, loopErr := editor.PasteBefore(); loopErr != nil {
					pasteErr = loopErr
					break
				}
			}
			cursor = buffer.GetCursor()
			skipCursorUpdate = true
		} else {
			count = len(content)
			_ = cursor.MoveRight(buffer, count, availableWidth)
		}

		if pasteErr != nil {
			err = &EditorError{
				id:  ErrFailedToPasteId,
				err: pasteErr,
			}
		} else {
			editor.DispatchSignal(PasteSignal{content: content})
		}

	case key.Rune == 'u': // Undo
		edits, undoErr := editor.Undo()
		if len(edits) > 0 {
			// Sent even on failure: a partial undo still changed the text.
			editor.DispatchSignal(UndoSignal{edits: edits})
		}
		if undoErr != nil {
			err = &EditorError{
				id:  ErrUndoFailedId,
				err: undoErr,
			}
		}
		skipCursorUpdate = true

	case key.Rune == 'U': // Undo all recent changes on the last-changed line
		if !state.WithInsertMode {
			return nil
		}

		edits, undoErr := editor.UndoLine()
		if len(edits) > 0 {
			// Sent even on failure: a partial undo still changed the text.
			editor.DispatchSignal(UndoLineSignal{edits: edits})
		}
		if undoErr != nil {
			err = &EditorError{
				id:  ErrUndoLineFailedId,
				err: undoErr,
			}
		}
		skipCursorUpdate = true

	case key.Rune == '.': // Repeat the last change, with count replacing its count
		n := takeCount(editor) // even when editing is off, so it can't reach the next command
		countWasPending = false
		if !state.WithInsertMode {
			return nil
		}
		if r, ok := editor.(changeRepeater); ok {
			err = r.repeatChange(n)
		}
		skipCursorUpdate = true

	case key.Key == KeyBackspace: // Delete character before cursor
		moveErr = cursor.MoveLeft(buffer, count, availableWidth)

	default:
		// Unknown key - clear pending state if an unrecognized key is pressed
		m.cmd.op = 0
	}

	// Reset count after any command that consumed it (excludes early returns for
	// digit accumulation and multi-key sequences like d/y/c which return early)
	if countWasPending {
		editor.ResetPendingCount()
	}

	boundaryErr := isBoundaryError(moveErr)

	// Update cursor in buffer if no error or only boundary error
	// SKIP THIS IF WE JUST DID UNDO/REDO
	if !skipCursorUpdate && ((err == nil && moveErr == nil) || boundaryErr) {
		buffer.SetCursor(cursor) // Update the buffer's cursor state

		// Don't return boundary errors to the editor loop
		if err != nil && !boundaryErr {
			return err // Return actual errors (e.g., from delete/insert)
		}

		return nil
	}

	return err // Return other errors
}

// changeRepeater is implemented by the editor so that normal mode can run '.'
// without it being part of the public Editor interface.
type changeRepeater interface {
	repeatChange(count int) *EditorError
}

// operatorName maps a d/y/c rune to its operation name string.
func operatorName(r rune) string {
	switch r {
	case 'd':
		return "delete"
	case 'y':
		return "yank"
	case 'c':
		return "change"
	}
	return ""
}

// handleCharSearchRepeat handles repeating (;) or reversing (,) the last character search.
func (m *normalMode) handleCharSearchRepeat(editor Editor, buffer Buffer, reverse bool) Cursor {
	state := editor.GetState()
	count := 1
	if state.PendingCount != nil {
		count = *state.PendingCount
		editor.ResetPendingCount()
	}

	repeatCharSearch(&m.charSearch, editor, buffer, count, reverse)

	return buffer.GetCursor() // Return refreshed cursor
}

// clearPendingState resets all pending state in normal mode, including the count.
func (m *normalMode) clearPendingState(editor Editor) {
	m.resetPending()
	editor.ResetPendingCount()
}

// resetPending clears every partly typed command. It is the one place that
// knows the full list, so a mode switch can't leave a stale prefix behind.
func (m *normalMode) resetPending() {
	m.cmd = pendingCmd{}
	m.charSearch.cancel() // keeps the last search for ; and ,
}

// startOperatorCount sets aside the count typed before an operator, so the
// count typed after it starts fresh (the 2 and the 3 in 2d3w).
func (m *normalMode) startOperatorCount(editor Editor) {
	m.cmd.count1 = 0
	state := editor.GetState()
	if state.PendingCount == nil {
		return
	}
	m.cmd.count1 = *state.PendingCount
	// Not ResetPendingCount: that clears the command line, which should
	// still show the count (2d).
	state.PendingCount = nil
	editor.SetState(state)
}

// composeOperatorCount multiplies the count typed before the operator into
// the one typed after it, as Vim does: 2d3w deletes 6 words. It runs on the
// first key after the operator that isn't a count digit, so the motion
// handlers read the effective count from State.PendingCount.
func (m *normalMode) composeOperatorCount(editor Editor) {
	if m.cmd.count1 == 0 {
		return
	}
	n := m.cmd.count1
	m.cmd.count1 = 0
	state := editor.GetState()
	if state.PendingCount != nil {
		n = mulCount(n, *state.PendingCount)
	}
	state.PendingCount = &n
	editor.SetState(state)
}
