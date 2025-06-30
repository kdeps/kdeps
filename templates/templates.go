package templates

import (
	"embed"
)

// Embed the templates directory.
//
//go:embed *.pkl
var TemplatesFS embed.FS
