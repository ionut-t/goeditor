package core

import (
	"fmt"
)

type Mode string

const (
	NormalMode     Mode = "normal"
	InsertMode     Mode = "insert"
	VisualMode     Mode = "visual"
	VisualLineMode Mode = "visual-line"
	CommandMode    Mode = "command"
	SearchMode     Mode = "search"
)

// EditorMode represents a Vim editing mode
type EditorMode interface {
	Name() Mode
	// HandleKey processes a key press. It can return an error or signal a mode change.
	// It takes the current editor *state* and returns the *new desired state*.
	// This makes state management more explicit.
	HandleKey(editor Editor, buffer Buffer, key KeyEvent) *EditorError
	Enter(editor Editor, buffer Buffer) // Called when entering the mode
	Exit(editor Editor, buffer Buffer)  // Called when exiting the mode

	// AwaitingLiteral reports whether the mode is waiting for a literal
	// character argument — the character after r, f, F, t or T. Key mappings do
	// not apply to it, since it is data rather than a command.
	AwaitingLiteral() bool

	// OperatorPending reports whether an operator (d/y/c) is waiting for a
	// motion, which selects the operator-pending mapping set.
	OperatorPending() bool
}

// baseMode provides the default pending-state answers for modes that have no
// multi-key sequences of their own.
type baseMode struct{}

func (baseMode) AwaitingLiteral() bool { return false }
func (baseMode) OperatorPending() bool { return false }

// pendingPrefix is the prefix key (g, z) a mode is waiting to complete, or 0.
// Like Vim's nv_g_cmd / nv_zet, the handler reads the second key itself; each
// mode switches on the prefix in its handlePrefix.
type pendingPrefix struct {
	prefix rune
}

// start records r as the pending prefix. The pending count is left alone so
// the prefix handler can use it (5gg, 10zz).
func (p *pendingPrefix) start(r rune) { p.prefix = r }

func (p *pendingPrefix) reset() { p.prefix = 0 }

// take returns the pending prefix, or 0, and clears it, so the handler can
// start a new one.
func (p *pendingPrefix) take() rune {
	r := p.prefix
	p.prefix = 0
	return r
}

// isPrefixKey reports whether key starts a two-key command in the visual
// modes. Normal mode starts its prefixes from its own key switch.
func isPrefixKey(key KeyEvent) bool {
	return (key.Rune == 'g' || key.Rune == 'z') && key.Modifiers&ModCtrl == 0
}

// showPrefix shows a just-typed prefix after any pending count (10g), as
// Vim's showcmd does.
func showPrefix(editor Editor, r rune) {
	if count := editor.GetState().PendingCount; count != nil {
		editor.UpdateCommand(fmt.Sprintf("%d%c", *count, r))
		return
	}
	editor.UpdateCommand(string(r))
}

// maxCount is the largest count Vim accepts. Counts that would grow past it
// are clamped to it, so a long run of digits can't overflow.
const maxCount = 999999999

// appendCountDigit returns count n with digit d typed after it.
func appendCountDigit(n, d int) int {
	if n > (maxCount-d)/10 {
		return maxCount
	}
	return n*10 + d
}

// mulCount multiplies two counts, as [count]op[count]motion does.
func mulCount(a, b int) int {
	if b != 0 && a > maxCount/b {
		return maxCount
	}
	return a * b
}

// accumulateCount adds a digit key to State.PendingCount and reports whether
// the key was consumed. '0' only counts once a count has started; on its own
// it is a motion.
func accumulateCount(editor Editor, key KeyEvent) bool {
	if key.Modifiers&ModCtrl != 0 {
		return false
	}
	state := editor.GetState()
	if key.Rune < '0' || key.Rune > '9' || (key.Rune == '0' && state.PendingCount == nil) {
		return false
	}
	n := 0
	if state.PendingCount != nil {
		n = *state.PendingCount
	}
	n = appendCountDigit(n, int(key.Rune-'0'))
	state.PendingCount = &n
	editor.SetState(state)
	return true
}

// takeCount consumes State.PendingCount. It returns 0 when no count was typed,
// so callers can tell "no count" from an explicit 1.
func takeCount(editor Editor) int {
	pending := editor.GetState().PendingCount
	if pending == nil {
		return 0
	}
	editor.ResetPendingCount()
	return *pending
}
