package web

import (
	"html/template"

	webassets "libratrack/web"
)

func parseTemplates() (*template.Template, error) {
	return template.ParseFS(webassets.Templates, "templates/layout.html", "templates/catalog.html", "templates/partials/book_row.html")
}
