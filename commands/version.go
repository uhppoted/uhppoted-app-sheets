package commands

import (
	"flag"
	"fmt"

	"github.com/uhppoted/uhppote-core/uhppote"
)

// VersionCmd is an initialized Version command for the main() command list
var VersionCmd = Version{}

// Version is a CLI command implementation that displays the CLI version information.
type Version struct {
}

func (c *Version) FlagSet() *flag.FlagSet {
	return flag.NewFlagSet("version", flag.ExitOnError)
}

// Execute prints the current 'uhppote' version
func (c *Version) Execute(...interface{}) error {
	fmt.Printf("%s\n", uhppote.VERSION)

	return nil
}

// Returns 'version'
func (c *Version) Name() string {
	return "version"
}

// Description returns the 'version' command short form help
func (c *Version) Description() string {
	return "Displays the current version"
}

// Usage returns the string describing the additional options for the 'version' command
func (c *Version) Usage() string {
	return ""
}

// Help returns the 'version' command long form help
func (c *Version) Help() {
	fmt.Println("Displays the uhppote-cli version in the format v<major>.<minor>.<build> e.g. v1.00.10")
	fmt.Println()
}
