package core

import (
	"fmt"
	"time"
)

// MapMode is a bitmask selecting which modes a mapping applies to. It mirrors
// Vim's :map prefixes — :nmap is MapNormal, :vmap is MapVisual, and so on.
type MapMode uint8

const (
	MapNormal MapMode = 1 << iota
	// MapVisual covers both visual and visual-line mode.
	MapVisual
	MapInsert
	// MapOperatorPending applies while an operator (d/y/c) waits for a motion.
	MapOperatorPending
)

// MapAll is the set targeted by an unprefixed :map — normal, visual and
// operator-pending, but deliberately not insert, matching Vim.
const MapAll = MapNormal | MapVisual | MapOperatorPending

// allMapModes lists the individual bits, for iterating a mask.
var allMapModes = []MapMode{MapNormal, MapVisual, MapInsert, MapOperatorPending}

// maxMapDepth bounds mapping expansion, mirroring Vim's 'maxmapdepth'. Exceeding
// it means the mappings are mutually recursive.
const maxMapDepth = 1000

// Mapping is a single key mapping: press LHS, get RHS. When NoRemap is set the
// RHS is delivered verbatim instead of being re-resolved (Vim's :noremap).
type Mapping struct {
	LHS     []KeyEvent
	RHS     []KeyEvent
	NoRemap bool
}

// String renders the mapping the way :map would list it.
func (m Mapping) String() string {
	return fmt.Sprintf("%s\t%s", FormatKeys(m.LHS), FormatKeys(m.RHS))
}

// mappingTable stores mappings per individual mode bit.
type mappingTable struct {
	byMode map[MapMode][]Mapping
	count  int
}

func newMappingTable() *mappingTable {
	return &mappingTable{byMode: make(map[MapMode][]Mapping)}
}

func (t *mappingTable) empty() bool { return t.count == 0 }

// set installs a mapping, replacing any existing one with the same LHS. Later
// definitions win, as in Vim.
func (t *mappingTable) set(mode MapMode, m Mapping) {
	existing := t.byMode[mode]
	for i := range existing {
		if keysEqual(existing[i].LHS, m.LHS) {
			existing[i] = m
			return
		}
	}
	t.byMode[mode] = append(existing, m)
	t.count++
}

// remove deletes the mapping with the given LHS, reporting whether one existed.
func (t *mappingTable) remove(mode MapMode, lhs []KeyEvent) bool {
	existing := t.byMode[mode]
	for i := range existing {
		if keysEqual(existing[i].LHS, lhs) {
			t.byMode[mode] = append(existing[:i], existing[i+1:]...)
			t.count--
			return true
		}
	}
	return false
}

func (t *mappingTable) clear(mode MapMode) {
	t.count -= len(t.byMode[mode])
	delete(t.byMode, mode)
}

// match resolves the keys typed so far. exact is the mapping whose LHS the keys
// match completely; hasLonger reports that at least one mapping starts with them
// but needs more keys, meaning the caller must wait before committing.
func (t *mappingTable) match(mode MapMode, pending []KeyEvent) (exact *Mapping, hasLonger bool) {
	candidates := t.byMode[mode]
	for i := range candidates {
		lhs := candidates[i].LHS
		if len(lhs) < len(pending) || !keysEqual(lhs[:len(pending)], pending) {
			continue
		}
		if len(lhs) == len(pending) {
			exact = &candidates[i]
		} else {
			hasLonger = true
		}
	}
	return exact, hasLonger
}

func keysEqual(a, b []KeyEvent) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Editor API ---

// Map installs a key mapping for every mode in modes, using Vim key notation on
// both sides (see ParseKeys). Set noremap to keep the RHS from being re-mapped,
// which is what :noremap and friends do.
func (e *editor) Map(modes MapMode, lhs, rhs string, noremap bool) error {
	lhsKeys, err := ParseKeys(lhs, e.mapLeader)
	if err != nil {
		return err
	}
	if len(lhsKeys) == 0 {
		return fmt.Errorf("%w: empty left-hand side", ErrInvalidMapping)
	}

	rhsKeys, err := ParseKeys(rhs, e.mapLeader)
	if err != nil {
		return err
	}

	for _, mode := range allMapModes {
		if modes&mode != 0 {
			e.mappings.set(mode, Mapping{LHS: lhsKeys, RHS: rhsKeys, NoRemap: noremap})
		}
	}

	return nil
}

