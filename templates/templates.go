package templates

import (
	"embed"
)

// TemplatesFS embeds the templates directory.
//
//go:embed *.pkl *.template
var TemplatesFS embed.FS
