package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#06B6D4") // Cyan
	Success   = lipgloss.Color("#10B981") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Error     = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F3F4F6") // Light gray

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	// Header style for table columns
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary).
			Padding(0, 1)

	// Cell styles
	CellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// Status styles
	StatusPending = lipgloss.NewStyle().
			Foreground(Muted).
			SetString("○ Pending")

	StatusRunning = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true).
			SetString("◐ Running")

	StatusSuccess = lipgloss.NewStyle().
			Foreground(Success).
			SetString("✓ Success")

	StatusSkipped = lipgloss.NewStyle().
			Foreground(Muted).
			SetString("⊘ Skipped")

	StatusFailed = lipgloss.NewStyle().
			Foreground(Error).
			SetString("✗ Failed")

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Footer style
	FooterStyle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1)

	// Progress bar styles
	ProgressFull = lipgloss.NewStyle().
			Foreground(Success)

	ProgressEmpty = lipgloss.NewStyle().
			Foreground(Muted)

	// Spinner frames for animation
	SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

