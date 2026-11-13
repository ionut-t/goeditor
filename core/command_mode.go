package core

type commandMode struct {
	baseMode
	line cmdline
}

func NewCommandMode() EditorMode  { return &commandMode{} }
func (m *commandMode) Name() Mode { return CommandMode }

func (m *commandMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterCommandModeSignal{})
	m.line.reset()
	editor.UpdateStatus("") // Clear status
	m.line.publish(editor, ":")
}

func (m *commandMode) Exit(editor Editor, buffer Buffer) {
	editor.UpdateCommand("") // Clear command line on exit
}

// HandleKey edits the command line (see cmdline.handleKey) and runs it on
// <CR>.
func (m *commandMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	switch m.line.handleKey(key) {
	case cmdlineAbandon:
		editor.SetNormalMode()
	case cmdlineEnter:
		cmd := string(m.line.text)
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
