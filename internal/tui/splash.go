package tui

import (
	"bytes"
	"embed"
	"fmt"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/blacktop/go-termimg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nfnt/resize"
)

//go:embed resources/kartoza-logo.png
var logoFS embed.FS

// SplashModel represents the splash screen state
type SplashModel struct {
	width          int
	height         int
	kittySupported bool
	logoImage      []byte
	showDuration   time.Duration
	startTime      time.Time
	elapsed        time.Duration
	done           bool
	scale          float64
	lastImageID    int
}

// splashDoneMsg signals the splash screen is complete
type splashDoneMsg struct{}

// splashTickMsg for animation updates
type splashTickMsg time.Time

// NewSplashModel creates a new splash screen model
func NewSplashModel(duration time.Duration) *SplashModel {
	sm := &SplashModel{
		showDuration:   duration,
		kittySupported: detectKittySupport(),
		scale:          0.05,
		lastImageID:    1000,
	}
	sm.loadLogo()
	return sm
}

// detectKittySupport checks if the terminal supports Kitty graphics protocol
func detectKittySupport() bool {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	kittyWindowID := os.Getenv("KITTY_WINDOW_ID")

	if kittyWindowID != "" {
		return true
	}
	if strings.Contains(term, "kitty") {
		return true
	}
	if termProgram == "kitty" {
		return true
	}

	protocol := termimg.DetectProtocol()
	return protocol == termimg.Kitty
}

// loadLogo loads the logo from embedded resources
func (sm *SplashModel) loadLogo() {
	data, err := logoFS.ReadFile("resources/kartoza-logo.png")
	if err != nil {
		return
	}
	sm.logoImage = data
}

// Init initializes the splash screen
func (sm *SplashModel) Init() tea.Cmd {
	sm.startTime = time.Now()
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

// Update handles messages for the splash screen
func (sm *SplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sm.width = msg.Width
		sm.height = msg.Height

	case tea.KeyMsg:
		sm.done = true
		return sm, tea.Quit

	case splashTickMsg:
		sm.elapsed = time.Since(sm.startTime)
		progress := float64(sm.elapsed) / float64(sm.showDuration)
		if progress >= 1.0 {
			sm.done = true
			return sm, tea.Quit
		}

		easedProgress := easeInOutCubic(progress)
		sm.scale = 0.05 + (easedProgress * 0.95)

		return sm, tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
			return splashTickMsg(t)
		})

	case splashDoneMsg:
		sm.done = true
		return sm, tea.Quit
	}

	return sm, nil
}

// View renders the splash screen
func (sm *SplashModel) View() string {
	if sm.width == 0 || sm.height == 0 {
		return ""
	}

	if sm.kittySupported && len(sm.logoImage) > 0 {
		return sm.renderWithKitty()
	}
	return sm.renderTextSplash()
}

// renderWithKitty renders the splash screen with the Kitty graphics protocol
func (sm *SplashModel) renderWithKitty() string {
	img, err := png.Decode(bytes.NewReader(sm.logoImage))
	if err != nil {
		return sm.renderTextSplash()
	}

	baseWidthCells := sm.width / 3
	if baseWidthCells < 20 {
		baseWidthCells = 20
	}
	if baseWidthCells > 60 {
		baseWidthCells = 60
	}

	logoWidthCells := int(float64(baseWidthCells) * sm.scale)
	if logoWidthCells < 2 {
		logoWidthCells = 2
	}

	bounds := img.Bounds()
	aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())
	logoHeightCells := int(float64(logoWidthCells) / aspectRatio / 2.0)
	if logoHeightCells < 1 {
		logoHeightCells = 1
	}

	pixelWidth := uint(logoWidthCells * 8)
	pixelHeight := uint(float64(pixelWidth) / aspectRatio)
	if pixelWidth < 8 {
		pixelWidth = 8
	}
	if pixelHeight < 8 {
		pixelHeight = 8
	}
	resizedLogo := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)

	var buf bytes.Buffer
	if err := png.Encode(&buf, resizedLogo); err != nil {
		return sm.renderTextSplash()
	}

	ti, err := termimg.From(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return sm.renderTextSplash()
	}

	sm.lastImageID++

	ti.Protocol(termimg.Kitty).
		Width(logoWidthCells).
		Height(logoHeightCells).
		Scale(termimg.ScaleFit).
		ImageNum(sm.lastImageID)

	rendered, err := ti.Render()
	if err != nil {
		return sm.renderTextSplash()
	}

	centerX := (sm.width - logoWidthCells) / 2
	centerY := (sm.height - logoHeightCells) / 2

	var output strings.Builder
	output.WriteString("\033_Ga=d\033\\")
	output.WriteString(fmt.Sprintf("\033[%d;%dH%s", centerY+1, centerX+1, rendered))

	return output.String()
}

