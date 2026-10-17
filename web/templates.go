package webassets

import "embed"

// Templates embeds the server-rendered catalog templates.
//
//go:embed templates/*.html templates/partials/*.html
var Templates embed.FS
