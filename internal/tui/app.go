package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/kartoza/kartoza-pg-ai/internal/postgres"
)

// Screen represents the current screen being displayed
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenQuery
	ScreenDatabase
	ScreenHistory
	ScreenSettings
	ScreenHarvest
	ScreenServiceEditor
)

// AppModel is the main application model
type AppModel struct {
	screen         Screen
	width          int
	height         int
	menu           *MenuModel
	database       *DatabaseModel
	query          *QueryModel
	harvest        *HarvestModel
	history        *HistoryModel
	settings       *SettingsModel
	serviceEditor  *ServiceEditorModel
	spinner        spinner.Model
	loading        bool
	loadingMessage string

	// State
	cfg            *config.Config
	activeService  *postgres.ServiceEntry
	activeSchema   *config.SchemaCache
	services       []postgres.ServiceEntry
	pendingScreen  Screen // Screen to navigate to after connection
}

// blinkTickMsg for status bar blinking
type blinkTickMsg time.Time

// goToMenuMsg indicates request to return to menu screen
type goToMenuMsg struct{}

// NewAppModel creates a new application model
func NewAppModel() *AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorOrange)

	cfg, _ := config.Load()

	return &AppModel{
		screen:   ScreenMenu,
		menu:     NewMenuModel(),
		database: NewDatabaseModel(),
		spinner:  s,
		cfg:      cfg,
	}
}

// Init initializes the application
func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.menu.Init(),
		m.startBlinkTicker(),
		m.loadInitialState(),
	)
}

func (m *AppModel) startBlinkTicker() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkTickMsg(t)
	})
}

func (m *AppModel) loadInitialState() tea.Cmd {
	// Don't auto-connect - stay on the menu and let user choose
	return nil
}

