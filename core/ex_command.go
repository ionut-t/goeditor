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
		e.state.Quit = true
		e.Quit()
		return nil

	case "q!", "quit!":
		e.state.Quit = true
		e.Quit()
		return nil

	case "w", "write":
		// Like Vim, write even when the buffer is unmodified.
		e.save(rest)
		return nil

	case "wq":
		e.save(rest)
		return e.ExecuteCommand("q")

	case "wq!":
		e.save(rest)
		return e.ExecuteCommand("q!")

	case "x", "xit":
		// Write only if modified, then quit
		if e.buffer.IsModified() {
			e.save(rest)
		}
		return e.ExecuteCommand("q")

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
			targetRow := max(lineNum-1, 0)
			targetRow = min(targetRow, e.buffer.LineCount()-1)
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
func (e *editor) save(path string) {
	if path == "" {
		e.Save(nil)
		return
	}
	e.Save(&path)
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
