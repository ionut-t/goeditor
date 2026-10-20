package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drainSignals discards all pending signals from the editor's signal channel.
func drainSignals(e Editor) {
	ch := e.GetUpdateSignalChan()
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// nextSignal returns the next signal from the editor's signal channel, or nil if none.
func nextSignal(e Editor) Signal {
	select {
	case s := <-e.GetUpdateSignalChan():
		return s
	default:
		return nil
	}
}

// --- Entering and exiting command mode ---

// TestCommandModeEnterExit tests entering and exiting command mode via ':' and Escape.
func TestCommandModeEnterExit(t *testing.T) {
	t.Run(": enters command mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		assert.True(t, e.IsCommandMode())
	})

	t.Run("command line shows ':' prompt on entry", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		assert.Equal(t, ":", e.GetState().CommandLine)
	})

	t.Run("Escape exits to normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("command line is cleared after Escape", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		escape(e)
		assert.Equal(t, "", e.GetState().CommandLine)
	})
}

// --- Typing in command mode ---

// TestCommandModeTyping tests that characters are appended to the command buffer.
func TestCommandModeTyping(t *testing.T) {
	t.Run("typing appends to command line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':', 'w', 'q')
		assert.Equal(t, ":wq", e.GetState().CommandLine)
	})

	t.Run("Backspace removes last character", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':', 'w', 'q')
		backspace(e)
		assert.Equal(t, ":w", e.GetState().CommandLine)
	})

	t.Run("Backspace on empty command buffer exits to normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		backspace(e)
		assert.True(t, e.IsNormalMode())
	})
}

// --- :q / :q! ---

// TestCommandModeQuit tests ':q' — quit when buffer is unmodified.
func TestCommandModeQuit(t *testing.T) {
	t.Run(":q on unmodified buffer sets Quit flag", func(t *testing.T) {
		e := newTestEditor("hello")
		drainSignals(e)
		keys(e, ':', 'q')
		enter(e)
		assert.True(t, e.GetState().Quit)
	})

	t.Run(":q on unmodified buffer dispatches QuitSignal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':', 'q')
		drainSignals(e) // drain EnterCommandModeSignal and any others before enter
		enter(e)
		sig := nextSignal(e)
		_, ok := sig.(QuitSignal)
		assert.True(t, ok)
	})

	t.Run(":q on modified buffer returns error (does not quit)", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		drainSignals(e)
		keys(e, ':', 'q')
		enter(e)
		assert.False(t, e.GetState().Quit)
	})

	t.Run(":q! force-quits even with unsaved changes", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		drainSignals(e)
		keys(e, ':', 'q', '!')
		enter(e)
		assert.True(t, e.GetState().Quit)
	})
}

// --- :w ---

// TestCommandModeWrite tests ':w' — write (save) the buffer.
func TestCommandModeWrite(t *testing.T) {
	t.Run(":w on modified buffer dispatches SaveSignal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer → "ello"
		keys(e, ':', 'w')
		drainSignals(e) // drain EnterCommandModeSignal and any others before enter
		enter(e)
		sig := nextSignal(e)
		save, ok := sig.(SaveSignal)
		assert.True(t, ok)
		path, savedContent := save.Value()
		assert.Nil(t, path)
		assert.Equal(t, "ello", savedContent)
	})

	t.Run(":w marks buffer as unmodified", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		assert.True(t, e.GetBuffer().IsModified())
		drainSignals(e)
		keys(e, ':', 'w')
		enter(e)
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run(":w on unmodified buffer still saves (like Vim)", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':', 'w')
		drainSignals(e)
		enter(e)
		save, ok := nextSignal(e).(SaveSignal)
		assert.True(t, ok)
		path, savedContent := save.Value()
		assert.Nil(t, path)
		assert.Equal(t, "hello", savedContent)
		assert.False(t, e.GetState().Quit)
	})

	t.Run(":w path saves to that path", func(t *testing.T) {
		e := newTestEditor("hello")
		drainSignals(e)
		assert.Nil(t, e.ExecuteCommand("w other.txt"))
		save, ok := nextSignal(e).(SaveSignal)
		assert.True(t, ok)
		path, _ := save.Value()
		assert.Equal(t, "other.txt", *path)
	})
}

// --- :wq / :wq! ---

// assertSavedThenQuit checks the next two signals are a SaveSignal with the
// given content followed by a QuitSignal.
func assertSavedThenQuit(t *testing.T, e Editor, wantContent string) {
	t.Helper()
	save, ok := nextSignal(e).(SaveSignal)
	require.True(t, ok, "expected SaveSignal")
	_, savedContent := save.Value()
	assert.Equal(t, wantContent, savedContent)
	_, ok = nextSignal(e).(QuitSignal)
	assert.True(t, ok, "expected QuitSignal")
	assert.True(t, e.GetState().Quit)
	assert.False(t, e.GetBuffer().IsModified())
}

