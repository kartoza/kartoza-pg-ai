# Libraries

This document describes the key libraries used in Kartoza PG AI.

## TUI Framework

### Bubble Tea

[github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)

The Elm Architecture for Go terminal applications.

```go
type Model interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

### Lipgloss

[github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)

CSS-like styling for terminal output.

```go
var style = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#DDA036"))
```

### Bubbles

[github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)

Pre-built components for Bubble Tea:

- `textarea` - Multi-line text input
- `spinner` - Loading indicators
- `list` - Scrollable lists

## Terminal Graphics

### go-termimg

[github.com/blacktop/go-termimg](https://github.com/blacktop/go-termimg)

Render images in terminal using Kitty graphics protocol.

### nfnt/resize

[github.com/nfnt/resize](https://github.com/nfnt/resize)

High-quality image resizing for splash screen animations.

## CLI Framework

### Cobra

[github.com/spf13/cobra](https://github.com/spf13/cobra)

CLI application framework with command hierarchy.

```go
var rootCmd = &cobra.Command{
    Use:   "kartoza-pg-ai",
    Short: "Natural language PostgreSQL interface",
}
```

## Database

### lib/pq

[github.com/lib/pq](https://github.com/lib/pq)

Pure Go PostgreSQL driver.

```go
import _ "github.com/lib/pq"

db, err := sql.Open("postgres", connectionString)
```

## Why These Libraries?

1. **Bubble Tea**: Modern, idiomatic Go TUI framework
2. **Lipgloss**: Powerful styling without ANSI escape codes
3. **Cobra**: Industry standard CLI framework
4. **lib/pq**: Reliable, well-maintained PostgreSQL driver
5. **go-termimg**: Enables beautiful splash screens on modern terminals
