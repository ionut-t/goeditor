# GoEditor

A feature-rich, Vim-inspired text editor library for Go, built on [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Multiple editing modes**: Normal, Insert, Visual, Visual Line, and Command modes
- **Vim-style keybindings**: Navigate and edit text efficiently with familiar Vim commands
- **Unicode support**: Full support for international characters and emojis
- **Undo/Redo**: Navigate through your editing history, including Vim's line-wise `U`
- **Key mapping**: Vim-compatible `:map` family plus a Go API for remapping any key
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
- **Completion menu**: Pluggable autocompletion with a scrollable, keyboard-navigable suggestion menu

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
    "github.com/ionut-t/goeditor"
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
- **Word movement**: `w`, `b`, `e`, `ge` and WORD variants `W`, `B`, `E`, `gE`
- **Line movement**: `0` (start), `$` (end), `^` (first non-blank)
- **Document movement**: `gg` (first line), `G` (last line), `{` / `}` (paragraph), `%` (matching bracket)
- **Character search**: `f`, `F`, `t`, `T`, repeated with `;` / `,`
- **Operators**: `d` (delete), `y` (yank), `c` (change) — combined with the motions above, counts (`d2w`, `3dd`), and text objects (`iw`, `aw`, `ip`, `i(`, `i"`, …)
- **Editing**: `x` / `X` (delete char), `s` / `S` (substitute), `r` (replace char), `J` (join lines), `D` / `C` (delete/change to end of line), `~`, `gu` / `gU` / `g~` (case)
- **Mode switching**: `i`, `I`, `a`, `A`, `o`, `O` (insert), `v` (visual), `V` (visual line), `:` (command), `/` / `?` (search)
- **Undo/Redo**: `u` (undo), `Ctrl+R` (redo), `U` (undo all recent changes on the last-changed line)
- **Copy/Paste**: `y` (yank), `p` / `P` (paste)

### Insert Mode

- Type normally to insert text
- `Esc` to return to Normal mode
- `Backspace` to delete characters
- Arrow keys for navigation
- `Ctrl+Space` to manually trigger autocompletion
- `Ctrl+N` / `Ctrl+P` to navigate suggestions
- `Ctrl+Y` to accept the selected suggestion
- `Ctrl+E` / `Esc` to dismiss the completion menu

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
- `:map`, `:nmap`, `:vmap`, `:imap`, `:omap` and their `noremap` / `unmap` / `mapclear` variants - see [Key Mapping](#key-mapping)

## Key Mapping

Keys are remapped the way Vim does it: a key sequence is replaced by another key
sequence, which is then re-fed through the editor. That means the right-hand side
can be any command, not just a single action — `:nmap Y y$` works because `y$`
re-enters the parser as if typed.

```
:nmap U <C-r>        " restore U as redo (it is undo-line by default)
:imap jk <Esc>       " leave insert mode with jk
:nnoremap Y y$       " yank to end of line
:nnoremap <Space> dw " map a named key
:nnoremap x <Nop>    " disable a key
:nunmap x            " put x back
:nmapclear           " drop every normal-mode mapping
```

The same table is available programmatically, which is usually how a library
consumer configures it:

```go
m := goeditor.New(80, 24)

m.Map(core.MapNormal, "U", "<C-r>", true)
m.Map(core.MapInsert, "jk", "<Esc>", true)
m.Map(core.MapNormal, "<leader>d", "dd", true)

m.SetMapLeader(",")                 // <leader> is "\" by default
m.Unmap(core.MapNormal, "U")
m.Mappings(core.MapNormal)          // inspect what is registered
```

**Modes** are selected with `core.MapNormal`, `core.MapVisual` (both visual and
visual-line mode), `core.MapInsert`, `core.MapOperatorPending` (while `d`/`y`/`c`
waits for a motion), or `core.MapAll`. As in Vim, `core.MapAll` and an unprefixed
`:map` cover normal, visual and operator-pending — but not insert.

**Recursion** follows Vim: `:map` re-resolves the replacement against the other
mappings, `:noremap` (and `noremap: true`) delivers it verbatim. Mutually
recursive mappings are stopped rather than hanging.

**Notation** accepts `<C-x>`, `<A-x>` / `<M-x>`, `<S-x>` (combinable as
`<C-A-x>`), `<Esc>`, `<CR>` / `<Enter>`, `<Tab>`, `<Space>`, `<BS>`, `<Del>`,
`<Insert>`, the arrow keys, `<Home>` / `<End>`, `<PageUp>` / `<PageDown>`,
`<leader>`, `<lt>` for a literal `<`, and `<Nop>` to disable a key.

Two limitations to be aware of:

- **No `timeoutlen` yet.** When the keys typed so far are both a complete mapping
  and the prefix of a longer one, Vim commits the shorter one after a timeout.
  Here it waits for the next key instead of timing out.
- **No `:cmap`.** Command-line and search input are not mapped.

Mappings never rewrite the character argument of `r`, `f`, `F`, `t` or `T` — that
is data, not a command, exactly as in Vim.

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

// Key Mapping
Map(modes core.MapMode, lhs, rhs string, noremap bool) error
Unmap(modes core.MapMode, lhs string) error
ClearMappings(modes core.MapMode)
Mappings(mode core.MapMode) []core.Mapping
SetMapLeader(leader string)

// Styling
WithTheme(theme Theme)
SetHighlightedWords(words map[string]lipgloss.Style)
SetPlaceholder(placeholder string)

// Focus Management
Focus()
Blur()
IsFocused() bool

// Completion
SetCompletions(completions []core.Completion, context core.CompletionContext)
WithCompletionAutoTrigger(enabled bool)
WithCompletionDebounce(duration time.Duration)
SetCompletionMenuMaxVisibleItems(max int)
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

## Autocompletion

GoEditor does not fetch completions itself — it dispatches a `CompletionRequestMsg` when the user triggers completion and expects the host application to respond with `SetCompletions`.

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case goeditor.CompletionRequestMsg:
        ctx := msg.Context
        return m, func() tea.Msg {
            completions := fetchCompletions(ctx) // your completion source
            return goeditor.CompletionResponseMsg{
                Completions: completions,
                Context:     ctx,
            }
        }

    case goeditor.CompletionResponseMsg:
        m.editor.SetCompletions(msg.Completions, msg.Context)
    }
}
```

Enable auto-trigger (fires on every keystroke in Insert mode):

```go
m.WithCompletionAutoTrigger(true)
m.WithCompletionDebounce(150 * time.Millisecond) // optional debounce
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
