package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// typeKeys sends a notation string through the editor as if typed.
func typeKeys(t *testing.T, e Editor, notation string) {
	t.Helper()
	events, err := ParseKeys(notation, e.MapLeader())
	require.NoError(t, err)
	for _, k := range events {
		e.HandleKey(k)
	}
}

// TestMapSingleKey covers the motivating case: rebinding a key to a command
// that already has its own default binding.
func TestMapSingleKey(t *testing.T) {
	t.Run("U can be mapped back to redo", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "U", "<C-r>", true))

		keys(e, 'd', 'd')
		keys(e, 'u')
		assert.Equal(t, "hello", content(e))

		keys(e, 'U') // now redo, not undo-line
		assert.Equal(t, "", content(e))
	})

	t.Run("unmapped keys keep their default behaviour", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "U", "<C-r>", true))

		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})
}

// TestMapToSequence is the property an action-enum keymap could not express:
// the right-hand side is keys, re-fed through the parser, so it can be a whole
// operator+motion command.
func TestMapToSequence(t *testing.T) {
	t.Run("Y maps to y$", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		require.NoError(t, e.Map(MapNormal, "Y", "y$", true))

		keys(e, 'Y')
		assert.Equal(t, "hello world", cb.content)
	})

	t.Run("a single key maps to a delete operator plus motion", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))

		keys(e, 'Q')
		assert.Equal(t, "world", content(e))
	})

	t.Run("mapping to a text object", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "diw", true))

		keys(e, 'Q')
		assert.Equal(t, " world", content(e))
	})

	t.Run("RHS may change mode partway through", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "Q", "ix", true))

		keys(e, 'Q')
		assertInsertMode(t, e)
		assert.Equal(t, "xhello", content(e))
	})
}

// TestMapMultiKeyLHS covers left-hand sides longer than one key, including the
// prefix handling that decides when to commit.
func TestMapMultiKeyLHS(t *testing.T) {
	t.Run("jk leaves insert mode", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapInsert, "jk", "<Esc>", true))

		keys(e, 'i')
		assertInsertMode(t, e)

		keys(e, 'j', 'k')
		assert.True(t, e.IsNormalMode(), "jk should have left insert mode")
		assert.Equal(t, "hello", content(e), "the mapped keys must not be typed")
	})

	t.Run("a partial prefix is delivered when it cannot complete", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapInsert, "jk", "<Esc>", true))

		keys(e, 'i')
		keys(e, 'j', 'a') // 'j' held, then 'a' rules out the mapping
		assert.Equal(t, "jahello", content(e))
		assertInsertMode(t, e)
	})

	t.Run("a held prefix waits rather than committing early", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapInsert, "jk", "<Esc>", true))

		keys(e, 'i')
		keys(e, 'j')
		assert.Equal(t, "hello", content(e), "'j' is held pending disambiguation")

		// FlushPendingMapping is what a 'timeoutlen' timer would call.
		e.FlushPendingMapping()
		assert.Equal(t, "jhello", content(e))
	})

	t.Run("flushed keys can start a new mapping", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "zz", "x", true))
		require.NoError(t, e.Map(MapNormal, "z", "dw", true))

		// 'z' matches exactly but 'zz' could still match, so it waits; the second
		// 'z' completes the longer mapping.
		keys(e, 'z', 'z')
		assert.Equal(t, "ello world", content(e))
	})
}

// TestMapModes verifies mappings are scoped per mode, which matters because the
// same key means different things in normal and visual mode.
func TestMapModes(t *testing.T) {
	t.Run("a normal-mode mapping does not apply in insert mode", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "q", "x", true))

		keys(e, 'i')
		keys(e, 'q')
		assert.Equal(t, "qhello", content(e), "q should be typed literally in insert mode")
	})

	t.Run("a visual mapping does not apply in normal mode", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		require.NoError(t, e.Map(MapVisual, "Q", "y", true))

		keys(e, 'Q') // no such normal-mode command
		assert.Equal(t, "", cb.content)

		keys(e, 'v', 'l', 'Q')
		assert.Equal(t, "he", cb.content)
	})

	t.Run("a visual mapping applies in visual line mode too", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello\nworld")
		require.NoError(t, e.Map(MapVisual, "Q", "y", true))

		keys(e, 'V', 'Q')
		assert.Equal(t, "hello\n", cb.content)
	})

	t.Run("MapAll covers normal and visual but not insert", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapAll, "q", "x", true))

		keys(e, 'q')
		assert.Equal(t, "ello", content(e))

		keys(e, 'i')
		keys(e, 'q')
		assert.Equal(t, "qello", content(e), "insert mode is excluded from MapAll")
	})
}

// TestMapOperatorPending covers mappings that apply only while an operator is
// waiting for a motion.
func TestMapOperatorPending(t *testing.T) {
	e := newTestEditor("hello world")
	require.NoError(t, e.Map(MapOperatorPending, "Q", "w", true))

	keys(e, 'd', 'Q')
	assert.Equal(t, "world", content(e))
}

