package goeditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithDarkBackgroundPicksTheme checks the option selects the theme
// without consulting the terminal.
func TestWithDarkBackgroundPicksTheme(t *testing.T) {
	for _, dark := range []bool{true, false} {
		m := New(40, 10, WithClipboard(&memClipboard{}), WithDarkBackground(dark))
		assert.Equal(t, DefaultTheme(dark).NormalModeStyle.GetBackground(), m.theme.NormalModeStyle.GetBackground())
	}
}

// TestNewDefaultsToDarkBackground checks New assumes a dark background
// instead of asking the terminal, which blocks and steals Bubble Tea's input.
func TestNewDefaultsToDarkBackground(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	assert.Equal(t, DefaultTheme(true).NormalModeStyle.GetBackground(), m.theme.NormalModeStyle.GetBackground())
}
