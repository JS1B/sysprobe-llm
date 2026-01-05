# sysprobe-llm

Work in progress.

A modular, manifest-driven system diagnostic tool optimized for LLM analysis. Capture your Linux system state in a single command and get a token-counted Markdown report perfect for pasting into AI chats.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

Run `sysprobe-llm --intro --no-ui` and paste the output at the start of any LLM conversation about your system. The AI will have instant context about your hardware, software versions, and any current issues.

## Features

- **Single Binary** — All YAML probe manifests embedded via `go:embed`. No external files needed.
- **Live TUI** — Terminal UI with real-time progress, powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea).
- **Token Counting** — Reports include GPT-4/Claude token counts using `cl100k_base` tokenizer.
- **Smart Filtering** — Automatically skips probes based on missing dependencies, privilege requirements, and environment tags.
- **LLM-Optimized Output** — Clean Markdown with structured sections, perfect for AI analysis.
- **Intro Mode** — Generate a concise system summary (~400 tokens) to prepend to any LLM chat.

## Quick Start

```bash
# Clone and build
git clone https://github.com/pkrzeminski/sysprobe-llm.git
cd sysprobe-llm
go build -o sysprobe-llm ./cmd/sysprobe-llm

# Run with interactive TUI
./sysprobe-llm

# Generate system intro for LLM chats
./sysprobe-llm --intro
```

## Usage

```
Usage of sysprobe-llm:
  -intro
        Generate only system intro for LLM chat context (~400 tokens)
  -minified
        Generate minified output for smaller token count
  -no-ui
        Disable interactive UI (print results to stdout)
  -o string
        Output file path for the report (default "sysprobe-report.md")
  -version
        Show version information
  -workers int
        Number of concurrent workers (default 4)
```

### Examples

```bash
# Full diagnostic with TUI (press Enter to exit when done)
./sysprobe-llm

# Quick system intro for starting LLM conversations
./sysprobe-llm --intro --no-ui
# Output: sysprobe-intro.md (~400 tokens)

# Compact output when token budget is tight
./sysprobe-llm --minified
```

## Output Modes

### Full Report (over 10k tokens)
Complete system diagnostic with 146 probes across 12 categories:
- System info, hardware, packages, services
- Graphics (GPU, drivers, Vulkan, VA-API)
- Audio (PipeWire, PulseAudio, ALSA)
- Network (interfaces, DNS, firewall, VPN)
- Boot (bootloader, mkinitcpio, systemd-analyze)
- Storage (BTRFS, ZFS, LVM, SMART, NVMe)
- Power (battery, thermal, suspend)
- Window Manager (Hyprland, Sway)

### Intro Mode (~400 tokens)
Perfect for starting LLM conversations:
```markdown
# System Context

Use this information to understand my environment when helping me.

## System Summary
- Hostname, User, Kernel, Architecture
- CPU, RAM, GPU
- Desktop environment, Shell

## Package Manager State
- Installed/AUR packages count
- Last update time

## Key Software Versions
- Kernel, Mesa, Hyprland, PipeWire, etc.

## Current Issues Summary
- Failed systemd units
- Boot errors count
```

## Probes

| Category | Description |
|----------|-------------|
| `intro` | System introduction for LLM context |
| `system` | Core system diagnostics |
| `graphics` | GPU, drivers, display info |
| `wm` | Hyprland/Sway/Wayland |
| `audio` | PipeWire, ALSA debugging |
| `boot` | Bootloader, initramfs |
| `network` | Connectivity, DNS, firewall |
| `bluetooth` | Bluetooth debugging |
| `power` | Battery, thermal, suspend |
| `packages` | Pacman, AUR, dependencies |
| `storage` | Disks, filesystems, SMART |

## How It Works

1. **Platform Detection** — Identifies distro (Arch), display server (Wayland), and WM (Hyprland)
2. **Probe Loading** — Loads embedded YAML manifests matching your platform
3. **Smart Filtering** — Skips probes with:
   - Missing dependencies (`requires: [binary]`)
   - Insufficient privileges (`privilege: sudo`)
   - Environment mismatch (`tags: [hyprland, wayland]`)
4. **Concurrent Execution** — Runs probes in parallel with configurable worker pool
5. **Report Generation** — Produces structured Markdown with token count

## Dependencies

Runtime: None (single static binary)

Build:
- Go 1.21+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [tiktoken-go](https://github.com/tiktoken-go/tokenizer) — Token counting
- [yaml.v3](https://gopkg.in/yaml.v3) — YAML parsing

## Platform Support

Currently built for **Arch Linux** with **Hyprland**.

The architecture supports multiple platforms via the `probes/<distro>/` directory structure. Contributions for other distros welcome!

## License

MIT License — See [LICENSE](LICENSE) for details.
