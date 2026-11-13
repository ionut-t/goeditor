package core

type commandMode struct {
	baseMode
	text   []rune
	cursor int // index into text, 0..len(text)
}

func NewCommandMode() EditorMode  { return &commandMode{} }
func (m *commandMode) Name() Mode { return CommandMode }

func (m *commandMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterCommandModeSignal{})
	m.text = nil
	m.cursor = 0
	editor.UpdateStatus("") // Clear status
	m.publish(editor)
}

func (m *commandMode) Exit(editor Editor, buffer Buffer) {
	editor.UpdateCommand("") // Clear command line on exit
}

// publish shows the prompt, the text and the cursor on the command line.
func (m *commandMode) publish(editor Editor) {
	// +1 for the ':' prompt.
	editor.UpdateCommandLine(":"+string(m.text), m.cursor+1)
}

// HandleKey edits the command line the way Vim's does (:help
// cmdline-editing). Keys that need features goeditor doesn't have yet
// (<C-R> registers, <C-V> literals, <C-K> digraphs, history, completion) are
// ignored, and <C-C> is left to the host.
func (m *commandMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	if key.IsCtrl('j') || key.IsCtrl('m') {
		key = KeyEvent{Key: KeyEnter}
	}

	switch key.Key {
	case KeyEscape:
		editor.SetNormalMode()
		return nil

	case KeyEnter:
		cmd := string(m.text)
		// Exit command mode *before* executing (usually)
		editor.SetNormalMode()
		err := editor.ExecuteCommand(cmd)
		if err != nil {
			editor.DispatchError(err.id, err.err)
		}
		return nil // Error handled by ExecuteCommand/SetMessage

	case KeyBackspace:
		if len(m.text) == 0 {
			// <BS> on an empty command line abandons it.
			editor.SetNormalMode()
			return nil
		}
		if m.cursor > 0 {
			m.delete(m.cursor-1, m.cursor)
		}

	case KeyDelete:
		switch {
		case len(m.text) == 0:
			editor.SetNormalMode()
			return nil
		case m.cursor == len(m.text):
			// At the end of the line <Del> deletes the character before.
			m.delete(m.cursor-1, m.cursor)
		default:
			m.delete(m.cursor, m.cursor+1)
		}

	case KeyLeft:
		if key.Modifiers&(ModShift|ModCtrl) != 0 {
			m.cursor = m.wordLeft()
		} else if m.cursor > 0 {
			m.cursor--
		}

	case KeyRight:
		if key.Modifiers&(ModShift|ModCtrl) != 0 {
			m.cursor = m.wordRight()
		} else if m.cursor < len(m.text) {
			m.cursor++
		}

	case KeyHome:
		m.cursor = 0

	case KeyEnd:
		m.cursor = len(m.text)

	default:
		switch {
		case key.IsCtrl('b'):
			m.cursor = 0
		case key.IsCtrl('e'):
			m.cursor = len(m.text)
		case key.IsCtrl('w'):
			m.delete(m.wordStartBefore(), m.cursor)
		case key.IsCtrl('u'):
			m.delete(0, m.cursor)
		case key.Modifiers&ModCtrl != 0:
			// Ctrl+<letter> carries the letter in Rune after normalisation;
			// it must not be typed into the command line.
			return nil
		case key.Rune != 0:
			m.text = append(m.text[:m.cursor], append([]rune{key.Rune}, m.text[m.cursor:]...)...)
			m.cursor++
		default:
			// Ignore unknown special keys
			return nil
		}
	}

	m.publish(editor)
	return nil
}

// delete removes text[from:to] and leaves the cursor at from.
func (m *commandMode) delete(from, to int) {
	m.text = append(m.text[:from], m.text[to:]...)
	m.cursor = from
}

// wordLeft is <S-Left>: back at least one character, then on until the
// character before the cursor is a space. Vim checks for ' ' only.
func (m *commandMode) wordLeft() int {
	pos := m.cursor
	if pos == 0 {
		return 0
	}
	pos--
	for pos > 0 && m.text[pos-1] != ' ' {
		pos--
	}
	return pos
}

// wordRight is <S-Right>: forward at least one character, then on until the
// cursor is on a space.
func (m *commandMode) wordRight() int {
	pos := m.cursor
	if pos == len(m.text) {
		return pos
	}
	pos++
	for pos < len(m.text) && m.text[pos] != ' ' {
		pos++
	}
	return pos
}

// wordStartBefore is where <C-W> deletes back to: over white space, then
// over characters that are all keyword characters or all not.
func (m *commandMode) wordStartBefore() int {
	pos := m.cursor
	for pos > 0 && isBlank(m.text[pos-1]) {
		pos--
	}
	if pos == 0 {
		return 0
	}
	word := isKeywordRune(m.text[pos-1])
	for pos > 0 && !isBlank(m.text[pos-1]) && isKeywordRune(m.text[pos-1]) == word {
		pos--
	}
	return pos
}

// isKeywordRune is Vim's vim_iswordc() with the default 'iskeyword'.
func isKeywordRune(r rune) bool {
	return utfClass(r) >= wordCharClass
}
