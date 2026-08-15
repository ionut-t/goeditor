package core

import (
	"fmt"
	"strings"
	"unicode"
)

// Key notation follows Vim's convention: printable characters stand for
// themselves, and anything else is written inside angle brackets — <Esc>, <CR>,
// <C-r>, <leader>. It is the notation accepted by :map and by Editor.Map.

// namedKeys maps the canonical <Name> spelling to its key code. The first entry
// for a given code is the one FormatKeys emits.
var namedKeys = []struct {
	name string
	code KeyCode
}{
	{"CR", KeyEnter},
	{"Enter", KeyEnter},
	{"Return", KeyEnter},
	{"Esc", KeyEscape},
	{"Escape", KeyEscape},
	{"Tab", KeyTab},
	{"BS", KeyBackspace},
	{"Backspace", KeyBackspace},
	{"Space", KeySpace},
	{"Del", KeyDelete},
	{"Delete", KeyDelete},
	{"Insert", KeyInsert},
	{"Up", KeyUp},
	{"Down", KeyDown},
	{"Left", KeyLeft},
	{"Right", KeyRight},
	{"Home", KeyHome},
	{"End", KeyEnd},
	{"PageUp", KeyPageUp},
	{"PageDown", KeyPageDown},
}

// modifierPrefixes maps a single-letter modifier prefix to its bit.
var modifierPrefixes = map[byte]KeyModifiers{
	'C': ModCtrl,
	'c': ModCtrl,
	'A': ModAlt,
	'a': ModAlt,
	'M': ModAlt, // Vim spells Alt as Meta too
	'm': ModAlt,
	'S': ModShift,
	's': ModShift,
}

// DefaultMapLeader is the <leader> expansion when none is configured, matching Vim.
const DefaultMapLeader = `\`

// ParseKeys converts Vim key notation into a sequence of key events.
//
// Recognised forms: printable characters; <Esc>, <CR>/<Enter>, <Tab>, <Space>,
// <BS>, <Del>, <Insert>, arrow keys, <Home>/<End>, <PageUp>/<PageDown>;
// modifiers <C-x>, <A-x>/<M-x>, <S-x>, which may be combined as <C-S-x>; <lt>
// for a literal '<'; <leader>, replaced by the given leader string; and <Nop>,
// which yields no key at all so a mapping can disable a binding.
//
// Events are returned in canonical form, so they compare equal to what arrives
// from the terminal.
func ParseKeys(notation string, leader string) ([]KeyEvent, error) {
	if leader == "" {
		leader = DefaultMapLeader
	}

	var keys []KeyEvent

	runes := []rune(notation)
	for i := 0; i < len(runes); {
		if runes[i] != '<' {
			keys = append(keys, normalizeKey(KeyEvent{Rune: runes[i]}))
			i++
			continue
		}

		end := indexRune(runes, '>', i+1)
		if end < 0 {
			// An unmatched '<' is a literal '<', as in Vim.
			keys = append(keys, normalizeKey(KeyEvent{Rune: '<'}))
			i++
			continue
		}

		name := string(runes[i+1 : end])

		switch {
		case strings.EqualFold(name, "Nop"):
			// Explicitly nothing — used to disable a key.
		case strings.EqualFold(name, "lt"):
			keys = append(keys, KeyEvent{Rune: '<'})
		case strings.EqualFold(name, "leader"):
			leaderKeys, err := ParseKeys(leader, DefaultMapLeader)
			if err != nil {
				return nil, fmt.Errorf("invalid mapleader %q: %w", leader, err)
			}
			keys = append(keys, leaderKeys...)
		default:
			key, err := parseBracketedKey(name)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		}

		i = end + 1
	}

	return keys, nil
}

// parseBracketedKey parses the inside of a <...> sequence: any number of
// modifier prefixes followed by either a named key or a single character.
func parseBracketedKey(name string) (KeyEvent, error) {
	var mods KeyModifiers

	for len(name) > 2 && name[1] == '-' {
		mod, ok := modifierPrefixes[name[0]]
		if !ok {
			break
		}
		mods |= mod
		name = name[2:]
	}

	for _, named := range namedKeys {
		if strings.EqualFold(name, named.name) {
			key := KeyEvent{Key: named.code, Modifiers: mods}
			if named.code == KeySpace {
				key.Rune = ' '
			}
			if named.code == KeyTab {
				key.Rune = '\t'
			}
			return normalizeKey(key), nil
		}
	}

	if runes := []rune(name); len(runes) == 1 {
		// <S-a> is just 'A'; fold it into the rune so it matches terminal input,
		// which reports shifted letters as uppercase with no modifier.
		r := runes[0]
		if mods&ModShift != 0 && unicode.IsLetter(r) {
			r = unicode.ToUpper(r)
			mods &^= ModShift
		}
		return normalizeKey(KeyEvent{Rune: r, Modifiers: mods}), nil
	}

	return KeyEvent{}, fmt.Errorf("%w: unknown key <%s>", ErrInvalidMapping, name)
}

// FormatKeys renders a key sequence back into Vim notation. It round-trips with
// ParseKeys, though not necessarily character-for-character: aliases collapse to
// their canonical spelling (<Enter> becomes <CR>).
func FormatKeys(keys []KeyEvent) string {
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(formatKey(key))
	}
	return b.String()
}

func formatKey(key KeyEvent) string {
	var mods string
	if key.Modifiers&ModCtrl != 0 {
		mods += "C-"
	}
	if key.Modifiers&ModAlt != 0 {
		mods += "A-"
	}
	if key.Modifiers&ModShift != 0 {
		mods += "S-"
	}

	for _, named := range namedKeys {
		if key.Key == named.code && key.Key != KeyUnknown {
			return "<" + mods + named.name + ">"
		}
	}

	if key.Rune == 0 {
		return ""
	}

	if mods == "" {
		if key.Rune == '<' {
			return "<lt>"
		}
		return string(key.Rune)
	}

	return "<" + mods + string(key.Rune) + ">"
}

func indexRune(runes []rune, target rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == target {
			return i
		}
	}
	return -1
}
