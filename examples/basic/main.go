package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	editor "github.com/ionut-t/goeditor"
	"github.com/ionut-t/goeditor/core"
)

const messageDuration = 3 * time.Second

// document is the file being edited. The editor's write function and the
// model share it, so it's held by pointer: Model is copied on every Update.
type document struct {
	name string
}

// write is the editor's write function (see SetWriteFunc in main). path is
// the file name typed after :w, or nil for the current file. A local file is
// quick to write, so it writes right here and returns no job.
func (d *document) write(path *string, content string) (core.WriteJob, error) {
	name := d.name
	if path != nil {
		name = *path
	}
	return nil, os.WriteFile(expandHome(name), []byte(content), 0o644)
}

type Model struct {
	editor editor.Model
	doc    *document
	lang   string
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.editor.CursorBlink(), tea.RequestBackgroundColor)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.editor.SetSize(msg.Width, msg.Height)

	case tea.BackgroundColorMsg:
		m.editor.WithTheme(editor.DefaultTheme(msg.IsDark()))
		m.editor.SetLanguage(m.lang, languageTheme(msg.IsDark()))

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
		// The file is already written (doc.write); a failed write comes as
		// an ErrorMsg instead.
		if msg.Path != nil {
			m.doc.name = *msg.Path
		}
		return m, m.editor.DispatchMessage(fmt.Sprintf("%q %dB written", m.doc.name, len(msg.Content)), messageDuration)

	case editor.RenameMsg:
		if err := os.Rename(m.doc.name, msg.FileName); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

	case editor.DeleteFileMsg:
		if err := os.Remove(m.doc.name); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

		return m, tea.Quit

	case editor.OpenMsg:
		if err := editor.OpenWithSystem(msg.Target); err != nil {
			return m, m.editor.DispatchError(err, messageDuration)
		}

	case editor.QuitMsg:
		return m, tea.Quit
	}

	var cmds []tea.Cmd

	editorModel, cmd := m.editor.Update(msg)
	cmds = append(cmds, cmd)
	m.editor = editorModel

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	v := tea.NewView(m.editor.View())
	v.AltScreen = true
	return v
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
	// Dark until the terminal reports its background colour.
	textEditor.SetLanguage(lang, languageTheme(true))
	textEditor.WithSearchOptions(editor.SearchOptions{
		IgnoreCase: true,
		SmartCase:  true,
		Wrap:       true,
	})
	textEditor.SetExtraWordChars('-')
	textEditor.SetMoveAcrossLines(true)

	if content, err := os.ReadFile(file); err == nil {
		textEditor.SetBytes(content)
	}

	// :w, :wq and :x write through doc.write: the buffer counts as saved
	// only when it succeeds, and :wq and :x stay open when it fails.
	doc := &document{name: file}
	textEditor.SetWriteFunc(doc.write)

	m := Model{
		editor: textEditor,
		doc:    doc,
		lang:   lang,
	}

	p := tea.NewProgram(m)

	_, err := p.Run()
	if err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
		os.Exit(1)
	}
}

// expandHome expands a leading ~/ to the home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

func languageTheme(isDark bool) string {
	if isDark {
		return "catppuccin-mocha"
	}

	return "catppuccin-latte"
}
