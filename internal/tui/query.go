package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/kartoza/kartoza-pg-ai/internal/llm"
	"github.com/kartoza/kartoza-pg-ai/internal/postgres"
	"github.com/kujtimiihoxha/vimtea"
)

// QueryModel represents the main query interface
type QueryModel struct {
	width       int
	height      int
	vimEditor   vimtea.Editor
	textArea    textarea.Model
	vimMode     bool // Whether to use vim editor or simple textarea
	spinner     spinner.Model
	loading     bool
	results     *QueryResults
	service     *postgres.ServiceEntry
	schema      *config.SchemaCache
	queryEngine *llm.QueryEngine
	error       string
	history     []ConversationEntry
	cfg         *config.Config
	db          *sql.DB // Persistent database connection
	// Pagination
	currentPage int
	rowsPerPage int
}

// QueryResults holds the results of a query
type QueryResults struct {
	Columns       []string
	Rows          [][]string
	RowCount      int
	ExecutionTime float64
	GeneratedSQL  string
	NaturalQuery  string
}

// ConversationEntry holds a conversation turn
type ConversationEntry struct {
	Query   string
	SQL     string
	Results *QueryResults
	Error   string
}

// queryExecutedMsg indicates a query was executed
type queryExecutedMsg struct {
	results *QueryResults
	err     error
}

// schemaLoadedMsg indicates schema was loaded
type schemaLoadedMsg struct {
	schema *config.SchemaCache
	err    error
}

// dbConnectedMsg indicates database connection was established
type dbConnectedMsg struct {
	db  *sql.DB
	err error
}

// NewQueryModel creates a new query model
func NewQueryModel(service *postgres.ServiceEntry, schema *config.SchemaCache) *QueryModel {
	cfg, _ := config.Load()

	// Determine if vim mode is enabled
	vimMode := true // default
	if cfg != nil {
		vimMode = cfg.Settings.VimModeEnabled
	}

	// Create vim-style editor with custom styling
	lineNumStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		PaddingRight(1)

	currentLineNumStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		PaddingRight(1)

	textStyle := lipgloss.NewStyle().
		Foreground(ColorWhite)

	statusStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 1)

	cursorStyle := lipgloss.NewStyle().
		Background(ColorOrange).
		Foreground(lipgloss.Color("#000000"))

	vimEditor := vimtea.NewEditor(
		vimtea.WithLineNumberStyle(lineNumStyle),
		vimtea.WithCurrentLineNumberStyle(currentLineNumStyle),
		vimtea.WithTextStyle(textStyle),
		vimtea.WithStatusStyle(statusStyle),
		vimtea.WithCursorStyle(cursorStyle),
		vimtea.WithRelativeNumbers(false), // Simpler display
		vimtea.WithEnableStatusBar(false), // We'll render our own status bar
	)

	// Create simple textarea as alternative
	ta := textarea.New()
	ta.Placeholder = "Enter your query here..."
	ta.ShowLineNumbers = false
	ta.SetHeight(5)
	ta.CharLimit = 0 // No limit
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorOrange)
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorGray)
	ta.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorOrange)

	return &QueryModel{
		vimEditor:   vimEditor,
		textArea:    ta,
		vimMode:     vimMode,
		spinner:     s,
		service:     service,
		schema:      schema,
		queryEngine: llm.NewQueryEngine(schema),
		history:     []ConversationEntry{},
		cfg:         cfg,
		db:          nil, // Will be established asynchronously in Init
		currentPage: 0,
		rowsPerPage: 15,
	}
}

// Init initializes the query model
func (m *QueryModel) Init() tea.Cmd {
	if m.vimMode {
		// Initialize vim editor and set to INSERT mode for immediate typing
		editorInit := m.vimEditor.Init()
		startInsertMode := m.vimEditor.SetMode(vimtea.ModeInsert)
		return tea.Batch(
			editorInit,
			startInsertMode,
			m.connectToDatabase(),
		)
	}
	// Initialize simple textarea
	return tea.Batch(
		textarea.Blink,
		m.connectToDatabase(),
	)
}

// connectToDatabase establishes the database connection asynchronously
func (m *QueryModel) connectToDatabase() tea.Cmd {
	return func() tea.Msg {
		if m.service == nil {
			return dbConnectedMsg{err: fmt.Errorf("no service configured")}
		}
		db, err := m.service.Connect()
		if err != nil {
			return dbConnectedMsg{err: err}
		}
		// Ping to verify connection
		if err := db.Ping(); err != nil {
			return dbConnectedMsg{err: err}
		}
		return dbConnectedMsg{db: db}
	}
}

