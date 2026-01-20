package tui

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/kartoza/kartoza-pg-ai/internal/postgres"
)

// HarvestState holds the current state of the harvest progress
type HarvestState struct {
	Current     int
	Total       int
	Message     string
	SchemaName  string
	EntityType  string
	EntityName  string
	mu          sync.Mutex
}

// harvestProgressMsg is sent during schema harvesting
type harvestProgressMsg struct {
	Current    int
	Total      int
	Message    string
	SchemaName string
	EntityType string
	EntityName string
}

// HarvestModel is the model for the harvest screen
type HarvestModel struct {
	width       int
	height      int
	progress    progress.Model
	current     int
	total       int
	message     string
	schemaName  string
	entityType  string
	entityName  string
	done        bool
	err         error
	schema      *config.SchemaCache
	service     postgres.ServiceEntry
	harvestChan chan harvestProgressMsg
}

// NewHarvestModel creates a new harvest model
func NewHarvestModel(service postgres.ServiceEntry) *HarvestModel {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)
	// Override the default colors with Kartoza colors
	prog.FullColor = string(ColorOrange)
	prog.EmptyColor = string(ColorDarkGray)

	return &HarvestModel{
		progress:    prog,
		service:     service,
		message:     "Initializing...",
		harvestChan: make(chan harvestProgressMsg, 100),
	}
}

// Init initializes the harvest model
func (m *HarvestModel) Init() tea.Cmd {
	return tea.Batch(
		m.startHarvest(),
		m.listenForProgress(),
	)
}

// listenForProgress listens for progress updates from the harvest goroutine
func (m *HarvestModel) listenForProgress() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.harvestChan
		if !ok {
			return nil
		}
		return msg
	}
}

// startHarvest starts the schema harvesting process
func (m *HarvestModel) startHarvest() tea.Cmd {
	return func() tea.Msg {
		db, err := m.service.Connect()
		if err != nil {
			return schemaLoadedMsg{err: err}
		}
		defer db.Close()

		harvester := postgres.NewSchemaHarvester(db)

		// Set up progress callback that sends to channel
		harvester.SetProgressCallback(func(current, total int, message string) {
			// Parse the message to extract schema/entity info
			schemaName := ""
			entityType := ""
			entityName := ""

			// Message format: "Table: schema.name" or "View: schema.name" or "Function: schema.name"
			if len(message) > 7 {
				if message[:6] == "Table:" {
					entityType = "Table"
					fullName := message[7:]
					schemaName, entityName = parseSchemaEntity(fullName)
				} else if message[:5] == "View:" {
					entityType = "View"
					fullName := message[6:]
					schemaName, entityName = parseSchemaEntity(fullName)
				} else if message[:9] == "Function:" {
					entityType = "Function"
					fullName := message[10:]
					schemaName, entityName = parseSchemaEntity(fullName)
				} else {
					// Other messages
					entityType = ""
					entityName = message
				}
			} else {
				entityName = message
			}

			select {
			case m.harvestChan <- harvestProgressMsg{
				Current:    current,
				Total:      total,
				Message:    message,
				SchemaName: schemaName,
				EntityType: entityType,
				EntityName: entityName,
			}:
			default:
				// Channel full, skip this update
			}
		})

		schema, err := harvester.Harvest(m.service.Name)
		close(m.harvestChan)
		return schemaLoadedMsg{schema: schema, err: err}
	}
}

// parseSchemaEntity parses "schema.entity" into separate parts
func parseSchemaEntity(fullName string) (schema, entity string) {
	for i, c := range fullName {
		if c == '.' {
			return fullName[:i], fullName[i+1:]
		}
	}
	return "", fullName
}

// harvestCancelledMsg indicates user cancelled harvesting
type harvestCancelledMsg struct{}

// Update handles messages for the harvest model
func (m *HarvestModel) Update(msg tea.Msg) (*HarvestModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(50, msg.Width-20)
		return m, nil

	case tea.KeyMsg:
		// Handle ctrl+c to cancel harvesting
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEscape {
			return m, func() tea.Msg {
				return harvestCancelledMsg{}
			}
		}

	case harvestProgressMsg:
		m.current = msg.Current
		m.total = msg.Total
		m.message = msg.Message
		m.schemaName = msg.SchemaName
		m.entityType = msg.EntityType
		m.entityName = msg.EntityName

		// Continue listening for more progress
		return m, m.listenForProgress()

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

// View renders the harvest screen
func (m *HarvestModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Schema Harvesting")

	// Calculate progress percentage
	percent := 0.0
	if m.total > 0 {
		percent = float64(m.current) / float64(m.total)
	}

	// Progress bar section
	progressBar := m.progress.ViewAs(percent)

	// Progress counter
	counterStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Align(lipgloss.Center)
	counter := counterStyle.Render(fmt.Sprintf("%d / %d objects", m.current, m.total))

	// Schema name display
	schemaStyle := lipgloss.NewStyle().
		Foreground(ColorBlue).
		Bold(true).
		Align(lipgloss.Center)

	schemaDisplay := ""
	if m.schemaName != "" {
		schemaDisplay = schemaStyle.Render(fmt.Sprintf("Schema: %s", m.schemaName))
	}

	// Entity display
	entityLabelStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center)

	entityValueStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Align(lipgloss.Center)

	entityDisplay := ""
	if m.entityType != "" && m.entityName != "" {
		entityLabel := entityLabelStyle.Render(fmt.Sprintf("%s: ", m.entityType))
		entityValue := entityValueStyle.Render(m.entityName)
		entityDisplay = entityLabel + entityValue
	} else if m.message != "" {
		entityDisplay = entityValueStyle.Render(m.message)
	}

	// Center all progress elements
	progressContainer := lipgloss.NewStyle().
		Width(60).
		Align(lipgloss.Center)

	// Build the content
	var contentLines []string
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, progressContainer.Render(progressBar))
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, progressContainer.Render(counter))
	contentLines = append(contentLines, "")
	if schemaDisplay != "" {
		contentLines = append(contentLines, progressContainer.Render(schemaDisplay))
	}
	if entityDisplay != "" {
		contentLines = append(contentLines, progressContainer.Render(entityDisplay))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, contentLines...)

	helpText := "ctrl+c/esc: cancel • Please wait while the schema is being harvested..."
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

