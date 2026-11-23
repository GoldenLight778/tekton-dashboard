//go:build !testenv && !prod

package config

// CompileEnv indicates the target environment chosen at compile-time via build tags.
const CompileEnv = "dev"
