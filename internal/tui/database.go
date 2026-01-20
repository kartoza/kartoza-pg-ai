package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/kartoza/kartoza-pg-ai/internal/postgres"
)

// DatabaseModel represents the database connection screen
type DatabaseModel struct {
	services       []postgres.ServiceEntry
	selectedItem   int
	width          int
	height         int
	loading        bool
	spinner        spinner.Model
	error          string
	testingService string
	cfg            *config.Config
}

// servicesLoadedMsg indicates services have been loaded
type servicesLoadedMsg struct {
	services []postgres.ServiceEntry
	err      error
}

// serviceTestedMsg indicates a service connection test completed
type serviceTestedMsg struct {
	serviceName string
	err         error
}

// serviceSelectedMsg indicates a service was selected
type serviceSelectedMsg struct {
	service postgres.ServiceEntry
}

// forceReharvest indicates user wants to reharvest schema for selected service
type forceReharvestMsg struct {
	service postgres.ServiceEntry
}

// editServiceMsg indicates user wants to edit a service entry
type editServiceMsg struct {
	service *postgres.ServiceEntry // nil for new entry
}

// newServiceMsg indicates user wants to create a new service entry
type newServiceMsg struct{}

// NewDatabaseModel creates a new database connection model
func NewDatabaseModel() *DatabaseModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorOrange)

	cfg, _ := config.Load()

	return &DatabaseModel{
		services:     []postgres.ServiceEntry{},
		selectedItem: 0,
		spinner:      s,
		loading:      true,
		cfg:          cfg,
	}
}

// Init initializes the database model
func (m *DatabaseModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadServices(),
	)
}

// loadServices loads the pg_service.conf file
func (m *DatabaseModel) loadServices() tea.Cmd {
	return func() tea.Msg {
		services, err := postgres.ParsePGServiceFile()
		return servicesLoadedMsg{services: services, err: err}
	}
}

// testService tests the connection to a service
func (m *DatabaseModel) testService(service postgres.ServiceEntry) tea.Cmd {
	return func() tea.Msg {
		err := service.TestConnection()
		return serviceTestedMsg{serviceName: service.Name, err: err}
	}
}

