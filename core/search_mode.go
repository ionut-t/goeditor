package core

type searchMode struct{}

func NewSearchMode() EditorMode  { return &searchMode{} }
func (m *searchMode) Name() Mode { return SearchMode }

func (m *searchMode) Enter(editor Editor, buffer Buffer) {
	editor.DispatchSignal(EnterSearchModeSignal{})
	editor.UpdateCommand("")
}

func (m *searchMode) Exit(editor Editor, buffer Buffer) {
	editor.DispatchSignal(ExitSearchModeSignal{})
}

func (m *searchMode) HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError {
	return nil
}
