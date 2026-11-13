package goeditor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

// TestNonASCIIKeyInput checks that a key's text reaches the core as whole
// runes, in Insert mode and on the command line, including a key press whose
// text holds several runes (a grapheme cluster, or a Kitty event's
// associated text).
func TestNonASCIIKeyInput(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		want string
	}{
		{"precomposed é", tea.KeyPressMsg{Code: 'é', Text: "é"}, "é"},
		{"CJK", tea.KeyPressMsg{Code: '日', Text: "日"}, "日"},
		{"emoji", tea.KeyPressMsg{Code: '😀', Text: "😀"}, "😀"},
		{"e + combining acute", tea.KeyPressMsg{Code: tea.KeyExtended, Text: "é"}, "é"},
		{"flag", tea.KeyPressMsg{Code: tea.KeyExtended, Text: "🇷🇴"}, "🇷🇴"},
		{"several characters", tea.KeyPressMsg{Code: 'a', Text: "aéb"}, "aéb"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/insert", func(t *testing.T) {
			m := newBottomRowsModel(40, 10, nil)
			m = typeRune('i')(m)
			m, _ = m.Update(tt.key)

			assert.Equal(t, tt.want+"line 1", string(m.editor.GetBuffer().GetLineRunes(0)))
			assert.Equal(t, len([]rune(tt.want)), m.GetCursorPosition().Col)
		})

		t.Run(tt.name+"/command line", func(t *testing.T) {
			m := newBottomRowsModel(40, 10, nil)
			m = typeRune(':')(m)
			m, _ = m.Update(tt.key)

			state := m.editor.GetState()
			assert.Equal(t, ":"+tt.want, state.CommandLine)
			assert.Equal(t, 1+len([]rune(tt.want)), state.CommandCursor, "cursor after the text, prompt included")
			row, _ := m.getCommandLine(state)
			assert.Equal(t, ":"+tt.want, strings.TrimRight(ansi.Strip(row), " "))
		})
	}
}
