package core

import (
	"fmt"
	"strings"
	"unicode"
)

// --- KeyCode, KeyModifiers, Key ---

// KeyCode represents non-character keys
type KeyCode int

const (
	KeyUnknown KeyCode = iota
	KeyEnter
	KeyTab
	KeyBackspace
	KeyEscape
	KeySpace

	// Arrow keys
	KeyUp
	KeyDown
	KeyLeft
	KeyRight

	// Navigation keys
	KeyHome // Often maps to ^ or 0
	KeyEnd  // Often maps to $
	KeyPageUp
	KeyPageDown

	// Editing keys
	KeyDelete
	KeyInsert

	// Ctrl+letter shortcuts.
	//
	// Deprecated: Ctrl+<letter> is now represented canonically as
	// KeyEvent{Rune: <letter>, Modifiers: ModCtrl}. These constants are still
	// accepted on input — normalizeKey converts them — but new code should use
	// the canonical form so that every Ctrl combination is representable.
	KeyCtrlD
	KeyCtrlU
	KeyCtrlN
	KeyCtrlP
	KeyCtrlY
	KeyCtrlE
)

// legacyCtrlKeys maps the deprecated Ctrl+<letter> key codes to their letter.
var legacyCtrlKeys = map[KeyCode]rune{
	KeyCtrlD: 'd',
	KeyCtrlU: 'u',
	KeyCtrlN: 'n',
	KeyCtrlP: 'p',
	KeyCtrlY: 'y',
	KeyCtrlE: 'e',
}

// Ctrl builds the canonical KeyEvent for Ctrl+<letter>.
func Ctrl(r rune) KeyEvent {
	return KeyEvent{Rune: unicode.ToLower(r), Modifiers: ModCtrl}
}

// IsCtrl reports whether k is Ctrl+r in canonical form.
func (k KeyEvent) IsCtrl(r rune) bool {
	return k.Modifiers&ModCtrl != 0 && k.Rune == unicode.ToLower(r)
}

// normalizeKey rewrites a KeyEvent into its canonical form so that events
// originating from different sources (the Bubble Tea bridge, the key-notation
// parser, consumer code using the deprecated KeyCtrl* constants) compare equal.
//
// Ctrl+<letter> canonicalises to {Rune: <lowercase letter>, Modifiers: ModCtrl}
// with Key cleared, which is what makes KeyEvent usable as a map/slice key in
// the mapping table.
func normalizeKey(key KeyEvent) KeyEvent {
	if r, ok := legacyCtrlKeys[key.Key]; ok {
		return KeyEvent{Rune: r, Modifiers: key.Modifiers | ModCtrl}
	}

	// Only Ctrl+<letter> is canonicalised. Combinations like Ctrl+Space carry a
	// meaningful KeyCode that must survive.
	if key.Modifiers&ModCtrl != 0 && unicode.IsLetter(key.Rune) {
		key.Rune = unicode.ToLower(key.Rune)
		key.Key = KeyUnknown
	}

	return key
}

// KeyModifiers represents modifier keys held during a keystroke
type KeyModifiers uint8

const (
	ModNone KeyModifiers = 0
	ModCtrl KeyModifiers = 1 << iota
	ModAlt
	ModShift
)

// KeyEvent represents a keyboard input event
type KeyEvent struct {
	Rune      rune
	Key       KeyCode
	Modifiers KeyModifiers
}

// String returns a string representation of a Key (Refined for clarity)
func (k KeyEvent) String() string {
	var parts []string

	// Modifiers first
	if k.Modifiers&ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if k.Modifiers&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if k.Modifiers&ModShift != 0 {
		parts = append(parts, "Shift")
	}

	// Key representation
	if k.Rune != 0 {
		parts = append(parts, string(k.Rune))
	} else {
		switch k.Key {
		case KeyEnter:
			parts = append(parts, "Enter")
		case KeyTab:
			parts = append(parts, "Tab")
		case KeyBackspace:
			parts = append(parts, "Backspace")
		case KeyEscape:
			parts = append(parts, "Escape")
		case KeySpace:
			parts = append(parts, "Space")
		case KeyUp:
			parts = append(parts, "Up")
		case KeyDown:
			parts = append(parts, "Down")
		case KeyLeft:
			parts = append(parts, "Left")
		case KeyRight:
			parts = append(parts, "Right")
		case KeyHome:
			parts = append(parts, "Home")
		case KeyEnd:
			parts = append(parts, "End")
		case KeyPageUp:
			parts = append(parts, "PageUp")
		case KeyPageDown:
			parts = append(parts, "PageDown")
		case KeyDelete:
			parts = append(parts, "Delete")
		case KeyInsert:
			parts = append(parts, "Insert")
		case KeyUnknown:
			parts = append(parts, "Unknown")
		default:
			parts = append(parts, fmt.Sprintf("SpecialKey(%d)", k.Key))
		}
	}

	return strings.Join(parts, "+")
}