// renderTextSplash renders a text-based splash screen (fallback)
func (sm *SplashModel) renderTextSplash() string {
	asciiLogo := `    ██████╗  ██████╗      █████╗ ██╗
    ██╔══██╗██╔════╝     ██╔══██╗██║
    ██████╔╝██║  ███╗    ███████║██║
    ██╔═══╝ ██║   ██║    ██╔══██║██║
    ██║     ╚██████╔╝    ██║  ██║██║
    ╚═╝      ╚═════╝     ╚═╝  ╚═╝╚═╝
    ██╗  ██╗ █████╗ ██████╗ ████████╗ ██████╗ ███████╗ █████╗
    ██║ ██╔╝██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗╚══███╔╝██╔══██╗
    █████╔╝ ███████║██████╔╝   ██║   ██║   ██║  ███╔╝ ███████║
    ██╔═██╗ ██╔══██║██╔══██╗   ██║   ██║   ██║ ███╔╝  ██╔══██║
    ██║  ██╗██║  ██║██║  ██║   ██║   ╚██████╔╝███████╗██║  ██║
    ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚══════╝╚═╝  ╚═╝`

	lines := strings.Split(asciiLogo, "\n")
	logoHeight := len(lines)
	logoWidth := 0
	for _, line := range lines {
		if len(line) > logoWidth {
			logoWidth = len(line)
		}
	}

	logoStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	styledLogo := logoStyle.Render(asciiLogo)

	centerX := (sm.width - logoWidth) / 2
	centerY := (sm.height - logoHeight) / 2

	var output strings.Builder
	styledLines := strings.Split(styledLogo, "\n")
	for i, line := range styledLines {
		row := centerY + i + 1
		col := centerX + 1
		if col < 1 {
			col = 1
		}
		if row < 1 {
			row = 1
		}
		output.WriteString(fmt.Sprintf("\033[%d;%dH%s", row, col, line))
	}

	return output.String()
}

// IsDone returns whether the splash screen is complete
func (sm *SplashModel) IsDone() bool {
	return sm.done
}

// ShowSplashScreen displays the splash screen as a standalone program
func ShowSplashScreen(duration time.Duration) error {
	splash := NewSplashModel(duration)
	p := tea.NewProgram(splash, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// easeInOutCubic provides smooth acceleration and deceleration
func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - (-2*t+2)*(-2*t+2)*(-2*t+2)/2
}

// ExitSplashModel represents the exit splash screen state
type ExitSplashModel struct {
	width          int
	height         int
	kittySupported bool
	logoImage      []byte
	showDuration   time.Duration
	startTime      time.Time
	elapsed        time.Duration
	done           bool
	scale          float64
	lastImageID    int
}

type exitSplashDoneMsg struct{}
type exitSplashTickMsg time.Time

// NewExitSplashModel creates a new exit splash screen model
func NewExitSplashModel(duration time.Duration) *ExitSplashModel {
	esm := &ExitSplashModel{
		showDuration:   duration,
		kittySupported: detectKittySupport(),
		scale:          1.0,
		lastImageID:    2000,
	}
	esm.loadLogo()
	return esm
}

func (esm *ExitSplashModel) loadLogo() {
	data, err := logoFS.ReadFile("resources/kartoza-logo.png")
	if err != nil {
		return
	}
	esm.logoImage = data
}

func (esm *ExitSplashModel) Init() tea.Cmd {
	esm.startTime = time.Now()
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return exitSplashTickMsg(t)
	})
}

func (esm *ExitSplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		esm.width = msg.Width
		esm.height = msg.Height

	case tea.KeyMsg:
		esm.done = true
		return esm, tea.Quit

	case exitSplashTickMsg:
		esm.elapsed = time.Since(esm.startTime)
		progress := float64(esm.elapsed) / float64(esm.showDuration)
		if progress >= 1.0 {
			esm.done = true
			return esm, tea.Quit
		}

		easedProgress := easeInOutCubic(progress)
		esm.scale = 1.0 - (easedProgress * 0.95)

		return esm, tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
			return exitSplashTickMsg(t)
		})

	case exitSplashDoneMsg:
		esm.done = true
		return esm, tea.Quit
	}

	return esm, nil
}

