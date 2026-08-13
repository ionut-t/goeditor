package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Editor options set with ':set'. Two forms are supported, matching Vim:
// boolean options toggled by name ("rnu" / "nornu") and valued options written
// as name=value ("mapleader=,").

// DefaultMapTimeoutLen is the default 'timeoutlen', matching Vim.
const DefaultMapTimeoutLen = time.Second

// executeSet applies a ':set' command. Vim accepts any number of options on one
// line and applies them left to right, stopping at the first bad one — options
// before it stay applied.
func (e *editor) executeSet(args []string) *EditorError {
	if len(args) == 0 {
		// Vim lists the modified options here. There is nowhere to render a
		// multi-line list, so this stays an error rather than a silent no-op.
		return &EditorError{
			id:  ErrInvalidCommandId,
			err: fmt.Errorf("%w: set requires an option, e.g. :set rnu", ErrInvalidCommand),
		}
	}

	for _, arg := range args {
		if err := e.setOne(arg); err != nil {
			return err
		}
	}

	return nil
}

// setOne applies a single ':set' argument.
func (e *editor) setOne(arg string) *EditorError {
	switch arg {
	case "relativenumber", "rnu":
		e.state.RelativeNumbers = true
		e.DispatchSignal(RelativeNumbersSignal{enabled: true})
		return nil

	case "norelativenumber", "nornu":
		e.state.RelativeNumbers = false
		e.DispatchSignal(RelativeNumbersSignal{enabled: false})
		return nil
		// Add 'number'/'nonu' later if needed

	case "timeout", "to":
		e.SetMapTimeout(true)
		return nil

	case "notimeout", "noto":
		e.SetMapTimeout(false)
		return nil
	}

	if name, value, ok := strings.Cut(arg, "="); ok {
		return e.setValueOption(name, value)
	}

	return unknownOptionError(arg)
}

// setValueOption applies an option written as name=value.
func (e *editor) setValueOption(name, value string) *EditorError {
	switch name {
	case "mapleader", "leader":
		// ParseKeys accepts an empty sequence (that is how <Nop> works), so an
		// empty leader has to be rejected here. Letting it through would store
		// "", which MapLeader reports as the default — silently resetting the
		// leader instead of reporting the mistake.
		if value == "" {
			return &EditorError{
				id:  ErrInvalidMappingId,
				err: fmt.Errorf("%w: mapleader requires a value, e.g. :set mapleader=,", ErrInvalidMapping),
			}
		}

		// Validate now so a bad leader is reported while the user is typing it,
		// rather than surfacing later on every mapping that uses <leader>.
		if _, err := ParseKeys(value, DefaultMapLeader); err != nil {
			return &EditorError{id: ErrInvalidMappingId, err: err}
		}

		e.SetMapLeader(value)
		return nil

	case "timeoutlen", "tm":
		ms, err := strconv.Atoi(value)
		if err != nil || ms < 0 {
			return &EditorError{
				id:  ErrInvalidCommandId,
				err: fmt.Errorf("%w: timeoutlen takes milliseconds, e.g. :set timeoutlen=500", ErrInvalidCommand),
			}
		}
		e.SetMapTimeoutLen(time.Duration(ms) * time.Millisecond)
		return nil
	}

	return unknownOptionError(name)
}

// SetMapTimeout sets 'timeout': whether keys held to disambiguate a mapping are
// committed once 'timeoutlen' expires. With it off they wait indefinitely for
// the key that decides them.
func (e *editor) SetMapTimeout(enabled bool) {
	e.mapTimeout = enabled
	if !enabled {
		// Any timer already scheduled must not fire against the keys it was
		// started for, now that they are meant to wait.
		e.mapPendingGen++
	}
}

// MapTimeoutEnabled reports the 'timeout' setting.
func (e *editor) MapTimeoutEnabled() bool { return e.mapTimeout }

// SetMapTimeoutLen sets 'timeoutlen', how long an ambiguous mapping waits.
func (e *editor) SetMapTimeoutLen(d time.Duration) {
	if d < 0 {
		d = 0
	}
	e.mapTimeoutLen = d
}

// MapTimeoutLen reports the 'timeoutlen' setting.
func (e *editor) MapTimeoutLen() time.Duration { return e.mapTimeoutLen }

func unknownOptionError(name string) *EditorError {
	return &EditorError{
		id:  ErrInvalidCommandId,
		err: fmt.Errorf("%w: unknown option %q", ErrInvalidCommand, name),
	}
}
