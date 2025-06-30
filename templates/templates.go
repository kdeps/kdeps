package templates

import (
	"embed"
)

// Embed the templates directory.
//
//go:embed *.pkl *.template
var TemplatesFS embed.FS
