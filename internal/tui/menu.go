package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents a menu option
type MenuItem int

const (
	MenuQuery MenuItem = iota
	MenuDatabases
	MenuHistory
	MenuSettings
	MenuQuit
)

// menuItem holds menu item data
type menuItem struct {
	label   string
	enabled bool
	action  MenuItem
	icon    string
}

// MenuModel represents the main menu screen
type MenuModel struct {
	selectedItem int
	menuItems    []menuItem
	width        int
	height       int
}

// NewMenuModel creates a new menu model
func NewMenuModel() *MenuModel {
	return &MenuModel{
		selectedItem: 0,
		menuItems: []menuItem{
			{label: "Query Database", enabled: true, action: MenuQuery, icon: "󰆼"},
			{label: "Database Connections", enabled: true, action: MenuDatabases, icon: "󰒋"},
			{label: "Query History", enabled: true, action: MenuHistory, icon: "󰋚"},
			{label: "Settings", enabled: true, action: MenuSettings, icon: "󰒓"},
			{label: "Quit", enabled: true, action: MenuQuit, icon: "󰗼"},
		},
	}
}

// Init initializes the menu
func (m *MenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the menu
func (m *MenuModel) Update(msg tea.Msg) (*MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			m.selectedItem--
			if m.selectedItem < 0 {
				m.selectedItem = len(m.menuItems) - 1
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			m.selectedItem++
			if m.selectedItem >= len(m.menuItems) {
				m.selectedItem = 0
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if m.selectedItem >= 0 && m.selectedItem < len(m.menuItems) {
				item := m.menuItems[m.selectedItem]
				if item.enabled {
					return m, m.handleSelection(item.action)
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// handleSelection handles menu item selection
func (m *MenuModel) handleSelection(action MenuItem) tea.Cmd {
	switch action {
	case MenuQuery:
		return func() tea.Msg {
			return menuActionMsg{action: MenuQuery}
		}
	case MenuDatabases:
		return func() tea.Msg {
			return menuActionMsg{action: MenuDatabases}
		}
	case MenuHistory:
		return func() tea.Msg {
			return menuActionMsg{action: MenuHistory}
		}
	case MenuSettings:
		return func() tea.Msg {
			return menuActionMsg{action: MenuSettings}
		}
	case MenuQuit:
		return tea.Quit
	}
	return nil
}

// View renders the menu
func (m *MenuModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Main Menu")
	menu := m.renderMenuItems()
	helpText := "↑/k: up • ↓/j: down • enter/space: select • q: quit"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, menu, footer, m.width, m.height)
}

// renderMenuItems renders the menu items
func (m *MenuModel) renderMenuItems() string {
	normalStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Padding(0, 2)

	selectedStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Padding(0, 2)

	disabledStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Padding(0, 2)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Align(lipgloss.Center)

	var items []string
	items = append(items, subtitleStyle.Render("Select an option from the menu below"))
	items = append(items, "")

	for i, item := range m.menuItems {
		prefix := "  "
		if i == m.selectedItem {
			prefix = "▶ "
		}

		var rendered string
		label := prefix + item.icon + " " + item.label
		if !item.enabled {
			rendered = disabledStyle.Render(label + " (disabled)")
		} else if i == m.selectedItem {
			rendered = selectedStyle.Render(label)
		} else {
			rendered = normalStyle.Render(label)
		}

		items = append(items, rendered)
	}

	return lipgloss.JoinVertical(lipgloss.Center, items...)
}

// SelectedAction returns the currently selected action
func (m *MenuModel) SelectedAction() MenuItem {
	if m.selectedItem >= 0 && m.selectedItem < len(m.menuItems) {
		return m.menuItems[m.selectedItem].action
	}
	return MenuQuery
}

// SetItemEnabled enables or disables a menu item
func (m *MenuModel) SetItemEnabled(action MenuItem, enabled bool) {
	for i := range m.menuItems {
		if m.menuItems[i].action == action {
			m.menuItems[i].enabled = enabled
			break
		}
	}
}

// menuActionMsg is sent when a menu item is selected
type menuActionMsg struct {
	action MenuItem
}

// bigASCIINumbers provides large ASCII art numbers for displays
var bigASCIINumbers = map[rune][]string{
	'0': {
		" ██████╗ ",
		"██╔═████╗",
		"██║██╔██║",
		"████╔╝██║",
		"╚██████╔╝",
		" ╚═════╝ ",
	},
	'1': {
		" ██╗",
		"███║",
		"╚██║",
		" ██║",
		" ██║",
		" ╚═╝",
	},
	'2': {
		"██████╗ ",
		"╚════██╗",
		" █████╔╝",
		"██╔═══╝ ",
		"███████╗",
		"╚══════╝",
	},
	'3': {
		"██████╗ ",
		"╚════██╗",
		" █████╔╝",
		" ╚═══██╗",
		"██████╔╝",
		"╚═════╝ ",
	},
	'4': {
		"██╗  ██╗",
		"██║  ██║",
		"███████║",
		"╚════██║",
		"     ██║",
		"     ╚═╝",
	},
	'5': {
		"███████╗",
		"██╔════╝",
		"███████╗",
		"╚════██║",
		"███████║",
		"╚══════╝",
	},
	'6': {
		" ██████╗",
		"██╔════╝",
		"███████╗",
		"██╔═══██╗",
		"╚██████╔╝",
		" ╚═════╝",
	},
	'7': {
		"███████╗",
		"╚════██║",
		"    ██╔╝",
		"   ██╔╝ ",
		"   ██║  ",
		"   ╚═╝  ",
	},
	'8': {
		" █████╗ ",
		"██╔══██╗",
		"╚█████╔╝",
		"██╔══██╗",
		"╚█████╔╝",
		" ╚════╝ ",
	},
	'9': {
		" █████╗ ",
		"██╔══██╗",
		"╚██████║",
		" ╚═══██║",
		" █████╔╝",
		" ╚════╝ ",
	},
}

// RenderBigNumber renders a number using ASCII art
func RenderBigNumber(num string) string {
	if len(num) == 0 {
		return ""
	}

	var lines [6]strings.Builder

	for _, r := range num {
		if digit, ok := bigASCIINumbers[r]; ok {
			for i, line := range digit {
				lines[i].WriteString(line)
				lines[i].WriteString(" ")
			}
		}
	}

	var result []string
	for _, line := range lines {
		result = append(result, line.String())
	}

	return strings.Join(result, "\n")
}
