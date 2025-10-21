package core

type searchMode struct {
	searchBuffer string
}

func NewSearchMode() EditorMode  { return &searchMode{} }
func (m *searchMode) Name() Mode { return SearchMode }

func (m *searchMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterSearchModeSignal{})
	m.searchBuffer = "" // Clear buffer on entry

	if editor.GetState().SearchQuery.Pattern != "" {
		editor.UpdateCommand("/" + editor.GetState().SearchQuery.Pattern) // Show previous search term
		m.searchBuffer = editor.GetState().SearchQuery.Pattern
	} else {
		editor.UpdateCommand("/") // Show prompt
	}
}

func (m *searchMode) Exit(editor Editor, buffer Buffer) {}

func (m *searchMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	switch key.Key {
	case KeyEscape:
		m.searchBuffer = ""
		editor.ExecuteSearch("")
		return nil

	case KeyBackspace:
		if len(m.searchBuffer) > 0 {
			// Handle UTF-8 correctly (remove last rune, not byte)
			runes := []rune(m.searchBuffer)
			runes = runes[:len(runes)-1]
			m.searchBuffer = string(runes)
			editor.UpdateCommand("/" + m.searchBuffer) // Update display
		}
		return nil

	case KeyEnter:
		query := m.searchBuffer
		editor.ExecuteSearch(query)
		editor.DispatchSignal(SearchResultsSignal{positions: editor.GetState().SearchResults})
		return nil

	default:
		if key.Rune != 0 {
			// Append character to command buffer
			m.searchBuffer += string(key.Rune)
			editor.UpdateCommand("/" + m.searchBuffer) // Update display
			return nil
		}
		// Ignore unknown special keys
		return nil
	}
}