// getEditorText returns the current text from whichever editor is active
func (m *QueryModel) getEditorText() string {
	if m.vimMode {
		return m.vimEditor.GetBuffer().Text()
	}
	return m.textArea.Value()
}

// SetInitialQuery sets the initial text in the editor
func (m *QueryModel) SetInitialQuery(query string) {
	if m.vimMode {
		m.vimEditor.GetBuffer().InsertAt(0, 0, query)
	} else {
		m.textArea.SetValue(query)
	}
}

// clearEditor clears the editor content
func (m *QueryModel) clearEditor() tea.Cmd {
	if m.vimMode {
		m.vimEditor = vimtea.NewEditor(
			vimtea.WithRelativeNumbers(false),
			vimtea.WithEnableStatusBar(false),
		)
		editorWidth := m.width - 10
		if editorWidth < 40 {
			editorWidth = 40
		}
		updatedEditor, sizeCmd := m.vimEditor.SetSize(editorWidth, 5)
		m.vimEditor = updatedEditor.(vimtea.Editor)
		return tea.Batch(m.vimEditor.Init(), sizeCmd, m.vimEditor.SetMode(vimtea.ModeInsert))
	}
	m.textArea.Reset()
	m.textArea.Focus()
	return nil
}

// Update handles messages for the query model
func (m *QueryModel) Update(msg tea.Msg) (*QueryModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update editor size - full width with padding, minimum 5 lines
		editorWidth := m.width - 10
		if editorWidth < 40 {
			editorWidth = 40
		}
		if m.vimMode {
			updatedEditor, cmd := m.vimEditor.SetSize(editorWidth, 5)
			m.vimEditor = updatedEditor.(vimtea.Editor)
			cmds = append(cmds, cmd)
		} else {
			m.textArea.SetWidth(editorWidth)
			m.textArea.SetHeight(5)
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case dbConnectedMsg:
		if msg.err != nil {
			m.error = "Connection failed: " + msg.err.Error()
		} else {
			m.db = msg.db
		}
		return m, nil

	case queryExecutedMsg:
		m.loading = false
		if msg.err != nil {
			m.error = msg.err.Error()
			m.history = append(m.history, ConversationEntry{
				Query: m.getEditorText(),
				Error: msg.err.Error(),
			})
		} else {
			m.results = msg.results
			m.error = ""
			m.history = append(m.history, ConversationEntry{
				Query:   msg.results.NaturalQuery,
				SQL:     msg.results.GeneratedSQL,
				Results: msg.results,
			})

			// Update global state
			GlobalAppState.QueryCount++
			GlobalAppState.LastQueryTime = msg.results.ExecutionTime

			// Save to config history
			if m.cfg != nil && m.service != nil {
				m.cfg.AddQueryToHistory(config.QueryHistoryEntry{
					Timestamp:     time.Now(),
					NaturalQuery:  msg.results.NaturalQuery,
					GeneratedSQL:  msg.results.GeneratedSQL,
					ServiceName:   m.service.Name,
					RowsAffected:  msg.results.RowCount,
					ExecutionTime: msg.results.ExecutionTime,
					Success:       true,
				})
				m.cfg.Save()
			}
		}
		// Clear editor content
		return m, m.clearEditor()

	case tea.KeyMsg:
		// Handle ctrl+c - cancel or quit (always intercept this, don't pass to editor)
		if msg.Type == tea.KeyCtrlC {
			if m.loading {
				m.loading = false
				return m, nil
			}
			return m, tea.Quit
		}

		// Handle F1 to go back to menu (vim-friendly) - don't pass to editor
		if msg.Type == tea.KeyF1 {
			return m, func() tea.Msg {
				return goToMenuMsg{}
			}
		}

		// Handle ctrl+s to execute query (works in any vim mode)
		if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))) {
			content := strings.TrimSpace(m.getEditorText())
			if !m.loading && content != "" {
				m.loading = true
				m.currentPage = 0 // Reset to first page on new query
				return m, tea.Batch(
					m.spinner.Tick,
					m.executeQuery(content),
				)
			}
			return m, nil
		}

		// Pagination controls - only when we have results
		if m.results != nil && len(m.results.Rows) > 0 {
			totalPages := (len(m.results.Rows) + m.rowsPerPage - 1) / m.rowsPerPage

			// Next page: n, Page Down, or Right arrow
			if key.Matches(msg, key.NewBinding(key.WithKeys("n", "pgdown", "right"))) {
				if m.currentPage < totalPages-1 {
					m.currentPage++
				}
				return m, nil
			}

			// Previous page: p, Page Up, or Left arrow
			if key.Matches(msg, key.NewBinding(key.WithKeys("p", "pgup", "left"))) {
				if m.currentPage > 0 {
					m.currentPage--
				}
				return m, nil
			}

			// First page: Home or g
			if key.Matches(msg, key.NewBinding(key.WithKeys("home", "g"))) {
				m.currentPage = 0
				return m, nil
			}

			// Last page: End or G
			if key.Matches(msg, key.NewBinding(key.WithKeys("end", "G"))) {
				m.currentPage = totalPages - 1
				return m, nil
			}
		}

		// Clear error on typing
		if m.error != "" {
			m.error = ""
		}
	}

	// Update the active editor (only for messages not handled above)
	var cmd tea.Cmd
	if m.vimMode {
		updatedEditor, cmd := m.vimEditor.Update(msg)
		m.vimEditor = updatedEditor.(vimtea.Editor)
		cmds = append(cmds, cmd)
	} else {
		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// executeQuery executes a natural language query
func (m *QueryModel) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		if m.db == nil {
			return queryExecutedMsg{err: fmt.Errorf("no database connection")}
		}

		// Generate SQL from natural language
		sqlQuery, err := m.queryEngine.GenerateSQL(query, m.getConversationContext())
		if err != nil {
			return queryExecutedMsg{err: fmt.Errorf("failed to generate SQL: %w", err)}
		}

		// Ensure connection is alive
		if err := m.db.Ping(); err != nil {
			// Try to reconnect
			if m.service != nil {
				m.db, _ = m.service.Connect()
			}
			if m.db == nil {
				return queryExecutedMsg{err: fmt.Errorf("failed to reconnect: %w", err)}
			}
		}

		startTime := time.Now()
		rows, err := m.db.Query(sqlQuery)
		if err != nil {
			return queryExecutedMsg{err: fmt.Errorf("query failed: %w", err)}
		}
		defer rows.Close()

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return queryExecutedMsg{err: fmt.Errorf("failed to get columns: %w", err)}
		}

		// Read results
		var results [][]string
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}

			row := make([]string, len(columns))
			for i, val := range values {
				row[i] = formatValue(val)
			}
			results = append(results, row)
		}

		executionTime := time.Since(startTime).Seconds() * 1000

		return queryExecutedMsg{
			results: &QueryResults{
				Columns:       columns,
				Rows:          results,
				RowCount:      len(results),
				ExecutionTime: executionTime,
				GeneratedSQL:  sqlQuery,
				NaturalQuery:  query,
			},
		}
	}
}

