package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"
)

var PutCmd = Put{
	command: command{
		workdir:     DEFAULT_WORKDIR,
		credentials: DEFAULT_CREDENTIALS,
		tokens:      "",
		url:         "",
		debug:       false,
	},

	area: "",
	file: "",
}

type Put struct {
	command
	area    string
	file    string
	withPIN bool
}

func (cmd *Put) Name() string {
	return "put"
}

func (cmd *Put) Description() string {
	return "Uploads a TSV file to a Google Sheets worksheet"
}

func (cmd *Put) Usage() string {
	return "--credentials <file> --url <url> --file <file>"
}

func (cmd *Put) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] put [options] --url <URL> --range <range> --file <file>\n", APP)
	fmt.Println()
	fmt.Println("  Uploads a TSV file to a Google Sheets worksheet")
	fmt.Println()

	helpOptions(cmd.FlagSet())

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println(`    uhppote-app-sheets --debug put --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println(`                                   --file "example.tsv"`)
	fmt.Println()
}

func (cmd *Put) FlagSet() *flag.FlagSet {
	flagset := cmd.flagset("put")

	flagset.StringVar(&cmd.area, "range", cmd.area, "Spreadsheet range e.g. 'AsIs!A2:E'")
	flagset.StringVar(&cmd.file, "file", cmd.file, "TSV file")
	flagset.BoolVar(&cmd.withPIN, "with-pin", cmd.withPIN, "Includes the card keypad PIN codes in the uploaded TSV file")

	return flagset
}

func (cmd *Put) Execute(args ...interface{}) error {
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

	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(cmd.area)
	if len(match) < 5 {
		return fmt.Errorf("Invalid spreadsheet range '%s'", cmd.area)
	}

	if strings.TrimSpace(cmd.file) == "" {
		return fmt.Errorf("--file is a required option")
	}

	match = regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(cmd.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]
	region := cmd.area

	if cmd.debug {
		debugf("Spreadsheet - ID:%s  range:%s", spreadsheetId, region)
	}

	// ... authorise
	tokens := cmd.tokens
	if tokens == "" {
		tokens = filepath.Join(cmd.workdir, ".google")
	}

	client, err := authorize(cmd.credentials, SHEETS, tokens)
	if err != nil {
		return fmt.Errorf("Authentication/authorization error (%v)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		return fmt.Errorf("Unable to create new Sheets client (%v)", err)
	}

	spreadsheet, err := getSpreadsheet(google, spreadsheetId)
	if err != nil {
		return err
	}

	f, err := os.Open(cmd.file)
	if err != nil {
		return err
	}

	defer f.Close()

	header, data, err := tsvToSheet(f, cmd.area)
	if err != nil {
		return err
	} else if header == nil {
		return fmt.Errorf("Invalid TSV file (%v)", err)
	}

	if err := cmd.clear(google, spreadsheet); err != nil {
		return err
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{header, data},
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Do(); err != nil {
		return err
	}

	infof("Uploaded TSV file %v to Google Sheets %v", cmd.file, cmd.area)

	return nil
}

func (cmd *Put) clear(google *sheets.Service, spreadsheet *sheets.Spreadsheet) error {
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(cmd.area)
	if len(match) < 5 {
		return fmt.Errorf("Invalid spreadsheet range '%s'", cmd.area)
	}

	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	data := fmt.Sprintf("%s!%s%v:%s", name, left, top+1, right)

	return clear(google, spreadsheet, []string{data})
}
