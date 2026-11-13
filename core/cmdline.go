package core

// cmdline is the text and cursor of a command line being edited: the ':'
// command line and the '/' and '?' search lines share it, as Vim's do
// (:help cmdline-editing).
type cmdline struct {
	text   []rune
	cursor int // index into text, 0..len(text)
}

// cmdlineResult is what a key did to the command line.
type cmdlineResult int

const (
	cmdlineEdited  cmdlineResult = iota // the text or the cursor may have changed
	cmdlineIgnored                      // the key does nothing on the command line
	cmdlineEnter                        // <CR>: run the line
	cmdlineAbandon                      // <Esc>, or <BS>/<Del> on an empty line
)

func (c *cmdline) reset() {
	c.text = nil
	c.cursor = 0
}

// set replaces the text, with the cursor at its end.
func (c *cmdline) set(text string) {
	c.text = []rune(text)
	c.cursor = len(c.text)
}

// publish shows prompt, the text and the cursor on the command line.
func (c *cmdline) publish(editor Editor, prompt string) {
	editor.UpdateCommandLine(prompt+string(c.text), c.cursor+len([]rune(prompt)))
}

// handleKey applies an editing key. Keys that need features goeditor doesn't
// have yet (<C-R> registers, <C-V> literals, <C-K> digraphs, history,
// completion) are ignored, and <C-C> is left to the host.
func (c *cmdline) handleKey(key KeyEvent) cmdlineResult {
	if key.IsCtrl('j') || key.IsCtrl('m') {
		key = KeyEvent{Key: KeyEnter}
	}

	switch key.Key {
	case KeyEscape:
		return cmdlineAbandon

	case KeyEnter:
		return cmdlineEnter

	case KeyBackspace:
		if len(c.text) == 0 {
			// <BS> on an empty command line abandons it.
			return cmdlineAbandon
		}
		if c.cursor > 0 {
			c.delete(c.cursor-1, c.cursor)
		}

	case KeyDelete:
		switch {
		case len(c.text) == 0:
			return cmdlineAbandon
		case c.cursor == len(c.text):
			// At the end of the line <Del> deletes the character before.
			c.delete(c.cursor-1, c.cursor)
		default:
			c.delete(c.cursor, c.cursor+1)
		}

	case KeyLeft:
		if key.Modifiers&(ModShift|ModCtrl) != 0 {
			c.cursor = c.wordLeft()
		} else if c.cursor > 0 {
			c.cursor--
		}

	case KeyRight:
		if key.Modifiers&(ModShift|ModCtrl) != 0 {
			c.cursor = c.wordRight()
		} else if c.cursor < len(c.text) {
			c.cursor++
		}

	case KeyHome:
		c.cursor = 0

	case KeyEnd:
		c.cursor = len(c.text)

	default:
		switch {
		case key.IsCtrl('b'):
			c.cursor = 0
		case key.IsCtrl('e'):
			c.cursor = len(c.text)
		case key.IsCtrl('w'):
			c.delete(c.wordStartBefore(), c.cursor)
		case key.IsCtrl('u'):
			c.delete(0, c.cursor)
		case key.Modifiers&ModCtrl != 0:
			// Ctrl+<letter> carries the letter in Rune after normalisation;
			// it must not be typed into the command line.
			return cmdlineIgnored
		case key.Rune != 0:
			c.text = append(c.text[:c.cursor], append([]rune{key.Rune}, c.text[c.cursor:]...)...)
			c.cursor++
		default:
			// Ignore unknown special keys
			return cmdlineIgnored
		}
	}

	return cmdlineEdited
}

// delete removes text[from:to] and leaves the cursor at from.
func (c *cmdline) delete(from, to int) {
	c.text = append(c.text[:from], c.text[to:]...)
	c.cursor = from
}

// wordLeft is <S-Left>: back at least one character, then on until the
// character before the cursor is a space. Vim checks for ' ' only.
func (c *cmdline) wordLeft() int {
	pos := c.cursor
	if pos == 0 {
		return 0
	}
	pos--
	for pos > 0 && c.text[pos-1] != ' ' {
		pos--
	}
	return pos
}

// wordRight is <S-Right>: forward at least one character, then on until the
// cursor is on a space.
func (c *cmdline) wordRight() int {
	pos := c.cursor
	if pos == len(c.text) {
		return pos
	}
	pos++
	for pos < len(c.text) && c.text[pos] != ' ' {
		pos++
	}
	return pos
}

// wordStartBefore is where <C-W> deletes back to: over white space, then
// over characters that are all keyword characters or all not.
func (c *cmdline) wordStartBefore() int {
	pos := c.cursor
	for pos > 0 && isBlank(c.text[pos-1]) {
		pos--
	}
	if pos == 0 {
		return 0
	}
	word := isKeywordRune(c.text[pos-1])
	for pos > 0 && !isBlank(c.text[pos-1]) && isKeywordRune(c.text[pos-1]) == word {
		pos--
	}
	return pos
}

// isKeywordRune is Vim's vim_iswordc() with the default 'iskeyword'.
func isKeywordRune(r rune) bool {
	return utfClass(r) >= wordCharClass
}
