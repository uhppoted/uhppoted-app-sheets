package commands

import (
	"path/filepath"
)

var _etc = workdir()
var _var = workdir()

var DEFAULT_WORKDIR = _var
var DEFAULT_CREDENTIALS = filepath.Join(_etc, "sheets", ".google", "credentials.json")
