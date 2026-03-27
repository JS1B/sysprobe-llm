package sysprobellm

import "embed"

//go:embed probes/*/*.yaml
var ProbeFS embed.FS