func (m *QueryModel) getConversationContext() string {
	if len(m.history) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("Previous conversation:\n")

	// Only include last 3 turns for context
	start := 0
	if len(m.history) > 3 {
		start = len(m.history) - 3
	}

	for _, entry := range m.history[start:] {
		context.WriteString(fmt.Sprintf("User: %s\n", entry.Query))
		if entry.SQL != "" {
			context.WriteString(fmt.Sprintf("SQL: %s\n", entry.SQL))
		}
		if entry.Results != nil && entry.Results.RowCount > 0 {
			context.WriteString(fmt.Sprintf("(Returned %d rows)\n", entry.Results.RowCount))
		}
		context.WriteString("\n")
	}

	return context.String()
}

// View renders the query interface
func (m *QueryModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := RenderHeader("Query")
	content := m.renderContent()
	var helpText string
	if m.vimMode {
		helpText = "ctrl+s: execute • F1: menu • ctrl+c: quit • vim: h/j/k/l, i, esc"
	} else {
		helpText = "ctrl+s: execute • F1: menu • ctrl+c: quit"
	}
	footer := RenderHelpFooter(helpText, m.width)

	return LayoutWithHeaderFooter(header, content, footer, m.width, m.height)
}

func (m *QueryModel) renderContent() string {
	var sections []string

	// Results area (takes most of the screen)
	resultsHeight := m.height - 25 // Leave room for header, prompt, footer
	if resultsHeight < 10 {
		resultsHeight = 10
	}

	if m.loading {
		loadingContent := lipgloss.NewStyle().
			Width(m.width - 10).
			Height(resultsHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View() + " Generating and executing query...")
		sections = append(sections, loadingContent)
	} else if m.results != nil {
		sections = append(sections, m.renderResults(resultsHeight))
	} else if len(m.history) == 0 {
		// Welcome message
		welcomeContent := m.renderWelcome(resultsHeight)
		sections = append(sections, welcomeContent)
	} else {
		// Show last result
		if len(m.history) > 0 {
			lastEntry := m.history[len(m.history)-1]
			if lastEntry.Results != nil {
				m.results = lastEntry.Results
				sections = append(sections, m.renderResults(resultsHeight))
			}
		}
	}

	// Error display
	if m.error != "" {
		errorBox := BoxStyle.Copy().
			BorderForeground(ColorRed).
			Width(min(60, m.width-10)).
			Align(lipgloss.Center)
		sections = append(sections, errorBox.Render(ErrorStyle.Render("Error: "+m.error)))
	}

	// Prompt area
	sections = append(sections, "")
	var promptLabel string
	if m.vimMode {
		promptLabel = PromptStyle.Render("🔮 Ask your database (vim mode):")
	} else {
		promptLabel = PromptStyle.Render("🔮 Ask your database:")
	}
	sections = append(sections, promptLabel)

	if m.vimMode {
		// Vim editor with border
		editorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorOrange).
			Width(m.width - 6).
			Padding(0, 1)

		sections = append(sections, editorBox.Render(m.vimEditor.View()))

		// Custom vim status bar with glyphs
		statusBar := m.renderVimStatusBar()
		sections = append(sections, statusBar)
	} else {
		// Simple textarea (already has border styling)
		m.textArea.SetWidth(m.width - 10)
		sections = append(sections, m.textArea.View())
	}

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

