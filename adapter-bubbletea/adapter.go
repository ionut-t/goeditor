package bubble_adapter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/core"
)

type Theme struct {
	NormalModeStyle        lipgloss.Style
	InsertModeStyle        lipgloss.Style
	VisualModeStyle        lipgloss.Style
	CommandModeStyle       lipgloss.Style
	StatusLineStyle        lipgloss.Style
	CommandLineStyle       lipgloss.Style
	MessageStyle           lipgloss.Style
	LineNumberStyle        lipgloss.Style
	CurrentLineNumberStyle lipgloss.Style
	SelectionStyle         lipgloss.Style
	ErrorStyle             lipgloss.Style
	HighlighYankStyle      lipgloss.Style
}

var DefaultTheme = Theme{
	NormalModeStyle:        lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")),
	InsertModeStyle:        lipgloss.NewStyle().Background(lipgloss.Color("26")).Foreground(lipgloss.Color("255")),
	VisualModeStyle:        lipgloss.NewStyle().Background(lipgloss.Color("127")).Foreground(lipgloss.Color("255")),
	CommandModeStyle:       lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("255")),
	CommandLineStyle:       lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("255")),
	StatusLineStyle:        lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("255")),
	MessageStyle:           lipgloss.NewStyle().Foreground(lipgloss.Color("34")),
	ErrorStyle:             lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
	LineNumberStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(4).Align(lipgloss.Right),
	CurrentLineNumberStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(4).Align(lipgloss.Right),
	SelectionStyle:         lipgloss.NewStyle().Background(lipgloss.Color("237")),
	HighlighYankStyle:      lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("0")).Bold(true),
}

type Model struct {
	editor          editor.Editor
	viewport        viewport.Model
	width           int
	height          int
	showLineNumbers bool
	showStatusLine  bool
	theme           Theme
	StatusLineFunc  func() string
	err             error
	message         string
	yanked          bool
	disableVimMode  bool
	showMessages    bool
}

type messageMsg string

type errMsg error

type SaveMsg string

type QuitMsg struct{}

type clearMsg struct{}

type yankMsg struct {
	message string
}

type clearYankMsg struct{}

func (m *Model) dispatchClearMsg() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return clearMsg{}
	})
}

func (m *Model) dispatchClearYankMsg() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return clearYankMsg{}
	})
}

type atottoClipboard struct{}

func (c *atottoClipboard) Write(text string) error {
	return clipboard.WriteAll(text)
}

func (c *atottoClipboard) Read() (string, error) {
	return clipboard.ReadAll()
}

func New(width, height int) Model {
	editor := editor.New(&atottoClipboard{})
	vp := viewport.New(width, height-2)

	model := Model{
		editor:          editor,
		viewport:        vp,
		showLineNumbers: true,
		showStatusLine:  true,
		theme:           DefaultTheme,
	}

	model.SetSize(width, height)

	return model
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 2 // Adjust for status and command lines

	// Calculate available width (like in updateViewport)
	lineNumWidth := 0
	if m.showLineNumbers {
		// ... (calculate lineNumWidth based on max lines / relative setting) ...
		// Example simplified calculation:
		maxLineNum := m.editor.GetBuffer().LineCount()
		maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
		lineNumWidth = max(4, maxWidth) + 1
		lineNumWidth = min(lineNumWidth, 10) // Example cap
	}
	availableWidth := m.viewport.Width - lineNumWidth
	if availableWidth <= 0 {
		availableWidth = 1
	}

	// Update editor state
	state := m.editor.GetState()
	state.ViewportWidth = m.viewport.Width
	state.AvailableWidth = availableWidth
	state.ViewportHeight = height - 2
	m.editor.SetState(state)

	m.updateViewport()
}

func (m *Model) SetContent(content []byte) {
	m.editor.SetContent(content)
	m.updateViewport()
}

func (m *Model) WithTheme(theme Theme) {
	m.theme = theme
}

func (m *Model) HideLineNumbers(hide bool) {
	m.showLineNumbers = !hide
}

