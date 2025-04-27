package core

import (
	"errors"
	"log"
)

var (
	ErrEndOfBuffer        = errors.New("end of buffer")
	ErrStartOfBuffer      = errors.New("start of buffer")
	ErrEndOfLine          = errors.New("end of line")
	ErrStartOfLine        = errors.New("start of line")
	ErrInvalidPosition    = errors.New("invalid position")
	ErrInvalidMode        = errors.New("invalid mode")
	ErrInvalidCommand     = errors.New("invalid command")
	ErrNoPendingOperation = errors.New("no pending operation")
	ErrInvalidMotion      = errors.New("invalid motion")
	ErrDeleteRunes        = errors.New("cannot delete runes")
	ErrNoChangesToSave    = errors.New("no changes to save")
)

type ErrorId int

const (
	ErrEndOfBufferId ErrorId = iota
	ErrStartOfBufferId
	ErrEndOfLineId
	ErrStartOfLineId
	ErrInvalidPositionId
	ErrInvalidModeId
	ErrInvalidCommandId
	ErrNoPendingOperationId
	ErrInvalidMotionId
	ErrDeleteRunesId
	ErrNoChangesToSaveId
	ErrFailedToSaveId
	ErrFailedToYankId
	ErrFailedToPasteId
	ErrUndoFailedId
	ErrRedoFailedId
	ErrCopyFailedId
)

type Error struct {
	id  ErrorId
	err error
}

func (e *editor) DispatchError(id ErrorId, err error) {
	select {
	case e.updateSignal <- ErrorSignal{id, err}:
	default:
		log.Println("Channel is full, unable to send error signal")
	}
}
