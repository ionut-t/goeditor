package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
)

type Model struct {
	editor editor.Model
}

func (m Model) Init() tea.Cmd {
	return nil
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

	case editor.SaveMsg:
		if err := os.WriteFile("test.md", []byte(msg), 0644); err != nil {
			log.Println("Error saving file:", err)
			os.Exit(1)
		}

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
	textEditor := editor.New(80, 20)
	textEditor.ShowMessages(true)
	textEditor.Focus()

	highlightedWords := map[string]lipgloss.Style{
		"TODO":  lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		"FIXME": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
	}

	textEditor.SetHighlightedWords(highlightedWords)

	if content, err := os.ReadFile("test.md"); err == nil {
		textEditor.SetContent(content)
	}

	m := Model{
		editor: textEditor,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err := p.Run()

	if err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
		os.Exit(1)
	}
}
