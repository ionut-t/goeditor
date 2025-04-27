package core

import (
	"fmt"
	"strings"
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
)

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
