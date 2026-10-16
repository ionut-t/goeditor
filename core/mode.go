package core

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

// prefixHandler runs the second key of a prefix command (gg, zz, ...) for a
// mode. Any count typed before the prefix is still in State.PendingCount.
type prefixHandler[M any] func(m M, editor Editor, buffer Buffer, key KeyEvent) *EditorError

// prefixTable maps a prefix key to the handler for the key that follows it.
// Vim marks these commands as taking a second character and reads it before
// calling the handler (nv_g_cmd, nv_zet); each mode keeps one such table.
type prefixTable[M any] map[rune]prefixHandler[M]

// pendingPrefix is the prefix key a mode is waiting to complete, or 0.
type pendingPrefix struct {
	prefix rune
}

// start records r as the pending prefix. The pending count is left alone so
// the prefix handler can use it (5gg, 10zz).
func (p *pendingPrefix) start(r rune) { p.prefix = r }

func (p *pendingPrefix) reset() { p.prefix = 0 }

// dispatchPrefix runs the handler for the pending prefix, if there is one, and
// reports whether it did. The prefix is cleared before the handler runs, so a
// handler can start a new one.
func dispatchPrefix[M any](table prefixTable[M], p *pendingPrefix, m M, editor Editor, buffer Buffer, key KeyEvent) (bool, *EditorError) {
	if p.prefix == 0 {
		return false, nil
	}
	handler, ok := table[p.prefix]
	p.reset()
	if !ok {
		return true, nil
	}
	return true, handler(m, editor, buffer, key)
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
	n := int(key.Rune - '0')
	if state.PendingCount != nil {
		n += *state.PendingCount * 10
	}
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