// Unmap removes the mapping for lhs from every mode in modes. It reports an
// error only when lhs is not valid notation; removing something that was never
// mapped is not an error.
func (e *editor) Unmap(modes MapMode, lhs string) error {
	lhsKeys, err := ParseKeys(lhs, e.mapLeader)
	if err != nil {
		return err
	}

	for _, mode := range allMapModes {
		if modes&mode != 0 {
			e.mappings.remove(mode, lhsKeys)
		}
	}

	// Removing a mapping can strand keys that were being held to disambiguate it.
	e.clearPendingMapKeys()

	return nil
}

// ClearMappings removes every mapping for the given modes (Vim's :mapclear).
func (e *editor) ClearMappings(modes MapMode) {
	for _, mode := range allMapModes {
		if modes&mode != 0 {
			e.mappings.clear(mode)
		}
	}
	e.clearPendingMapKeys()
}

// Mappings returns the mappings registered for a single mode.
func (e *editor) Mappings(mode MapMode) []Mapping {
	src := e.mappings.byMode[mode]
	out := make([]Mapping, len(src))
	copy(out, src)
	return out
}

// SetMapLeader sets what <leader> expands to in mappings defined from now on.
// Existing mappings keep the leader they were defined with, as in Vim.
func (e *editor) SetMapLeader(leader string) {
	e.mapLeader = leader
}

// MapLeader returns the current <leader> expansion.
func (e *editor) MapLeader() string {
	if e.mapLeader == "" {
		return DefaultMapLeader
	}
	return e.mapLeader
}

// --- Resolution ---

// activeMapMode reports which mapping set applies to the current editor mode,
// or 0 when mappings do not apply at all.
func (e *editor) activeMapMode() MapMode {
	switch e.currentMode.Name() {
	case NormalMode:
		if e.currentMode.OperatorPending() {
			return MapOperatorPending
		}
		return MapNormal
	case VisualMode, VisualLineMode:
		return MapVisual
	case InsertMode:
		return MapInsert
	}
	// Command and search mode collect literal text; :cmap is not supported.
	return 0
}

// handleKeyMapped runs a key through the mapping layer. When remap is false the
// key goes straight to the current mode — that is how the RHS of a :noremap is
// delivered.
func (e *editor) handleKeyMapped(key KeyEvent, remap bool) *EditorError {
	if !remap || e.mappings.empty() {
		return e.dispatchKey(key)
	}

	// The character argument of r/f/F/t/T is data, not a command, so mappings
	// must not rewrite it. Vim behaves the same way.
	if e.currentMode.AwaitingLiteral() {
		return e.dispatchKey(key)
	}

	mode := e.activeMapMode()
	if mode == 0 {
		return e.dispatchKey(key)
	}

	e.holdKey(key)

	exact, hasLonger := e.mappings.match(mode, e.pendingMapKeys)

	// A longer mapping could still match, so hold the keys until the next one
	// decides it — or until 'timeoutlen' expires and TimeoutPendingMapping
	// settles for what has been typed. With 'notimeout' the wait is indefinite.
	if hasLonger {
		return nil
	}

	if exact != nil {
		return e.applyMapping(exact)
	}

	return e.FlushPendingMapping()
}

// applyMapping expands a matched mapping, releasing the keys that produced it.
func (e *editor) applyMapping(m *Mapping) *EditorError {
	// Copied before the pending keys are cleared: m points into the table, which
	// the expansion below is free to modify (an RHS can run :nmap).
	rhs, noremap := m.RHS, m.NoRemap
	e.clearPendingMapKeys()
	return e.feedKeys(rhs, !noremap)
}

// holdKey adds a key to the run being held while a longer mapping might match.
func (e *editor) holdKey(key KeyEvent) {
	e.pendingMapKeys = append(e.pendingMapKeys, key)
	e.mapPendingGen++
}

