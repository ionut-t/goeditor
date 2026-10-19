package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Visual mode (character-wise) ---

// TestVisualModeEnterExit tests entering and exiting character-wise visual mode.
func TestVisualModeEnterExit(t *testing.T) {
	t.Run("v enters visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		assert.True(t, e.IsVisualMode())
	})

	t.Run("Escape exits visual mode to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v while in visual mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'v')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V switches from visual to visual-line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'V')
		assert.True(t, e.IsVisualLineMode())
	})
}

// TestVisualModeDelete tests 'd'/'x' in character-wise visual mode.
func TestVisualModeDelete(t *testing.T) {
	t.Run("d deletes single selected char", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', 'd')
		assert.Equal(t, "ello", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v+l+l then d deletes 3 chars", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'd')
		assert.Equal(t, "lo world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+$ then d deletes to end of line", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'v', '$', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("v+w then d deletes up to and including first char of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w', 'd')
		// w is a plain motion in visual mode: cursor lands on 'w' of "world" (col 6),
		// so the inclusive selection covers "hello w" (cols 0–6).
		assert.Equal(t, "orld", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeDeleteMultiLine tests multi-line deletion in character-wise visual mode.
func TestVisualModeDeleteMultiLine(t *testing.T) {
	t.Run("v+j then d deletes from cursor to same col on next line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'v', 'j', 'd')
		// selection: {0,0}→{1,0}; deletes "first" from row 0 and 's' from row 1, then merges
		assert.Equal(t, "econd", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualModeYank tests 'y' in character-wise visual mode.
func TestVisualModeYank(t *testing.T) {
	t.Run("y yanks selected text to clipboard", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'y') // select "hello"
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, "hello world", content(e)) // content unchanged
	})

	t.Run("y with $ selects to end of line", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello")
		keys(e, 'v', '$', 'y')
		assert.Equal(t, "hello", cb.content)
	})

	t.Run("y stays in visual mode when the copy fails", func(t *testing.T) {
		e := newTestEditor("abcdef") // no clipboard
		keys(e, 'v', 'l')
		assert.NotNil(t, e.HandleKey(KeyEvent{Rune: 'y'}))
		assert.True(t, e.IsVisualMode())
	})
}

// TestVisualYankEndsVisual checks that y in either visual mode returns to
// normal mode with the cursor where Vim leaves it: the start of a charwise
// selection; for a linewise one, column 0 of the anchor's line unless the
// cursor was above the anchor, where it stays.
func TestVisualYankEndsVisual(t *testing.T) {
	const text = "abcdef\nabcdef\nabcdef"
	tests := []struct {
		keys string
		want Position
	}{
		{"vly", Position{0, 0}},
		{"lllvhhy", Position{0, 1}},
		{"lllvy", Position{0, 3}},
		{"lllvjhy", Position{0, 3}},
		{"jlllvkly", Position{0, 4}},
		{"lllvhhyj", Position{1, 1}}, // j keeps the column y moved to
		{"lllVy", Position{0, 0}},
		{"lllVhy", Position{0, 0}},
		{"lllVly", Position{0, 0}},
		{"lllVjy", Position{0, 0}},
		{"lljVjy", Position{1, 0}},
		{"jjllVky", Position{1, 2}},
		{"jlllVkhy", Position{0, 2}},
		{"jjlllVkkly", Position{0, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.keys, func(t *testing.T) {
			e, _ := newTestEditorWithClipboard(text)
			typeKeys(t, e, tt.keys)
			assert.True(t, e.IsNormalMode())
			assert.Equal(t, tt.want, cursorPos(e))
		})
	}

	t.Run("the yanked text stays highlighted", func(t *testing.T) {
		tests := []struct {
			keys string
			want SelectionType
			on   []Position // highlighted
			off  []Position // not highlighted
		}{
			{"lvly", SelectionCharacter, []Position{{0, 1}, {0, 2}}, []Position{{0, 0}, {0, 3}}},
			{"lllvjhy", SelectionCharacter, []Position{{0, 3}, {1, 2}}, []Position{{0, 2}, {1, 3}}},
			{"jVjy", SelectionLine, []Position{{1, 0}, {2, 5}}, []Position{{0, 0}}},
			{"jjVky", SelectionLine, []Position{{1, 0}, {2, 5}}, []Position{{0, 0}}},
		}
		for _, tt := range tests {
			t.Run(tt.keys, func(t *testing.T) {
				e, _ := newTestEditorWithClipboard(text)
				typeKeys(t, e, tt.keys)
				assert.Equal(t, tt.want, e.GetState().YankSelection)
				for _, p := range tt.on {
					assert.Equal(t, tt.want, e.GetSelectionStatus(p), "%v", p)
				}
				for _, p := range tt.off {
					assert.Equal(t, SelectionNone, e.GetSelectionStatus(p), "%v", p)
				}
			})
		}
	})

	t.Run("a new selection is not taken for the last yank", func(t *testing.T) {
		for _, yank := range []string{"yy", "Vy", "vly"} {
			e, cb := newTestEditorWithClipboard(text)
			typeKeys(t, e, yank+"vly")
			assert.Equal(t, "ab", cb.content, yank)
		}
	})

	t.Run("keys after y act in normal mode", func(t *testing.T) {
		// What a mapping such as Q → vlyx types in one go: x deletes one
		// character at the cursor, not the selection.
		e, cb := newTestEditorWithClipboard("abcdef")
		typeKeys(t, e, "vlyx")
		assert.Equal(t, "ab", cb.content)
		assert.Equal(t, "bcdef", content(e))
	})
}

// TestVisualModeChange tests 'c' in character-wise visual mode.
func TestVisualModeChange(t *testing.T) {
	t.Run("c deletes selection and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c') // select "hello"
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c replaces selection", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'l', 'l', 'l', 'l', 'c')
		keys(e, 'h', 'i')
		assert.Equal(t, "hi world", content(e))
	})
}

// TestVisualModeTextObjects tests text object selections in character-wise visual mode.
func TestVisualModeTextObjects(t *testing.T) {
	// viw / vaw

	t.Run("viw selects inside word from start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("viw selects inside word from mid-word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'v', 'i', 'w', 'd')
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw selects around word including trailing space", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'a', 'w', 'd')
		assert.Equal(t, "world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vaw on middle word includes leading space", func(t *testing.T) {
		e := newTestEditor("one two three")
		keys(e, 'w', 'v', 'a', 'w', 'd')
		assert.Equal(t, "one three", content(e))
	})

	// vip / vap

	t.Run("vip switches to visual line mode", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("vip then d deletes paragraph", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\nfoo")
		keys(e, 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vap then d deletes paragraph and surrounding blanks", func(t *testing.T) {
		e := newTestEditor("hello\nworld\n\n\nfoo")
		keys(e, 'v', 'a', 'p', 'd')
		assert.Equal(t, "foo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip from mid-paragraph selects whole paragraph", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree\n\nfoo")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "\nfoo", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("vip on blank line selects just the blank line", func(t *testing.T) {
		e := newTestEditor("hello\n\nworld")
		keys(e, 'j', 'v', 'i', 'p', 'd')
		assert.Equal(t, "hello\nworld", content(e))
	})
}

// --- Visual line mode ---

// TestVisualLineModeEnterExit tests entering and exiting visual line mode.
func TestVisualLineModeEnterExit(t *testing.T) {
	t.Run("V enters visual line mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		assert.True(t, e.IsVisualLineMode())
	})

	t.Run("Escape exits to normal mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V')
		escape(e)
		assert.True(t, e.IsNormalMode())
	})

	t.Run("V while in visual-line mode exits to normal", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'V')
		assert.True(t, e.IsNormalMode())
	})

	t.Run("v switches from visual-line to character-wise visual mode", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'v')
		assert.True(t, e.IsVisualMode())
	})
}

// TestVisualLineModeDelete tests 'd' in visual line mode.
func TestVisualLineModeDelete(t *testing.T) {
	t.Run("d deletes current line (single line buffer)", func(t *testing.T) {
		e := newTestEditor("hello")
		keys(e, 'V', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assert.True(t, e.IsNormalMode())
	})

	t.Run("d deletes current line (multi-line buffer)", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'd')
		assert.Equal(t, "second\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j then d deletes two lines", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'd')
		assert.Equal(t, "third", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("V+j+j then d deletes three lines", func(t *testing.T) {
		e := newTestEditor("one\ntwo\nthree")
		keys(e, 'V', 'j', 'j', 'd')
		assert.Equal(t, "", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("d from last line moves cursor to previous line", func(t *testing.T) {
		e := newTestEditor("first\nsecond")
		keys(e, 'j', 'V', 'd')
		assert.Equal(t, "first", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
}

// TestVisualLineModeYank tests 'y' in visual line mode.
func TestVisualLineModeYank(t *testing.T) {
	t.Run("y yanks current line with trailing newline", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello\nworld")
		keys(e, 'V', 'y')
		assert.Equal(t, "hello\n", cb.content)
		assert.Equal(t, "hello\nworld", content(e)) // unchanged
	})

	t.Run("V+j then y yanks two lines", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("one\ntwo\nthree")
		keys(e, 'V', 'j', 'y')
		assert.Equal(t, "one\ntwo\n", cb.content)
		assert.Equal(t, "one\ntwo\nthree", content(e)) // unchanged
	})
}

// TestVisualLineModeChange tests 'c' in visual line mode.
func TestVisualLineModeChange(t *testing.T) {
	t.Run("c empties selected lines to one and enters insert mode", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		assert.Equal(t, "\nthird", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("typing after c fills the emptied line", func(t *testing.T) {
		e := newTestEditor("first\nsecond\nthird")
		keys(e, 'V', 'j', 'c')
		keys(e, 'n', 'e', 'w')
		assert.Equal(t, "new\nthird", content(e))
	})
}

// TestVisualModeMovementSequences tests that navigation keys work after shared-motion keys.
func TestVisualModeMovementSequences(t *testing.T) {
	t.Run("v+j+l: j then l both move cursor", func(t *testing.T) {
		e := newTestEditor("abcd\nefgh")
		keys(e, 'v', 'j') // enter visual, move down
		assert.Equal(t, Position{1, 0}, cursorPos(e))
		keys(e, 'l') // move right
		assert.Equal(t, Position{1, 1}, cursorPos(e))
	})
	t.Run("v+w: word forward moves cursor to start of next word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'w')
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
	t.Run("v+w+w: second w advances to start of second next word", func(t *testing.T) {
		e := newTestEditor("hello world foo")
		keys(e, 'v', 'w', 'w')
		assert.Equal(t, Position{0, 12}, cursorPos(e))
	})
	t.Run("v+e: word end moves to last char of word", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'v', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("v+b: word backward from middle moves to word start", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'v', 'b') // cursor at col 3, then v+b
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})
	t.Run("V+j: j in visual line mode extends selection", func(t *testing.T) {
		e := newTestEditor("line1\nline2\nline3")
		keys(e, 'V', 'j', 'd') // visual line, extend down, delete 2 lines
		assert.Equal(t, "line3", content(e))
	})
	t.Run("V+w: w moves cursor forward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world\nfoo")
		keys(e, 'V', 'w') // enter visual line, move forward one word
		assert.Equal(t, Position{0, 6}, cursorPos(e))
		assert.True(t, e.IsVisualLineMode())
	})
	t.Run("V+e: e moves cursor to word end in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'V', 'e')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})
	t.Run("V+b: b moves cursor word backward in visual line mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'l', 'V', 'b') // cursor at col 8 (mid "world"), then V+b → start of "world"
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

func TestVisualGGWithCountGoesToLine(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo\nthree\nfour\nfive\nsix")

			keys(e, mode, '4', 'g', 'g')

			assert.Equal(t, 3, cursorPos(e).Row)
			assert.Nil(t, e.GetState().PendingCount)
		})
	}
}

func TestVisualGWithCountGoesToLine(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo\nthree\nfour\nfive\nsix")

			keys(e, mode, '3', 'G')
			assert.Equal(t, 2, cursorPos(e).Row)
			assert.Nil(t, e.GetState().PendingCount)

			keys(e, 'G')
			assert.Equal(t, 5, cursorPos(e).Row, "G with no count goes to the last line")
		})
	}
}

func TestVisualEnterMovesDown(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\n  two\n    three\n four\nfive")
			setCursorRow(e, 1)

			keys(e, mode)
			enter(e)
			assert.Equal(t, Position{2, 4}, cursorPos(e), "no count moves down one line")

			keys(e, '2')
			enter(e)
			assert.Equal(t, Position{4, 0}, cursorPos(e), "count moves down count lines")
			assert.Nil(t, e.GetState().PendingCount)
			assert.Equal(t, Position{Row: 1, Col: 0}, e.GetState().VisualStart)
		})
	}
}

func TestPrefixDoesNotSurviveModeSwitch(t *testing.T) {
	e := newTestEditor("one\ntwo\nthree\nfour")
	setCursorRow(e, 2)

	keys(e, 'g')
	e.SetInsertMode()
	escape(e)
	keys(e, 'g')

	assert.Equal(t, 2, cursorPos(e).Row, "the g typed before the switch must not complete a gg")
}

func TestVisualTextObjectKeyIsNotAPrefix(t *testing.T) {
	e := newTestEditor("one\ntwo three\nfour")
	setCursorRow(e, 1)

	keys(e, 'v', 'i', 'g', 'w')

	assert.Equal(t, Position{Row: 1, Col: 4}, cursorPos(e), "g after i is the (unknown) text object, so w is a plain motion")
	assert.Equal(t, Position{Row: 1, Col: 0}, e.GetState().VisualStart)
}

func TestPrefixShowsPendingCount(t *testing.T) {
	e := newTestEditor("one\ntwo")
	keys(e, '1', '0', 'g')
	assert.Equal(t, "10g", e.GetState().CommandLine)
	escape(e)

	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("one\ntwo")
			keys(e, mode, '1', '0', 'z')
			assert.Equal(t, "10z", e.GetState().CommandLine)
		})
	}
}

// Re-entering visual mode (e.g. a host refocusing) must not reset the anchor.
func TestSetVisualModeWhileInVisualKeepsSelection(t *testing.T) {
	e := newTestEditor("abc")
	keys(e, 'v', 'l')
	e.SetVisualMode()
	assert.True(t, e.IsVisualMode())
	assert.Equal(t, Position{0, 0}, e.GetState().VisualStart)
	assert.Equal(t, Position{0, 1}, cursorPos(e))
}
