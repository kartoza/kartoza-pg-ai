package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

// HistoryModel represents the query history screen
type HistoryModel struct {
	width        int
	height       int
	entries      []config.QueryHistoryEntry
	selectedItem int
	serviceName  string
	cfg          *config.Config
}

// rerunQueryMsg indicates user wants to rerun a query
type rerunQueryMsg struct {
	query string
}

// NewHistoryModel creates a new history model for a specific service
func NewHistoryModel(serviceName string) *HistoryModel {
	cfg, _ := config.Load()

	// Filter entries for this service
	var entries []config.QueryHistoryEntry
	if cfg != nil {
		for _, entry := range cfg.QueryHistory {
			if entry.ServiceName == serviceName {
				entries = append(entries, entry)
			}
		}
	}

	return &HistoryModel{
		entries:      entries,
		selectedItem: 0,
		serviceName:  serviceName,
		cfg:          cfg,
	}
}

// Init initializes the history model
func (m *HistoryModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the history model
func (m *HistoryModel) Update(msg tea.Msg) (*HistoryModel, tea.Cmd) {
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
			if len(m.entries) > 0 {
				m.selectedItem--
				if m.selectedItem < 0 {
					m.selectedItem = len(m.entries) - 1
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if len(m.entries) > 0 {
				m.selectedItem++
				if m.selectedItem >= len(m.entries) {
					m.selectedItem = 0
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if len(m.entries) > 0 && m.selectedItem < len(m.entries) {
				entry := m.entries[m.selectedItem]
				return m, func() tea.Msg {
					return rerunQueryMsg{query: entry.NaturalQuery}
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			// Delete selected entry
			if len(m.entries) > 0 && m.selectedItem < len(m.entries) {
				m.deleteEntry(m.selectedItem)
				if m.selectedItem >= len(m.entries) && m.selectedItem > 0 {
					m.selectedItem--
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *HistoryModel) deleteEntry(index int) {
	if m.cfg == nil || index >= len(m.entries) {
		return
	}

	// Remove from local list
	entry := m.entries[index]
	m.entries = append(m.entries[:index], m.entries[index+1:]...)

	// Remove from config (find matching entry)
	for i, e := range m.cfg.QueryHistory {
		if e.Timestamp == entry.Timestamp && e.NaturalQuery == entry.NaturalQuery {
			m.cfg.QueryHistory = append(m.cfg.QueryHistory[:i], m.cfg.QueryHistory[i+1:]...)
			m.cfg.Save()
			break
		}
	}
}

// View renders the history screen
func (m *HistoryModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Query History - " + m.serviceName)
	content := m.renderContent()
	helpText := "↑/k: up • ↓/j: down • enter: rerun • d: delete • esc: back • ctrl+c: quit"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

func (m *HistoryModel) renderContent() string {
	if len(m.entries) == 0 {
		noHistory := lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Render("No query history for " + m.serviceName)
		return noHistory
	}

	// Table styles
	borderStyle := lipgloss.NewStyle().
		Foreground(ColorOrange)

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	// Build table header
	headerLine := borderStyle.Render("┌") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 18)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 45)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 8)) +
		borderStyle.Render("┐")

	headerContent := borderStyle.Render("│") +
		headerStyle.Render(padRight(" ", 3)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Time", 18)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Query", 45)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Rows", 8)) +
		borderStyle.Render("│")

	separatorLine := borderStyle.Render("├") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 18)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 45)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 8)) +
		borderStyle.Render("┤")

	var rows []string
	rows = append(rows, headerLine)
	rows = append(rows, headerContent)
	rows = append(rows, separatorLine)

	// Build entry rows (limit to visible)
	maxVisible := 15
	for i, entry := range m.entries {
		if i >= maxVisible {
			moreStyle := lipgloss.NewStyle().
				Foreground(ColorGray).
				Italic(true)
			rows = append(rows, moreStyle.Render(fmt.Sprintf("... and %d more entries", len(m.entries)-maxVisible)))
			break
		}

		isSelected := i == m.selectedItem

		// Success indicator - render separately to avoid ANSI code issues
		var statusIcon string
		if entry.Success {
			statusIcon = lipgloss.NewStyle().Foreground(ColorGreen).Render("●")
		} else {
			statusIcon = lipgloss.NewStyle().Foreground(ColorRed).Render("○")
		}

		// Selector
		var selector string
		if isSelected {
			selector = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render("▶")
		} else {
			selector = " "
		}

		// Build first cell: selector + status icon
		firstCell := selector + statusIcon + " "

		// Query style
		queryStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if isSelected {
			queryStyle = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		}

		timeStr := entry.Timestamp.Format("01-02 15:04")
		rowsStr := fmt.Sprintf("%d", entry.RowsAffected)

		row := borderStyle.Render("│") +
			firstCell +
			borderStyle.Render("│") +
			lipgloss.NewStyle().Foreground(ColorGray).Render(padRight(" "+timeStr, 18)) +
			borderStyle.Render("│") +
			queryStyle.Render(padRight(" "+truncateStr(entry.NaturalQuery, 43), 45)) +
			borderStyle.Render("│") +
			lipgloss.NewStyle().Foreground(ColorBlue).Render(padRight(" "+rowsStr, 8)) +
			borderStyle.Render("│")

		rows = append(rows, row)
	}

	// Footer line
	footerLine := borderStyle.Render("└") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 18)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 45)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 8)) +
		borderStyle.Render("┘")
	rows = append(rows, footerLine)

	// Legend
	rows = append(rows, "")
	legendStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center)
	rows = append(rows, legendStyle.Render("● success  ○ failed"))

	// Show details of selected entry
	if m.selectedItem < len(m.entries) {
		entry := m.entries[m.selectedItem]
		rows = append(rows, "")

		detailBox := BoxStyle.Copy().
			BorderForeground(ColorBlue).
			Width(min(80, m.width-10))

		sqlStyle := lipgloss.NewStyle().Foreground(ColorCyan)
		labelStyle := lipgloss.NewStyle().Foreground(ColorGray)

		details := lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("Generated SQL:"),
			sqlStyle.Render(entry.GeneratedSQL),
			"",
			labelStyle.Render(fmt.Sprintf("Execution time: %.2fms", entry.ExecutionTime)),
		)

		rows = append(rows, detailBox.Render(details))
	}

	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}
