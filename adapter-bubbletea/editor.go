package adapter_bubbletea

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/goeditor/adapter-bubbletea/highlighter"
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
	HighlightYankStyle     lipgloss.Style
	PlaceholderStyle       lipgloss.Style
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
	HighlightYankStyle:     lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.Color("0")).Bold(true),
	PlaceholderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
}

type cursorBlinkMsg struct{}
type cursorBlinkCanceledMsg struct{}
type resumeBlinkCycleMsg struct{}

type CursorMode int

const (
	CursorBlink CursorMode = iota
	CursorSteady
)

const cursorBlinkInterval = 500 * time.Millisecond
const cursorActivityResetDelay = 250 * time.Millisecond

type cursorBlinkContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Model struct {
	editor                  editor.Editor
	viewport                viewport.Model
	width                   int
	height                  int
	showLineNumbers         bool
	showTildeIndicator      bool
	showStatusLine          bool
	theme                   Theme
	StatusLineFunc          func() string
	err                     error
	message                 string
	yanked                  bool
	disableVimMode          bool
	showMessages            bool
	fullVisualLayoutHeight  int              // Total number of visual lines in the entire buffer
	cursorAbsoluteVisualRow int              // Cursor's current row index in the full visual layout
	currentVisualTopLine    int              // Top line of the current visual slice
	visualLayoutCache       []VisualLineInfo // Cache of visual line information for the current slice
	clampedCursorLogicalCol int              // Clamped cursor column in the current visual slice
	highlightedWords        map[string]lipgloss.Style
	isFocused               bool
	placeholder             string
	cursorMode              CursorMode
	cursorVisible           bool
	cursorBlinkContext      *cursorBlinkContext
	highlighter             *highlighter.Highlighter
	language                string
	highlighterTheme        string
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

type RenameMsg struct {
	FileName string
}

type DeleteFileMsg struct{}

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

type clipboardImpl struct{}

func (c *clipboardImpl) Write(text string) error {
	return clipboard.WriteAll(text)
}

func (c *clipboardImpl) Read() (string, error) {
	return clipboard.ReadAll()
}

func New(width, height int) Model {
	editor := editor.New(&clipboardImpl{})
	vp := viewport.New(width, height-2)

	m := Model{
		editor:           editor,
		viewport:         vp,
		showLineNumbers:  true,
		showStatusLine:   true,
		theme:            DefaultTheme,
		highlightedWords: make(map[string]lipgloss.Style),
		cursorMode:       CursorSteady,
		cursorVisible:    true,
		cursorBlinkContext: &cursorBlinkContext{
			ctx: context.Background(),
		},
	}

	m.SetSize(width, height)

	return m
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 2

	lineNumWidth := 0
	if m.showLineNumbers {
		maxLineNum := m.editor.GetBuffer().LineCount()
		maxWidth := len(strconv.Itoa(max(1, maxLineNum)))
		lineNumWidth = max(4, maxWidth) + 1
		lineNumWidth = min(lineNumWidth, 10)
	}
	availableWidth := m.viewport.Width - lineNumWidth
	if availableWidth <= 0 {
		availableWidth = 1
	}

	state := m.editor.GetState()
	state.ViewportWidth = m.viewport.Width
	state.AvailableWidth = availableWidth
	state.ViewportHeight = height - 2
	m.editor.SetState(state)

	if m.fullVisualLayoutHeight > 0 {
		if m.cursorAbsoluteVisualRow < m.currentVisualTopLine {
			m.currentVisualTopLine = m.cursorAbsoluteVisualRow
		} else if m.cursorAbsoluteVisualRow >= m.currentVisualTopLine+m.viewport.Height {
			m.currentVisualTopLine = m.cursorAbsoluteVisualRow - m.viewport.Height + 1
		}

		maxPossibleTopLine := 0
		if m.fullVisualLayoutHeight > m.viewport.Height {
			maxPossibleTopLine = m.fullVisualLayoutHeight - m.viewport.Height
		}
		if m.currentVisualTopLine > maxPossibleTopLine {
			m.currentVisualTopLine = maxPossibleTopLine
		}
		if m.currentVisualTopLine < 0 {
			m.currentVisualTopLine = 0
		}
	} else {
		m.currentVisualTopLine = 0
	}

	m.viewport.YOffset = 0
}

// SetBytes sets the content of the editor.
func (m *Model) SetBytes(content []byte) {
	if len(content) == 0 {
		content = []byte("\n")
	}
	m.editor.SetContent(content)
	m.handleContentChange()
}

// SetContent sets the content of the editor from a string.
func (m *Model) SetContent(content string) {
	m.SetBytes([]byte(content))
}

// WithTheme allows setting a custom theme for the editor.
func (m *Model) WithTheme(theme Theme) {
	m.theme = theme
}

// SetLanguage sets the programming language for syntax highlighting.
//
// If the language is empty, syntax highlighting will be disabled.
//
// The theme parameter allows specifying a Chroma theme for the syntax highlighter.
// For a full list of available themes, see: https://github.com/alecthomas/chroma/blob/master/styles
func (m *Model) SetLanguage(language string, theme string) {
	if m.language == language && m.highlighterTheme == theme {
		return
	}

	m.language = language
	m.highlighterTheme = theme
	if language == "" {
		m.highlighter = nil
		return
	}

	m.highlighter = highlighter.New(language, theme)
}

// WithSyntaxHighlighter allows setting a custom syntax highlighter.
func (m *Model) WithSyntaxHighlighter(highlighter *highlighter.Highlighter) {
	m.highlighter = highlighter
}

// HideLineNumbers controls whether to show line numbers in the viewport.
func (m *Model) HideLineNumbers(hide bool) {
	m.showLineNumbers = !hide
}

// ShowLineNumbers controls whether to show relative line numbers in the viewport.
// If Vim mode is disabled, this will not have any effect.
// If line numbers are hidden, this will not have any effect.
func (m *Model) ShowRelativeLineNumbers(show bool) {
	if m.disableVimMode {
		return
	}

	m.editor.ShowRelativeLineNumbers(show)
}

// ShowTildeIndicator controls whether to show the tilde indicator in the viewport.
// If line numbers are hidden, this will not have any effect.
func (m *Model) ShowTildeIndicator(show bool) {
	m.showTildeIndicator = show
}

// HideStatusLine controls whether to show the status line at the bottom of the viewport.
// If Vim mode is disabled, this will not have any effect.
func (m *Model) HideStatusLine(hide bool) {
	m.showStatusLine = !hide
}

// ShowMessages controls whether to show messages in the command line.
// This is useful for displaying messages like "1 line yanked" or "File saved successfully".
// If Vim mode is disabled, this will not have any effect.
// If set to false, messages will not be displayed in the command line.
// Instead, they will be handled internally and not shown to the user.
func (m *Model) ShowMessages(show bool) {
	m.showMessages = show
}

// GetSavedContent returns the saved content of the editor buffer
// This content is what was last saved to disk, and may not reflect the current state of the editor.
// It is useful for operations that require the last saved state, such as saving to a file.
func (m *Model) GetSavedContent() string {
	return m.editor.GetBuffer().GetSavedContent()
}

// GetCurrentContent returns the current content of the editor buffer.
// This content may not be saved yet, as it reflects the current state of the editor.
func (m *Model) GetCurrentContent() string {
	return m.editor.GetBuffer().GetCurrentContent()
}

// HasChanges checks if the editor has unsaved changes
func (m *Model) HasChanges() bool {
	return m.editor.GetBuffer().IsModified()
}

// GetEditor returns the underlying editor instance
func (m *Model) GetEditor() editor.Editor {
	return m.editor
}

// DisableVimMode allows disabling Vim mode in the editor.
// This will disable all Vim-specific features and revert to a simpler text editor mode.
// If Vim mode is disabled, the editor will not respond to Vim keybindings.
func (m *Model) DisableVimMode(disable bool) {
	m.disableVimMode = disable
	m.editor.DisableVimMode(disable)
}

// DisableCommandMode allows disabling command mode in the editor.
// This will disable the command mode functionality, meaning the editor will not respond to command mode keybindings.
func (m *Model) DisableCommandMode(disable bool) {
	m.editor.DisableCommandMode(disable)
}

// SetHighlightedWords allows setting highlighted words in the editor.
// These words will be styled with the provided lipgloss styles.
// This is useful for highlighting specific keywords or phrases in the text.
func (m *Model) SetHighlightedWords(words map[string]lipgloss.Style) {
	m.highlightedWords = words
}

// Focus sets the editor to focused state.
func (m *Model) Focus() {
	m.isFocused = true
}

// Blur sets the editor to unfocused state.
func (m *Model) Blur() {
	m.isFocused = false
}

// IsFocused returns whether the editor is currently focused.
func (m *Model) IsFocused() bool {
	return m.isFocused
}

// IsNormalMode returns whether the editor is in normal mode.
func (m *Model) IsNormalMode() bool {
	return m.editor.IsNormalMode()
}

// IsInsertMode returns whether the editor is in insert mode.
func (m *Model) IsInsertMode() bool {
	return m.editor.IsInsertMode()
}

// IsVisualMode returns whether the editor is in visual mode.
func (m *Model) IsVisualMode() bool {
	return m.editor.IsVisualMode()
}

// IsVisualLineMode returns whether the editor is in visual line mode.
func (m *Model) IsVisualLineMode() bool {
	return m.editor.IsVisualLineMode()
}

// IsCommandMode returns whether the editor is in command mode.
func (m *Model) IsCommandMode() bool {
	return m.editor.IsCommandMode()
}

// SetNormalMode sets the editor to normal mode.
func (m *Model) SetNormalMode() {
	m.editor.SetNormalMode()
}

// SetInsertMode sets the editor to insert mode.
func (m *Model) SetInsertMode() {
	m.editor.SetInsertMode()
}

// SetVisualMode sets the editor to visual mode.
func (m *Model) SetVisualMode() {
	m.editor.SetVisualMode()
}

// SetVisualLineMode sets the editor to visual line mode.
func (m *Model) SetVisualLineMode() {
	m.editor.SetVisualLineMode()
}

// SetCommandMode sets the editor to command mode.
func (m *Model) SetCommandMode() {
	m.editor.SetCommandMode()
}

// SetPlaceholder sets the placeholder text for the editor.
func (m *Model) SetPlaceholder(placeholder string) {
	m.placeholder = placeholder
}

// IsEmpty checks if the editor buffer is empty.
func (m *Model) IsEmpty() bool {
	return m.editor.GetBuffer().IsEmpty()
}

// SetCursorMode sets the cursor mode for the editor.
func (m *Model) SetCursorMode(mode CursorMode) {
	m.cursorMode = mode
	if mode == CursorBlink {
		m.cursorVisible = true
	} else {
		m.cursorVisible = m.isFocused
	}
}

// SetCursorBlinkMode sets the cursor mode to blinking or steady.
func (m *Model) SetCursorBlinkMode(blink bool) {
	if blink {
		m.SetCursorMode(CursorBlink)
	} else {
		m.SetCursorMode(CursorSteady)
	}
}

// SetCursorPosition sets the cursor position in the editor.
func (m *Model) SetCursorPosition(row, col int) error {
	if row < 0 || col < 0 {
		return fmt.Errorf("invalid cursor position: (%d, %d)", row, col)
	}

	if m.editor.GetBuffer().IsEmpty() {
		return fmt.Errorf("cannot set cursor position on an empty buffer")
	}

	cursor := m.editor.GetBuffer().GetCursor()
	cursor.Position.Row = row
	cursor.Position.Col = col

	cursor.Position.Row = max(0, cursor.Position.Row)
	cursor.Position.Col = max(0, cursor.Position.Col)

	m.editor.GetBuffer().SetCursor(cursor)

	return nil
}

// SetCursorPositionEnd sets the cursor position to the end of the editor buffer.
func (m *Model) SetCursorPositionEnd() error {
	if m.editor.GetBuffer().IsEmpty() {
		return fmt.Errorf("cannot set cursor position on an empty buffer")
	}

	cursor := m.editor.GetBuffer().GetCursor()
	lastLine := m.editor.GetBuffer().LineCount() - 1
	cursor.Position.Row = max(0, lastLine)
	cursor.Position.Col = m.editor.GetBuffer().LineRuneCount(lastLine)

	m.editor.GetBuffer().SetCursor(cursor)

	return nil
}

func (m Model) Init() tea.Cmd {
	return m.listenForEditorUpdate()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.IsFocused() {
			break
		}

		if m.editor.GetState().Quit {
			return m, tea.Quit
		}

		keyEvent := convertBubbleKey(msg)
		err := m.editor.HandleKey(keyEvent)
		if err != nil {
			cmds = append(cmds, func() tea.Msg {
				return errMsg(err)
			})
		}

		/* TODO: Optimize to only tokenize changed lines if possible. */
		m.handleContentChange()

		m.cursorVisible = true
		if m.cursorBlinkContext != nil && m.cursorBlinkContext.cancel != nil {
			m.cursorBlinkContext.cancel()
		}

		if m.cursorMode == CursorBlink {
			cmds = append(cmds, m.restartBlinkCycleCmd())
		}

		m.editor.ScrollViewport()

		m.calculateVisualMetrics()

		m.updateVisualTopLine()

	case messageMsg:
		m.message = string(msg)
		m.err = nil
		cmds = append(cmds, m.dispatchClearMsg())

	case errMsg:
		m.message = ""
		m.err = msg
		cmds = append(cmds, m.dispatchClearMsg())

	case yankMsg:
		if m.showMessages {
			m.message = msg.message
		}
		m.err = nil
		m.yanked = true
		cmds = append(cmds, m.dispatchClearMsg(), m.dispatchClearYankMsg())

	case clearMsg:
		m.message = ""
		m.err = nil

	case clearYankMsg:
		m.yanked = false
		m.editor.SetNormalMode()

	case cursorBlinkMsg:
		if m.isFocused && m.cursorMode == CursorBlink {
			m.cursorVisible = !m.cursorVisible
			cmds = append(cmds, m.CursorBlink())
		} else {
			m.cursorVisible = m.isFocused
		}

	case resumeBlinkCycleMsg:
		if m.isFocused && m.cursorMode == CursorBlink {
			m.cursorVisible = true
			cmds = append(cmds, m.CursorBlink())
		}
	}

	cmds = append(cmds, m.listenForEditorUpdate())

	var viewportCmd tea.Cmd
	m.viewport, viewportCmd = m.viewport.Update(msg)

	cmds = append(cmds, viewportCmd)

	m.calculateVisualMetrics()
	m.renderVisibleSlice()

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	state := m.editor.GetState()

	content := m.viewport.View()

	var commandLine string

	if !m.disableVimMode {
		commandLine = m.theme.CommandLineStyle.Render(state.CommandLine)
	}

	if m.message != "" {
		commandLine = m.theme.MessageStyle.
			Background(m.theme.CommandLineStyle.GetBackground()).
			Render(m.message)
	}

	if m.err != nil {
		commandLine = m.theme.ErrorStyle.
			Background(m.theme.CommandLineStyle.GetBackground()).
			Render(m.err.Error())
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

	cursor := m.editor.GetBuffer().GetCursor()

	cursorInfo := fmt.Sprintf("%d/%d ", cursor.Position.Row+1, cursor.Position.Col+1)

	width := m.width - (lipgloss.Width(cursorInfo) + lipgloss.Width(statusLine))
	gap := strings.Repeat(" ", max(0, width))

	statusLine += m.theme.StatusLineStyle.Render(
		gap + cursorInfo,
	)

	return statusLine
}

// SetMaxHistory sets the maximum number of history entries for undo/redo.
// This allows controlling how many undo steps are kept in memory.
// If set to 0, no history will be kept.
// The default value is 1000.
// If the number of history entries exceeds this limit, the oldest entries will be removed.
// This is useful for managing memory usage in the editor.
func (m *Model) SetMaxHistory(max uint32) {
	m.editor.SetMaxHistory(max)
}

func (m *Model) listenForEditorUpdate() tea.Cmd {
	return func() tea.Msg {
		editorChan := m.editor.GetUpdateSignalChan()
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

		case editor.RenameSignal:
			return RenameMsg{FileName: signal.Value()}

		case editor.DeleteFileSignal:
			return DeleteFileMsg{}
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

	if msg.Alt {
		key.Modifiers |= editor.ModAlt
	}

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

// CursorBlink is the main command for the blinking cursor effect (toggling visibility)
func (m *Model) CursorBlink() tea.Cmd {
	if m.cursorMode != CursorBlink || !m.isFocused {
		m.cursorVisible = m.isFocused
		return nil
	}

	if m.cursorBlinkContext != nil && m.cursorBlinkContext.cancel != nil {
		m.cursorBlinkContext.cancel()
	}

	ctx, cancel := context.WithTimeout(m.cursorBlinkContext.ctx, cursorBlinkInterval)
	m.cursorBlinkContext.cancel = cancel

	return func() tea.Msg {
		defer cancel()
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			return cursorBlinkMsg{}
		}
		return cursorBlinkCanceledMsg{}
	}
}

// restartBlinkCycleCmd is used after user activity to delay the resumption of blinking.
func (m *Model) restartBlinkCycleCmd() tea.Cmd {
	if m.cursorMode != CursorBlink || !m.isFocused {
		m.cursorVisible = m.isFocused
		return nil
	}

	return tea.Tick(cursorActivityResetDelay, func(t time.Time) tea.Msg {
		return resumeBlinkCycleMsg{}
	})
}