// TestCommandModeWriteQuit tests ':wq' and ':wq!' — Vim always writes then
// quits, whether or not the buffer is modified.
func TestCommandModeWriteQuit(t *testing.T) {
	for _, cmd := range []string{"wq", "wq!"} {
		t.Run(":"+cmd+" on modified buffer saves and quits", func(t *testing.T) {
			e := newTestEditor("hello")
			keys(e, 'x') // modify buffer → "ello"
			keys(e, ':')
			keys(e, []rune(cmd)...)
			drainSignals(e)
			enter(e)
			assertSavedThenQuit(t, e, "ello")
		})

		t.Run(":"+cmd+" on unmodified buffer saves and quits", func(t *testing.T) {
			e := newTestEditor("hello")
			keys(e, ':')
			keys(e, []rune(cmd)...)
			drainSignals(e)
			enter(e)
			assertSavedThenQuit(t, e, "hello")
		})
	}

	t.Run(":wq path saves to that path and quits", func(t *testing.T) {
		e := newTestEditor("hello")
		drainSignals(e)
		assert.Nil(t, e.ExecuteCommand("wq other.txt"))
		save, ok := nextSignal(e).(SaveSignal)
		assert.True(t, ok)
		path, _ := save.Value()
		assert.Equal(t, "other.txt", *path)
		assert.True(t, e.GetState().Quit)
	})

	t.Run("write commands take the rest of the line as one path (like Vim)", func(t *testing.T) {
		for _, cmd := range []string{"w", "wq", "wq!", "x"} {
			e := newTestEditor("hello")
			keys(e, 'x') // modify buffer so :x writes
			drainSignals(e)
			assert.Nil(t, e.ExecuteCommand(cmd+"  my file.txt  "), cmd)
			save, ok := nextSignal(e).(SaveSignal)
			if assert.True(t, ok, cmd) {
				path, _ := save.Value()
				assert.Equal(t, "my file.txt", *path, cmd)
			}
		}
	})
}

// --- :x / :xit ---

// TestCommandModeXit tests ':x' — write only if modified, then quit.
func TestCommandModeXit(t *testing.T) {
	t.Run(":x on modified buffer saves and quits", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		drainSignals(e)
		keys(e, ':', 'x')
		enter(e)
		assert.True(t, e.GetState().Quit)
		assert.False(t, e.GetBuffer().IsModified())
	})

	t.Run(":x on unmodified buffer quits without save signal", func(t *testing.T) {
		e := newTestEditor("hello")
		drainSignals(e)
		keys(e, ':', 'x')
		enter(e)
		assert.True(t, e.GetState().Quit)
		// No save signal should be dispatched (buffer was clean)
		sig := nextSignal(e)
		_, isSave := sig.(SaveSignal)
		assert.False(t, isSave)
	})

	t.Run(":xit is an alias for :x", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		drainSignals(e)
		keys(e, ':', 'x', 'i', 't')
		enter(e)
		assert.True(t, e.GetState().Quit)
	})
}

// --- Enter with empty command ---

// TestCommandModeEmptyEnter tests that pressing Enter on an empty command is a no-op.
func TestCommandModeEmptyEnter(t *testing.T) {
	t.Run("Enter on empty command returns to normal mode without error", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		enter(e)
		assert.True(t, e.IsNormalMode())
		assert.Equal(t, "hello", content(e))
	})
}

// TestCommandModeGotoLine tests :{number}, which jumps to that line.
func TestCommandModeGotoLine(t *testing.T) {
	const text = "one\ntwo\nthree\nfour\nfive"

	t.Run("jumps to the given line", func(t *testing.T) {
		e := newTestEditor(text)
		assert.Nil(t, e.ExecuteCommand("3"))
		assert.Equal(t, Position{2, 0}, cursorPos(e))
	})

	t.Run(":0 jumps to the first line", func(t *testing.T) {
		e := newTestEditor(text)
		keys(e, 'G')
		assert.Nil(t, e.ExecuteCommand("0"))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("line past the end jumps to the last line", func(t *testing.T) {
		e := newTestEditor(text)
		assert.Nil(t, e.ExecuteCommand("99"))
		assert.Equal(t, Position{4, 0}, cursorPos(e))
	})

	t.Run("number too large for int jumps to the last line", func(t *testing.T) {
		e := newTestEditor(text)
		assert.Nil(t, e.ExecuteCommand("99999999999999999999999"))
		assert.Equal(t, Position{4, 0}, cursorPos(e))
	})

	for _, cmd := range []string{"10abc", "3 abc", "3x", "+2", "-1", "1e2"} {
		t.Run("rejects "+cmd, func(t *testing.T) {
			e := newTestEditor(text)
			keys(e, 'j')
			err := e.ExecuteCommand(cmd)
			require.NotNil(t, err)
			assert.Equal(t, ErrInvalidCommandId, err.ID())
			assert.Equal(t, Position{1, 0}, cursorPos(e))
		})
	}
}
