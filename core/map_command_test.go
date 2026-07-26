package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCommand executes a :-command as if typed in command mode.
func runCommand(t *testing.T, e Editor, cmd string) *EditorError {
	t.Helper()
	return e.ExecuteCommand(cmd)
}

func TestMapCommandSet(t *testing.T) {
	t.Run("nmap", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "nmap U <C-r>"))

		keys(e, 'd', 'd')
		keys(e, 'u')
		keys(e, 'U')
		assert.Equal(t, "", content(e), "U should redo")
	})

	t.Run("imap", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "imap jk <Esc>"))

		keys(e, 'i')
		keys(e, 'j', 'k')
		assert.True(t, e.IsNormalMode())
		assert.Equal(t, "hello", content(e))
	})

	t.Run("vmap", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		require.Nil(t, runCommand(t, e, "vmap Q y"))

		keys(e, 'v', 'l', 'Q')
		assert.Equal(t, "he", cb.content)
	})

	t.Run("unprefixed map covers normal but not insert", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "map Q x"))

		keys(e, 'Q')
		assert.Equal(t, "ello", content(e))

		keys(e, 'i')
		keys(e, 'Q')
		assert.Equal(t, "Qello", content(e))
	})

	t.Run("nnoremap does not re-expand the RHS", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "nnoremap b dw"))
		require.Nil(t, runCommand(t, e, "nnoremap Q b"))

		keys(e, 'Q')
		assert.Equal(t, "hello world", content(e))
	})

	t.Run("nmap does re-expand the RHS", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "nnoremap b dw"))
		require.Nil(t, runCommand(t, e, "nmap Q b"))

		keys(e, 'Q')
		assert.Equal(t, "world", content(e))
	})

	t.Run("omap applies while an operator is pending", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "onoremap Q w"))

		keys(e, 'd', 'Q')
		assert.Equal(t, "world", content(e))
	})
}

// TestMapCommandArgumentParsing verifies the raw-argument handling that :map
// needs — the generic strings.Fields path would mangle these.
func TestMapCommandArgumentParsing(t *testing.T) {
	t.Run("extra whitespace between arguments is tolerated", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "nnoremap    Q     dw"))

		keys(e, 'Q')
		assert.Equal(t, "world", content(e))
	})

	t.Run("a space can be mapped via <Space>", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "nnoremap <Space> dw"))

		e.HandleKey(KeyEvent{Key: KeySpace, Rune: ' '})
		assert.Equal(t, "world", content(e))
	})

	t.Run("RHS keeps its internal characters", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		require.Nil(t, runCommand(t, e, "nnoremap Y y$"))

		keys(e, 'Y')
		assert.Equal(t, "hello world", cb.content)
	})
}

func TestMapCommandUnmapAndClear(t *testing.T) {
	t.Run("nunmap restores the default", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "nnoremap x <Nop>"))
		keys(e, 'x')
		assert.Equal(t, "hello", content(e))

		require.Nil(t, runCommand(t, e, "nunmap x"))
		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})

	t.Run("nmapclear removes every normal mapping", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "nnoremap x <Nop>"))
		require.Nil(t, runCommand(t, e, "nnoremap Q dd"))

		require.Nil(t, runCommand(t, e, "nmapclear"))
		assert.Empty(t, e.Mappings(MapNormal))
	})
}

func TestMapCommandErrors(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"missing both arguments", "nmap"},
		{"missing the replacement", "nmap U"},
		{"unmap without a key", "nunmap"},
		{"invalid notation on the left", "nmap <Bogus> x"},
		{"invalid notation on the right", "nmap U <Bogus>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEditor("hello")
			err := runCommand(t, e, tc.cmd)
			require.NotNil(t, err, "expected an error")
			assert.Equal(t, ErrInvalidMappingId, err.ID())
		})
	}
}

// TestMapCommandDoesNotShadowOtherCommands guards the early-return added to
// ExecuteCommand for the :map family.
func TestMapCommandDoesNotShadowOtherCommands(t *testing.T) {
	e := newTestEditor("hello")

	require.Nil(t, runCommand(t, e, "set relativenumber"))
	assert.True(t, e.GetState().RelativeNumbers)

	err := runCommand(t, e, "definitelynotacommand")
	require.NotNil(t, err)
	assert.Equal(t, ErrInvalidCommandId, err.ID())
}
