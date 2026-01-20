package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/postgres"
)

// ServiceEditorModel represents the service entry editor
type ServiceEditorModel struct {
	width        int
	height       int
	inputs       []textinput.Model
	focusedInput int
	isNew        bool // true if creating new, false if editing
	originalName string
	error        string
}

// Field indices
const (
	fieldName = iota
	fieldHost
	fieldPort
	fieldDBName
	fieldUser
	fieldPassword
	fieldSSLMode
)

// serviceSavedMsg indicates service was saved
type serviceSavedMsg struct {
	entry postgres.ServiceEntry
}

// serviceEditorCancelledMsg indicates editor was cancelled
type serviceEditorCancelledMsg struct{}

// NewServiceEditorModel creates a new service editor
func NewServiceEditorModel(entry *postgres.ServiceEntry) *ServiceEditorModel {
	inputs := make([]textinput.Model, 7)

	// Service Name
	inputs[fieldName] = textinput.New()
	inputs[fieldName].Placeholder = "my-database"
	inputs[fieldName].CharLimit = 50
	inputs[fieldName].Width = 40
	inputs[fieldName].Prompt = ""

	// Host
	inputs[fieldHost] = textinput.New()
	inputs[fieldHost].Placeholder = "localhost"
	inputs[fieldHost].CharLimit = 100
	inputs[fieldHost].Width = 40
	inputs[fieldHost].Prompt = ""

	// Port
	inputs[fieldPort] = textinput.New()
	inputs[fieldPort].Placeholder = "5432"
	inputs[fieldPort].CharLimit = 10
	inputs[fieldPort].Width = 40
	inputs[fieldPort].Prompt = ""

	// Database Name
	inputs[fieldDBName] = textinput.New()
	inputs[fieldDBName].Placeholder = "postgres"
	inputs[fieldDBName].CharLimit = 100
	inputs[fieldDBName].Width = 40
	inputs[fieldDBName].Prompt = ""

	// User
	inputs[fieldUser] = textinput.New()
	inputs[fieldUser].Placeholder = "postgres"
	inputs[fieldUser].CharLimit = 100
	inputs[fieldUser].Width = 40
	inputs[fieldUser].Prompt = ""

	// Password
	inputs[fieldPassword] = textinput.New()
	inputs[fieldPassword].Placeholder = "••••••••"
	inputs[fieldPassword].CharLimit = 100
	inputs[fieldPassword].Width = 40
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].Prompt = ""

	// SSL Mode
	inputs[fieldSSLMode] = textinput.New()
	inputs[fieldSSLMode].Placeholder = "prefer"
	inputs[fieldSSLMode].CharLimit = 20
	inputs[fieldSSLMode].Width = 40
	inputs[fieldSSLMode].Prompt = ""

	isNew := entry == nil
	originalName := ""

	// Pre-populate if editing
	if entry != nil {
		inputs[fieldName].SetValue(entry.Name)
		inputs[fieldHost].SetValue(entry.Host)
		inputs[fieldPort].SetValue(entry.Port)
		inputs[fieldDBName].SetValue(entry.DBName)
		inputs[fieldUser].SetValue(entry.User)
		inputs[fieldPassword].SetValue(entry.Password)
		inputs[fieldSSLMode].SetValue(entry.SSLMode)
		originalName = entry.Name
	} else {
		// Set defaults for new entry
		inputs[fieldPort].SetValue("5432")
		inputs[fieldSSLMode].SetValue("prefer")
	}

	// Focus first input
	inputs[fieldName].Focus()

	return &ServiceEditorModel{
		inputs:       inputs,
		focusedInput: fieldName,
		isNew:        isNew,
		originalName: originalName,
	}
}

