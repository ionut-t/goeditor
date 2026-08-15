package core

import (
	"errors"
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
	ErrDeleteRunes        = errors.New("cannot delete runes")
	ErrNoChangesToSave    = errors.New("no changes to save")
	ErrUnsavedChanges     = errors.New("unsaved changes (use :q! to override)")
	ErrRenameFailed       = errors.New("rename requires a single argument (rename new_filename)")
	ErrNoLineToRestore    = errors.New("no line to restore")
	ErrInvalidMapping     = errors.New("invalid mapping")
	ErrMapRecursion       = errors.New("recursive mapping")
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
	ErrCharNotFoundId
	ErrDeleteRunesId
	ErrFailedToSaveId
	ErrNoChangesToSaveId
	ErrUnsavedChangesId
	ErrFailedToYankId
	ErrFailedToPasteId
	ErrUndoFailedId
	ErrRedoFailedId
	ErrUndoLineFailedId
	ErrInvalidMappingId
	ErrMapRecursionId
	ErrCopyFailedId
	ErrRenameFailedId
)

type EditorError struct {
	id  ErrorId
	err error
}

func (e EditorError) ID() ErrorId {
	return e.id
}

func (e EditorError) Error() error {
	return e.err
}

func (e *editor) DispatchError(id ErrorId, err error) {
	select {
	case e.updateSignal <- ErrorSignal{id, err}:
	default: // Ignore if the channel is full
	}
}

func isBoundaryError(err error) bool {
	return errors.Is(err, ErrEndOfBuffer) ||
		errors.Is(err, ErrStartOfBuffer) ||
		errors.Is(err, ErrEndOfLine) ||
		errors.Is(err, ErrStartOfLine)
}
