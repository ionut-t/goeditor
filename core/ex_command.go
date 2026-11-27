package core

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// ExecuteCommand executes a command string (typically entered in command mode)
func (e *editor) ExecuteCommand(cmd string) *EditorError {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	// The :map family must be handled before the Fields split below, which
	// collapses whitespace — mappings need their arguments kept verbatim. Vim
	// parses these grammars separately too: :map takes literal trailing text,
	// :set takes whitespace-separated options.
	name, rest := splitCommandWord(cmd)
	if handled, err := e.executeMapCommand(name, rest); handled {
		return err
	}

	parts := strings.Fields(cmd)
	command := parts[0]
	args := parts[1:]

	// TODO: Add range parsing (e.g., :%s/foo/bar/g)

	switch command {
	case "q", "quit":
		if e.buffer.IsModified() {
			return &EditorError{
				id:  ErrUnsavedChangesId,
				err: ErrUnsavedChanges,
			}
		}
		// Waits for pending writes, so quitting never cuts one off.
		return e.requestQuit()

	case "q!", "quit!":
		e.abandonWrites()
		e.Quit()
		return nil

	case "w", "write":
		// Like Vim, write even when the buffer is unmodified.
		return e.save(rest)

	// Like Vim, :wq and :x quit only once the write worked: a failed write
	// leaves the editor open with the buffer still modified. The ! of :wq!
	// forces the write in Vim (a read-only file), not the quit.
	case "wq", "wq!":
		if err := e.save(rest); err != nil {
			return err
		}
		return e.requestQuit()

	case "x", "xit":
		// Write only if modified (and not already being written), then quit.
		if e.buffer.IsModified() && !e.writeQueued(pathArg(rest)) {
			if err := e.save(rest); err != nil {
				return err
			}
		}
		return e.requestQuit()

		// Add more commands: e, edit, r, read, s, substitute etc.
		// case "s": return e.executeSubstitute(args)

	case "set":
		return e.executeSet(args)

	case "rename":
		if len(args) != 1 {
			return &EditorError{
				id:  ErrRenameFailedId,
				err: ErrRenameFailed,
			}
		}

		e.DispatchSignal(RenameSignal{
			fileName: args[0],
		})

		return nil

	case "delete", "del":
		e.DispatchSignal(DeleteFileSignal{})
		return nil

	default:
		// Handle line number navigation (e.g., ":10")
		if lineNum, ok := parseLineNumber(command); ok && len(args) == 0 {
			// User enters 1-based, we use 0-based. Like Vim, :0 goes to the
			// first line and a number past the end goes to the last line.
			maxRow := max(0, e.buffer.LineCount()-1)
			targetRow := min(max(0, lineNum-1), maxRow)
			cursor := e.buffer.GetCursor()

			cursor.Position.Row = targetRow
			cursor.Position.Col = 0 // Move to start of that line
			// Try moving to first non-blank instead?
			// cursor.MoveToFirstNonBlank(e.buffer)
			e.buffer.SetCursor(cursor)
			e.ScrollViewport() // Ensure jumped-to line is visible
			return nil
		}
		return &EditorError{
			id:  ErrInvalidCommandId,
			err: ErrInvalidCommand,
		}
	}
}

// save writes the buffer for :w, :wq and :x. Like Vim, the rest of the
// command line is a single path, so it may contain spaces. An empty path
// writes the current file.
func (e *editor) save(path string) *EditorError {
	return e.write(pathArg(path))
}

// pathArg returns a write command's path argument, nil when empty.
func pathArg(path string) *string {
	if path == "" {
		return nil
	}
	return &path
}

// parseLineNumber parses a :{number} line address. Only plain digits are
// accepted: signs are relative addresses in Vim, which aren't supported yet.
// A number too large for an int returns math.MaxInt, so it clamps to the
// last line.
func parseLineNumber(s string) (int, bool) {
	if s == "" || strings.TrimLeft(s, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if errors.Is(err, strconv.ErrRange) {
		return math.MaxInt, true
	}
	return n, err == nil
}
