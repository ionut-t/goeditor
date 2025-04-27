package core

type Signal any

type YankSignal struct {
	totalLines   int
	isVisualLine bool
}

func (y YankSignal) Value() (totalLines int, isVisualLine bool) {
	totalLines = y.totalLines
	isVisualLine = y.isVisualLine

	return totalLines, isVisualLine
}

type PasteSignal struct {
	totalLines int
}

func (p PasteSignal) Value() int {
	return p.totalLines
}

type DeleteSignal struct {
	totalLines int
}

func (d DeleteSignal) Value() int {
	return d.totalLines
}

type UndoSignal struct{}

func (u UndoSignal) Value() {}

type RedoSignal struct{}

func (r RedoSignal) Value() {}

type MessageSignal struct {
	id    string
	value string
}

func (m MessageSignal) Value() (id, message string) {
	id = m.id
	message = m.value

	return id, message
}

type SaveSignal struct {
	content string
}

func (s SaveSignal) Value() string {
	return s.content
}

type QuitSignal struct{}

type ErrorSignal Error

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
