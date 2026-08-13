package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fireMapTimeout runs the pending mapping timer the way the UI layer does:
// ask whether one should be running, then hand the token straight back.
func fireMapTimeout(t *testing.T, e Editor) {
	t.Helper()
	_, token, ok := e.PendingMapTimeout()
	require.True(t, ok, "expected keys to be held for a timeout")
	e.TimeoutPendingMapping(token)
}

// TestPendingMapTimeout covers when a timer is asked for. Keys are only held
// when a longer mapping could still match; anything else resolves immediately.
func TestPendingMapTimeout(t *testing.T) {
	t.Run("no timer when nothing is held", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))

		_, _, ok := e.PendingMapTimeout()
		assert.False(t, ok, "no keys typed yet")

		keys(e, 'Q')
		_, _, ok = e.PendingMapTimeout()
		assert.False(t, ok, "the mapping was complete and unambiguous")
	})

	t.Run("timer requested while a longer mapping could still match", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		d, _, ok := e.PendingMapTimeout()
		require.True(t, ok)
		assert.Equal(t, DefaultMapTimeoutLen, d)
		assert.Equal(t, "hello world", content(e), "nothing has run yet")
	})

	t.Run("timer reports the configured timeoutlen", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.SetMapTimeoutLen(250 * time.Millisecond)
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		d, _, ok := e.PendingMapTimeout()
		require.True(t, ok)
		assert.Equal(t, 250*time.Millisecond, d)
	})

	t.Run("no timer when timeout is off", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.SetMapTimeout(false)
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		_, _, ok := e.PendingMapTimeout()
		assert.False(t, ok, "with notimeout the keys wait indefinitely")
		assert.Equal(t, "hello world", content(e))
	})
}

// TestTimeoutPendingMapping covers what expiry actually does with the held
// keys — the whole point of the option.
func TestTimeoutPendingMapping(t *testing.T) {
	t.Run("runs the shorter mapping when it is complete", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		fireMapTimeout(t, e)
		assert.Equal(t, "world", content(e), "Q should run on its own")
	})

	t.Run("the longer mapping still wins when the next key arrives", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q', 'Q')
		assert.Equal(t, "", content(e), "QQ should delete the line")
	})

	t.Run("delivers the keys unmapped when they complete nothing", func(t *testing.T) {
		e := newTestEditor("hello world")
		// 'x' is a real command and also the start of a mapping; timing out has
		// to fall back to the command rather than swallowing the key.
		require.NoError(t, e.Map(MapNormal, "xq", "dd", true))

		keys(e, 'x')
		fireMapTimeout(t, e)
		assert.Equal(t, "ello world", content(e))
	})

	t.Run("a stale token is ignored", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		_, token, ok := e.PendingMapTimeout()
		require.True(t, ok)

		// The second Q resolves the run, so the timer started for the first one
		// must not also fire — that would run 'dw' on top of the 'dd'.
		keys(e, 'Q')
		require.Equal(t, "", content(e))

		e.TimeoutPendingMapping(token)
		assert.Equal(t, "", content(e), "the late timer did nothing")
	})

	t.Run("a token is ignored once nothing is pending", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		_, token, _ := e.PendingMapTimeout()
		fireMapTimeout(t, e)
		require.Equal(t, "world", content(e))

		e.TimeoutPendingMapping(token)
		assert.Equal(t, "world", content(e), "the same timer cannot fire twice")
	})

	t.Run("turning timeout off invalidates a running timer", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "Q", "dw", true))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		_, token, _ := e.PendingMapTimeout()

		e.SetMapTimeout(false)
		e.TimeoutPendingMapping(token)
		assert.Equal(t, "hello world", content(e), "the keys keep waiting")

		keys(e, 'Q')
		assert.Equal(t, "", content(e))
	})

	t.Run("the expanded mapping is re-mapped unless it is a noremap", func(t *testing.T) {
		e := newTestEditor("hello world")
		require.NoError(t, e.Map(MapNormal, "b", "dw", true))
		require.NoError(t, e.Map(MapNormal, "Q", "b", false))
		require.NoError(t, e.Map(MapNormal, "QQ", "dd", true))

		keys(e, 'Q')
		fireMapTimeout(t, e)
		assert.Equal(t, "world", content(e), "Q -> b -> dw")
	})

	t.Run("a leader mapping that prefixes another still fires", func(t *testing.T) {
		e := newTestEditor("hello world")
		e.SetMapLeader(",")
		require.NoError(t, e.Map(MapNormal, "<leader>d", "dw", true))
		require.NoError(t, e.Map(MapNormal, "<leader>dd", "dd", true))

		keys(e, ',', 'd')
		fireMapTimeout(t, e)
		assert.Equal(t, "world", content(e))
	})
}

// TestSetTimeoutOptions covers the ':set' side of the two options.
func TestSetTimeoutOptions(t *testing.T) {
	t.Run("timeoutlen", func(t *testing.T) {
		e := newTestEditor("hello")
		require.Nil(t, runCommand(t, e, "set timeoutlen=250"))
		assert.Equal(t, 250*time.Millisecond, e.MapTimeoutLen())

		require.Nil(t, runCommand(t, e, "set tm=0"))
		assert.Equal(t, time.Duration(0), e.MapTimeoutLen())
	})

	t.Run("timeout and notimeout", func(t *testing.T) {
		e := newTestEditor("hello")
		assert.True(t, e.MapTimeoutEnabled(), "on by default, as in Vim")

		require.Nil(t, runCommand(t, e, "set notimeout"))
		assert.False(t, e.MapTimeoutEnabled())

		require.Nil(t, runCommand(t, e, "set timeout"))
		assert.True(t, e.MapTimeoutEnabled())

		require.Nil(t, runCommand(t, e, "set noto"))
		assert.False(t, e.MapTimeoutEnabled())

		require.Nil(t, runCommand(t, e, "set to"))
		assert.True(t, e.MapTimeoutEnabled())
	})

	t.Run("rejects a value that is not milliseconds", func(t *testing.T) {
		for _, cmd := range []string{"set timeoutlen=abc", "set timeoutlen=-1", "set timeoutlen="} {
			e := newTestEditor("hello")
			err := runCommand(t, e, cmd)
			require.NotNil(t, err, cmd)
			assert.Equal(t, ErrInvalidCommandId, err.ID(), cmd)
			assert.Equal(t, DefaultMapTimeoutLen, e.MapTimeoutLen(), cmd)
		}
	})
}
