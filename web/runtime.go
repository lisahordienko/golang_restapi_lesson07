package webassets

import "embed"

// Runtime embeds the variant configuration and SQL migrations.
//
//go:embed VARIANT migrations/*.sql
var Runtime embed.FS
