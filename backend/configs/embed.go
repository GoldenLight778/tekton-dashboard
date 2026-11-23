package configs

import "embed"

//go:embed *.yaml
var ConfigFS embed.FS
