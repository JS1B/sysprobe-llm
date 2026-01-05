package platform

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// Platform holds detected system information
type Platform struct {
	OS        string // e.g., "linux", "darwin", "windows"
	Distro    string // e.g., "arch", "ubuntu", "fedora"
	DistroID  string // e.g., "arch_linux"
	WM        string // e.g., "hyprland", "sway", "gnome"
	IsRoot    bool
	IsWayland bool
}

// Detect returns information about the current platform
func Detect() Platform {
	p := Platform{
		OS:     runtime.GOOS,
		IsRoot: os.Geteuid() == 0,
	}

	// Parse /etc/os-release for distro info
	if osRelease, err := parseOSRelease(); err == nil {
		p.Distro = osRelease["ID"]
		p.DistroID = osRelease["ID"]
		if idLike, ok := osRelease["ID_LIKE"]; ok && p.Distro == "" {
			p.Distro = idLike
		}
	}

	// Normalize distro ID
	if p.DistroID != "" {
		p.DistroID = strings.ToLower(p.DistroID) + "_linux"
	}

	// Detect display server
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		p.IsWayland = true
	}

	// Detect window manager/desktop environment
	p.WM = detectWM()

	return p
}

// parseOSRelease reads and parses /etc/os-release
func parseOSRelease() (map[string]string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], "\"'")
		result[key] = value
	}

	return result, scanner.Err()
}

// detectWM attempts to detect the current window manager
func detectWM() string {
	// Check common environment variables
	if desktop := os.Getenv("XDG_CURRENT_DESKTOP"); desktop != "" {
		return strings.ToLower(desktop)
	}

	if session := os.Getenv("XDG_SESSION_DESKTOP"); session != "" {
		return strings.ToLower(session)
	}

	if desktop := os.Getenv("DESKTOP_SESSION"); desktop != "" {
		return strings.ToLower(desktop)
	}

	// Check for Hyprland specifically
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return "hyprland"
	}

	// Check for Sway
	if os.Getenv("SWAYSOCK") != "" {
		return "sway"
	}

	return ""
}

// MatchesTags checks if the platform matches any of the given tags
func (p Platform) MatchesTags(tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	for _, tag := range tags {
		tag = strings.ToLower(tag)
		switch tag {
		case "wayland":
			if p.IsWayland {
				return true
			}
		case "x11":
			if !p.IsWayland && p.WM != "" {
				return true
			}
		case "hyprland":
			if strings.Contains(p.WM, "hyprland") {
				return true
			}
		case "sway":
			if strings.Contains(p.WM, "sway") {
				return true
			}
		case "gnome":
			if strings.Contains(p.WM, "gnome") {
				return true
			}
		case "kde", "plasma":
			if strings.Contains(p.WM, "kde") || strings.Contains(p.WM, "plasma") {
				return true
			}
		default:
			// Generic match against distro or WM
			if strings.Contains(p.Distro, tag) || strings.Contains(p.WM, tag) {
				return true
			}
		}
	}

	return false
}

