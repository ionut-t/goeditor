package core

import "strconv"

// Repeating the last change with '.' works like Vim's redo buffer: the keys of
// the last command that changed the text are kept and fed back through
// dispatchKey. They are the keys after mapping expansion, so a mapped ,d → dw
// repeats as dw, and the replay itself runs with mappings off.
//
// A command starts on a key that reaches normal mode with nothing pending and
// ends once normal mode is idle again. That takes in the insert session a
// command opens, so cwfoo<Esc> repeats as a whole. Count digits are kept out of
// the keys: the counts typed before and after an operator multiply into one
// count (2d3w is 6dw), which a count on '.' replaces.
//
// A command that goes through any other mode (visual, command line, search)
// is not recorded yet.

// repeatRecord is a change '.' can repeat.
type repeatRecord struct {
	keys  []KeyEvent // without count digits
	count int        // 0 if none was typed
}

// commandRecording collects the keys of the command in progress.
type commandRecording struct {
	active     bool
	keys       []KeyEvent
	count      int  // product of the counts typed so far, 0 if none
	run        int  // count being typed, while inRun
	inRun      bool // the last key was a count digit
	edits      int  // e.editCount when the command started
	repeatable bool // cleared by commands '.' must not repeat (U, '.' itself)
}

// recordKeyStart runs before a key is dispatched. It starts a recording when
// the key begins a new normal-mode command, and returns State.PendingCount
// for recordKeyEnd.
func (e *editor) recordKeyStart() (pendingCount *int) {
	if e.replaying {
		return nil
	}
	rec := &e.recording
	if !rec.active {
		if !e.normalIdle() {
			return nil
		}
		*rec = commandRecording{
			active:     true,
			keys:       rec.keys[:0],
			edits:      e.editCount,
			repeatable: true,
		}
	}
	return e.state.PendingCount
}

// recordKeyEnd runs after key was dispatched in mode. before is
// State.PendingCount as it was before. Count digits go into the count, other
// keys into the keys, and once the command is over a command that changed
// the text becomes the record '.' repeats.
func (e *editor) recordKeyEnd(mode EditorMode, key KeyEvent, before *int) {
	rec := &e.recording
	if e.replaying || !rec.active {
		return
	}

	if mode.Name() == NormalMode && isCountKey(key, before, e.state.PendingCount) {
		rec.run, rec.inRun = *e.state.PendingCount, true
	} else {
		if rec.inRun {
			rec.count = mulCount(max(rec.count, 1), rec.run)
			rec.inRun = false
		}
		rec.keys = append(rec.keys, key)
	}

	switch {
	case e.state.Mode != NormalMode && e.state.Mode != InsertMode:
		rec.active = false
	case e.normalIdle():
		rec.active = false
		if rec.repeatable && e.editCount != rec.edits {
			e.repeat = repeatRecord{
				keys:  append(e.repeat.keys[:0], rec.keys...),
				count: rec.count,
			}
		}
	}
}

// isCountKey reports whether key was taken as a count digit: State.PendingCount
// went from before to after by appending it.
func isCountKey(key KeyEvent, before, after *int) bool {
	if after == nil || key.Rune < '0' || key.Rune > '9' || key.Modifiers != 0 {
		return false
	}
	n := 0
	if before != nil {
		n = *before
	}
	return *after == appendCountDigit(n, int(key.Rune-'0'))
}

// normalIdle reports whether the editor is in normal mode with no command
// part way typed, count included.
func (e *editor) normalIdle() bool {
	m, ok := e.currentMode.(*normalMode)
	return ok && e.state.Mode == NormalMode && m.Idle() && e.state.PendingCount == nil
}

// noRepeat keeps the command in progress from becoming the record '.'
// repeats, though it changes the text.
func (e *editor) noRepeat() {
	e.recording.repeatable = false
}

// repeatChange replays the last change. A count other than 0 replaces the
// recorded one, for this and later repeats, as in Vim. The replay is one undo
// step. If a key fails, or the command ends before its last key (a motion
// that found nothing ends it early), the rest is dropped and the editor goes
// back to normal mode; the record is kept.
func (e *editor) repeatChange(count int) *EditorError {
	e.noRepeat()
	if len(e.repeat.keys) == 0 {
		return nil
	}
	if count > 0 {
		e.repeat.count = count
	}

	e.replaying = true
	defer func() { e.replaying = false }()

	keys := make([]KeyEvent, 0, len(e.repeat.keys)+4)
	if e.repeat.count > 0 {
		for _, r := range strconv.Itoa(e.repeat.count) {
			keys = append(keys, KeyEvent{Rune: r})
		}
	}
	keys = append(keys, e.repeat.keys...)

	for i, key := range keys {
		if err := e.dispatchKey(key); err != nil {
			e.abortReplay()
			return err
		}
		if i < len(keys)-1 && e.normalIdle() {
			break
		}
	}
	return nil
}

// abortReplay drops whatever command a failed replay left part way.
func (e *editor) abortReplay() {
	if m, ok := e.currentMode.(*normalMode); ok {
		m.clearPendingState(e)
		return
	}
	e.SetNormalMode()
}

// resetRepeat forgets the record, for new content.
func (e *editor) resetRepeat() {
	e.repeat = repeatRecord{}
	e.recording = commandRecording{}
}
