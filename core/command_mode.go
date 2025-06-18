package core

type commandMode struct {
	commandBuffer string
}

func NewCommandMode() EditorMode  { return &commandMode{} }
func (m *commandMode) Name() Mode { return CommandMode }

func (m *commandMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterCommandModeSignal{})
	m.commandBuffer = ""      // Clear buffer on entry
	editor.UpdateStatus("")   // Clear status
	editor.UpdateCommand(":") // Show prompt
}

func (m *commandMode) Exit(editor Editor, buffer Buffer) {
	editor.UpdateCommand("") // Clear command line on exit
}

func (m *commandMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *Error {
	switch key.Key {
	case KeyEscape:
		editor.SetVisualMode()
		return nil

	case KeyBackspace:
		if len(m.commandBuffer) > 0 {
			// Handle UTF-8 correctly (remove last rune, not byte)
			runes := []rune(m.commandBuffer)
			runes = runes[:len(runes)-1]
			m.commandBuffer = string(runes)
			editor.UpdateCommand(":" + m.commandBuffer) // Update display
		} else {
			// Backspace on empty command line goes back to normal mode
			editor.SetVisualMode()
		}
		return nil

	case KeyEnter:
		cmd := m.commandBuffer
		// Exit command mode *before* executing (usually)
		editor.SetNormalMode()
		// Execute the command
		err := editor.ExecuteCommand(cmd)
		if err != nil {
			editor.DispatchError(ErrInvalidCommandId, err)
		}
		return nil // Error handled by ExecuteCommand/SetMessage

	// Add history navigation (Up/Down arrows) here later

	default:
		if key.Rune != 0 {
			// Append character to command buffer
			m.commandBuffer += string(key.Rune)
			editor.UpdateCommand(":" + m.commandBuffer) // Update display
			return nil
		}
		// Ignore unknown special keys
		return nil
	}
}
