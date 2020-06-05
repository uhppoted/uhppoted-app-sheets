package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	//	"regexp"
	//	"strings"
	"time"

	"github.com/uhppoted/uhppoted-api/command"
	"github.com/uhppoted/uhppoted-app-sheets/commands"
)

var cli = []uhppoted.Command{
	&commands.VersionCmd,
	&commands.GetACLCmd,
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
	//	flag.StringVar(&options.credentials, "credentials", options.credentials, "Path for the 'credentials.json' file")
	//	flag.StringVar(&options.url, "url", options.url, "Spreadsheet URL")
	//	flag.StringVar(&options.region, "range", options.region, "Spreadsheet range e.g. 'Class Data!A2:E'")
	//	flag.StringVar(&options.file, "file", options.file, "TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")
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
