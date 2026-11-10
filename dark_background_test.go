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
