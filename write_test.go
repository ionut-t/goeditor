package goeditor

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ionut-t/goeditor/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newWriteModel returns a focused Model on "hello" writing through f, with
// x typed (so modified) and no messages pending.
func newWriteModel(f core.WriteFunc) Model {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	m.Focus()
	m.SetContent("hello")
	m.SetWriteFunc(f)
	m = typeText("x")(m)
	drainEditorMsgs(m)
	return m
}

// drainEditorMsgs discards every message waiting on the editor's channel.
func drainEditorMsgs(m Model) {
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		<-m.GetEditor().GetUpdateSignalChan()
	}
}

// editorMsgs returns the messages waiting on the editor's channel, without
// its internal commandMsg and clearMsg.
func editorMsgs(m Model) []any {
	var msgs []any
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		switch msg := m.listenForEditorUpdate()(); msg {
		case commandMsg{}, clearMsg{}:
		default:
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// writeDone runs the job of a write's tea.Cmd (from writeCmd) and returns its
// writeDoneMsg. The job comes first in the batch, so the indicator's tick
// never runs.
func writeDone(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	batch, ok := cmd().(tea.BatchMsg)
	require.True(t, ok)
	msg := batch[0]()
	require.IsType(t, writeDoneMsg{}, msg)
	return msg
}

// command types a : command and presses Enter.
func command(m Model, cmd string) Model {
	return pressSpecial(tea.KeyEnter)(typeText(":" + cmd)(m))
}

// TestWriteThroughUpdate checks the plumbing: Update starts the queued write
// (the request is taken), and the job's result fed back through Update
// settles it.
func TestWriteThroughUpdate(t *testing.T) {
	var written string
	m := newWriteModel(func(_ *string, content string) (core.WriteJob, error) {
		return func(context.Context) error {
			written = content
			return nil
		}, nil
	})

	m = command(m, "wq")

	assert.True(t, m.Writing())
	assert.True(t, m.HasChanges(), "not saved before the job ran")
	_, ok := m.GetEditor().NextWrite()
	assert.False(t, ok, "Update already started the write")

	// What the write's tea.Cmd returns once the job ran.
	m, _ = m.Update(writeDoneMsg{editor: m.editor, id: 1, err: nil})

	assert.False(t, m.Writing())
	assert.False(t, m.HasChanges())
	msgs := editorMsgs(m)
	require.Len(t, msgs, 2)
	assert.Equal(t, SaveMsg{Path: nil, Content: "ello"}, msgs[0])
	assert.Equal(t, QuitMsg{}, msgs[1])
	assert.Empty(t, written, "the job runs in the Cmd, not in Update")
}

// TestWriteCmd checks the tea.Cmd that runs a write: it runs the job and
// returns its result for Update.
func TestWriteCmd(t *testing.T) {
	errWrite := errors.New("E212: Cannot open file for writing")
	m := newWriteModel(func(*string, string) (core.WriteJob, error) {
		return func(context.Context) error { return errWrite }, nil
	})
	require.Nil(t, m.GetEditor().ExecuteCommand("wq"))

	cmd := m.writeCmd()
	require.NotNil(t, cmd)
	assert.Nil(t, m.writeCmd(), "one write at a time")
	msg := writeDone(t, cmd)
	assert.Equal(t, writeDoneMsg{editor: m.editor, id: 1, err: errWrite}, msg)

	m, _ = m.Update(msg)

	msgs := editorMsgs(m)
	require.Len(t, msgs, 1)
	assert.Equal(t, ErrorMsg{ID: core.ErrFailedToSaveId, Error: errWrite}, msgs[0])
	assert.True(t, m.HasChanges())
	assert.False(t, m.Writing())
}

// TestWriteDoneOtherEditor checks a result goes only to the editor whose
// write it is, for programs with several editors.
func TestWriteDoneOtherEditor(t *testing.T) {
	job := func(*string, string) (core.WriteJob, error) {
		return func(context.Context) error { return nil }, nil
	}
	a, b := newWriteModel(job), newWriteModel(job)
	a, b = command(a, "w"), command(b, "w")

	a, _ = a.Update(writeDoneMsg{editor: b.editor, id: 1})

	assert.True(t, a.Writing(), "b's result doesn't finish a's write")
	a, _ = a.Update(writeDoneMsg{editor: a.editor, id: 1})
	assert.False(t, a.Writing())
}

// TestSlowWriteDoesNotBlock checks typing goes on while a job runs.
func TestSlowWriteDoesNotBlock(t *testing.T) {
	release := make(chan struct{})
	m := newWriteModel(func(*string, string) (core.WriteJob, error) {
		return func(ctx context.Context) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}, nil
	})
	require.Nil(t, m.GetEditor().ExecuteCommand("w"))
	cmd := m.writeCmd()
	done := make(chan tea.Msg, 1)
	go func() { done <- writeDone(t, cmd) }()

	m = typeText("x")(m) // "llo", while the job waits

	assert.Equal(t, "llo", m.GetCurrentContent())
	close(release)
	m, _ = m.Update(<-done)
	assert.False(t, m.Writing())
	assert.True(t, m.HasChanges(), "only the text written counts as saved")
	assert.Equal(t, "ello", m.GetSavedContent())
}

// TestWriteNoWriteFuncModel checks a Model with no write function: :w fails
// with an ErrorMsg and the buffer stays modified.
func TestWriteNoWriteFuncModel(t *testing.T) {
	m := newWriteModel(nil)

	m = command(m, "w")

	var ids []core.ErrorId
	for _, msg := range editorMsgs(m) {
		if e, ok := msg.(ErrorMsg); ok {
			ids = append(ids, e.ID)
		}
	}
	assert.Equal(t, []core.ErrorId{core.ErrNoWriteFuncId}, ids)
	assert.True(t, m.HasChanges())
}

// TestWritingIndicator checks "writing…" shows only once a job has run for
// writingIndicatorDelay, and goes when it finishes.
func TestWritingIndicator(t *testing.T) {
	commandLine := func(m Model) string {
		line, _ := m.getCommandLine(m.GetEditor().GetState())
		return line
	}
	m := newWriteModel(func(*string, string) (core.WriteJob, error) {
		return func(context.Context) error { return nil }, nil
	})
	m = command(m, "w")
	assert.NotContains(t, commandLine(m), "writing…", "not before the delay")

	other := newWriteModel(nil)
	m, _ = m.Update(writingSlowMsg{editor: other.editor, id: 1})
	assert.NotContains(t, commandLine(m), "writing…", "another editor's tick")

	m, _ = m.Update(writingSlowMsg{editor: m.editor, id: 1})
	assert.Contains(t, commandLine(m), "writing…")

	m, _ = m.Update(writeDoneMsg{editor: m.editor, id: 1})
	assert.NotContains(t, commandLine(m), "writing…")

	m, _ = m.Update(writingSlowMsg{editor: m.editor, id: 1})
	assert.NotContains(t, commandLine(m), "writing…", "a late tick")
}

// TestWriteInFunction checks a write done inside the function, with no job:
// it finishes inside the command, with no Cmd and no indicator.
func TestWriteInFunction(t *testing.T) {
	var written string
	m := newWriteModel(func(_ *string, content string) (core.WriteJob, error) {
		written = content
		return nil, nil
	})

	m = command(m, "wq")

	assert.Equal(t, "ello", written)
	assert.False(t, m.Writing())
	assert.False(t, m.HasChanges())
	assert.Equal(t, []any{SaveMsg{Content: "ello"}, QuitMsg{}}, editorMsgs(m))
}
