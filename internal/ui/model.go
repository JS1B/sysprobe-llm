package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkrzeminski/sysprobe/internal/probe"
)

// Message types
type TaskStartMsg struct {
	Name string
}

type TaskDoneMsg struct {
	Result probe.TaskResult
}

type AllDoneMsg struct {
	Results []probe.TaskResult
}

type ReportDoneMsg struct {
	ReportPath string
	TokenCount int
}

type TickMsg time.Time

// Model represents the UI state
type Model struct {
	tasks        []probe.TaskResult
	taskIndex    map[string]int
	completed    int
	total        int
	startTime    time.Time
	quitting     bool
	done         bool
	waitingInput bool // Wait for user input before exiting
	err          error
	width        int
	spinnerIdx   int
	reportPath   string
	tokenCount   int
}

// NewModel creates a new UI model
func NewModel(taskNames []string) Model {
	tasks := make([]probe.TaskResult, len(taskNames))
	taskIndex := make(map[string]int)

	for i, name := range taskNames {
		tasks[i] = probe.TaskResult{
			Name:   name,
			Status: probe.StatusPending,
		}
		taskIndex[name] = i
	}

	return Model{
		tasks:     tasks,
		taskIndex: taskIndex,
		total:     len(taskNames),
		startTime: time.Now(),
		width:     80,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.EnterAltScreen)
}

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter", " ":
			// Exit on enter/space if waiting for input
			if m.waitingInput {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case TickMsg:
		m.spinnerIdx = (m.spinnerIdx + 1) % len(SpinnerFrames)
		if !m.done {
			return m, tickCmd()
		}

	case TaskStartMsg:
		if idx, ok := m.taskIndex[msg.Name]; ok {
			m.tasks[idx].Status = probe.StatusRunning
		}

	case TaskDoneMsg:
		if idx, ok := m.taskIndex[msg.Result.Name]; ok {
			m.tasks[idx] = msg.Result
			m.completed++
		}

	case AllDoneMsg:
		m.done = true
		// Update any remaining tasks with final results
		for _, result := range msg.Results {
			if idx, ok := m.taskIndex[result.Name]; ok {
				m.tasks[idx] = result
			}
		}
		// Wait for report to be generated
		return m, nil

	case ReportDoneMsg:
		m.reportPath = msg.ReportPath
		m.tokenCount = msg.TokenCount
		m.waitingInput = true
		return m, nil
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "\n  Interrupted. Partial results may be available.\n\n"
	}

	var b strings.Builder

	// Title
	title := TitleStyle.Render("🔍 SysProbe Diagnostic Scanner")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Progress bar
	progress := m.renderProgress()
	b.WriteString(progress)
	b.WriteString("\n\n")

	// Task table
	table := m.renderTable()
	b.WriteString(table)
	b.WriteString("\n")

	// Footer
	elapsed := time.Since(m.startTime).Round(time.Millisecond)
	footer := FooterStyle.Render(fmt.Sprintf("Elapsed: %s", elapsed))
	b.WriteString(footer)

	if m.done {
		b.WriteString("\n\n")
		doneMsg := lipgloss.NewStyle().
			Foreground(Success).
			Bold(true).
			Render("✓ Scan complete!")
		b.WriteString(doneMsg)
		if m.reportPath != "" {
			b.WriteString(fmt.Sprintf(" Report saved to: %s", m.reportPath))
		}
		if m.tokenCount > 0 {
			b.WriteString(fmt.Sprintf(" (%d tokens)", m.tokenCount))
		}
		b.WriteString("\n\n")
		hint := FooterStyle.Render("Press Enter or Space to exit...")
		b.WriteString(hint)
		b.WriteString("\n")
	}

	return b.String()
}

// renderProgress renders the progress bar
func (m Model) renderProgress() string {
	percent := 0
	if m.total > 0 {
		percent = m.completed * 100 / m.total
	}

	barWidth := 40
	filled := barWidth * m.completed / max(m.total, 1)
	empty := barWidth - filled

	bar := ProgressFull.Render(strings.Repeat("█", filled)) +
		ProgressEmpty.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("  Progress: [%s] %d/%d (%d%%)", bar, m.completed, m.total, percent)
}

// renderTable renders the task status table
func (m Model) renderTable() string {
	var rows []string

	// Header
	header := fmt.Sprintf("  %-40s %-12s %-10s",
		HeaderStyle.Render("Task"),
		HeaderStyle.Render("Status"),
		HeaderStyle.Render("Duration"))
	rows = append(rows, header)
	rows = append(rows, "  "+strings.Repeat("─", 64))

	// Task rows (show last 15 tasks to fit in terminal)
	startIdx := 0
	if len(m.tasks) > 15 {
		startIdx = len(m.tasks) - 15
	}

	for i := startIdx; i < len(m.tasks); i++ {
		task := m.tasks[i]
		row := m.renderTaskRow(task)
		rows = append(rows, row)
	}

	if startIdx > 0 {
		rows = append(rows, FooterStyle.Render(fmt.Sprintf("  ... and %d more tasks above", startIdx)))
	}

	return strings.Join(rows, "\n")
}

// renderTaskRow renders a single task row
func (m Model) renderTaskRow(task probe.TaskResult) string {
	name := task.Name
	if len(name) > 38 {
		name = name[:35] + "..."
	}

	var status string
	switch task.Status {
	case probe.StatusPending:
		status = StatusPending.String()
	case probe.StatusRunning:
		spinner := lipgloss.NewStyle().Foreground(Warning).Render(SpinnerFrames[m.spinnerIdx])
		status = spinner + " Running"
	case probe.StatusSuccess:
		status = StatusSuccess.String()
	case probe.StatusSkipped:
		status = StatusSkipped.String()
	case probe.StatusFailed:
		status = StatusFailed.String()
	}

	duration := ""
	if task.Duration > 0 {
		duration = task.Duration.Round(time.Millisecond).String()
	}

	return fmt.Sprintf("  %-40s %-12s %-10s", name, status, duration)
}

// SetReportPath sets the report output path
func (m *Model) SetReportPath(path string) {
	m.reportPath = path
}

// SetTokenCount sets the token count for display
func (m *Model) SetTokenCount(count int) {
	m.tokenCount = count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

