package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea/v2"
	core "github.com/ionut-t/goeditor/core"
)

const messageDuration = 2 * time.Second

type Model struct {
	editor editor.Model
}

func (m Model) Init() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case editor.CompletionRequestMsg:
		// Handle completion request and provide completions
		completions := getCompletions(msg.Context)

		// Dispatch completion response signal back to editor
		m.editor.GetEditor().DispatchSignal(
			core.NewCompletionResponseSignal(completions, msg.Context),
		)
		return m, nil

	case editor.ErrorMsg:
		return m, m.editor.DispatchError(msg.Error, messageDuration)

	case editor.YankMsg:
		return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes yanked", len(msg.Content)), messageDuration)

	case editor.DeleteMsg:
		return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes deleted", len(msg.Content)), messageDuration)

	case editor.QuitMsg:
		return m, tea.Quit
	}

	editorModel, cmd := m.editor.Update(msg)
	m.editor = editorModel.(editor.Model)
	return m, cmd
}

func (m Model) View() tea.View {
	v := m.editor.View()
	v.AltScreen = true
	return v
}

// getCompletions returns completions based on the context
func getCompletions(ctx core.CompletionContext) []core.Completion {
	text := strings.ToLower(ctx.TextBeforeCursor)
	var completions []core.Completion

	// SQL-like keywords
	keywords := []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE",
		"JOIN", "LEFT JOIN", "RIGHT JOIN", "INNER JOIN",
		"ORDER BY", "GROUP BY", "HAVING", "LIMIT",
	}

	// Sample table names
	tables := []string{
		"users", "posts", "comments", "categories",
		"products", "orders", "customers",
	}

	// Sample column names
	columns := []string{
		"id", "name", "email", "created_at", "updated_at",
		"title", "content", "author_id", "status",
	}

	// Sample functions
	functions := []string{
		"COUNT()", "SUM()", "AVG()", "MAX()", "MIN()",
		"CONCAT()", "UPPER()", "LOWER()", "LENGTH()",
	}

	// Determine what to complete based on context
	if strings.Contains(text, "select") || strings.Contains(text, "from") {
		// After SELECT or FROM, suggest columns and tables
		for _, col := range columns {
			if strings.HasPrefix(strings.ToLower(col), getLastWord(text)) {
				completions = append(completions, core.Completion{
					Text:        col,
					Label:       col,
					Description: "Column",
					Type:        "column",
					Score:       1.0,
				})
			}
		}

		for _, table := range tables {
			if strings.HasPrefix(strings.ToLower(table), getLastWord(text)) {
				completions = append(completions, core.Completion{
					Text:        table,
					Label:       table,
					Description: "Table",
					Type:        "table",
					Score:       0.9,
				})
			}
		}
	}

	// Always suggest keywords
	lastWord := getLastWord(text)
	for _, kw := range keywords {
		if strings.HasPrefix(strings.ToLower(kw), lastWord) {
			completions = append(completions, core.Completion{
				Text:        kw,
				Label:       kw,
				Description: "SQL Keyword",
				Type:        "keyword",
				Score:       0.8,
			})
		}
	}

	// Suggest functions if typing a function-like pattern
	if strings.Contains(text, "(") || lastWord != "" {
		for _, fn := range functions {
			if strings.HasPrefix(strings.ToLower(fn), lastWord) {
				completions = append(completions, core.Completion{
					Text:        fn,
					Label:       fn,
					Description: "SQL Function",
					Type:        "function",
					Score:       0.85,
				})
			}
		}
	}

	// If no specific matches, return all completions
	if len(completions) == 0 && len(lastWord) > 0 {
		// Add all keywords
		for _, kw := range keywords {
			completions = append(completions, core.Completion{
				Text:        kw,
				Label:       kw,
				Description: "SQL Keyword",
				Type:        "keyword",
				Score:       0.8,
			})
		}
	}

	// Limit to top 10
	if len(completions) > 10 {
		completions = completions[:10]
	}

	return completions
}

// getLastWord extracts the last word from the text (partial word being typed)
func getLastWord(text string) string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '(' || r == ',' || r == '\t' || r == '\n'
	})
	if len(words) > 0 {
		return strings.ToLower(words[len(words)-1])
	}
	return ""
}

func main() {
	// Create editor
	m := editor.New(80, 24)
	isDark := lipgloss.HasDarkBackground(os.Stdout, os.Stderr)
	m.SetLanguage("sql", languageTheme(isDark))

	// Enable auto-trigger completions
	m.WithAutoTrigger(true)
	m.WithCompletionDebounce(200 * time.Millisecond)

	// Set some initial content to help demonstrate
	m.SetContent(`-- SQL Completion Demo
-- Try typing:
--   - "SELECT " to see columns/tables
--   - "FROM " to see table names
--   - Press Ctrl+Space for manual completion
--   - Use Up/Down arrows to navigate
--   - Press Enter or Tab to insert completion
--   - Press Escape to close menu

`)
	m.Focus()

	model := Model{editor: m}

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func languageTheme(isDark bool) string {
	if isDark {
		return "catppuccin-mocha"
	}

	return "catppuccin-latte"
}
