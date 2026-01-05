package sysprobe

import "embed"

//go:embed probes/*/*.yaml
var ProbeFS embed.FS
