package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	editor "github.com/ionut-t/goeditor"
	"github.com/ionut-t/goeditor/core"
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
		// msg.Context is passed through to the response unchanged so the editor
		// can match the RequestID and discard stale responses. This is especially
		// important for async providers (e.g. LSP, HTTP) where an earlier response
		// can arrive after a newer one.
		completions := getCompletions(msg.Context)
		m.editor.SetCompletions(completions, msg.Context)
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
	m.editor = editorModel
	return m, cmd
}

func (m Model) View() tea.View {
	v := tea.NewView(m.editor.View())
	v.AltScreen = true
	return v
}

// getCompletions returns completions based on the context
func getCompletions(ctx core.CompletionContext) []core.Completion {
	text := strings.ToLower(ctx.TextBeforeCursor)
	lastWord := getLastWord(text)

	// Don't suggest anything when the user hasn't started a word yet —
	// returning everything on an empty prefix floods the menu after every space.
	if lastWord == "" {
		return nil
	}

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

	var completions []core.Completion

	// Use the last SQL keyword on the line (not the entire text before cursor)
	// to decide which category of completions is most relevant. Checking the
	// whole text would incorrectly activate the column/table branch whenever
	// "select" or "from" appears anywhere earlier on the line.
	lastKeyword := getLastSQLKeyword(text)
	if lastKeyword == "select" || lastKeyword == "from" || lastKeyword == "where" {
		for _, col := range columns {
			if strings.HasPrefix(col, lastWord) {
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
			if strings.HasPrefix(table, lastWord) {
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

	// Always suggest matching keywords
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

	// Suggest matching functions
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

	// Limit to top 10
	if len(completions) > 10 {
		completions = completions[:10]
	}

	return completions
}

// getLastSQLKeyword returns the last SQL keyword found in text (lowercase).
// This is used to determine completion context without being confused by
// keywords that appeared earlier in the line.
func getLastSQLKeyword(text string) string {
	sqlKeywords := []string{
		"select", "from", "where", "insert", "update", "delete",
		"join", "order by", "group by", "having", "limit",
	}
	last := ""
	lastIdx := -1
	for _, kw := range sqlKeywords {
		if idx := strings.LastIndex(text, kw); idx > lastIdx {
			lastIdx = idx
			last = kw
		}
	}
	return last
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
	m.WithCompletionAutoTrigger(true)
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