func (m *Model) ShowRelativeLineNumbers(show bool) {
	if m.disableVimMode {
		return
	}

	m.editor.ShowRelativeLineNumbers(show)
}

func (m *Model) HideStatusLine(hide bool) {
	m.showStatusLine = !hide
}

func (m *Model) ShowMessages(show bool) {
	m.showMessages = show
}

func (m *Model) GetSavedContent() string {
	return m.editor.GetBuffer().GetSavedContent()
}

func (m *Model) GetCurrentContent() string {
	return m.editor.GetBuffer().GetCurrentContent()
}

func (m *Model) HasChanges() bool {
	return m.editor.GetBuffer().IsModified()
}

func (m *Model) GetEditor() editor.Editor {
	return m.editor
}

func (m *Model) DisableVimMode(disable bool) {
	m.disableVimMode = disable
	m.editor.DisableVimMode(disable)
}

// Init initializes the Bubbletea application
func (m Model) Init() tea.Cmd {
	return m.listenForEditorUpdate()
}

// Update processes messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editor.GetState().Quit {
			return m, tea.Quit
		}

		// Handle key message by converting to editor.Key
		key := convertBubbleKey(msg)

		// Process the key in the editor
		err := m.editor.HandleKey(key)
		if err != nil {
			return m, func() tea.Msg {
				return errMsg(err)
			}
		}

		// Update viewport content
		m.updateViewport()

		// Check if editor requested to quit
		if m.editor.GetState().Quit {
			// return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// // Handle window resize
		// m.width = msg.Width
		// m.height = msg.Height

		// // Resize viewport while leaving room for status/command lines
		// m.viewport.Width = msg.Width
		// m.viewport.Height = msg.Height - 2 // Save space for status and command lines

		// // Update editor's viewport size too
		// state := m.editor.GetState()
		// state.ViewportWidth = msg.Width
		// state.ViewportHeight = msg.Height - 2
		// m.editor.SetState(state)

		// // Update viewport content
		// m.updateViewport()

	case messageMsg:
		// The editor signaled that its state (e.g., cleared message) changed.
		// We just need to ensure the UI redraws with the latest state.
		m.message = string(msg)
		m.err = nil
		m.updateViewport()

		return m, m.dispatchClearMsg()

	case yankMsg:
		if m.showMessages {
			m.message = msg.message
		}

		m.err = nil
		m.yanked = true
		m.updateViewport()

		return m, tea.Batch(
			m.dispatchClearMsg(),
			m.dispatchClearYankMsg(),
		)

	case errMsg:
		// The editor signaled that its state (e.g., cleared message) changed.
		// We just need to ensure the UI redraws with the latest state.
		m.message = ""
		m.err = msg
		m.updateViewport()

		return m, m.dispatchClearMsg()

	case clearMsg:
		m.message = ""
		m.err = nil

	case clearYankMsg:
		m.yanked = false
		m.editor.SetNormalMode()
		m.updateViewport()

	}

	cmds = append(cmds, m.listenForEditorUpdate())

	// Handle viewport updates
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	state := m.editor.GetState()

	content := m.viewport.View()

	var commandLine string

	if !m.disableVimMode {
		commandLine = state.CommandLine
	}

	if m.message != "" {
		commandLine = m.theme.MessageStyle.Render(m.message)
	}

	if m.err != nil {
		commandLine = m.theme.ErrorStyle.Render(m.err.Error())
	}

	statusLine := m.getStatusLine()

	paddingWidth := m.width - lipgloss.Width(statusLine)
	if paddingWidth > 0 {
		statusLine += m.theme.StatusLineStyle.Render(strings.Repeat(" ", paddingWidth))
	}

	paddingWidth = m.width - lipgloss.Width(commandLine)
	if paddingWidth > 0 {
		commandLine += m.theme.CommandLineStyle.Render(strings.Repeat(" ", paddingWidth))
	}

	if m.disableVimMode {
		return content
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusLine,
		commandLine,
	)
}

