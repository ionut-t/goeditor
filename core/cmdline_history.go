package core

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// HistoryKind selects which history to read or write: commands typed after
// ':' or searches typed after '/' or '?'. Forward and backward searches share
// one history.
type HistoryKind int

const (
	CommandHistory HistoryKind = iota // commands typed after ':'
	SearchHistory                     // searches from '/', '?', '*' and '#'
	historyKinds
)

// DefaultHistorySize is how many entries each history keeps by default. It
// matches Vim's default.
const DefaultHistorySize = 200

// histories are the command-line history lists, oldest entry first.
type histories struct {
	lists [historyKinds][]string
	size  int // 'history'
}

// add appends entry to the list of kind, as Vim's add_to_history: an empty
// entry is not kept, an entry already in the list moves to the end, and the
// oldest entries go once the list is over 'history'.
func (h *histories) add(kind HistoryKind, entry string) {
	if entry == "" || kind < 0 || kind >= historyKinds {
		return
	}
	list := slices.DeleteFunc(h.lists[kind], func(s string) bool { return s == entry })
	h.lists[kind] = h.trim(append(list, entry))
}

// trim drops the oldest entries of list past 'history'.
func (h *histories) trim(list []string) []string {
	if over := len(list) - h.size; over > 0 {
		return slices.Delete(list, 0, over)
	}
	return list
}

// CommandLineHistory returns the command or search history, oldest entry
// first. The returned slice is a copy, so it is safe to keep or modify.
func (e *editor) CommandLineHistory(kind HistoryKind) []string {
	if kind < 0 || kind >= historyKinds {
		return nil
	}
	return slices.Clone(e.histories.lists[kind])
}

// SetCommandLineHistory replaces the command or search history with entries,
// listed oldest first. Empty entries are skipped, duplicates are kept once at
// their latest position, and only the newest entries that fit the history
// size are kept.
func (e *editor) SetCommandLineHistory(kind HistoryKind, entries []string) {
	if kind < 0 || kind >= historyKinds {
		return
	}
	e.histories.lists[kind] = nil
	for _, entry := range entries {
		e.histories.add(kind, entry)
	}
}

// SetCommandLineHistorySize sets how many entries each history keeps, the
// same setting as ":set history". Histories that are already longer lose their
// oldest entries. 0 turns history off.
func (e *editor) SetCommandLineHistorySize(n int) {
	e.histories.size = max(n, 0)
	for kind := range e.histories.lists {
		e.histories.lists[kind] = e.histories.trim(e.histories.lists[kind])
	}
}

// CommandLineHistorySize returns how many entries each history keeps.
func (e *editor) CommandLineHistorySize() int {
	return e.histories.size
}

// setHistoryOption applies ":set history=N". Vim caps it at 10000.
func (e *editor) setHistoryOption(value string) *EditorError {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 || n > 10000 {
		return &EditorError{
			id:  ErrInvalidCommandId,
			err: fmt.Errorf("%w: history takes 0 to 10000 entries, e.g. :set history=200", ErrInvalidCommand),
		}
	}
	e.SetCommandLineHistorySize(n)
	return nil
}

// historyBrowse is a command line's place in its history while <Up> and
// <Down> walk it, as Vim's getcmdline() hiscnt and lookfor.
type historyBrowse struct {
	// index is the entry shown, len(list) for the line being typed.
	index int
	// lookfor is the line up to the cursor when the walk started: the prefix
	// <Up> and <Down> look for, and the line <Down> past the newest entry
	// brings back. browsing is false once a key other than the history keys,
	// <Left> and <Right> has been typed; the next history key starts a new
	// walk from index, with the line as it is then.
	lookfor  string
	browsing bool
}

// historyKey reports whether key walks the history: older reports the
// direction, and prefix whether only entries starting with the typed text
// count (<Up> and <Down>; <S-Up>, <S-Down>, <PageUp>, <PageDown>, <C-P> and
// <C-N> take every entry).
func historyKey(key KeyEvent) (ok, older, prefix bool) {
	shifted := key.Modifiers&ModShift != 0
	switch {
	case key.Key == KeyUp:
		return true, true, !shifted
	case key.Key == KeyDown:
		return true, false, !shifted
	case key.Key == KeyPageUp, key.IsCtrl('p'):
		return true, true, false
	case key.Key == KeyPageDown, key.IsCtrl('n'):
		return true, false, false
	}
	return false, false, false
}

// browseHistory moves the command line to the next older or newer entry of
// list, as Vim's <Up>/<Down> on the command line, and reports whether it
// moved. Past the newest entry is the line that was typed. With no entry
// that way the line stays as it is.
func (c *cmdline) browseHistory(list []string, older, prefix bool) bool {
	b := &c.browse
	b.index = min(b.index, len(list))
	if !b.browsing {
		b.lookfor = string(c.text[:c.cursor])
		b.browsing = true
	}

	j := b.index
	for {
		if older {
			if j == 0 {
				return false
			}
			j--
		} else {
			if j == len(list) {
				return false
			}
			j++
		}
		if j == len(list) || !prefix || strings.HasPrefix(list[j], b.lookfor) {
			break
		}
	}

	b.index = j
	if j == len(list) {
		c.set(b.lookfor)
	} else {
		c.set(list[j])
	}
	return true
}

// reset starts the walk past the newest entry.
func (b *historyBrowse) reset() {
	*b = historyBrowse{index: math.MaxInt}
}
