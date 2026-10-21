package goeditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetContentEmpty checks that empty content gives one editable empty line.
func TestSetContentEmpty(t *testing.T) {
	for _, content := range []string{"", "\n"} {
		m := New(40, 10)
		m.Focus()
		m.SetContent(content)
		assert.Equal(t, 1, m.GetEditor().GetBuffer().LineCount())
		assert.Empty(t, m.GetCurrentContent())

		m = typeText("ihi")(m)
		assert.Equal(t, "hi", m.GetCurrentContent())
	}
}
