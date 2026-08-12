package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetMultipleOptions covers ':set' with more than one option. Vim applies
// them left to right, which previously was not possible here at all — the
// handler only ever looked at a single argument.
func TestSetMultipleOptions(t *testing.T) {
	t.Run("applies several options in one command", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "set rnu mapleader=,"))

		assert.True(t, e.GetState().RelativeNumbers)
		assert.Equal(t, ",", e.MapLeader())
	})

	t.Run("applies options left to right", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "set rnu nornu"))
		assert.False(t, e.GetState().RelativeNumbers, "the later option wins")
	})

	t.Run("stops at the first bad option, keeping earlier ones", func(t *testing.T) {
		e := newTestEditor("hello")
		err := runCommand(t, e, "set rnu nosuchoption mapleader=,")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidCommandId, err.ID())

		assert.True(t, e.GetState().RelativeNumbers, "the option before the error is applied")
		assert.Equal(t, DefaultMapLeader, e.MapLeader(), "the option after the error is not")
	})

	t.Run("several valued options together", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "set mapleader=, leader=;"))
		assert.Equal(t, ";", e.MapLeader(), "the later assignment wins")
	})
}

func TestSetSingleOptions(t *testing.T) {
	t.Run("boolean options", func(t *testing.T) {
		e := newTestEditor("hello")

		require.Nil(t, runCommand(t, e, "set rnu"))
		assert.True(t, e.GetState().RelativeNumbers)

		require.Nil(t, runCommand(t, e, "set nornu"))
		assert.False(t, e.GetState().RelativeNumbers)

		require.Nil(t, runCommand(t, e, "set relativenumber"))
		assert.True(t, e.GetState().RelativeNumbers)

		require.Nil(t, runCommand(t, e, "set norelativenumber"))
		assert.False(t, e.GetState().RelativeNumbers)
	})

	t.Run("bare set is rejected", func(t *testing.T) {
		e := newTestEditor("hello")
		err := runCommand(t, e, "set")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidCommandId, err.ID())
	})

	t.Run("unknown boolean option is rejected", func(t *testing.T) {
		e := newTestEditor("hello")
		err := runCommand(t, e, "set nosuchoption")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidCommandId, err.ID())
	})

	t.Run("unknown valued option is rejected", func(t *testing.T) {
		e := newTestEditor("hello")
		err := runCommand(t, e, "set nosuchoption=1")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidCommandId, err.ID())
	})
}

// TestSetMapLeader covers ':set mapleader=X'. Without it the leader was
// reachable only from Go, so a mapping typed at the : prompt could never use
// <leader>.
func TestSetMapLeader(t *testing.T) {
	t.Run("sets the leader used by later mappings", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "set mapleader=,"))
		assert.Equal(t, ",", e.MapLeader())

		require.Nil(t, runCommand(t, e, "nnoremap <leader>d dw"))
		keys(e, ',', 'd')
		assert.Equal(t, "world", content(e))
	})

	t.Run("accepts key notation", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "set mapleader=<Space>"))
		require.Nil(t, runCommand(t, e, "nnoremap <leader>d dw"))

		e.HandleKey(KeyEvent{Key: KeySpace, Rune: ' '})
		e.HandleKey(KeyEvent{Rune: 'd'})
		assert.Equal(t, "world", content(e))
	})

	t.Run("leader is resolved when the mapping is defined, not when used", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.Nil(t, runCommand(t, e, "set mapleader=,"))
		require.Nil(t, runCommand(t, e, "nnoremap <leader>d dw"))

		// Changing it afterwards must not disturb the existing mapping — this is
		// why a sourced script has to set the leader before its mappings.
		require.Nil(t, runCommand(t, e, "set mapleader=;"))

		keys(e, ',', 'd')
		assert.Equal(t, "world", content(e), "the old leader still triggers it")
	})

	t.Run("the alias 'leader' works too", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "set leader=,"))
		assert.Equal(t, ",", e.MapLeader())
	})

	t.Run("rejects an empty value", func(t *testing.T) {
		// ParseKeys accepts an empty sequence, so this has to be caught by the
		// option handler; otherwise the leader would silently reset to default.
		e := newTestEditor("hello")
		err := runCommand(t, e, "set mapleader=")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidMappingId, err.ID())
		assert.Equal(t, DefaultMapLeader, e.MapLeader())
	})

	t.Run("rejects invalid notation at set time", func(t *testing.T) {
		e := newTestEditor("hello")
		err := runCommand(t, e, "set mapleader=<Bogus>")
		require.NotNil(t, err)
		assert.Equal(t, ErrInvalidMappingId, err.ID())
		assert.Equal(t, DefaultMapLeader, e.MapLeader(), "a rejected value must not be stored")
	})
}