// Update handles all messages for the application
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propagate to all sub-models
		m.menu.width = msg.Width
		m.menu.height = msg.Height
		m.database.width = msg.Width
		m.database.height = msg.Height
		if m.query != nil {
			m.query.width = msg.Width
			m.query.height = msg.Height
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case blinkTickMsg:
		GlobalAppState.BlinkOn = !GlobalAppState.BlinkOn
		return m, m.startBlinkTicker()

	case menuActionMsg:
		switch msg.action {
		case MenuQuery:
			if m.activeService != nil && m.activeSchema != nil {
				m.screen = ScreenQuery
				if m.query == nil {
					m.query = NewQueryModel(m.activeService, m.activeSchema)
				}
				m.query.width = m.width
				m.query.height = m.height
				return m, m.query.Init()
			}
			// No service connected, go to database selection
			m.screen = ScreenDatabase
			m.database.width = m.width
			m.database.height = m.height
			return m, m.database.Init()

		case MenuDatabases:
			m.screen = ScreenDatabase
			m.database = NewDatabaseModel()
			m.database.width = m.width
			m.database.height = m.height
			return m, m.database.Init()

		case MenuHistory:
			if m.activeService != nil {
				m.screen = ScreenHistory
				m.history = NewHistoryModel(m.activeService.Name)
				m.history.width = m.width
				m.history.height = m.height
				return m, m.history.Init()
			}
			// No active service - go to database selection first, then history
			m.pendingScreen = ScreenHistory
			m.screen = ScreenDatabase
			m.database = NewDatabaseModel()
			m.database.width = m.width
			m.database.height = m.height
			return m, m.database.Init()

		case MenuSettings:
			m.screen = ScreenSettings
			m.settings = NewSettingsModel(m.cfg)
			m.settings.width = m.width
			m.settings.height = m.height
			return m, m.settings.Init()

		case MenuQuit:
			return m, tea.Quit
		}

	case serviceSelectedMsg:
		m.activeService = &msg.service
		GlobalAppState.IsConnected = true
		GlobalAppState.ActiveService = msg.service.Name
		GlobalAppState.Status = "Connected"

		// Save to config
		if m.cfg != nil {
			m.cfg.ActiveService = msg.service.Name
			m.cfg.Save()
		}

		// Load or use cached schema
		if m.cfg != nil && m.cfg.IsSchemaCacheValid(msg.service.Name) {
			m.activeSchema = m.cfg.CachedSchemas[msg.service.Name]
			GlobalAppState.SchemaLoaded = true
			GlobalAppState.TablesCount = len(m.activeSchema.Tables)
			GlobalAppState.HasPostGIS = m.activeSchema.HasPostGIS

			// Check if we have a pending screen to navigate to
			if m.pendingScreen == ScreenHistory {
				m.pendingScreen = ScreenMenu // Reset pending
				m.screen = ScreenHistory
				m.history = NewHistoryModel(m.activeService.Name)
				m.history.width = m.width
				m.history.height = m.height
				return m, m.history.Init()
			}

			// Go to query screen
			m.screen = ScreenQuery
			m.query = NewQueryModel(m.activeService, m.activeSchema)
			m.query.width = m.width
			m.query.height = m.height
			return m, m.query.Init()
		}

		// Need to harvest schema - show harvest screen with progress bar
		m.screen = ScreenHarvest
		m.harvest = NewHarvestModel(msg.service)
		m.harvest.width = m.width
		m.harvest.height = m.height
		return m, m.harvest.Init()

	case schemaLoadedMsg:
		m.loading = false
		if msg.err == nil && msg.schema != nil {
			m.activeSchema = msg.schema
			GlobalAppState.SchemaLoaded = true
			GlobalAppState.TablesCount = len(msg.schema.Tables)
			GlobalAppState.HasPostGIS = msg.schema.HasPostGIS

			// Cache the schema
			if m.cfg != nil && m.activeService != nil {
				m.cfg.CachedSchemas[m.activeService.Name] = msg.schema
				m.cfg.Save()
			}

			// Check if we have a pending screen to navigate to
			if m.pendingScreen == ScreenHistory {
				m.pendingScreen = ScreenMenu // Reset pending
				m.screen = ScreenHistory
				m.history = NewHistoryModel(m.activeService.Name)
				m.history.width = m.width
				m.history.height = m.height
				return m, m.history.Init()
			}

			// Go to query screen
			m.screen = ScreenQuery
			m.query = NewQueryModel(m.activeService, m.activeSchema)
			m.query.width = m.width
			m.query.height = m.height
			return m, m.query.Init()
		} else if msg.err != nil {
			// Handle error - go back to database selection
			m.screen = ScreenDatabase
			m.database = NewDatabaseModel()
			m.database.width = m.width
			m.database.height = m.height
			return m, m.database.Init()
		}

	case rerunQueryMsg:
		// User wants to rerun a query from history
		if m.activeService != nil && m.activeSchema != nil {
			m.screen = ScreenQuery
			m.query = NewQueryModel(m.activeService, m.activeSchema)
			// Set the query text in the editor
			m.query.SetInitialQuery(msg.query)
			m.query.width = m.width
			m.query.height = m.height
			return m, m.query.Init()
		}

	case harvestCancelledMsg:
		// User cancelled harvesting - go back to database selection
		m.activeService = nil
		GlobalAppState.IsConnected = false
		GlobalAppState.ActiveService = ""
		GlobalAppState.Status = "Ready"
		m.screen = ScreenDatabase
		m.database = NewDatabaseModel()
		m.database.width = m.width
		m.database.height = m.height
		return m, m.database.Init()

	case goToMenuMsg:
		// Return to menu screen
		m.screen = ScreenMenu
		return m, nil

	case editServiceMsg:
		// Open service editor for editing
		m.screen = ScreenServiceEditor
		m.serviceEditor = NewServiceEditorModel(msg.service)
		m.serviceEditor.width = m.width
		m.serviceEditor.height = m.height
		return m, m.serviceEditor.Init()

	case newServiceMsg:
		// Open service editor for new entry
		m.screen = ScreenServiceEditor
		m.serviceEditor = NewServiceEditorModel(nil)
		m.serviceEditor.width = m.width
		m.serviceEditor.height = m.height
		return m, m.serviceEditor.Init()

	case serviceSavedMsg:
		// Service was saved, return to database screen and refresh
		m.screen = ScreenDatabase
		m.database = NewDatabaseModel()
		m.database.width = m.width
		m.database.height = m.height
		return m, m.database.Init()

	case serviceEditorCancelledMsg:
		// Return to database screen without changes
		m.screen = ScreenDatabase
		return m, nil
	}

	// Route to current screen
	switch m.screen {
	case ScreenMenu:
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		cmds = append(cmds, cmd)

	case ScreenDatabase:
		var cmd tea.Cmd
		m.database, cmd = m.database.Update(msg)
		cmds = append(cmds, cmd)

	case ScreenQuery:
		if m.query != nil {
			var cmd tea.Cmd
			m.query, cmd = m.query.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ScreenHarvest:
		if m.harvest != nil {
			var cmd tea.Cmd
			m.harvest, cmd = m.harvest.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ScreenHistory:
		if m.history != nil {
			var cmd tea.Cmd
			m.history, cmd = m.history.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ScreenSettings:
		if m.settings != nil {
			var cmd tea.Cmd
			m.settings, cmd = m.settings.Update(msg)
			cmds = append(cmds, cmd)
		}

	case ScreenServiceEditor:
		if m.serviceEditor != nil {
			var cmd tea.Cmd
			m.serviceEditor, cmd = m.serviceEditor.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) harvestSchema(service postgres.ServiceEntry) tea.Cmd {
	return func() tea.Msg {
		db, err := service.Connect()
		if err != nil {
			return schemaLoadedMsg{err: err}
		}
		defer db.Close()

		harvester := postgres.NewSchemaHarvester(db)
		schema, err := harvester.Harvest(service.Name)
		return schemaLoadedMsg{schema: schema, err: err}
	}
}

// View renders the application
func (m *AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.loading {
		return m.renderLoading()
	}

	switch m.screen {
	case ScreenMenu:
		return m.menu.View()
	case ScreenDatabase:
		return m.database.View()
	case ScreenQuery:
		if m.query != nil {
			return m.query.View()
		}
		return m.menu.View()
	case ScreenHarvest:
		if m.harvest != nil {
			return m.harvest.View()
		}
		return m.menu.View()
	case ScreenHistory:
		if m.history != nil {
			return m.history.View()
		}
		return m.menu.View()
	case ScreenSettings:
		if m.settings != nil {
			return m.settings.View()
		}
		return m.menu.View()
	case ScreenServiceEditor:
		if m.serviceEditor != nil {
			return m.serviceEditor.View()
		}
		return m.database.View()
	default:
		return m.menu.View()
	}
}

func (m *AppModel) renderLoading() string {
	header := RenderHeader("Loading")

	loadingStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Align(lipgloss.Center)

	content := loadingStyle.Render(m.spinner.View() + " " + m.loadingMessage)

	helpText := "Please wait..."
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}


func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// RunApp runs the main TUI application
func RunApp() error {
	app := NewAppModel()
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
