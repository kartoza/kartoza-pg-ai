package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

// SettingItem represents a single setting
type SettingItem struct {
	Name        string
	Description string
	Type        string // "toggle", "number", "text"
	GetValue    func(*config.Config) string
	Toggle      func(*config.Config) // For toggle types
}

// SettingsModel represents the settings screen
type SettingsModel struct {
	width        int
	height       int
	cfg          *config.Config
	selectedItem int
	items        []SettingItem
}

// settingsChangedMsg indicates settings were changed
type settingsChangedMsg struct{}

// NewSettingsModel creates a new settings model
func NewSettingsModel(cfg *config.Config) *SettingsModel {
	items := []SettingItem{
		{
			Name:        "Vim Mode",
			Description: "Use vim-style keybindings in query editor",
			Type:        "toggle",
			GetValue: func(c *config.Config) string {
				if c.Settings.VimModeEnabled {
					return "Enabled"
				}
				return "Disabled"
			},
			Toggle: func(c *config.Config) {
				c.Settings.VimModeEnabled = !c.Settings.VimModeEnabled
			},
		},
		{
			Name:        "Spatial Operations",
			Description: "Enable PostGIS spatial query support",
			Type:        "toggle",
			GetValue: func(c *config.Config) string {
				if c.Settings.EnableSpatialOps {
					return "Enabled"
				}
				return "Disabled"
			},
			Toggle: func(c *config.Config) {
				c.Settings.EnableSpatialOps = !c.Settings.EnableSpatialOps
			},
		},
		{
			Name:        "Neural Network",
			Description: "Use NN for query generation (requires training)",
			Type:        "toggle",
			GetValue: func(c *config.Config) string {
				if c.Settings.NeuralNetEnabled {
					return "Enabled"
				}
				return "Disabled"
			},
			Toggle: func(c *config.Config) {
				c.Settings.NeuralNetEnabled = !c.Settings.NeuralNetEnabled
			},
		},
		{
			Name:        "Max History Size",
			Description: "Maximum number of queries to keep in history",
			Type:        "display",
			GetValue: func(c *config.Config) string {
				return fmt.Sprintf("%d", c.Settings.MaxHistorySize)
			},
		},
		{
			Name:        "Default Row Limit",
			Description: "Default number of rows to fetch per page",
			Type:        "display",
			GetValue: func(c *config.Config) string {
				return fmt.Sprintf("%d", c.Settings.DefaultRowLimit)
			},
		},
		{
			Name:        "Schema Cache",
			Description: "Schema is cached until manually refreshed (R on connections screen)",
			Type:        "display",
			GetValue: func(c *config.Config) string {
				return "Persistent"
			},
		},
		{
			Name:        "NN Training Status",
			Description: "Train NN model (needs 10+ queries in history)",
			Type:        "display",
			GetValue: func(c *config.Config) string {
				historyCount := len(c.QueryHistory)
				if historyCount >= 10 {
					return fmt.Sprintf("%d queries (ready)", historyCount)
				}
				return fmt.Sprintf("%d queries (need 10+)", historyCount)
			},
		},
	}

	return &SettingsModel{
		cfg:          cfg,
		selectedItem: 0,
		items:        items,
	}
}

// Init initializes the settings model
func (m *SettingsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings model
func (m *SettingsModel) Update(msg tea.Msg) (*SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, func() tea.Msg {
				return goToMenuMsg{}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if len(m.items) > 0 {
				m.selectedItem--
				if m.selectedItem < 0 {
					m.selectedItem = len(m.items) - 1
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if len(m.items) > 0 {
				m.selectedItem++
				if m.selectedItem >= len(m.items) {
					m.selectedItem = 0
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if m.selectedItem < len(m.items) {
				item := m.items[m.selectedItem]
				if item.Type == "toggle" && item.Toggle != nil {
					item.Toggle(m.cfg)
					m.cfg.Save()
					return m, func() tea.Msg {
						return settingsChangedMsg{}
					}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the settings screen
func (m *SettingsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Settings")
	content := m.renderContent()
	helpText := "↑/k: up • ↓/j: down • enter/space: toggle • esc: back • ctrl+c: quit"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

func (m *SettingsModel) renderContent() string {
	var sections []string

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Align(lipgloss.Center)
	sections = append(sections, subtitleStyle.Render("Configure application preferences"))
	sections = append(sections, "")

	if m.cfg == nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(ColorRed).
			Align(lipgloss.Center)
		sections = append(sections, errorStyle.Render("Error: Configuration not loaded"))
		return lipgloss.JoinVertical(lipgloss.Center, sections...)
	}

	// Settings table
	sections = append(sections, m.renderSettingsTable())

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func (m *SettingsModel) renderSettingsTable() string {
	// Table styles
	borderStyle := lipgloss.NewStyle().
		Foreground(ColorOrange)

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	// Column widths
	nameWidth := 22
	valueWidth := 15
	descWidth := 40

	// Build table header
	headerLine := borderStyle.Render("┌") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", nameWidth)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", valueWidth)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", descWidth)) +
		borderStyle.Render("┐")

	headerContent := borderStyle.Render("│") +
		headerStyle.Render(padRight(" ", 3)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Setting", nameWidth)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Value", valueWidth)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Description", descWidth)) +
		borderStyle.Render("│")

	separatorLine := borderStyle.Render("├") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", nameWidth)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", valueWidth)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", descWidth)) +
		borderStyle.Render("┤")

	var rows []string
	rows = append(rows, headerLine)
	rows = append(rows, headerContent)
	rows = append(rows, separatorLine)

	// Build setting rows
	for i, item := range m.items {
		isSelected := i == m.selectedItem

		// Selector
		var selector string
		if isSelected {
			selector = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render("▶")
		} else {
			selector = " "
		}

		// Type indicator
		var typeIcon string
		if item.Type == "toggle" {
			typeIcon = lipgloss.NewStyle().Foreground(ColorBlue).Render("◉")
		} else {
			typeIcon = lipgloss.NewStyle().Foreground(ColorGray).Render("○")
		}

		firstCell := selector + typeIcon + " "

		// Name style
		nameStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if isSelected {
			nameStyle = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		}

		// Value style - green for enabled, gray for disabled/display
		value := item.GetValue(m.cfg)
		valueStyle := lipgloss.NewStyle().Foreground(ColorGray)
		if value == "Enabled" {
			valueStyle = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
		} else if value == "Disabled" {
			valueStyle = lipgloss.NewStyle().Foreground(ColorRed)
		} else {
			valueStyle = lipgloss.NewStyle().Foreground(ColorCyan)
		}

		// Description style
		descStyle := lipgloss.NewStyle().Foreground(ColorGray)

		row := borderStyle.Render("│") +
			firstCell +
			borderStyle.Render("│") +
			nameStyle.Render(padRight(" "+truncateStr(item.Name, nameWidth-2), nameWidth)) +
			borderStyle.Render("│") +
			valueStyle.Render(padRight(" "+truncateStr(value, valueWidth-2), valueWidth)) +
			borderStyle.Render("│") +
			descStyle.Render(padRight(" "+truncateStr(item.Description, descWidth-2), descWidth)) +
			borderStyle.Render("│")

		rows = append(rows, row)
	}

	// Footer line
	footerLine := borderStyle.Render("└") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", nameWidth)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", valueWidth)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", descWidth)) +
		borderStyle.Render("┘")
	rows = append(rows, footerLine)

	// Legend
	rows = append(rows, "")
	legendStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center)
	rows = append(rows, legendStyle.Render("◉ toggleable  ○ read-only"))

	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}
