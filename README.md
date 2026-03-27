# sysprobe-llm

Work in progress.

`sysprobe-llm` collects Linux system information and writes it as Markdown for troubleshooting or sharing with an LLM.

Current target platform: Arch Linux with Hyprland. The probe layout is extensible, but other platforms are not documented or supported yet.

## Build

```bash
git clone https://github.com/JS1B/sysprobe-llm
cd sysprobe-llm
go build -o sysprobe-llm ./cmd/sysprobe-llm
```

Requires Go 1.21+.

## Common Usage

```bash
# Full report
./sysprobe-llm

# Short context for an LLM chat
./sysprobe-llm --intro --no-ui

# Smaller output
./sysprobe-llm --minified
```

Default output files:

- `sysprobe-report.md`
- `sysprobe-intro.md`

## Flags

```log
Usage of sysprobe-llm:
  -intro
        Generate only system intro for LLM chat context (~400 tokens)
  -minified
        Generate minified output for smaller token count
  -no-ui
        Disable interactive UI (print results to stdout)
  -o string
        Output file path for the report (default "sysprobe-(report|intro).md")
  -version
        Show version information
  -workers int
        Number of concurrent workers (default 4)
```

## Output

There are two useful modes:

- Full report: broad system diagnostics across system, graphics, audio, boot, network, packages, storage, power, and window manager state.
- Intro mode: short summary intended to be pasted at the start of a support or LLM conversation.

Reports include token counts and skip probes that do not apply because of missing binaries, privileges, or environment tags.

## Probe Categories

| Category | Purpose |
| -------- | ------- |
| `intro` | Short system context |
| `system` | Core system information |
| `graphics` | GPU, drivers, display stack |
| `wm` | Wayland, Hyprland, Sway |
| `audio` | PipeWire, PulseAudio, ALSA |
| `boot` | Bootloader and initramfs |
| `network` | Interfaces, DNS, firewall, VPN |
| `bluetooth` | Bluetooth state |
| `power` | Battery, thermal, suspend |
| `packages` | Pacman, AUR, dependencies |
| `storage` | Disks, filesystems, SMART |

## Notes

- Probe manifests are embedded in the binary.
- Execution is concurrent and worker count is configurable.
- The interactive UI is optional.

## License

MIT. See `LICENSE`.
