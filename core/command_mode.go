package core

type commandMode struct {
	baseMode
	line cmdline

	histories *histories // the editor's, nil for NewCommandMode
	mapDepth  *int       // the editor's mapping depth: keys from a mapping when > 0
	typed     bool       // a key of this line came from the user, not a mapping
}

func NewCommandMode() EditorMode  { return newCommandMode(nil, nil) }
func (m *commandMode) Name() Mode { return CommandMode }

func newCommandMode(h *histories, mapDepth *int) *commandMode {
	m := &commandMode{histories: h, mapDepth: mapDepth}
	if h != nil {
		m.line.history = &h.lists[CommandHistory]
	}
	return m
}

func (m *commandMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterCommandModeSignal{})
	m.line.reset()
	m.typed = false
	editor.UpdateStatus("") // Clear status
	m.line.publish(editor, ":")
}

func (m *commandMode) Exit(editor Editor, buffer Buffer) {
	editor.UpdateCommand("") // Clear command line on exit
}

// HandleKey edits the command line (see cmdline.handleKey) and runs it on
// <CR>. A line run or abandoned goes into the history, as Vim's, unless every
// key of it came from a mapping.
func (m *commandMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	if m.mapDepth == nil || *m.mapDepth == 0 {
		m.typed = true
	}
	switch m.line.handleKey(key) {
	case cmdlineAbandon:
		m.addHistory()
		editor.SetNormalMode()
	case cmdlineEnter:
		cmd := string(m.line.text)
		m.addHistory()
		// Exit command mode *before* executing (usually)
		editor.SetNormalMode()
		if err := editor.ExecuteCommand(cmd); err != nil {
			editor.DispatchError(err.id, err.err)
		}
	case cmdlineEdited:
		m.line.publish(editor, ":")
	}
	return nil // Errors are dispatched, not returned
}

func (m *commandMode) addHistory() {
	if m.histories != nil && m.typed {
		m.histories.add(CommandHistory, string(m.line.text))
	}
}