// clearPendingMapKeys ends the current run of held keys, invalidating any
// 'timeoutlen' timer still outstanding for it.
func (e *editor) clearPendingMapKeys() {
	e.pendingMapKeys = e.pendingMapKeys[:0]
	e.mapPendingGen++
}

// PendingMapTimeout reports how long to wait before giving up on a longer
// mapping and committing the keys held so far, along with a token identifying
// this run of held keys. ok is false when nothing is held or 'timeout' is off,
// meaning no timer should run.
//
// The core cannot start timers, so the UI layer is expected to call this after
// every key, schedule a one-shot timer when ok, and hand the token back to
// TimeoutPendingMapping when it fires.
func (e *editor) PendingMapTimeout() (d time.Duration, token uint64, ok bool) {
	if len(e.pendingMapKeys) == 0 || !e.mapTimeout {
		return 0, 0, false
	}
	return e.mapTimeoutLen, e.mapPendingGen, true
}

// TimeoutPendingMapping resolves the held keys the way 'timeoutlen' expiring
// does in Vim: if they form a complete mapping it runs, otherwise they are
// delivered unmapped. A token from a run that has already been resolved is
// ignored, so a timer that fires late is harmless.
func (e *editor) TimeoutPendingMapping(token uint64) *EditorError {
	if len(e.pendingMapKeys) == 0 || token != e.mapPendingGen {
		return nil
	}

	// The keys were never delivered, so the mode cannot have changed since they
	// were held — but resolve against the live mode regardless, for the same
	// reason feedKeys does.
	if mode := e.activeMapMode(); mode != 0 {
		if exact, _ := e.mappings.match(mode, e.pendingMapKeys); exact != nil {
			return e.applyMapping(exact)
		}
	}

	return e.FlushPendingMapping()
}

// FlushPendingMapping delivers keys that are being held while waiting to see
// whether they complete a longer mapping, giving up on the longer match and on
// any mapping the held keys already complete.
//
// It is called automatically once the held keys can no longer match anything.
// TimeoutPendingMapping is the one to call from a 'timeoutlen' timer: this one
// discards a complete match rather than running it.
func (e *editor) FlushPendingMapping() *EditorError {
	if len(e.pendingMapKeys) == 0 {
		return nil
	}

	pending := make([]KeyEvent, len(e.pendingMapKeys))
	copy(pending, e.pendingMapKeys)
	e.clearPendingMapKeys()

	// The first key has already failed to start a mapping, so deliver it as-is.
	// The rest re-enter resolution because they may begin a new one.
	err := e.dispatchKey(pending[0])
	for _, key := range pending[1:] {
		if mapErr := e.handleKeyMapped(key, true); mapErr != nil && err == nil {
			err = mapErr
		}
	}

	return err
}

// feedKeys delivers an expanded right-hand side one key at a time. Each key is
// resolved against the mode that is current when it is delivered, so a mapping
// whose RHS changes mode (":nmap gv V") behaves like the keys being typed.
func (e *editor) feedKeys(keys []KeyEvent, remap bool) *EditorError {
	if e.mapDepth >= maxMapDepth {
		e.clearPendingMapKeys()
		e.mapDepth = 0
		return &EditorError{id: ErrMapRecursionId, err: ErrMapRecursion}
	}

	e.mapDepth++
	defer func() { e.mapDepth-- }()

	var firstErr *EditorError
	for _, key := range keys {
		if err := e.handleKeyMapped(key, remap); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// The expansion is complete, so nothing further is coming to disambiguate
	// keys it left pending.
	if err := e.FlushPendingMapping(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// dispatchKey hands a key to the current mode, with the bookkeeping that has to
// happen once per key actually delivered rather than once per HandleKey call —
// otherwise every key of an expanded mapping would share one cursor snapshot.
func (e *editor) dispatchKey(key KeyEvent) *EditorError {
	// Snapshot cursor before any change so SaveHistory can record the pre-change position.
	e.preChangeCursor = e.buffer.GetCursor()

	err := e.currentMode.HandleKey(e, e.buffer, key)

	// Update derived state AFTER handling key
	e.ScrollViewport() // Ensure cursor is visible after potential movement

	return err
}
