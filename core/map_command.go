package core

import (
	"fmt"
	"strings"
)

// mapCommand describes one entry in the :map family.
type mapCommand struct {
	modes   MapMode
	noremap bool
	action  mapAction
}

type mapAction int

const (
	mapActionSet mapAction = iota
	mapActionUnmap
	mapActionClear
)

// mapCommands is the :map family. Vim's unprefixed :map deliberately covers
// normal, visual and operator-pending mode but not insert mode.
var mapCommands = map[string]mapCommand{
	"map":  {modes: MapAll, noremap: false, action: mapActionSet},
	"nmap": {modes: MapNormal, noremap: false, action: mapActionSet},
	"vmap": {modes: MapVisual, noremap: false, action: mapActionSet},
	"xmap": {modes: MapVisual, noremap: false, action: mapActionSet},
	"omap": {modes: MapOperatorPending, noremap: false, action: mapActionSet},
	"imap": {modes: MapInsert, noremap: false, action: mapActionSet},

	"noremap":  {modes: MapAll, noremap: true, action: mapActionSet},
	"nnoremap": {modes: MapNormal, noremap: true, action: mapActionSet},
	"vnoremap": {modes: MapVisual, noremap: true, action: mapActionSet},
	"xnoremap": {modes: MapVisual, noremap: true, action: mapActionSet},
	"onoremap": {modes: MapOperatorPending, noremap: true, action: mapActionSet},
	"inoremap": {modes: MapInsert, noremap: true, action: mapActionSet},

	"unmap":  {modes: MapAll, action: mapActionUnmap},
	"nunmap": {modes: MapNormal, action: mapActionUnmap},
	"vunmap": {modes: MapVisual, action: mapActionUnmap},
	"xunmap": {modes: MapVisual, action: mapActionUnmap},
	"ounmap": {modes: MapOperatorPending, action: mapActionUnmap},
	"iunmap": {modes: MapInsert, action: mapActionUnmap},

	"mapclear":  {modes: MapAll, action: mapActionClear},
	"nmapclear": {modes: MapNormal, action: mapActionClear},
	"vmapclear": {modes: MapVisual, action: mapActionClear},
	"xmapclear": {modes: MapVisual, action: mapActionClear},
	"omapclear": {modes: MapOperatorPending, action: mapActionClear},
	"imapclear": {modes: MapInsert, action: mapActionClear},
}

// splitCommandWord separates the leading command word from the rest of the line
// without disturbing the whitespace inside the remainder.
func splitCommandWord(cmd string) (word, rest string, ok bool) {
	idx := strings.IndexFunc(cmd, func(r rune) bool { return r == ' ' || r == '\t' })
	if idx < 0 {
		return cmd, "", cmd != ""
	}
	return cmd[:idx], strings.TrimLeft(cmd[idx:], " \t"), true
}

// executeMapCommand handles the :map family. It reports whether name was one of
// those commands, so the caller can fall through to the rest of :-commands.
func (e *editor) executeMapCommand(name, args string) (bool, *EditorError) {
	cmd, ok := mapCommands[name]
	if !ok {
		return false, nil
	}

	switch cmd.action {
	case mapActionClear:
		e.ClearMappings(cmd.modes)
		return true, nil

	case mapActionUnmap:
		lhs := strings.TrimSpace(args)
		if lhs == "" {
			return true, mapUsageError("%s requires a key to unmap", name)
		}
		if err := e.Unmap(cmd.modes, lhs); err != nil {
			return true, &EditorError{id: ErrInvalidMappingId, err: err}
		}
		return true, nil

	case mapActionSet:
		lhs, rhs, ok := splitCommandWord(args)
		if !ok || rhs == "" {
			// Listing existing mappings is not supported: there is no multi-line
			// display surface. Mappings() serves that need programmatically.
			return true, mapUsageError("%s requires a key and a replacement, e.g. :%s U <C-r>", name, name)
		}
		if err := e.Map(cmd.modes, lhs, rhs, cmd.noremap); err != nil {
			return true, &EditorError{id: ErrInvalidMappingId, err: err}
		}
		return true, nil
	}

	return false, nil
}

func mapUsageError(format string, args ...any) *EditorError {
	return &EditorError{
		id:  ErrInvalidMappingId,
		err: fmt.Errorf("%w: %s", ErrInvalidMapping, fmt.Sprintf(format, args...)),
	}
}
