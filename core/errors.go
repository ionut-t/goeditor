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
	ErrUndoCorrupt        = errors.New("undo list corrupt")
	ErrTextTooLong        = errors.New("resulting text too long")
	// ErrNoPreviousPattern is Vim's E35: an empty search, n or N with no
	// last search to reuse.
	ErrNoPreviousPattern = errors.New("no previous regular expression")
	// ErrNoStringUnderCursor is Vim's E348: * or # with only blanks from the
	// cursor to the end of the line.
	ErrNoStringUnderCursor = errors.New("no string under cursor")
	// ErrPatternNotFound is Vim's E486: a search, n or N found no match
	// anywhere in the buffer. ErrSearchHitBottom (E385) and ErrSearchHitTop
	// (E384) are the same with Wrap off, going forward and backward. The
	// error dispatched wraps them with the pattern ("pattern not found: bar").
	ErrPatternNotFound = errors.New("pattern not found")
	ErrSearchHitBottom = errors.New("search hit BOTTOM without match")
	ErrSearchHitTop    = errors.New("search hit TOP without match")
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
	ErrTextTooLongId
	ErrNoPreviousPatternId
	ErrNoStringUnderCursorId
	ErrPatternNotFoundId
	ErrSearchHitBottomId
	ErrSearchHitTopId
)

// EditorError is an error with an ErrorId saying which operation failed. It
// wraps the underlying error, so errors.Is and errors.As see through it.
type EditorError struct {
	id  ErrorId
	err error
}

func (e *EditorError) ID() ErrorId {
	return e.id
}

func (e *EditorError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Err returns the underlying error.
func (e *EditorError) Err() error {
	return e.err
}

func (e *EditorError) Unwrap() error {
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
