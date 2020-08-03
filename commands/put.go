package commands

import (
	"context"
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
	workdir:     DEFAULT_WORKDIR,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	area:        "",
	file:        "",
	debug:       false,
}

type Put struct {
	workdir     string
	credentials string
	url         string
	area        string
	file        string
	debug       bool
}

func (c *Put) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("put", flag.ExitOnError)

	flagset.StringVar(&c.workdir, "workdir", c.workdir, "Directory for working files (tokens, revisions, etc)'")
	flagset.StringVar(&c.credentials, "credentials", c.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&c.url, "url", c.url, "Spreadsheet URL")
	flagset.StringVar(&c.area, "range", c.area, "Spreadsheet range e.g. 'AsIs!A2:E'")
	flagset.StringVar(&c.file, "file", c.file, "TSV file")

	return flagset
}

func (cmd *Put) Execute(ctx context.Context, options ...interface{}) error {
	if len(options) > 0 {
		if opt, ok := options[0].(*Options); ok {
			cmd.debug = opt.Debug
		}
	}

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
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s", spreadsheetId, region))
	}

	client, err := authorize(cmd.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(cmd.workdir, ".google"))
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

	if err := cmd.clear(google, spreadsheet, ctx); err != nil {
		return err
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{header, data},
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	info(fmt.Sprintf("Uploaded TSV file %v to Google Sheets %v", cmd.file, cmd.area))

	return nil
}

func (c *Put) clear(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ctx context.Context) error {
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(c.area)
	if len(match) < 5 {
		return fmt.Errorf("Invalid spreadsheet range '%s'", c.area)
	}

	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	data := fmt.Sprintf("%s!%s%v:%s", name, left, top+1, right)

	return clear(google, spreadsheet, []string{data}, ctx)
}

func (c *Put) Name() string {
	return "put"
}

func (c *Put) Description() string {
	return "Uploads a TSV file to a Google Sheets worksheet"
}

func (c *Put) Usage() string {
	return "--credentials <file> --url <url> --file <file>"
}

func (c *Put) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] put --credentials <credentials> --url <URL> --range <range> --file <file>\n", APP)
	fmt.Println()
	fmt.Println("  Uploads a TSV file to a Google Sheets worksheet")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-12s %s\n", f.Name, f.Usage)
	})

	fmt.Println(helpOptions())

	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug put --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println(`                                   --file "example.tsv"`)
	fmt.Println()
}