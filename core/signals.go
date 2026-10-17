package core

type Signal any

type YankSignal struct {
	content string
}

type PasteSignal struct {
	content string
}

func (p PasteSignal) Value() string {
	return p.content
}

type CommandSignal struct{}

func (y YankSignal) Value() string {
	return y.content
}

type DeleteSignal struct {
	content string
}

func (d DeleteSignal) Value() string {
	return d.content
}

type RelativeNumbersSignal struct {
	enabled bool
}

func (r RelativeNumbersSignal) Value() bool {
	return r.enabled
}

type UndoSignal struct {
	edits []TextEdit
}

// Value returns the edits applied to the buffer, in order (see TextEdit).
func (u UndoSignal) Value() []TextEdit {
	return u.edits
}

type RedoSignal struct {
	edits []TextEdit
}

// Value returns the edits applied to the buffer, in order (see TextEdit).
func (r RedoSignal) Value() []TextEdit {
	return r.edits
}

// UndoLineSignal is dispatched when 'U' restores the last-changed line.
type UndoLineSignal struct {
	edits []TextEdit
}

// Value returns the edits applied to the buffer, in order (see TextEdit).
func (u UndoLineSignal) Value() []TextEdit {
	return u.edits
}

type RenameSignal struct {
	fileName string
}

func (r RenameSignal) Value() string {
	return r.fileName
}

type DeleteFileSignal struct{}

func (d DeleteFileSignal) Value() {}

type SaveSignal struct {
	path    *string
	content string
}

func (s SaveSignal) Value() (path *string, content string) {
	path = s.path
	content = s.content

	return path, content
}

type QuitSignal struct{}

type ErrorSignal EditorError

func (e ErrorSignal) Value() (id ErrorId, err error) {
	id = e.id
	err = e.err

	return id, err
}

type EnterCommandModeSignal struct{}

type EnterSearchModeSignal struct {
	Backwards bool // true when entering with '?', false for '/'
}

type ExitSearchModeSignal struct{}

type SearchResultsSignal struct {
	positions []Position
}

func (s SearchResultsSignal) Value() []Position {
	return s.positions
}

type CompletionRequestSignal struct {
	context CompletionContext
}

func (c CompletionRequestSignal) Context() CompletionContext {
	return c.context
}

type CompletionResponseSignal struct {
	completions []Completion
	context     CompletionContext
}

func NewCompletionResponseSignal(completions []Completion, context CompletionContext) CompletionResponseSignal {
	return CompletionResponseSignal{
		completions: completions,
		context:     context,
	}
}

func (c CompletionResponseSignal) Value() ([]Completion, CompletionContext) {
	return c.completions, c.context
}

func (e *editor) DispatchSignal(signal Signal) {
	select {
	case e.updateSignal <- signal:
	default: // Ignore if the channel is full
	}
}
