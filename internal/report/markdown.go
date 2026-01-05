package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkrzeminski/sysprobe/internal/platform"
	"github.com/pkrzeminski/sysprobe/internal/probe"
)

// MarkdownReport generates LLM-friendly markdown reports
type MarkdownReport struct {
	Platform platform.Platform
	Results  []probe.TaskResult
	Generated time.Time
}

// NewMarkdownReport creates a new report generator
func NewMarkdownReport(p platform.Platform, results []probe.TaskResult) *MarkdownReport {
	return &MarkdownReport{
		Platform:  p,
		Results:   results,
		Generated: time.Now(),
	}
}

// Generate creates the markdown report
func (r *MarkdownReport) Generate() (string, int, error) {
	var b strings.Builder

	// Generate report content first (without token count)
	content := r.generateContent()
	
	// Count tokens
	tc, err := NewTokenCounter()
	if err != nil {
		// Fall back to rough estimate if tokenizer fails
		tokenCount := len(content) / 4
		return r.generateWithTokenCount(tokenCount), tokenCount, nil
	}
	
	// Count tokens of content
	tokenCount := tc.Count(content)
	
	// Regenerate with actual token count
	report := r.generateWithTokenCount(tokenCount)
	
	// Recount with token count included
	finalTokenCount := tc.Count(report)
	
	_ = b // unused
	return report, finalTokenCount, nil
}

// generateContent generates the report without the header token count
func (r *MarkdownReport) generateContent() string {
	var b strings.Builder
	
	// Group results by category
	categories := r.groupByCategory()
	
	// Generate each category
	for _, cat := range categories {
		b.WriteString(fmt.Sprintf("\n## %s\n", cat.name))
		for _, result := range cat.results {
			r.writeTaskResult(&b, result)
		}
	}
	
	// Errors and skipped section
	r.writeErrorsSection(&b)
	
	return b.String()
}

// generateWithTokenCount generates the full report with token count in header
func (r *MarkdownReport) generateWithTokenCount(tokenCount int) string {
	var b strings.Builder
	
	// Header
	b.WriteString("# SysProbe Diagnostic Report\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n", r.Generated.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Platform: %s", r.Platform.DistroID))
	if r.Platform.WM != "" {
		b.WriteString(fmt.Sprintf(" (%s)", r.Platform.WM))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Token Count: %d\n", tokenCount))
	
	// Content
	b.WriteString(r.generateContent())
	
	return b.String()
}

type categoryGroup struct {
	name    string
	results []probe.TaskResult
}

// groupByCategory organizes results by their category
func (r *MarkdownReport) groupByCategory() []categoryGroup {
	categoryMap := make(map[string][]probe.TaskResult)
	categoryOrder := []string{}
	
	for _, result := range r.Results {
		cat := result.Category
		if cat == "" {
			cat = "General"
		}
		cat = strings.Title(cat)
		
		if _, exists := categoryMap[cat]; !exists {
			categoryOrder = append(categoryOrder, cat)
		}
		categoryMap[cat] = append(categoryMap[cat], result)
	}
	
	// Sort categories for consistent output
	sort.Strings(categoryOrder)
	
	var groups []categoryGroup
	for _, name := range categoryOrder {
		groups = append(groups, categoryGroup{
			name:    name,
			results: categoryMap[name],
		})
	}
	
	return groups
}

// writeTaskResult writes a single task result to the report
func (r *MarkdownReport) writeTaskResult(b *strings.Builder, result probe.TaskResult) {
	// Skip failed/skipped tasks in main output (they go to errors section)
	if result.Status == probe.StatusFailed || result.Status == probe.StatusSkipped {
		return
	}
	
	b.WriteString(fmt.Sprintf("\n### %s\n", result.Name))
	b.WriteString(fmt.Sprintf("```\n$ %s\n", result.Command))
	
	if result.Output != "" {
		b.WriteString(result.Output)
		if !strings.HasSuffix(result.Output, "\n") {
			b.WriteString("\n")
		}
	} else {
		b.WriteString("[no output]\n")
	}
	
	b.WriteString("```\n")
}

// writeErrorsSection writes the errors and skipped tasks section
func (r *MarkdownReport) writeErrorsSection(b *strings.Builder) {
	var errors, skipped []probe.TaskResult
	
	for _, result := range r.Results {
		switch result.Status {
		case probe.StatusFailed:
			errors = append(errors, result)
		case probe.StatusSkipped:
			skipped = append(skipped, result)
		}
	}
	
	if len(errors) == 0 && len(skipped) == 0 {
		return
	}
	
	b.WriteString("\n## Errors & Skipped\n\n")
	
	for _, result := range errors {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "Unknown error"
		}
		b.WriteString(fmt.Sprintf("- **%s**: Failed (%s)\n", result.Name, errMsg))
	}
	
	for _, result := range skipped {
		reason := result.SkipReason
		if reason == "" {
			reason = "Unknown reason"
		}
		b.WriteString(fmt.Sprintf("- **%s**: Skipped (%s)\n", result.Name, reason))
	}
}

// GenerateMinified creates a more compact version for constrained contexts
func (r *MarkdownReport) GenerateMinified() (string, int, error) {
	var b strings.Builder
	
	b.WriteString("# SysProbe Report\n")
	b.WriteString(fmt.Sprintf("Time:%s Platform:%s\n", 
		r.Generated.Format("2006-01-02T15:04"),
		r.Platform.DistroID))
	
	for _, result := range r.Results {
		if result.Status == probe.StatusSuccess && result.Output != "" {
			b.WriteString(fmt.Sprintf("\n## %s\n```\n%s\n```\n", 
				result.Name, 
				strings.TrimSpace(result.Output)))
		}
	}
	
	content := b.String()
	
	tc, err := NewTokenCounter()
	if err != nil {
		return content, len(content) / 4, nil
	}
	
	tokenCount := tc.Count(content)
	return content, tokenCount, nil
}

// GenerateIntro creates a concise system introduction for LLM chat context
func (r *MarkdownReport) GenerateIntro() (string, int, error) {
	var b strings.Builder
	
	b.WriteString("# System Context\n\n")
	b.WriteString("Use this information to understand my environment when helping me.\n\n")
	
	// Only include intro category results
	for _, result := range r.Results {
		if result.Category != "intro" {
			continue
		}
		if result.Status == probe.StatusSuccess && result.Output != "" {
			b.WriteString(fmt.Sprintf("## %s\n", result.Name))
			b.WriteString("```\n")
			b.WriteString(strings.TrimSpace(result.Output))
			b.WriteString("\n```\n\n")
		}
	}
	
	content := b.String()
	
	tc, err := NewTokenCounter()
	if err != nil {
		return content, len(content) / 4, nil
	}
	
	tokenCount := tc.Count(content)
	return content, tokenCount, nil
}

