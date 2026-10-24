package goeditor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGxSendsOpenMsg checks that gx reaches the host as an OpenMsg.
func TestGxSendsOpenMsg(t *testing.T) {
	m := New(40, 10, WithClipboard(&memClipboard{}))
	m.Focus()
	m.SetContent("see https://example.com/a.")
	m = typeText("w")(m)
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		<-m.GetEditor().GetUpdateSignalChan()
	}

	m = typeText("gx")(m)

	var msg any
	for len(m.GetEditor().GetUpdateSignalChan()) > 0 {
		if msg = m.listenForEditorUpdate()(); msg != (commandMsg{}) {
			break
		}
	}
	assert.Equal(t, OpenMsg{Target: "https://example.com/a"}, msg)
}

func TestOpenWithSystemEmptyTarget(t *testing.T) {
	assert.ErrorIs(t, OpenWithSystem(""), ErrNothingToOpen)
}

// TestOpenCommand checks the opener run on each OS. The target is always one
// argument, with no shell in between, and never starts with '-' where the
// opener would read it as a flag.
func TestOpenCommand(t *testing.T) {
	tests := []struct {
		name   string
		goos   string
		target string
		want   []string
	}{
		{"macOS url", "darwin", "https://x.com/a?b=1#c", []string{"open", "https://x.com/a?b=1#c"}},
		{"macOS flag-like name", "darwin", "--help.txt", []string{"open", "./--help.txt"}},
		{"linux url", "linux", "https://x.com", []string{"xdg-open", "https://x.com"}},
		{"linux flag-like name", "linux", "-a", []string{"xdg-open", "./-a"}},
		{"bsd uses xdg-open", "freebsd", "x.txt", []string{"xdg-open", "x.txt"}},
		{"shell characters stay literal", "linux", "a;rm -rf ~$(id)`x` b.txt", []string{"xdg-open", "a;rm -rf ~$(id)`x` b.txt"}},
		{"windows", "windows", "https://x.com", []string{"rundll32", "url.dll,FileProtocolHandler", "https://x.com"}},
		{"windows has no flag problem", "windows", "-a", []string{"rundll32", "url.dll,FileProtocolHandler", "-a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, openCommand(tt.goos, tt.target).Args)
		})
	}
}
