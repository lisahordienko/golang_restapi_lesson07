package webassets

import "embed"

// Static embeds browser-side assets served by the application.
//
//go:embed static/*
var Static embed.FS
