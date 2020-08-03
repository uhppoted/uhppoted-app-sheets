package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/uhppoted/uhppoted-api/command"
	"github.com/uhppoted/uhppoted-app-sheets/commands"
)

var cli = []uhppoted.CommandV{
	&commands.VersionCmd,
	&commands.GetCmd,
	&commands.PutCmd,
	&commands.LoadACLCmd,
	&commands.CompareACLCmd,
	&commands.UploadACLCmd,
}

var options = commands.Options{
	Debug: false,
}

var help = uhppoted.NewHelpV("uhppoted-app-sheets", cli, nil)

func main() {
	flag.BoolVar(&options.Debug, "debug", options.Debug, "Enable debugging information")
	flag.Parse()

	cmd, err := uhppoted.ParseV(cli, nil, help)
	if err != nil {
		fmt.Printf("\nError parsing command line: %v\n\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	if cmd == nil {
		help.Execute(ctx)
		os.Exit(1)
	}

	if err = cmd.Execute(ctx, &options); err != nil {
		log.Fatalf("ERROR: %v", err)
		os.Exit(1)
	}
}
