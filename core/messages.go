package core

import "log"

var (
	EmptyMessage                   = ""
	ChangesSavedMessage            = "changes saved"
	RelativeNumbersEnabledMessage  = "relative line numbers enabled"
	RelativeNumbersDisabledMessage = "relative line numbers disabled"
	LinesDeletedMessage            = "lines deleted"
	YankMessage                    = "selection yanked"
)

func (e *editor) DispatchMessage(args ...string) {
	id := args[0]
	value := id
	if len(args) > 1 {
		value = args[1]
	}
	select {
	case e.updateSignal <- MessageSignal{id, value}:
	default:
		log.Println("Channel is full, unable to send message signal")
	}
}
