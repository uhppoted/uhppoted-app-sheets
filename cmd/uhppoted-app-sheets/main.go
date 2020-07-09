package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/uhppoted/uhppoted-api/command"
	"github.com/uhppoted/uhppoted-app-sheets/commands"
)

var cli = []uhppoted.Command{
	&commands.VersionCmd,
	&commands.GetACLCmd,
	&commands.LoadACLCmd,
	&commands.CompareACLCmd,
}

var options = struct {
	credentials string
	url         string
	region      string
	file        string
	debug       bool
}{
	credentials: "",
	url:         "",
	region:      "",
	file:        time.Now().Format("2006-01-02T150405.acl"),
	debug:       false,
}

var help = uhppoted.NewHelp("uhppoted-app-sheets", cli, nil)

func main() {
	flag.BoolVar(&options.debug, "debug", options.debug, "Enable debugging information")
	flag.Parse()

	cmd, err := uhppoted.Parse(cli, nil, help)
	if err != nil {
		fmt.Printf("\nError parsing command line: %v\n\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if cmd == nil {
		help.Execute(ctx)
		os.Exit(1)
	}

	if err = cmd.Execute(ctx); err != nil {
		log.Fatalf("ERROR: %v", err)
		os.Exit(1)
	}

}
