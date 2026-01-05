package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	sysprobe "github.com/pkrzeminski/sysprobe"
	"github.com/pkrzeminski/sysprobe/internal/platform"
	"github.com/pkrzeminski/sysprobe/internal/probe"
	"github.com/pkrzeminski/sysprobe/internal/report"
	"github.com/pkrzeminski/sysprobe/internal/ui"
)

var (
	version = "dev"
)

// ReportMode indicates which report format to generate
type ReportMode int

const (
	ReportFull ReportMode = iota
	ReportMinified
	ReportIntro
)

func main() {
	// CLI flags
	outputFile := flag.String("o", "sysprobe-report.md", "Output file path for the report")
	noUI := flag.Bool("no-ui", false, "Disable interactive UI (print results to stdout)")
	minified := flag.Bool("minified", false, "Generate minified output for smaller token count")
	intro := flag.Bool("intro", false, "Generate only system intro for LLM chat context")
	showVersion := flag.Bool("version", false, "Show version information")
	workers := flag.Int("workers", 4, "Number of concurrent workers")
	flag.Parse()

	if *showVersion {
		fmt.Printf("sysprobe %s\n", version)
		os.Exit(0)
	}

	// Determine report mode
	mode := ReportFull
	if *intro {
		mode = ReportIntro
	} else if *minified {
		mode = ReportMinified
	}

	// Detect platform
	plat := platform.Detect()

	// Load probes
	loader := probe.NewLoader(sysprobe.ProbeFS, plat)
	tasks, err := loader.GetAllTasks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading probes: %v\n", err)
		os.Exit(1)
	}

	// Filter to intro tasks only if in intro mode
	if mode == ReportIntro {
		var introTasks []probe.Task
		for _, t := range tasks {
			if t.Category == "intro" {
				introTasks = append(introTasks, t)
			}
		}
		tasks = introTasks
		// Default output file for intro mode
		if *outputFile == "sysprobe-report.md" {
			*outputFile = "sysprobe-intro.md"
		}
	}

	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks found for this platform")
		os.Exit(1)
	}

	// Run with or without UI
	if *noUI {
		results := runWithoutUI(plat, tasks, *workers)

		// Generate report
		rep := report.NewMarkdownReport(plat, results)
		var content string
		var tokenCount int

		switch mode {
		case ReportIntro:
			content, tokenCount, err = rep.GenerateIntro()
		case ReportMinified:
			content, tokenCount, err = rep.GenerateMinified()
		default:
			content, tokenCount, err = rep.Generate()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}

		// Write report
		if err := os.WriteFile(*outputFile, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n✓ Report saved to: %s (%d tokens)\n", *outputFile, tokenCount)
	} else {
		// UI mode - report is generated inside runWithUI
		runWithUI(plat, tasks, *workers, *outputFile, mode)
	}
}

// runWithUI runs the diagnostic with the Bubble Tea UI
func runWithUI(plat platform.Platform, tasks []probe.Task, workerCount int, outputFile string, mode ReportMode) []probe.TaskResult {
	// Create task name list for UI
	taskNames := make([]string, len(tasks))
	for i, t := range tasks {
		taskNames[i] = t.Name
	}

	// Create model
	model := ui.NewModel(taskNames)

	// Create program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Results channel
	resultsChan := make(chan probe.TaskResult, len(tasks))
	var results []probe.TaskResult
	var resultsMu sync.Mutex

	// Start workers
	var wg sync.WaitGroup
	taskChan := make(chan probe.Task, len(tasks))

	// Spawn workers
	runner := probe.NewRunner(plat)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Notify UI that task started
				p.Send(ui.TaskStartMsg{Name: task.Name})

				// Run task
				result := runner.Run(task)
				resultsChan <- result

				// Notify UI that task is done
				p.Send(ui.TaskDoneMsg{Result: result})
			}
		}()
	}

	// Feed tasks
	go func() {
		for _, task := range tasks {
			taskChan <- task
		}
		close(taskChan)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect all results and generate report in background
	go func() {
		for result := range resultsChan {
			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()
		}
		p.Send(ui.AllDoneMsg{Results: results})

		// Generate report in background
		resultsMu.Lock()
		resultsCopy := make([]probe.TaskResult, len(results))
		copy(resultsCopy, results)
		resultsMu.Unlock()

		rep := report.NewMarkdownReport(plat, resultsCopy)
		var content string
		var tokenCount int
		var err error

		switch mode {
		case ReportIntro:
			content, tokenCount, err = rep.GenerateIntro()
		case ReportMinified:
			content, tokenCount, err = rep.GenerateMinified()
		default:
			content, tokenCount, err = rep.Generate()
		}

		if err == nil {
			_ = os.WriteFile(outputFile, []byte(content), 0644)
			p.Send(ui.ReportDoneMsg{ReportPath: outputFile, TokenCount: tokenCount})
		}
	}()

	// Run UI
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
	}

	return results
}

// runWithoutUI runs diagnostics without the TUI
func runWithoutUI(plat platform.Platform, tasks []probe.Task, workerCount int) []probe.TaskResult {
	fmt.Printf("Running %d diagnostic tasks...\n", len(tasks))

	runner := probe.NewRunner(plat)
	results := make([]probe.TaskResult, len(tasks))

	// Create work queue
	taskChan := make(chan int, len(tasks))
	var wg sync.WaitGroup

	// Spawn workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range taskChan {
				result := runner.Run(tasks[idx])
				results[idx] = result

				// Print progress
				status := "✓"
				if result.Status == probe.StatusFailed {
					status = "✗"
				} else if result.Status == probe.StatusSkipped {
					status = "⊘"
				}
				fmt.Printf("  %s %s\n", status, result.Name)
			}
		}()
	}

	// Feed work
	for i := range tasks {
		taskChan <- i
	}
	close(taskChan)

	wg.Wait()
	return results
}