func (esm *ExitSplashModel) View() string {
	if esm.width == 0 || esm.height == 0 {
		return ""
	}

	if esm.kittySupported && len(esm.logoImage) > 0 {
		return esm.renderWithKitty()
	}
	return esm.renderTextSplash()
}

func (esm *ExitSplashModel) renderWithKitty() string {
	img, err := png.Decode(bytes.NewReader(esm.logoImage))
	if err != nil {
		return esm.renderTextSplash()
	}

	baseWidthCells := esm.width / 3
	if baseWidthCells < 20 {
		baseWidthCells = 20
	}
	if baseWidthCells > 60 {
		baseWidthCells = 60
	}

	logoWidthCells := int(float64(baseWidthCells) * esm.scale)
	if logoWidthCells < 2 {
		logoWidthCells = 2
	}

	bounds := img.Bounds()
	aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())
	logoHeightCells := int(float64(logoWidthCells) / aspectRatio / 2.0)
	if logoHeightCells < 1 {
		logoHeightCells = 1
	}

	pixelWidth := uint(logoWidthCells * 8)
	pixelHeight := uint(float64(pixelWidth) / aspectRatio)
	if pixelWidth < 8 {
		pixelWidth = 8
	}
	if pixelHeight < 8 {
		pixelHeight = 8
	}
	resizedLogo := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)

	var buf bytes.Buffer
	if err := png.Encode(&buf, resizedLogo); err != nil {
		return esm.renderTextSplash()
	}

	ti, err := termimg.From(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return esm.renderTextSplash()
	}

	esm.lastImageID++

	ti.Protocol(termimg.Kitty).
		Width(logoWidthCells).
		Height(logoHeightCells).
		Scale(termimg.ScaleFit).
		ImageNum(esm.lastImageID)

	rendered, err := ti.Render()
	if err != nil {
		return esm.renderTextSplash()
	}

	centerX := (esm.width - logoWidthCells) / 2
	centerY := (esm.height - logoHeightCells) / 2

	var output strings.Builder
	output.WriteString("\033_Ga=d\033\\")
	output.WriteString(fmt.Sprintf("\033[%d;%dH%s", centerY+1, centerX+1, rendered))

	return output.String()
}

func (esm *ExitSplashModel) renderTextSplash() string {
	asciiLogo := `    ██████╗  ██████╗      █████╗ ██╗
    ██╔══██╗██╔════╝     ██╔══██╗██║
    ██████╔╝██║  ███╗    ███████║██║
    ██╔═══╝ ██║   ██║    ██╔══██║██║
    ██║     ╚██████╔╝    ██║  ██║██║
    ╚═╝      ╚═════╝     ╚═╝  ╚═╝╚═╝
    ██╗  ██╗ █████╗ ██████╗ ████████╗ ██████╗ ███████╗ █████╗
    ██║ ██╔╝██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗╚══███╔╝██╔══██╗
    █████╔╝ ███████║██████╔╝   ██║   ██║   ██║  ███╔╝ ███████║
    ██╔═██╗ ██╔══██║██╔══██╗   ██║   ██║   ██║ ███╔╝  ██╔══██║
    ██║  ██╗██║  ██║██║  ██║   ██║   ╚██████╔╝███████╗██║  ██║
    ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚══════╝╚═╝  ╚═╝`

	lines := strings.Split(asciiLogo, "\n")
	logoHeight := len(lines)
	logoWidth := 0
	for _, line := range lines {
		if len(line) > logoWidth {
			logoWidth = len(line)
		}
	}

	logoStyle := lipgloss.NewStyle().
		Foreground(ColorOrange).
		Bold(true)

	styledLogo := logoStyle.Render(asciiLogo)

	centerX := (esm.width - logoWidth) / 2
	centerY := (esm.height - logoHeight) / 2

	var output strings.Builder
	styledLines := strings.Split(styledLogo, "\n")
	for i, line := range styledLines {
		row := centerY + i + 1
		col := centerX + 1
		if col < 1 {
			col = 1
		}
		if row < 1 {
			row = 1
		}
		output.WriteString(fmt.Sprintf("\033[%d;%dH%s", row, col, line))
	}

	return output.String()
}

func (esm *ExitSplashModel) IsDone() bool {
	return esm.done
}

func ShowExitSplashScreen(duration time.Duration) error {
	splash := NewExitSplashModel(duration)
	p := tea.NewProgram(splash, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