// renderVimStatusBar renders a custom vim-style status bar with glyphs
func (m *QueryModel) renderVimStatusBar() string {
	// Get vim mode from editor
	mode := m.vimEditor.GetMode()

	// Mode icons and colors
	var modeIcon string
	var modeText string
	var modeColor lipgloss.Color

	switch mode.String() {
	case "NORMAL":
		modeIcon = "󰰓" // Normal mode icon
		modeText = "NORMAL"
		modeColor = ColorBlue
	case "INSERT":
		modeIcon = "󰰄" // Insert mode icon
		modeText = "INSERT"
		modeColor = ColorGreen
	case "VISUAL":
		modeIcon = "󰰫" // Visual mode icon
		modeText = "VISUAL"
		modeColor = ColorOrange
	case "COMMAND":
		modeIcon = "󰘳" // Command mode icon
		modeText = "COMMAND"
		modeColor = ColorCyan
	default:
		modeIcon = "󰰓"
		modeText = mode.String()
		modeColor = ColorGray
	}

	// Mode section
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(modeColor).
		Bold(true).
		Padding(0, 1)

	modeSeparator := lipgloss.NewStyle().
		Foreground(modeColor).
		Background(ColorDarkGray).
		Render("")

	// Info section with glyphs
	infoStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Background(ColorDarkGray).
		Padding(0, 1)

	// Lines info
	buffer := m.vimEditor.GetBuffer()
	lineCount := buffer.LineCount()
	linesIcon := "󰯎" // Lines icon
	lines := fmt.Sprintf("%s %d lines", linesIcon, lineCount)

	// Hint text
	hintIcon := "󰌌" // Keyboard icon
	hint := fmt.Sprintf("%s i:insert  esc:normal  ctrl+s:execute", hintIcon)

	// Build status bar
	modeSection := modeStyle.Render(modeIcon + " " + modeText)
	infoSection := infoStyle.Render(lines + "  " + hint)

	// Calculate padding
	statusWidth := m.width - 8
	contentWidth := lipgloss.Width(modeSection) + lipgloss.Width(modeSeparator) + lipgloss.Width(infoSection)
	padding := statusWidth - contentWidth
	if padding < 0 {
		padding = 0
	}

	paddingStyle := lipgloss.NewStyle().
		Background(ColorDarkGray)

	return modeSection + modeSeparator + paddingStyle.Render(strings.Repeat(" ", padding)) + infoSection
}

func (m *QueryModel) renderWelcome(height int) string {
	// Welcome title
	titleStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	// Schema info
	var schemaInfo string
	if m.schema != nil {
		tablesCount := len(m.schema.Tables)
		viewsCount := len(m.schema.Views)
		functionsCount := len(m.schema.Functions)
		postgisStatus := "No"
		if m.schema.HasPostGIS {
			postgisStatus = "Yes"
		}
		schemaInfo = fmt.Sprintf("Tables: %d • Views: %d • Functions: %d • PostGIS: %s",
			tablesCount, viewsCount, functionsCount, postgisStatus)
	}

	schemaStyle := lipgloss.NewStyle().
		Foreground(ColorBlue)

	// Example queries box
	examplesTitle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true).
		Render("Example queries:")

	examples := []string{
		"• \"How many records are in each table?\"",
		"• \"Show me the first 10 customers\"",
		"• \"What are the columns in users?\"",
		"• \"Describe the products table\"",
	}

	if m.schema != nil && m.schema.HasPostGIS {
		examples = append(examples, "• \"Find all roads within 1km\" (PostGIS)")
		examples = append(examples, "• \"What is the total length of roads?\"")
	}

	examplesStyle := lipgloss.NewStyle().
		Foreground(ColorGray)

	examplesContent := lipgloss.JoinVertical(lipgloss.Left,
		examplesTitle,
		"",
		examplesStyle.Render(strings.Join(examples, "\n")),
	)

	examplesBox := BoxStyle.Copy().
		BorderForeground(ColorGray).
		Width(50).
		Padding(1, 2)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("Welcome to the Query Interface"),
		"",
		subtitleStyle.Render("Ask questions about your database in natural language"),
		"",
		schemaStyle.Render(schemaInfo),
		"",
		"",
		examplesBox.Render(examplesContent),
	)

	return lipgloss.NewStyle().
		Width(m.width - 10).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

