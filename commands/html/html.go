package html

import (
	"embed"
)

//go:embed images css fonts favicon.ico manifest.json auth.html
var HTML embed.FS