// TestMapRemapSemantics covers :map vs :noremap.
func TestMapRemapSemantics(t *testing.T) {
	t.Run("a recursive mapping expands the RHS again", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "b", "dw", true))
		require.NoError(t, e.Map(MapNormal, "Q", "b", false)) // recursive

		keys(e, 'Q')
		assert.Equal(t, "world", content(e), "Q → b → dw")
	})

	t.Run("noremap delivers the RHS verbatim", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "b", "dw", true))
		require.NoError(t, e.Map(MapNormal, "Q", "b", true)) // non-recursive

		keys(e, 'Q')
		assert.Equal(t, "hello world", content(e), "Q → literal b, which just moves back a word")
	})

	t.Run("a self-referential noremap does not loop", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "x", "x", true))

		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})

	t.Run("mutually recursive mappings are stopped", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "a", "b", false))
		require.NoError(t, e.Map(MapNormal, "b", "a", false))

		err := e.HandleKey(KeyEvent{Rune: 'a'})
		require.NotNil(t, err)
		assert.Equal(t, ErrMapRecursionId, err.ID())

		// The message has to name one of the mappings in the cycle, or there is
		// nothing to go on when it is one of a hundred lines of config.
		assert.Regexp(t, `^recursive mapping: [ab] expanded past 1000 levels$`, err.Error())
	})
}

// TestMapLiteralArgument verifies mappings do not rewrite the character
// argument of r/f/t, which is data rather than a command.
func TestMapLiteralArgument(t *testing.T) {
	t.Run("r takes the literal character", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "x", "dd", true))

		keys(e, 'r', 'x')
		assert.Equal(t, "xello", content(e), "x should be inserted, not run as a mapping")
	})

	t.Run("f searches for the literal character", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "l", "dd", true))

		keys(e, 'f', 'l')
		assert.Equal(t, "hello", content(e))
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})
}

// TestMapNop verifies a key can be disabled.
func TestMapNop(t *testing.T) {
	e := newTestEditor("hello")
	require.NoError(t, e.Map(MapNormal, "x", "<Nop>", true))

	keys(e, 'x')
	assert.Equal(t, "hello", content(e), "x should do nothing")
}

// TestUnmapAndClear covers removing mappings.
func TestUnmapAndClear(t *testing.T) {
	t.Run("unmap restores the default", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "x", "<Nop>", true))
		keys(e, 'x')
		assert.Equal(t, "hello", content(e))

		require.NoError(t, e.Unmap(MapNormal, "x"))
		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})

	t.Run("unmapping something unmapped is not an error", func(t *testing.T) {
		e := newTestEditor("hello")
		assert.NoError(t, e.Unmap(MapNormal, "Q"))
	})

	t.Run("clear removes every mapping for the mode", func(t *testing.T) {
		e := newTestEditor("hello")
		require.NoError(t, e.Map(MapNormal, "x", "<Nop>", true))
		require.NoError(t, e.Map(MapNormal, "Q", "dd", true))

		e.ClearMappings(MapNormal)
		assert.Empty(t, e.Mappings(MapNormal))

		keys(e, 'x')
		assert.Equal(t, "ello", content(e))
	})
}

// TestMapRedefinition verifies that the most recent definition wins.
func TestMapRedefinition(t *testing.T) {
	e := newTestEditor("hello world")
	require.NoError(t, e.Map(MapNormal, "Q", "x", true))
	require.NoError(t, e.Map(MapNormal, "Q", "dw", true))

	require.Len(t, e.Mappings(MapNormal), 1)

	keys(e, 'Q')
	assert.Equal(t, "world", content(e))
}

// TestMapLeader covers <leader> expansion.
func TestMapLeader(t *testing.T) {
	t.Run("defaults to backslash", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "<leader>d", "dw", true))

		typeKeys(t, e, `\d`)
		assert.Equal(t, "world", content(e))
	})

	t.Run("honours a configured leader", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.SetMapLeader(",")
		require.NoError(t, e.Map(MapNormal, "<leader>d", "dw", true))

		keys(e, ',', 'd')
		assert.Equal(t, "world", content(e))
	})
}

// TestMapInvalidNotation verifies bad notation is rejected at definition time
// rather than silently producing a mapping that can never fire.
func TestMapInvalidNotation(t *testing.T) {
	e := newTestEditor("hello")

	assert.ErrorIs(t, e.Map(MapNormal, "<Bogus>", "x", true), ErrInvalidMapping)
	assert.ErrorIs(t, e.Map(MapNormal, "x", "<Bogus>", true), ErrInvalidMapping)
	assert.ErrorIs(t, e.Map(MapNormal, "", "x", true), ErrInvalidMapping)
}

// TestMapWithCount verifies that a count typed before a mapped key still
// applies to the command the mapping expands to.
func TestMapWithCount(t *testing.T) {
	e := newTestEditor("hello")
	require.NoError(t, e.Map(MapNormal, "Q", "x", true))

	keys(e, '3', 'Q')
	assert.Equal(t, "lo", content(e))
}

// TestMappingsListing verifies the accessor used for displaying mappings.
func TestMappingsListing(t *testing.T) {
	e := newTestEditor("hello")
	require.NoError(t, e.Map(MapNormal, "U", "<C-r>", true))

	list := e.Mappings(MapNormal)
	require.Len(t, list, 1)
	assert.Equal(t, "U", FormatKeys(list[0].LHS))
	assert.Equal(t, "<C-r>", FormatKeys(list[0].RHS))
	assert.True(t, list[0].NoRemap)
}
