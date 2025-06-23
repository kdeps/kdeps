package templates

import (
	"embed"
)

// Embed the templates directory.
//
//go:embed *.pkl Dockerfile
var TemplatesFS embed.FS
