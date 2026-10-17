package libratrack

import "embed"

// Files contains runtime configuration and SQL migrations required by the binary.
//
//go:embed VARIANT migrations/*.sql
var Files embed.FS
