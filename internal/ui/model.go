package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkrzeminski/sysprobe-llm/internal/probe"
)

// Message types
type TaskStartMsg struct {
	Index int
	Name  string
}

type TaskDoneMsg struct {
	Index  int
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
	startedAt    []time.Time // wall time when task Index entered running; zero if not running
	completed    int
	total        int
	workerCount  int
	startTime    time.Time
	quitting     bool
	done         bool
	waitingInput bool // Wait for user input before exiting
	err          error
	width        int
	height       int
	spinnerIdx   int
	reportPath   string
	tokenCount   int
}

// NewModel creates a new UI model
func NewModel(taskNames []string, workerCount int) Model {
	tasks := make([]probe.TaskResult, len(taskNames))
	startedAt := make([]time.Time, len(taskNames))

	for i, name := range taskNames {
		tasks[i] = probe.TaskResult{
			Name:   name,
			Status: probe.StatusPending,
		}
	}

	return Model{
		tasks:       tasks,
		startedAt:   startedAt,
		workerCount: workerCount,
		total:       len(taskNames),
		startTime:   time.Now(),
		width:       80,
		height:      0,
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
		m.height = msg.Height

	case TickMsg:
		m.spinnerIdx = (m.spinnerIdx + 1) % len(SpinnerFrames)
		if !m.done {
			return m, tickCmd()
		}

	case TaskStartMsg:
		if msg.Index >= 0 && msg.Index < len(m.tasks) {
			m.tasks[msg.Index].Status = probe.StatusRunning
			m.startedAt[msg.Index] = time.Now()
		}

	case TaskDoneMsg:
		if msg.Index >= 0 && msg.Index < len(m.tasks) {
			if msg.Index < len(m.startedAt) {
				m.startedAt[msg.Index] = time.Time{}
			}
			m.tasks[msg.Index] = msg.Result
			m.completed++
		}

	case AllDoneMsg:
		m.done = true
		n := len(m.tasks)
		for i := 0; i < n && i < len(msg.Results); i++ {
			m.tasks[i] = msg.Results[i]
		}
		for i := range m.startedAt {
			m.startedAt[i] = time.Time{}
		}
		m.completed = n

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
	now := time.Now()

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
	elapsed := now.Sub(m.startTime).Round(time.Millisecond)
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

	runningN := m.countRunning()
	line := fmt.Sprintf("  Progress: [%s] %d/%d (%d%%)  ·  workers: %d  ·  active: %d",
		bar, m.completed, m.total, percent, m.workerCount, runningN)
	return line
}

func (m Model) countRunning() int {
	n := 0
	for _, t := range m.tasks {
		if t.Status == probe.StatusRunning {
			n++
		}
	}
	return n
}

// tableVisibleRows chooses how many task rows to show based on terminal height
func (m Model) tableVisibleRows() int {
	if m.height <= 0 {
		return 15
	}
	// Title, progress, table header, footer, margins
	overhead := 16
	n := m.height - overhead
	if n < 8 {
		n = 8
	}
	if n > 45 {
		n = 45
	}
	return n
}

// tableDisplayOrder returns task indices with unfinished first: running, then pending, then
// finished (success / failed / skipped), each subgroup in stable probe order.
func (m Model) tableDisplayOrder() []int {
	n := len(m.tasks)
	var running, pending, finished []int
	for i := 0; i < n; i++ {
		switch m.tasks[i].Status {
		case probe.StatusRunning:
			running = append(running, i)
		case probe.StatusPending:
			pending = append(pending, i)
		default:
			finished = append(finished, i)
		}
	}
	order := make([]int, 0, n)
	order = append(order, running...)
	order = append(order, pending...)
	order = append(order, finished...)
	return order
}

// renderTable renders the task table; unfinished tasks stay at the top when the view is truncated.
func (m Model) renderTable() string {
	var rows []string

	limit := m.tableVisibleRows()
	order := m.tableDisplayOrder()
	visible := order
	hidden := 0
	if len(visible) > limit {
		hidden = len(visible) - limit
		visible = visible[:limit]
	}

	header := fmt.Sprintf("  %-40s %-12s %-10s",
		HeaderStyle.Render("Task"),
		HeaderStyle.Render("Status"),
		HeaderStyle.Render("Duration"))
	rows = append(rows, header)
	rows = append(rows, "  "+strings.Repeat("─", 64))
	if hidden > 0 {
		rows = append(rows, FooterStyle.Render(fmt.Sprintf("  … %d more below (unfinished listed first)", hidden)))
	}

	for _, i := range visible {
		task := m.tasks[i]
		row := m.renderTaskRow(i, task)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

// renderTaskRow renders a single task row (index shown for disambiguation)
func (m Model) renderTaskRow(index int, task probe.TaskResult) string {
	name := fmt.Sprintf("#%d %s", index+1, task.Name)
	name = truncateRunes(name, 38)

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
	if task.Status == probe.StatusRunning {
		if index < len(m.startedAt) && !m.startedAt[index].IsZero() {
			duration = time.Since(m.startedAt[index]).Round(time.Millisecond).String()
		}
	} else if task.Duration > 0 {
		duration = task.Duration.Round(time.Millisecond).String()
	}

	return fmt.Sprintf("  %-40s %-12s %-10s", name, status, duration)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
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
