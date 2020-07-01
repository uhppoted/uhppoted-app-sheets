package commands

import (
	"path/filepath"
)

var DEFAULT_WORKDIR = workdir()
var DEFAULT_REVISION_FILE = filepath.Join(workdir(), "sheets", "uhppoted-app-sheets.revision")
