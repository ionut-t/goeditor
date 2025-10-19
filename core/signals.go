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
	contentBefore string
}

func (u UndoSignal) Value() string {
	return u.contentBefore
}

type RedoSignal struct {
	contentBefore string
}

func (r RedoSignal) Value() string {
	return r.contentBefore
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

func (e *editor) DispatchSignal(signal Signal) {
	select {
	case e.updateSignal <- signal:
	default: // Ignore if the channel is full
	}
}