func (m *Model) getStatusLine() string {
	if !m.showStatusLine {
		return ""
	}

	if m.StatusLineFunc != nil {
		return m.StatusLineFunc()
	}

	state := m.editor.GetState()

	// Construct status line
	var statusLine string
	switch state.Mode {
	case editor.NormalMode:
		statusLine = m.theme.NormalModeStyle.Render(" NORMAL ")
	case editor.InsertMode:
		statusLine = m.theme.InsertModeStyle.Render(" INSERT ")
	case editor.VisualMode:
		statusLine = m.theme.VisualModeStyle.Render(" VISUAL ")
	case editor.VisualLineMode:
		statusLine = m.theme.VisualModeStyle.Render(" VISUAL LINE ")
	case editor.CommandMode:
		statusLine = m.theme.CommandModeStyle.Render(" COMMAND ")
	}

	// Add rest of status line info
	cursor := m.editor.GetBuffer().GetCursor()

	cursorInfo := fmt.Sprintf("%d/%d ", cursor.Position.Row+1, cursor.Position.Col+1)

	width := m.width - (lipgloss.Width(cursorInfo) + lipgloss.Width(statusLine))
	gap := strings.Repeat(" ", max(0, width))

	statusLine += m.theme.StatusLineStyle.Render(
		gap + cursorInfo,
	)

	return statusLine
}

func (m *Model) listenForEditorUpdate() tea.Cmd {
	return func() tea.Msg {
		editorChan := m.editor.GetUpdateSignalChan() // Get the channel via interface method
		// Block here waiting for a signal from the editor's Goroutine
		signal := <-editorChan

		switch signal := signal.(type) {
		case editor.MessageSignal:
			_, message := signal.Value()
			if m.showMessages {
				return messageMsg(message)
			}

			return nil

		case editor.ErrorSignal:
			_, err := signal.Value()
			return errMsg(err)

		case editor.YankSignal:
			totalLines, isVisualLine := signal.Value()
			message := ""
			if isVisualLine {
				if totalLines == 1 {
					message = "1 line yanked"
				} else {
					message = fmt.Sprintf("%d lines yanked", totalLines)
				}
			} else {
				message = "selection yanked"
			}

			return yankMsg{message}

		case editor.SaveSignal:
			return SaveMsg(signal.Value())

		case editor.EnterCommandModeSignal:
			return messageMsg("")

		case editor.QuitSignal:
			return QuitMsg{}
		}

		return nil
	}
}

// Convert Bubbletea key to editor.Key
func convertBubbleKey(msg tea.KeyMsg) editor.KeyEvent {
	key := editor.KeyEvent{}

	if len(msg.Runes) > 0 {
		key.Rune = rune(msg.Runes[0])
	}

	// Set modifiers
	if msg.Alt {
		key.Modifiers |= editor.ModAlt
	}

	// Handle special keys
	switch msg.Type {
	case tea.KeyEnter:
		key.Key = editor.KeyEnter
	case tea.KeySpace:
		key.Key = editor.KeySpace
		key.Rune = ' '
	case tea.KeyEsc:
		key.Key = editor.KeyEscape
	case tea.KeyBackspace:
		key.Key = editor.KeyBackspace
	case tea.KeyTab:
		key.Key = editor.KeyTab
		key.Rune = '\t'
	case tea.KeyUp:
		key.Key = editor.KeyUp
	case tea.KeyDown:
		key.Key = editor.KeyDown
	case tea.KeyLeft:
		key.Key = editor.KeyLeft
	case tea.KeyRight:
		key.Key = editor.KeyRight
	case tea.KeyHome:
		key.Key = editor.KeyHome
	case tea.KeyEnd:
		key.Key = editor.KeyEnd
	case tea.KeyDelete:
		key.Key = editor.KeyDelete
	case tea.KeyPgUp:
		key.Key = editor.KeyPageUp
	case tea.KeyPgDown:
		key.Key = editor.KeyPageDown
	}

	return key
}
