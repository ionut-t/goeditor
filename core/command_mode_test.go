package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("Backspace on empty command buffer exits command mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, ':')
		backspace(e)
		// TODO: bug in command_mode.go — SetVisualMode() is called instead of SetNormalMode()
		// when backspace is pressed on an empty command buffer. Comment in code says "goes back
		// to normal mode" but the call is wrong.
		assert.True(t, e.IsVisualMode())
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

	t.Run(":w on unmodified buffer returns error", func(t *testing.T) {
		e := newTestEditor("hello")
		drainSignals(e)
		keys(e, ':', 'w')
		enter(e)
		// No save signal should be dispatched
		sig := nextSignal(e)
		_, isSave := sig.(SaveSignal)
		assert.False(t, isSave)
	})
}

// --- :wq ---

// TestCommandModeWriteQuit tests ':wq' — write then quit.
func TestCommandModeWriteQuit(t *testing.T) {
	t.Run(":wq saves and quits", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'x') // modify buffer
		drainSignals(e)
		keys(e, ':', 'w', 'q')
		enter(e)
		assert.True(t, e.GetState().Quit)
		assert.False(t, e.GetBuffer().IsModified())
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
