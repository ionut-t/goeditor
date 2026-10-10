// Layout shows how the status line and command line can be arranged below the
// editor, and how each setting gives its row back to the editor when it isn't
// needed. Every setting can be toggled while the program runs; the header
// shows what is currently on.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	editor "github.com/ionut-t/goeditor"
)

const messageDuration = 3 * time.Second

const intro = `# Layout

The two rows under the editor are the status line and the command line.
The header's keys switch them on and off while you type. Whatever they
give up goes to the editor; watch the ruler at the bottom of this text.

## The keys

^G  Shared row. The command line gives up its own row and shares the
    status line's, like Vim's cmdheight=0. The status line shows until
    you type ':', search with '/', or a message arrives, then comes back.
^T  Status line. Hidden, its row goes to the editor; the command line
    keeps its row.
^K  Command mode. Turning it off alone keeps the command line row,
    because '/' search still needs it.
^L  Search mode. Once command and search are both off the command line
    row goes away, and messages appear on the status line instead.
^X  Vim mode. Off, the editor is a plain text area and both rows go.

Every toggle reports itself as a message, so you can see where messages
end up in each layout. With the status line hidden and command and
search off there is nowhere to show them.

## Try

1. Press ^G, then type :w and Esc to watch the bottom row change hands.
2. Press ^K then ^L; the command line row disappears.
3. Press ^T as well; now every row belongs to the editor.

Press ^C to quit.

## Ruler

Scroll down here and toggle the rows: the line count shifts with them.
`

// sampleContent adds a numbered ruler to the intro, long enough to fill any
// terminal, so each row the editor gains or loses is visible.
func sampleContent() string {
	var b strings.Builder
	b.WriteString(intro)
	for i := 1; i <= 200; i++ {
		fmt.Fprintf(&b, "ruler %03d\n", i)
	}
	return b.String()
}

type toggle struct {
	key   string
	label string
	on    bool
}

type Model struct {
	editor editor.Model
	width  int
	height int

	statusLine    toggle
	sharedRow     toggle
	commandMode   toggle
	searchMode    toggle
	vimMode       toggle
	headerStyle   lipgloss.Style
	onStyle       lipgloss.Style
	offStyle      lipgloss.Style
	keyStyle      lipgloss.Style
	separatorText string
}

func (m Model) Init() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.editor.SetSize(m.width, m.editorHeight())
		return m, nil

	case tea.KeyPressMsg:
		// Handle the toggles here so the editor never sees these keys.
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case m.statusLine.key:
			m.statusLine.on = !m.statusLine.on
			m.editor.HideStatusLine(!m.statusLine.on)
			return m, m.announce(m.statusLine)

		case m.sharedRow.key:
			m.sharedRow.on = !m.sharedRow.on
			height := 1
			if m.sharedRow.on {
				height = 0
			}
			m.editor.SetCommandLineHeight(height)
			return m, m.announce(m.sharedRow)

		case m.commandMode.key:
			m.commandMode.on = !m.commandMode.on
			m.editor.DisableCommandMode(!m.commandMode.on)
			return m, m.announce(m.commandMode)

		case m.searchMode.key:
			m.searchMode.on = !m.searchMode.on
			m.editor.DisableSearchMode(!m.searchMode.on)
			return m, m.announce(m.searchMode)

		case m.vimMode.key:
			m.vimMode.on = !m.vimMode.on
			m.editor.DisableVimMode(!m.vimMode.on)
			return m, m.announce(m.vimMode)
		}

	case editor.ErrorMsg:
		return m, m.editor.DispatchError(msg.Error, messageDuration)

	case editor.YankMsg:
		return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes yanked", len(msg.Content)), messageDuration)

	case editor.SearchResultsMsg:
		if len(msg.Positions) == 0 {
			return m, m.editor.DispatchError(errors.New("no search results"), messageDuration)
		}

	case editor.SaveMsg:
		return m, m.editor.DispatchMessage("nothing to save in this example", messageDuration)

	case editor.QuitMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

// announce reports a toggle through the editor, which shows where messages
// end up for the current layout: the command line, the status line or, with
// both rows gone, nowhere.
func (m *Model) announce(t toggle) tea.Cmd {
	state := "off"
	if t.on {
		state = "on"
	}
	return m.editor.DispatchMessage(fmt.Sprintf("%s %s", t.label, state), messageDuration)
}

func (m Model) editorHeight() int {
	return max(0, m.height-1)
}

func (m Model) header() string {
	toggles := []toggle{m.statusLine, m.sharedRow, m.commandMode, m.searchMode, m.vimMode}

	parts := make([]string, 0, len(toggles))
	for _, t := range toggles {
		style := m.offStyle
		if t.on {
			style = m.onStyle
		}
		hint := "^" + strings.ToUpper(strings.TrimPrefix(t.key, "ctrl+"))
		parts = append(parts, m.keyStyle.Render(hint)+" "+style.Render(t.label))
	}

	// The editor is sized for exactly one header row, so never let it wrap.
	return ansi.Truncate(m.headerStyle.Render(strings.Join(parts, m.separatorText)), m.width, "…")
}

func (m Model) View() tea.View {
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, m.header(), m.editor.View()))
	v.AltScreen = true
	return v
}

func main() {
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	lightDark := lipgloss.LightDark(isDark)

	textEditor := editor.New(80, 20)
	textEditor.Focus()
	textEditor.SetCursorMode(editor.CursorBlink)
	textEditor.SetLanguage("markdown", languageTheme(isDark))
	textEditor.SetContent(sampleContent())

	m := Model{
		editor:      textEditor,
		statusLine:  toggle{key: "ctrl+t", label: "status", on: true},
		sharedRow:   toggle{key: "ctrl+g", label: "shared row", on: false},
		commandMode: toggle{key: "ctrl+k", label: "command", on: true},
		searchMode:  toggle{key: "ctrl+l", label: "search", on: true},
		vimMode:     toggle{key: "ctrl+x", label: "vim", on: true},

		headerStyle:   lipgloss.NewStyle().Padding(0, 1),
		keyStyle:      lipgloss.NewStyle().Bold(true),
		onStyle:       lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("#1e7f3c"), lipgloss.Color("#7ee2a8"))),
		offStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Strikethrough(true),
		separatorText: " · ",
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
	}
}

func languageTheme(isDark bool) string {
	if isDark {
		return "catppuccin-mocha"
	}

	return "catppuccin-latte"
}
