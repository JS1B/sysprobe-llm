package probe

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkrzeminski/sysprobe/internal/platform"
	"gopkg.in/yaml.v3"
)

// Loader handles loading and filtering probe profiles
type Loader struct {
	fs       embed.FS
	platform platform.Platform
}

// NewLoader creates a new probe loader
func NewLoader(probeFS embed.FS, p platform.Platform) *Loader {
	return &Loader{
		fs:       probeFS,
		platform: p,
	}
}

// LoadAll loads all profiles that match the current platform
func (l *Loader) LoadAll() ([]Profile, error) {
	var profiles []Profile

	err := fs.WalkDir(l.fs, "probes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		profile, err := l.loadProfile(path)
		if err != nil {
			return fmt.Errorf("loading %s: %w", path, err)
		}

		// Check if profile matches current platform
		if l.matchesPlatform(profile) {
			profiles = append(profiles, profile)
		}

		return nil
	})

	return profiles, err
}

// loadProfile reads and parses a single YAML profile
func (l *Loader) loadProfile(path string) (Profile, error) {
	var profile Profile

	data, err := l.fs.ReadFile(path)
	if err != nil {
		return profile, err
	}

	if err := yaml.Unmarshal(data, &profile); err != nil {
		return profile, err
	}

	// Set category from filename if not specified in tasks
	category := strings.TrimSuffix(filepath.Base(path), ".yaml")
	for i := range profile.Tasks {
		if profile.Tasks[i].Category == "" {
			profile.Tasks[i].Category = category
		}
	}

	return profile, nil
}

// matchesPlatform checks if a profile matches the current platform
func (l *Loader) matchesPlatform(profile Profile) bool {
	if profile.Platform == "" {
		return true
	}

	// Normalize platform strings for comparison
	profilePlatform := strings.ToLower(profile.Platform)
	currentPlatform := strings.ToLower(l.platform.DistroID)

	// Direct match
	if profilePlatform == currentPlatform {
		return true
	}

	// Partial match (e.g., "arch" matches "arch_linux")
	if strings.Contains(currentPlatform, profilePlatform) {
		return true
	}

	// Generic linux match
	if profilePlatform == "linux" && l.platform.OS == "linux" {
		return true
	}

	return false
}

// FilterTasks returns only the tasks that can run on the current system
func (l *Loader) FilterTasks(profiles []Profile) []Task {
	var tasks []Task
	runner := NewRunner(l.platform)

	for _, profile := range profiles {
		for _, task := range profile.Tasks {
			// We include all tasks, but mark ones that can't run
			// The runner will skip them with appropriate messages
			tasks = append(tasks, task)
			_ = runner // Used for potential pre-filtering
		}
	}

	return tasks
}

// GetAllTasks loads all matching profiles and returns their tasks
func (l *Loader) GetAllTasks() ([]Task, error) {
	profiles, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	return l.FilterTasks(profiles), nil
}

