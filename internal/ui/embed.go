package ui

import "embed"

// staticFiles holds the embedded workbench UI.
// When the SvelteKit frontend has been built (task ui:build), its output
// replaces the placeholder files in static/. The Go binary always embeds
// whatever is present in static/ at compile time.
//
//go:embed all:static
var staticFiles embed.FS