func (m *QueryModel) renderResults(height int) string {
	if m.results == nil {
		return ""
	}

	var sections []string

	// Show generated SQL
	sqlBox := BoxStyle.Copy().
		BorderForeground(ColorCyan).
		Width(min(80, m.width-10))
	sqlContent := SQLStyle.Render("SQL: " + m.results.GeneratedSQL)
	sections = append(sections, sqlBox.Render(sqlContent))

	// Stats line
	statsStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)
	stats := fmt.Sprintf("%d rows • %.2fms", m.results.RowCount, m.results.ExecutionTime)
	sections = append(sections, statsStyle.Render(stats))
	sections = append(sections, "")

	// Results table
	if len(m.results.Rows) > 0 {
		table := m.renderTable()
		sections = append(sections, table)
	} else {
		noResults := lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true).
			Render("No results returned")
		sections = append(sections, noResults)
	}

	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func (m *QueryModel) renderTable() string {
	if m.results == nil || len(m.results.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	colWidths := make([]int, len(m.results.Columns))
	for i, col := range m.results.Columns {
		colWidths[i] = len(col)
	}
	for _, row := range m.results.Rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Limit column widths
	maxColWidth := 30
	for i := range colWidths {
		if colWidths[i] > maxColWidth {
			colWidths[i] = maxColWidth
		}
	}

	// Build table
	var lines []string

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange)

	var headerCells []string
	for i, col := range m.results.Columns {
		cell := padOrTruncate(col, colWidths[i])
		headerCells = append(headerCells, headerStyle.Render(cell))
	}
	lines = append(lines, strings.Join(headerCells, " │ "))

	// Separator
	var sepParts []string
	for _, w := range colWidths {
		sepParts = append(sepParts, strings.Repeat("─", w))
	}
	sepStyle := lipgloss.NewStyle().Foreground(ColorGray)
	lines = append(lines, sepStyle.Render(strings.Join(sepParts, "─┼─")))

	// Pagination calculation
	totalRows := len(m.results.Rows)
	totalPages := (totalRows + m.rowsPerPage - 1) / m.rowsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	startIdx := m.currentPage * m.rowsPerPage
	endIdx := startIdx + m.rowsPerPage
	if endIdx > totalRows {
		endIdx = totalRows
	}

	// Rows for current page
	rowStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	for i := startIdx; i < endIdx; i++ {
		row := m.results.Rows[i]
		var cells []string
		for j, cell := range row {
			if j < len(colWidths) {
				cells = append(cells, padOrTruncate(cell, colWidths[j]))
			}
		}
		lines = append(lines, rowStyle.Render(strings.Join(cells, " │ ")))
	}

	// Pagination info
	if totalPages > 1 {
		lines = append(lines, "")
		paginationStyle := lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true)
		paginationInfo := fmt.Sprintf("Page %d of %d (rows %d-%d of %d) • n/→: next • p/←: prev • g: first • G: last",
			m.currentPage+1, totalPages, startIdx+1, endIdx, totalRows)
		lines = append(lines, paginationStyle.Render(paginationInfo))
	}

	return strings.Join(lines, "\n")
}

// formatValue formats a SQL value for display
func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case sql.NullString:
		if v.Valid {
			return v.String
		}
		return "NULL"
	case sql.NullInt64:
		if v.Valid {
			return fmt.Sprintf("%d", v.Int64)
		}
		return "NULL"
	case sql.NullFloat64:
		if v.Valid {
			return fmt.Sprintf("%.2f", v.Float64)
		}
		return "NULL"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func padOrTruncate(s string, width int) string {
	if len(s) > width {
		return s[:width-1] + "…"
	}
	return s + strings.Repeat(" ", width-len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
