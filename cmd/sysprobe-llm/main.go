package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	sysprobellm "github.com/pkrzeminski/sysprobe-llm"
	"github.com/pkrzeminski/sysprobe-llm/internal/platform"
	"github.com/pkrzeminski/sysprobe-llm/internal/probe"
	"github.com/pkrzeminski/sysprobe-llm/internal/report"
	"github.com/pkrzeminski/sysprobe-llm/internal/ui"
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

type indexedTask struct {
	idx  int
	task probe.Task
}

type indexedResult struct {
	idx    int
	result probe.TaskResult
}

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
		fmt.Printf("sysprobe-llm %s\n", version)
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
	loader := probe.NewLoader(sysprobellm.ProbeFS, plat)
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
func runWithUI(plat platform.Platform, tasks []probe.Task, workerCount int, outputFile string, mode ReportMode) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create task name list for UI
	taskNames := make([]string, len(tasks))
	for i, t := range tasks {
		taskNames[i] = t.Name
	}

	// Create model
	model := ui.NewModel(taskNames, workerCount)

	// Create program
	p := tea.NewProgram(model, tea.WithAltScreen())

	resultsChan := make(chan indexedResult, len(tasks))
	ordered := make([]probe.TaskResult, len(tasks))
	var orderedMu sync.Mutex

	var wg sync.WaitGroup
	taskChan := make(chan indexedTask, len(tasks))

	runner := probe.NewRunner(plat)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wi := range taskChan {
				p.Send(ui.TaskStartMsg{Index: wi.idx, Name: wi.task.Name})
				result := runner.Run(ctx, wi.task)
				resultsChan <- indexedResult{idx: wi.idx, result: result}
				p.Send(ui.TaskDoneMsg{Index: wi.idx, Result: result})
			}
		}()
	}

	go func() {
		for i, task := range tasks {
			taskChan <- indexedTask{idx: i, task: task}
		}
		close(taskChan)
	}()

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var uiFinished atomic.Bool
	go func() {
		for o := range resultsChan {
			orderedMu.Lock()
			ordered[o.idx] = o.result
			orderedMu.Unlock()
		}
		if uiFinished.Load() {
			return
		}
		orderedMu.Lock()
		resultsCopy := make([]probe.TaskResult, len(ordered))
		copy(resultsCopy, ordered)
		orderedMu.Unlock()

		p.Send(ui.AllDoneMsg{Results: resultsCopy})

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
			if !uiFinished.Load() {
				p.Send(ui.ReportDoneMsg{ReportPath: outputFile, TokenCount: tokenCount})
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "UI error: %v\n", err)
	}
	uiFinished.Store(true)
	cancel()
}

// runWithoutUI runs diagnostics without the TUI
func runWithoutUI(plat platform.Platform, tasks []probe.Task, workerCount int) []probe.TaskResult {
	fmt.Printf("Running %d diagnostic tasks...\n", len(tasks))

	ctx := context.Background()
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
				fmt.Printf("  → start [%d/%d] %s\n", idx+1, len(tasks), tasks[idx].Name)
				t0 := time.Now()
				result := runner.Run(ctx, tasks[idx])
				dur := time.Since(t0).Round(time.Millisecond)

				results[idx] = result

				status := "✓"
				if result.Status == probe.StatusFailed {
					status = "✗"
				} else if result.Status == probe.StatusSkipped {
					status = "⊘"
				}
				fmt.Printf("  %s %s (%s)\n", status, result.Name, dur)
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
