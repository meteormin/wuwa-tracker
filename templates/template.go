package templates

import "embed"

//go:embed html/*.tmpl
var HTML embed.FS
