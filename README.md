# GoEditor

A feature-rich, Vim-inspired text editor library for Go.

## Features

### Core Editor (`core` package)

- **Multiple editing modes**: Normal, Insert, Visual, Visual Line, and Command modes
- **Vim-style keybindings**: Navigate and edit text efficiently with familiar Vim commands
- **Unicode support**: Full support for international characters and emojis
- **Undo/Redo**: Navigate through your editing history
- **Search functionality**: Find text within your document (in progress)
- **Clipboard integration**: Copy, cut, and paste with system clipboard support
- **Line wrapping**: Automatic word-wrap for long lines
- **Extensible architecture**: Easy to add new modes and commands

### [Bubble Tea](https://github.com/charmbracelet/bubbletea) Adapter (`editor` package)

- **Custom Themes**: Customizable color schemes and styles with [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Line numbers**: Optional absolute or relative line numbering
- **Syntax highlighting**: Customizable word highlighting with styles
- **Status line**: Shows current mode, cursor position, and file status
- **Command Line**: Display messages and command input.
- **Responsive**: Adapts to terminal size changes
- **Line Wrapping**: Automatically wraps long lines to fit the terminal width
- **Cursor modes**: Blinking or steady cursor with mode-specific styling
- **Focus/Blur**: Programmatic focus management.
- **Placeholder text**: Display helpful text when the buffer is empty

## Installation

```bash
go get github.com/ionut-t/goeditor/adapter-bubbletea
```

## Quick Start

### Basic Usage

```go
package main

import (
    "log"

    tea "github.com/charmbracelet/bubbletea"
    editor "github.com/ionut-t/goeditor/adapter-bubbletea"
)

func main() {
    // Create a new editor with specified dimensions
    m := editor.New(80, 24)

    // Set initial content
    m.SetContent("Hello, World!\nWelcome to GoEditor.")

    // Create the Bubble Tea program
    p := tea.NewProgram(m)

    // Run the editor
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Advanced Configuration

```go
// Disable Vim mode for a simpler editing experience
m.DisableVimMode(true)

// Show relative line numbers
m.ShowRelativeLineNumbers(true)

// Hide line numbers entirely
m.HideLineNumbers(true)

// Set placeholder text
m.SetPlaceholder("Start typing...")

// Highlight specific words
highlights := map[string]lipgloss.Style{
    "TODO":     lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
    "FIXME":    lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
    "NOTE":     lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true),
}
m.SetHighlightedWords(highlights)

// Set cursor to blink
m.SetCursorBlinkMode(true)

// Custom theme
theme := editor.Theme{
    NormalModeStyle:  lipgloss.NewStyle().Background(lipgloss.Color("22")),
    InsertModeStyle:  lipgloss.NewStyle().Background(lipgloss.Color("28")),
    SelectionStyle:   lipgloss.NewStyle().Background(lipgloss.Color("240")),
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
SetNormalMode() error
SetInsertMode() error
SetVisualMode() error
SetCommandMode() error
DisableVimMode(disable bool)

// Display Options
HideLineNumbers(hide bool)
ShowRelativeLineNumbers(show bool)
ShowTildeIndicator(show bool)
HideStatusLine(hide bool)
ShowMessages(show bool)

// Cursor Control
SetCursorPosition(row, col int) error
SetCursorPositionEnd() error
SetCursorBlinkMode(blink bool)

// Styling
WithTheme(theme Theme)
SetHighlightedWords(words map[string]lipgloss.Style)
SetPlaceholder(placeholder string)

// Focus Management
Focus()
Blur()
IsFocused() bool
```

### Listening for Editor Events

The editor sends various messages that can be handled in your Bubble Tea Update method:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case editor.SaveMsg:
        // Handle save event
        content := string(msg)
        // Save to file...

    case editor.QuitMsg:
        // Handle quit event
        return m, tea.Quit

    case editor.RenameMsg:
        // Handle rename event
        newName := msg.FileName
        // Rename file...

    case editor.DeleteFileMsg:
        // Handle delete event
        // Delete file...
    }
}
```

## Core Library Usage

For direct usage of the core editor without the Bubble Tea UI:

```go
import (
    "github.com/atotto/clipboard"
    "github.com/ionut-t/goeditor/core"
)

// Implement core.Clipboard interface
type atottoClipboard struct{}

func (c *atottoClipboard) Write(text string) error {
	return clipboard.WriteAll(text)
}

func (c *atottoClipboard) Read() (string, error) {
	return clipboard.ReadAll()
}

// Create a new buffer
buffer := core.NewBuffer()
buffer.SetContent([]byte("Hello, World!"))

// Create an editor
clipboard := &atottoClipboard{}
editor := core.New(clipboard)

// Handle key events
key := core.KeyEvent{Rune: 'i'}
editor.HandleKey(key)

// Get buffer content
content := editor.GetBuffer().GetCurrentContent()
```

## Examples

See [adapter-bubbletea/example/main.go](adapter-bubbletea/example/main.go).

## License

[MIT](LICENSE)

