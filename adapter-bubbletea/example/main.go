package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/goeditor/core"
)

const messageDuration = 3 * time.Second

type Model struct {
	editor editor.Model
	file   string
}

func (m Model) Init() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor.SetSize(msg.Width-4, msg.Height-2)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case editor.ErrorMsg:
		return m, m.editor.DispatchError(msg.Error, messageDuration)

	case editor.YankMsg:
		return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes yanked", len(msg.Content)), messageDuration)

	case editor.DeleteMsg:
		return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes deleted", len(msg.Content)), messageDuration)

	case editor.SearchResultsMsg:
		if len(msg.Positions) == 0 {
			return m, m.editor.DispatchError(errors.New("no search results"), messageDuration)
		}

	case editor.SaveMsg:
		if msg.Path != nil {
			m.file = *msg.Path
		}

		filePath := m.file
		if strings.HasPrefix(filePath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return m, m.editor.DispatchError(err, messageDuration)
			} else {
				filePath = filepath.Join(homeDir, filePath[2:])
			}
		}

		if err := os.WriteFile(filePath, []byte(msg.Content), 0644); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

		return m, m.editor.DispatchMessage(fmt.Sprintf("file saved to %s", m.file), messageDuration)

	case editor.RenameMsg:
		if err := os.Rename(m.file, msg.FileName); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

	case editor.DeleteFileMsg:
		if err := os.Remove(m.file); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

		return m, tea.Quit

	case editor.QuitMsg:
		return m, tea.Quit
	}

	var cmds []tea.Cmd

	editorModel, cmd := m.editor.Update(msg)
	cmds = append(cmds, cmd)
	m.editor = editorModel.(editor.Model)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(m.editor.View())
}

func main() {
	lang := "markdown"

	if len(os.Args) > 1 {
		lang = os.Args[1]
	}

	file := "test.md"

	if lang == "sql" {
		file = "test.sql"
	}

	textEditor := editor.New(80, 20)
	textEditor.Focus()
	textEditor.SetCursorMode(editor.CursorBlink)
	textEditor.SetLanguage(lang, "catppuccin-mocha")
	textEditor.WithSearchOptions(core.SearchOptions{
		IgnoreCase: true,
		SmartCase:  true,
		Wrap:       true,
		Backwards:  true,
	})

	if content, err := os.ReadFile(file); err == nil {
		textEditor.SetBytes(content)
	}

	m := Model{
		editor: textEditor,
		file:   file,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err := p.Run()

	if err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
		os.Exit(1)
	}
}
