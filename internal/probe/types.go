package probe

import "time"

// Status represents the execution status of a task
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusSkipped
	StatusFailed
)

// String returns the string representation of a Status
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusSuccess:
		return "Success"
	case StatusSkipped:
		return "Skipped"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// Task represents a single diagnostic command to execute
type Task struct {
	Name      string   `yaml:"name"`
	Command   string   `yaml:"command"`
	Privilege string   `yaml:"privilege,omitempty"` // "sudo" or empty
	MaxLines  int      `yaml:"max_lines,omitempty"`
	MaxBytes  int      `yaml:"max_bytes,omitempty"`
	Requires  []string `yaml:"requires,omitempty"` // binary dependencies
	Tags      []string `yaml:"tags,omitempty"`     // e.g., ["hyprland", "wayland"]
	Category  string   `yaml:"category,omitempty"` // for grouping in report
}

// Profile represents a collection of tasks for a specific platform
type Profile struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Platform    string `yaml:"platform"` // e.g., "arch_linux"
	Tasks       []Task `yaml:"tasks"`
}

// TaskResult holds the result of executing a task
type TaskResult struct {
	Name       string
	Command    string
	Category   string
	Status     Status
	Output     string
	Error      string
	Duration   time.Duration
	SkipReason string
}

