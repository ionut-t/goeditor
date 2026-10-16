package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// "hello world": h=0 e=1 l=2 l=3 o=4 ' '=5 w=6 o=7 r=8 l=9 d=10

// TestFindForward tests 'f{char}' — move to next occurrence of char on line.
func TestFindForward(t *testing.T) {
	t.Run("moves to first occurrence", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'f', 'o')
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run("from mid-line finds next occurrence", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 'f', 'o') // cursor at col 6; next 'o' is at col 7
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("char not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'f', 'z')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2fo finds second occurrence", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '2', 'f', 'o')
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})
}

// TestFindBackward tests 'F{char}' — move to previous occurrence of char on line.
func TestFindBackward(t *testing.T) {
	t.Run("moves to previous occurrence", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'o') // start at col 10; 'o' at col 7
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("finds occurrence earlier on line", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'l') // 'l' at col 9, 3, 2; nearest backward from 10 is col 9
		assert.Equal(t, Position{0, 9}, cursorPos(e))
	})

	t.Run("char not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'z')
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})

	t.Run("count: 2Fl finds second occurrence backward", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', '2', 'F', 'l') // from col 10; 2nd 'l' backward: col 9, then col 3
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})
}

// TestTillForward tests 't{char}' — move to one before next occurrence.
func TestTillForward(t *testing.T) {
	t.Run("stops one before the char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 't', 'o') // 'o' at col 4; stop at col 3
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})

	t.Run("from mid-line", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'w', 't', 'd') // cursor at col 6; 'd' at col 10; stop at col 9
		assert.Equal(t, Position{0, 9}, cursorPos(e))
	})

	t.Run("char not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 't', 'z')
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("count: 2to stops before second occurrence", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '2', 't', 'o') // 2nd 'o' is at col 7; stop at col 6
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})
}

// TestTillBackward tests 'T{char}' — move to one after previous occurrence.
func TestTillBackward(t *testing.T) {
	t.Run("stops one after the char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'T', 'o') // 'o' at col 7 (searching back from col 10); stop at col 8
		assert.Equal(t, Position{0, 8}, cursorPos(e))
	})

	t.Run("char not found leaves cursor unchanged", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'T', 'z')
		assert.Equal(t, Position{0, 10}, cursorPos(e))
	})
}

// TestRepeatCharSearch tests ';' and ',' — repeat last char search.
// "hello world": h=0 e=1 l=2 l=3 o=4 ' '=5 w=6 o=7 r=8 l=9 d=10
func TestRepeatCharSearch(t *testing.T) {
	t.Run("; repeats last f search forward", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'f', 'o') // → col 4
		keys(e, ';')      // repeat: next 'o' from col 4 → col 7
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run(", reverses last f search to F", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'o') // → col 7
		keys(e, ',')           // reverse F→f: next 'o' forward from col 7 → not found, stays
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("; after F repeats backward", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'o') // → col 7
		keys(e, ';')           // repeat F: previous 'o' from col 7 → col 4
		assert.Equal(t, Position{0, 4}, cursorPos(e))
	})

	t.Run(", after F reverses to f", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'o') // → col 7
		keys(e, ',')           // reversed F→f: next 'o' forward from col 7 → not found
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("; repeats t search, skipping past adjacent char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 't', 'l') // 'l' at col 2; stop at col 1
		assert.Equal(t, Position{0, 1}, cursorPos(e))
		keys(e, ';') // repeat: next 'l' at col 3; stop at col 2
		assert.Equal(t, Position{0, 2}, cursorPos(e))
	})

	t.Run("; repeats T search, skipping past adjacent char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, '$', 'F', 'o') // → col 7
		keys(e, 'T', 'l')      // 'l' at col 3 (searching back from col 7); stop at col 4
		assert.Equal(t, Position{0, 4}, cursorPos(e))
		keys(e, ';') // repeat: prev 'l' at col 2; stop at col 3
		assert.Equal(t, Position{0, 3}, cursorPos(e))
	})
}

