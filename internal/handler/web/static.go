package web

import (
	"io/fs"

	webassets "libratrack/web"
)

func assets() (fs.FS, error) {
	return fs.Sub(webassets.Static, "static")
}
