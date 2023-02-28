package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var GetCmd = Get{
	command: command{
		workdir:     DEFAULT_WORKDIR,
		credentials: DEFAULT_CREDENTIALS,
		tokens:      "",
		url:         "",
		debug:       false,
	},

	area: "",
	file: time.Now().Format("2006-01-02T150405.tsv"),
}

type Get struct {
	command
	area    string
	file    string
	withPIN bool
}

func (cmd *Get) Name() string {
	return "get"
}

func (cmd *Get) Description() string {
	return "Retrieves an access control list from a Google Sheets worksheet and stores it to a local file"
}

func (cmd *Get) Usage() string {
	return "--credentials <file> --url <url> --file <file>"
}

func (cmd *Get) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] get [options] --url <URL> --range <range> --file <file>\n", APP)
	fmt.Println()
	fmt.Println("  Downloads a Google Sheets worksheet to a TSV file")
	fmt.Println()

	helpOptions(cmd.FlagSet())

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println(`    uhppote-app-sheets --debug get --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println(`                                   --file "example.tsv"`)
	fmt.Println()
}

func (cmd *Get) FlagSet() *flag.FlagSet {
	flagset := cmd.flagset("get")

	flagset.StringVar(&cmd.area, "range", cmd.area, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.StringVar(&cmd.file, "file", cmd.file, "TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")
	flagset.BoolVar(&cmd.withPIN, "with-pin", cmd.withPIN, "Includes the card keypad PIN codes in the retrieved ACL file")

	return flagset
}

func (cmd *Get) Execute(args ...interface{}) error {
	options := args[0].(*Options)

	cmd.debug = options.Debug

	// ... check parameters
	if strings.TrimSpace(cmd.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(cmd.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(cmd.area) == "" {
		return fmt.Errorf("--range is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(cmd.url)
	if len(match) < 2 {
		return fmt.Errorf("invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheet := match[1]
	area := cmd.area

	if cmd.debug {
		debugf("Spreadsheet - ID:%s  range:%s", spreadsheet, area)
	}

	// ... authorise
	tokens := cmd.tokens
	if tokens == "" {
		tokens = filepath.Join(cmd.workdir, ".google")
	}

	client, err := authorize(cmd.credentials, SHEETS, tokens)
	if err != nil {
		return fmt.Errorf("authentication/authorization error (%v)", err)
	}

	google, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to create new Sheets client (%v)", err)
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet, area).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return fmt.Errorf("no data in spreadsheet/range")
	}

	tmp, err := os.CreateTemp(os.TempDir(), "ACL")
	if err != nil {
		return err
	}

	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if err := sheetToTSV(tmp, response, cmd.withPIN); err != nil {
		return fmt.Errorf("error creating TSV file (%v)", err)
	}

	tmp.Close()

	dir := filepath.Dir(cmd.file)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), cmd.file); err != nil {
		return err
	}

	infof("Retrieved ACL to file %s", cmd.file)

	return nil
}
