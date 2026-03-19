# GoEditor

A feature-rich, Vim-inspired text editor library for Go, built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Multiple editing modes**: Normal, Insert, Visual, Visual Line, and Command modes
- **Vim-style keybindings**: Navigate and edit text efficiently with familiar Vim commands
- **Unicode support**: Full support for international characters and emojis
- **Undo/Redo**: Navigate through your editing history
- **Search functionality**: Find text within your document
- **Clipboard integration**: Copy, cut, and paste with system clipboard support
- **Line wrapping**: Automatic word-wrap for long lines
- **Custom Themes**: Customizable color schemes and styles with [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Line numbers**: Optional absolute or relative line numbering
- **Syntax highlighting**: Automatic syntax highlighting for various languages (Go, Python, Markdown, etc.)
- **Customizable word highlighting**: Highlight specific words with custom styles
- **Status line**: Shows current mode, cursor position, and file status
- **Responsive**: Adapts to terminal size changes
- **Cursor modes**: Blinking or steady cursor with mode-specific styling
- **Focus/Blur**: Programmatic focus management
- **Placeholder text**: Display helpful text when the buffer is empty

## Installation

```bash
go get github.com/ionut-t/goeditor
```

## Quick Start

```go
package main

import (
    "log"

    tea "charm.land/bubbletea/v2"
    goeditor "github.com/ionut-t/goeditor"
)

func main() {
    m := goeditor.New(80, 24)
    m.SetContent("Hello, World!\nWelcome to GoEditor.")
    m.Focus()

    p := tea.NewProgram(m)

    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## Configuration

```go
// Disable Vim mode for a simpler editing experience
m.DisableVimMode(true)

// Show relative line numbers
m.ShowRelativeLineNumbers(true)

// Hide line numbers entirely
m.HideLineNumbers(true)

// Set placeholder text
m.SetPlaceholder("Start typing...")

// Set language for syntax highlighting
m.SetLanguage("go", "catppuccin-mocha")

// Highlight specific words
highlights := map[string]lipgloss.Style{
    "TODO":  lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
    "FIXME": lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
    "NOTE":  lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true),
}
m.SetHighlightedWords(highlights)

// Set cursor to blink
m.SetCursorMode(goeditor.CursorBlink)

// Custom theme
theme := goeditor.Theme{
    NormalModeStyle: lipgloss.NewStyle().Background(lipgloss.Color("22")),
    InsertModeStyle: lipgloss.NewStyle().Background(lipgloss.Color("28")),
    SelectionStyle:  lipgloss.NewStyle().Background(lipgloss.Color("240")),
    // ... customize other styles
}
m.WithTheme(theme)
```

## Vim Keybindings

### Normal Mode

- **Movement**: `h`, `j`, `k`, `l` or arrow keys
- **Word movement**: `w` (forward), `b` (backward), `e` (end of word)
- **Line movement**: `0` (start), `$` (end), `^` (first non-blank)
- **Document movement**: `g` (first line), `G` (last line)
- **Editing**: `x` (delete char), `dd` (delete line), `D` (delete to end of line)
- **Mode switching**: `i` (insert), `v` (visual), `V` (visual line), `:` (command)
- **Undo/Redo**: `u` (undo), `U` (redo)
- **Copy/Paste**: `y` (yank), `p` (paste)

### Insert Mode

- Type normally to insert text
- `Esc` to return to Normal mode
- `Backspace` to delete characters
- Arrow keys for navigation

### Visual Mode

- Select text character by character
- `d` or `x` to delete selection
- `y` to copy selection
- `Esc` to cancel selection

### Command Mode

- `:w` - Save file
- `:q` - Quit
- `:wq` - Save and quit
- `:q!` - Force quit without saving
- `:set rnu` - Enable relative line numbers
- `:set nornu` - Disable relative line numbers

## API Reference

### Editor Model Methods

```go
// Content Management
SetContent(content string)
SetBytes(content []byte)
GetCurrentContent() string
GetSavedContent() string
HasChanges() bool
IsEmpty() bool

// Mode Control
SetNormalMode()
SetInsertMode()
SetVisualMode()
SetCommandMode()
DisableVimMode(disable bool)

// Display Options
HideLineNumbers(hide bool)
ShowRelativeLineNumbers(show bool)
ShowTildeIndicator(show bool)
HideStatusLine(hide bool)

// Cursor Control
SetCursorPosition(row, col int) error
SetCursorPositionEnd() error
SetCursorMode(mode CursorMode)

// Styling
WithTheme(theme Theme)
SetHighlightedWords(words map[string]lipgloss.Style)
SetPlaceholder(placeholder string)

// Focus Management
Focus()
Blur()
IsFocused() bool
```

### Handling Editor Events

Handle events in your Bubble Tea `Update` method:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case goeditor.SaveMsg:
        content := msg.Content
        // Save to file...

    case goeditor.QuitMsg:
        return m, tea.Quit

    case goeditor.YankMsg:
        return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes yanked", len(msg.Content)), 3*time.Second)

    case goeditor.DeleteMsg:
        return m, m.editor.DispatchMessage(fmt.Sprintf("%d bytes deleted", len(msg.Content)), 3*time.Second)

    case goeditor.ErrorMsg:
        return m, m.editor.DispatchError(msg.Error, 3*time.Second)
    }
}
```

## Core Package

The `core` package contains the editor engine with no UI dependencies and can be used independently:

```go
import "github.com/ionut-t/goeditor/core"

// Implement core.Clipboard interface
type clipboardImpl struct{}

func (c *clipboardImpl) Write(text string) error { ... }
func (c *clipboardImpl) Read() (string, error)   { ... }

ed := core.New(&clipboardImpl{})
ed.SetContent([]byte("Hello, World!"))
ed.HandleKey(core.KeyEvent{Rune: 'i'})

content := ed.GetBuffer().GetCurrentContent()
```

## Examples

See [examples/basic](examples/basic/main.go) and [examples/completion](examples/completion/main.go).

## Acknowledgements

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): A powerful TUI framework for Go.
- [Chroma](https://github.com/alecthomas/chroma): A general purpose syntax highlighter in pure Go.
- [Lip Gloss](https://github.com/charmbracelet/lipgloss): Style definitions for nice terminal layouts.
- [atotto/clipboard](https://github.com/atotto/clipboard): A cross-platform clipboard package for Go.

## License

[MIT](LICENSE)
