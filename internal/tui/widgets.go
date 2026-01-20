package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ========================================
// Brand Colors - Kartoza standard palette
// ========================================

var (
	ColorOrange   = lipgloss.Color("#DDA036") // Primary/Active
	ColorBlue     = lipgloss.Color("#569FC6") // Secondary/Links
	ColorGray     = lipgloss.Color("#9A9EA0") // Inactive/Subtle
	ColorWhite    = lipgloss.Color("#FFFFFF") // Text
	ColorDarkGray = lipgloss.Color("#3A3A3A") // Background
	ColorRed      = lipgloss.Color("#E95420") // Error/Active
	ColorGreen    = lipgloss.Color("#4CAF50") // Success
	ColorCyan     = lipgloss.Color("#00BCD4") // Info/SQL
)

// HeaderWidth is the standard width for the header
const HeaderWidth = 64

// ========================================
// Global Application State for Header
// ========================================

// AppState contains the global application state shown in all headers
type AppState struct {
	IsConnected     bool
	ActiveService   string
	SchemaLoaded    bool
	TablesCount     int
	QueryCount      int
	HasPostGIS      bool
	Status          string // e.g., "Ready", "Querying", "Connected"
	BlinkOn         bool   // For blinking indicator
	LastQueryTime   float64
}

// Global app state - updated by the main app model
var GlobalAppState = &AppState{
	IsConnected:   false,
	ActiveService: "",
	SchemaLoaded:  false,
	TablesCount:   0,
	QueryCount:    0,
	HasPostGIS:    false,
	Status:        "Ready",
	BlinkOn:       true,
}

// ========================================
// Header Rendering (DRY Implementation)
// ========================================

// RenderHeader renders the standard application header for ALL pages
// pageTitle is the name of the current page (e.g., "Main Menu", "Query")
//
// Format:
//
//	Kartoza PG AI - Page Title
//	Natural Language PostgreSQL Interface
//	────────────────────────────────────────────────────────────────
//	DB: myservice | Tables: 42 | PostGIS: ✓ | Status: Connected
//	────────────────────────────────────────────────────────────────
func RenderHeader(pageTitle string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorOrange).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	mottoStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	dividerStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	statusStyle := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Align(lipgloss.Center).
		Width(HeaderWidth)

	// Line 1: Application Name - Page Title
	title := titleStyle.Render(fmt.Sprintf("Kartoza PG AI - %s", pageTitle))

	// Line 2: Motto
	motto := mottoStyle.Render("Natural Language PostgreSQL Interface")

	// Line 3: Divider
	divider := dividerStyle.Render("────────────────────────────────────────────────────────────────")

	// Line 4: Status bar
	dbStatus := "Not connected"
	dbColor := ColorGray
	if GlobalAppState.IsConnected {
		dbStatus = GlobalAppState.ActiveService
		dbColor = ColorGreen
	}

	dbStyled := lipgloss.NewStyle().
		Foreground(dbColor).
		Bold(GlobalAppState.IsConnected).
		Render(dbStatus)

	// PostGIS status
	postgisStatus := "-"
	postgisColor := ColorGray
	if GlobalAppState.HasPostGIS {
		postgisStatus = "✓"
		postgisColor = ColorGreen
	}
	postgisStyled := lipgloss.NewStyle().
		Foreground(postgisColor).
		Render(postgisStatus)

	// Schema status
	schemaCount := "-"
	if GlobalAppState.SchemaLoaded {
		schemaCount = fmt.Sprintf("%d", GlobalAppState.TablesCount)
	}

	statusLine := fmt.Sprintf("DB: %s | Tables: %s | PostGIS: %s | Queries: %d",
		dbStyled,
		schemaCount,
		postgisStyled,
		GlobalAppState.QueryCount,
	)
	status := statusStyle.Render(statusLine)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		motto,
		divider,
		status,
		divider,
	)
}

// ========================================
// Footer Rendering
// ========================================

// RenderHelpFooter renders the standard help footer at the bottom of the screen
func RenderHelpFooter(helpText string, width int) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(ColorGray).
		Italic(true)

	footerStyle := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)

	return footerStyle.Render(helpStyle.Render(helpText))
}

// ========================================
// Layout Helpers
// ========================================

// LayoutWithHeaderFooter creates a standard layout with header at top and footer at bottom
func LayoutWithHeaderFooter(header, content, footer string, width, height int) string {
	// Center the header horizontally
	centeredHeader := lipgloss.PlaceHorizontal(width, lipgloss.Center, header)

	// Center the content horizontally
	centeredContent := lipgloss.PlaceHorizontal(width, lipgloss.Center, content)

	// Center the footer horizontally
	centeredFooter := lipgloss.PlaceHorizontal(width, lipgloss.Center, footer)

	// Calculate heights
	headerHeight := lipgloss.Height(centeredHeader)
	footerHeight := lipgloss.Height(centeredFooter)
	contentAreaHeight := height - headerHeight - footerHeight - 2 // 2 for spacing

	if contentAreaHeight < 1 {
		contentAreaHeight = 1
	}

	// Place content in its area (top-aligned within content area)
	contentArea := lipgloss.Place(
		width,
		contentAreaHeight,
		lipgloss.Center,
		lipgloss.Top,
		centeredContent,
	)

	// Join everything vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		centeredHeader,
		"",
		contentArea,
		centeredFooter,
	)
}

// CenterContent centers content both horizontally and vertically
func CenterContent(content string, width, height int) string {
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// ========================================
// Common Styles
// ========================================

// Box style for content areas
var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorOrange).
	Padding(1, 2)

// Title style for section headings
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorOrange)

// Subtitle style
var SubtitleStyle = lipgloss.NewStyle().
	Foreground(ColorBlue)

// Label style for form labels
var LabelStyle = lipgloss.NewStyle().
	Foreground(ColorGray)

// Value style for displaying values
var ValueStyle = lipgloss.NewStyle().
	Foreground(ColorWhite)

// Active style for active/selected items
var ActiveStyle = lipgloss.NewStyle().
	Foreground(ColorOrange).
	Bold(true)

// Inactive style for inactive items
var InactiveStyle = lipgloss.NewStyle().
	Foreground(ColorGray)

// Error style for error messages
var ErrorStyle = lipgloss.NewStyle().
	Foreground(ColorRed).
	Bold(true)

// Success style for success messages
var SuccessStyle = lipgloss.NewStyle().
	Foreground(ColorGreen).
	Bold(true)

// SQL style for SQL code
var SQLStyle = lipgloss.NewStyle().
	Foreground(ColorCyan)

// Prompt style for input prompts
var PromptStyle = lipgloss.NewStyle().
	Foreground(ColorOrange).
	Bold(true)