// Update handles messages for the database model
func (m *DatabaseModel) Update(msg tea.Msg) (*DatabaseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case servicesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			// Check if it's because file doesn't exist - auto-prompt to create
			if !postgres.PGServiceFileExists() {
				return m, func() tea.Msg {
					return newServiceMsg{}
				}
			}
			m.error = msg.err.Error()
		} else {
			m.services = msg.services
			// If file exists but is empty, prompt to add first entry
			if len(m.services) == 0 {
				return m, func() tea.Msg {
					return newServiceMsg{}
				}
			}
		}
		return m, nil

	case serviceTestedMsg:
		m.testingService = ""
		if msg.err != nil {
			m.error = fmt.Sprintf("Connection to '%s' failed: %v", msg.serviceName, msg.err)
		} else {
			// Connection successful, select this service
			for _, s := range m.services {
				if s.Name == msg.serviceName {
					return m, func() tea.Msg {
						return serviceSelectedMsg{service: s}
					}
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Clear error on any key
		if m.error != "" && msg.String() != "enter" && msg.String() != " " {
			m.error = ""
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, func() tea.Msg {
				return goToMenuMsg{} // Go back to menu
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if len(m.services) > 0 {
				m.selectedItem--
				if m.selectedItem < 0 {
					m.selectedItem = len(m.services) - 1
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if len(m.services) > 0 {
				m.selectedItem++
				if m.selectedItem >= len(m.services) {
					m.selectedItem = 0
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", " "))):
			if len(m.services) > 0 && m.selectedItem < len(m.services) {
				service := m.services[m.selectedItem]
				m.testingService = service.Name
				m.error = ""
				return m, tea.Batch(
					m.spinner.Tick,
					m.testService(service),
				)
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			m.error = ""
			return m, tea.Batch(
				m.spinner.Tick,
				m.loadServices(),
			)

		case key.Matches(msg, key.NewBinding(key.WithKeys("R"))):
			// Force reharvest schema for selected service
			if len(m.services) > 0 && m.selectedItem < len(m.services) {
				service := m.services[m.selectedItem]
				// Invalidate cache for this service
				if m.cfg != nil {
					delete(m.cfg.CachedSchemas, service.Name)
					m.cfg.Save()
				}
				m.testingService = service.Name
				m.error = ""
				return m, tea.Batch(
					m.spinner.Tick,
					m.testService(service),
				)
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
			// Edit selected service
			if len(m.services) > 0 && m.selectedItem < len(m.services) {
				service := m.services[m.selectedItem]
				return m, func() tea.Msg {
					return editServiceMsg{service: &service}
				}
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			// Create new service
			return m, func() tea.Msg {
				return newServiceMsg{}
			}
		}
	}

	return m, nil
}

// View renders the database connection screen
func (m *DatabaseModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Database Connections")
	content := m.renderContent()
	helpText := "↑/k: up • ↓/j: down • enter: connect • n: new • e: edit • r: refresh • R: reharvest • esc: back"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

func (m *DatabaseModel) renderContent() string {
	var sections []string

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Align(lipgloss.Center)
	sections = append(sections, subtitleStyle.Render("Select a database service from pg_service.conf"))
	sections = append(sections, "")

	// Loading state
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(ColorOrange).
			Align(lipgloss.Center)
		sections = append(sections, loadingStyle.Render(m.spinner.View()+" Loading pg_service.conf..."))
		return lipgloss.JoinVertical(lipgloss.Center, sections...)
	}

	// Error display
	if m.error != "" {
		errorBox := BoxStyle.Copy().
			BorderForeground(ColorRed).
			Width(50).
			Align(lipgloss.Center)
		sections = append(sections, errorBox.Render(ErrorStyle.Render("Error: "+m.error)))
		sections = append(sections, "")
	}

	// Testing connection
	if m.testingService != "" {
		testingStyle := lipgloss.NewStyle().
			Foreground(ColorOrange).
			Align(lipgloss.Center)
		sections = append(sections, testingStyle.Render(m.spinner.View()+" Testing connection to '"+m.testingService+"'..."))
		sections = append(sections, "")
	}

	// Services list
	if len(m.services) == 0 {
		noServicesStyle := lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Align(lipgloss.Center)
		sections = append(sections, noServicesStyle.Render("No services found in pg_service.conf"))
		sections = append(sections, "")
		sections = append(sections, noServicesStyle.Render("Create ~/.pg_service.conf with your database connections"))
	} else {
		sections = append(sections, m.renderServicesList())
	}

	// Legend
	sections = append(sections, "")
	legendStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center)
	sections = append(sections, legendStyle.Render("● cached schema  ○ not harvested"))

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func (m *DatabaseModel) renderServicesList() string {
	// Table styles
	tableWidth := 70
	borderStyle := lipgloss.NewStyle().
		Foreground(ColorOrange)

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	// Build table header
	headerLine := borderStyle.Render("┌") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 20)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 25)) +
		borderStyle.Render("┬") +
		borderStyle.Render(repeatChar("─", 15)) +
		borderStyle.Render("┐")

	headerContent := borderStyle.Render("│") +
		headerStyle.Render(padRight(" ", 3)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Service", 20)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Host", 25)) +
		borderStyle.Render("│") +
		headerStyle.Render(padRight(" Database", 15)) +
		borderStyle.Render("│")

	separatorLine := borderStyle.Render("├") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 20)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 25)) +
		borderStyle.Render("┼") +
		borderStyle.Render(repeatChar("─", 15)) +
		borderStyle.Render("┤")

	var rows []string
	rows = append(rows, headerLine)
	rows = append(rows, headerContent)
	rows = append(rows, separatorLine)

	// Build service rows
	for i, service := range m.services {
		// Check if schema is cached
		isCached := m.cfg != nil && m.cfg.IsSchemaCacheValid(service.Name)

		// Determine colors based on selection
		isSelected := i == m.selectedItem

		// Status icon - render separately to avoid ANSI code issues with padding
		var statusIcon string
		if isCached {
			statusIcon = lipgloss.NewStyle().Foreground(ColorGreen).Render("●")
		} else {
			statusIcon = lipgloss.NewStyle().Foreground(ColorGray).Render("○")
		}

		// Selector
		var selector string
		if isSelected {
			selector = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true).Render("▶")
		} else {
			selector = " "
		}

		// Service name style
		nameStyle := lipgloss.NewStyle().Foreground(ColorBlue)
		if isSelected {
			nameStyle = lipgloss.NewStyle().Foreground(ColorOrange).Bold(true)
		}

		// Build the first cell content: selector + status icon (fixed width cell)
		// We render each part separately and then combine
		firstCell := selector + statusIcon + " "

		row := borderStyle.Render("│") +
			firstCell +
			borderStyle.Render("│") +
			nameStyle.Render(padRight(" "+truncateStr(service.Name, 18), 20)) +
			borderStyle.Render("│") +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(padRight(" "+truncateStr(service.Host, 23), 25)) +
			borderStyle.Render("│") +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(padRight(" "+truncateStr(service.DBName, 13), 15)) +
			borderStyle.Render("│")

		rows = append(rows, row)
	}

	// Footer line
	footerLine := borderStyle.Render("└") +
		borderStyle.Render(repeatChar("─", 3)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 20)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 25)) +
		borderStyle.Render("┴") +
		borderStyle.Render(repeatChar("─", 15)) +
		borderStyle.Render("┘")
	rows = append(rows, footerLine)

	// Container for the table
	tableContainer := lipgloss.NewStyle().
		Width(tableWidth).
		Align(lipgloss.Center)

	return tableContainer.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// Helper functions for table rendering
func repeatChar(char string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	result := s
	for len(result) < length {
		result += " "
	}
	return result
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-2] + ".."
	}
	return s
}