// Init initializes the service editor
func (m *ServiceEditorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the service editor
func (m *ServiceEditorModel) Update(msg tea.Msg) (*ServiceEditorModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, func() tea.Msg {
				return serviceEditorCancelledMsg{}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "down"))):
			m.nextInput()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab", "up"))):
			m.prevInput()
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s", "enter"))):
			// Validate and save
			if m.inputs[fieldName].Value() == "" {
				m.error = "Service name is required"
				return m, nil
			}
			if m.inputs[fieldHost].Value() == "" {
				m.error = "Host is required"
				return m, nil
			}

			entry := postgres.ServiceEntry{
				Name:     m.inputs[fieldName].Value(),
				Host:     m.inputs[fieldHost].Value(),
				Port:     m.inputs[fieldPort].Value(),
				DBName:   m.inputs[fieldDBName].Value(),
				User:     m.inputs[fieldUser].Value(),
				Password: m.inputs[fieldPassword].Value(),
				SSLMode:  m.inputs[fieldSSLMode].Value(),
				Options:  make(map[string]string),
			}

			// If editing and name changed, delete old entry first
			if !m.isNew && m.originalName != entry.Name {
				postgres.DeleteServiceEntry(m.originalName)
			}

			if err := postgres.SaveServiceEntry(entry); err != nil {
				m.error = "Failed to save: " + err.Error()
				return m, nil
			}

			return m, func() tea.Msg {
				return serviceSavedMsg{entry: entry}
			}
		}

		// Clear error on typing
		if m.error != "" {
			m.error = ""
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.inputs[m.focusedInput], cmd = m.inputs[m.focusedInput].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ServiceEditorModel) nextInput() {
	m.inputs[m.focusedInput].Blur()
	m.focusedInput = (m.focusedInput + 1) % len(m.inputs)
	m.inputs[m.focusedInput].Focus()
}

func (m *ServiceEditorModel) prevInput() {
	m.inputs[m.focusedInput].Blur()
	m.focusedInput--
	if m.focusedInput < 0 {
		m.focusedInput = len(m.inputs) - 1
	}
	m.inputs[m.focusedInput].Focus()
}

// View renders the service editor
func (m *ServiceEditorModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var title string
	if m.isNew {
		title = "New Connection"
	} else {
		title = "Edit Connection"
	}
	header := RenderHeader(title)
	content := m.renderContent()
	helpText := "tab/↓: next • shift+tab/↑: prev • ctrl+s/enter: save • esc: cancel"
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

func (m *ServiceEditorModel) renderContent() string {
	var sections []string

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Align(lipgloss.Center)
	if m.isNew {
		sections = append(sections, subtitleStyle.Render("Enter connection details for a new PostgreSQL service"))
	} else {
		sections = append(sections, subtitleStyle.Render("Edit connection details for: "+m.originalName))
	}
	sections = append(sections, "")

	// Error display
	if m.error != "" {
		errorBox := BoxStyle.Copy().
			BorderForeground(ColorRed).
			Width(50).
			Align(lipgloss.Center)
		sections = append(sections, errorBox.Render(ErrorStyle.Render("Error: "+m.error)))
		sections = append(sections, "")
	}

	// Form fields
	labelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(15).
		Align(lipgloss.Right).
		PaddingRight(1)

	focusedLabelStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Width(15).
		Align(lipgloss.Right).
		PaddingRight(1)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray).
		Padding(0, 1)

	focusedInputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange).
		Padding(0, 1)

	labels := []string{
		"Service Name:",
		"Host:",
		"Port:",
		"Database:",
		"User:",
		"Password:",
		"SSL Mode:",
	}

	for i, label := range labels {
		var labelStr string
		var inputBox string

		if i == m.focusedInput {
			labelStr = focusedLabelStyle.Render(label)
			inputBox = focusedInputStyle.Render(m.inputs[i].View())
		} else {
			labelStr = labelStyle.Render(label)
			inputBox = inputStyle.Render(m.inputs[i].View())
		}

		row := lipgloss.JoinHorizontal(lipgloss.Center, labelStr, inputBox)
		sections = append(sections, row)
	}

	// SSL Mode hints
	sections = append(sections, "")
	hintStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true).
		Align(lipgloss.Center)
	sections = append(sections, hintStyle.Render("SSL modes: disable, allow, prefer, require, verify-ca, verify-full"))

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}