// TestCharSearchWithOperators tests operator + char search combinations (df, dt, yf, yt, cf, ct).
func TestCharSearchWithOperators(t *testing.T) {
	t.Run("df deletes to and including char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 'f', 'o') // delete from col 0 to col 4 inclusive → "hello" deleted
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("dt deletes up to but not including char", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'd', 't', 'o') // delete from col 0 to col 3 → "hell" deleted
		assert.Equal(t, "o world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
	})

	t.Run("dF deletes backward to and including found char and cursor char", func(t *testing.T) {
		e := newTestEditor("hello world")
		// cursor at col 10 ('d'); 'w' at col 6; dF deletes cols 6–10 ("world"), leaving "hello "
		keys(e, '$', 'd', 'F', 'w')
		assert.Equal(t, "hello ", content(e))
		assert.Equal(t, Position{0, 6}, cursorPos(e))
	})

	t.Run("dT deletes backward one-after found char up to but not including cursor char", func(t *testing.T) {
		e := newTestEditor("hello world")
		// cursor at col 10 ('d'); 'w' at col 6; T lands at col 7; dT deletes cols 7–9 ("orl"), leaving "hello wd"
		keys(e, '$', 'd', 'T', 'w')
		assert.Equal(t, "hello wd", content(e))
		assert.Equal(t, Position{0, 7}, cursorPos(e))
	})

	t.Run("yf yanks to and including char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 'f', 'o') // yank cols 0–4 inclusive → "hello"
		assert.Equal(t, "hello", cb.content)
		assert.Equal(t, "hello world", content(e)) // unchanged
	})

	t.Run("yt yanks up to but not including char", func(t *testing.T) {
		e, cb := newTestEditorWithClipboard("hello world")
		keys(e, 'y', 't', 'o') // yank cols 0–3 → "hell"
		assert.Equal(t, "hell", cb.content)
	})

	t.Run("cf deletes to char and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 'f', 'o') // delete cols 0–4 inclusive, enter insert
		assert.Equal(t, " world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})

	t.Run("ct deletes till char and enters insert mode", func(t *testing.T) {
		e := newTestEditor("hello world")
		keys(e, 'c', 't', 'o') // delete cols 0–3, enter insert
		assert.Equal(t, "o world", content(e))
		assert.Equal(t, Position{0, 0}, cursorPos(e))
		assertInsertMode(t, e)
	})
}

func TestVisualCharSearchWithCount(t *testing.T) {
	for _, mode := range []rune{'v', 'V'} {
		t.Run(string(mode), func(t *testing.T) {
			e := newTestEditor("a.b.c.d.e")

			keys(e, mode, '3', 'f')
			assert.Equal(t, "3f", e.GetState().CommandLine)

			keys(e, '.')
			assert.Equal(t, 5, cursorPos(e).Col, "3f. goes to the third '.'")
			assert.Nil(t, e.GetState().PendingCount)
		})
	}
}

func TestCharSearchShowsPendingCount(t *testing.T) {
	e := newTestEditor("a.b.c.d.e")

	keys(e, '2', 't')
	assert.Equal(t, "2t", e.GetState().CommandLine)

	keys(e, '.')
	assert.Equal(t, 2, cursorPos(e).Col)
}

// The last f/F/t/T search is editor-wide, as in Vim: a mode switch must not
// make ; and , forget it.
func TestCharSearchRepeatSurvivesModeSwitch(t *testing.T) {
	t.Run("normal, insert, normal", func(t *testing.T) {
		e := newTestEditor("a.b.c.d")
		keys(e, 'f', '.', 'i')
		escape(e)
		keys(e, ';')
		assert.Equal(t, 3, cursorPos(e).Col, "; repeats f. after i<Esc>")
	})

	for _, mode := range []rune{'v', 'V'} {
		t.Run("normal to "+string(mode), func(t *testing.T) {
			e := newTestEditor("a.b.c.d")
			keys(e, 'f', '.', mode, ';')
			assert.Equal(t, 3, cursorPos(e).Col)
			keys(e, ',')
			assert.Equal(t, 1, cursorPos(e).Col)
		})

		t.Run(string(mode)+" to normal", func(t *testing.T) {
			e := newTestEditor("a.b.c.d")
			keys(e, mode, 'f', '.')
			escape(e)
			keys(e, ';')
			assert.Equal(t, 3, cursorPos(e).Col)
		})
	}

	t.Run("operator search, then visual", func(t *testing.T) {
		e := newTestEditor("a.b.c.d")
		keys(e, 'd', 't', '.', 'v', ';')
		assert.Equal(t, ".b.c.d", content(e))
		assert.Equal(t, 1, cursorPos(e).Col, "; repeats t. from the d")
	})
}

func TestCancelledCharSearchKeepsLastSearch(t *testing.T) {
	e := newTestEditor("a.b.c.d")
	keys(e, 'f', '.', 'f')
	escape(e)
	keys(e, ';')
	assert.Equal(t, 3, cursorPos(e).Col, "f<Esc> leaves the last search as f.")

	keys(e, 'v', 'f')
	escape(e) // cancels the search and leaves visual mode
	escape(e)
	keys(e, ';')
	assert.Equal(t, 5, cursorPos(e).Col)
}

// Vim records an f/F/t/T search even when its target isn't found.
func TestFailedCharSearchIsStillRepeated(t *testing.T) {
	e := newTestEditor("a.b\nzz")
	keys(e, 'f', 'z', 'j', '0', ';')
	assert.Equal(t, 1, cursorPos(e).Col, "; repeats fz on the next line")
}
