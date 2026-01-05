package probe

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkrzeminski/sysprobe/internal/platform"
)

const (
	DefaultTimeout  = 30 * time.Second
	DefaultMaxLines = 500
	DefaultMaxBytes = 64 * 1024 // 64KB
)

// Runner executes diagnostic tasks
type Runner struct {
	Platform platform.Platform
	Timeout  time.Duration
}

// NewRunner creates a new task runner
func NewRunner(p platform.Platform) *Runner {
	return &Runner{
		Platform: p,
		Timeout:  DefaultTimeout,
	}
}

// CanRun checks if a task can be executed on the current system
func (r *Runner) CanRun(task Task) (bool, string) {
	// Check privilege requirements
	if task.Privilege == "sudo" && !r.Platform.IsRoot {
		return false, "Requires sudo privileges"
	}

	// Check binary dependencies
	for _, req := range task.Requires {
		if _, err := exec.LookPath(req); err != nil {
			return false, "Missing dependency: " + req
		}
	}

	// Check tag requirements
	if len(task.Tags) > 0 && !r.Platform.MatchesTags(task.Tags) {
		return false, "Environment mismatch: requires " + strings.Join(task.Tags, " or ")
	}

	return true, ""
}

// Run executes a single task and returns the result
func (r *Runner) Run(task Task) TaskResult {
	result := TaskResult{
		Name:     task.Name,
		Command:  task.Command,
		Category: task.Category,
		Status:   StatusPending,
	}

	// Check if we can run this task
	canRun, reason := r.CanRun(task)
	if !canRun {
		result.Status = StatusSkipped
		result.SkipReason = reason
		return result
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, "sh", "-c", task.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	start := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(start)

	// Get output
	result.Output = truncateOutput(stdout.String(), task.MaxLines, task.MaxBytes)
	result.Error = truncateOutput(stderr.String(), task.MaxLines, task.MaxBytes)

	// Determine status
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = StatusFailed
		result.Error = "Command timed out after " + r.Timeout.String()
	} else if err != nil {
		result.Status = StatusFailed
		if result.Error == "" {
			result.Error = err.Error()
		}
	} else {
		result.Status = StatusSuccess
	}

	return result
}

// truncateOutput limits output by lines and bytes
func truncateOutput(output string, maxLines, maxBytes int) string {
	if maxLines <= 0 {
		maxLines = DefaultMaxLines
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}

	// Truncate by bytes first
	if len(output) > maxBytes {
		output = output[:maxBytes] + "\n... [truncated: exceeded " + formatBytes(maxBytes) + "]"
	}

	// Truncate by lines
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		output = strings.Join(lines[:maxLines], "\n") + "\n... [truncated: exceeded " + string(rune(maxLines+'0')) + " lines]"
	}

	return strings.TrimSpace(output)
}

// formatBytes returns a human-readable byte count
func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return string(rune(b+'0')) + "B"
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return string(rune(b/div+'0')) + string("KMGTPE"[exp]) + "B"
}

// CheckBinary verifies if a binary exists in PATH
func CheckBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ExpandHome expands ~ to the user's home directory
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

