package commands

import (
	"path/filepath"
)

var DEFAULT_WORKDIR = workdir()
var DEFAULT_REVISION_FILE = filepath.Join(workdir(), ".google", "uhppoted-app-sheets.revision")
