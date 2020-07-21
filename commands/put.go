package commands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

func (c *Put) Execute(ctx context.Context) error {
	if strings.TrimSpace(c.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(c.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(c.area) == "" {
		return fmt.Errorf("--range is a required option")
	}

	if strings.TrimSpace(c.file) == "" {
		return fmt.Errorf("--file is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(c.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]
	region := c.area

	if c.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s", spreadsheetId, region))
	}

	client, err := authorize(c.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(c.workdir, ".google"))
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

	f, err := os.Open(c.file)
	if err != nil {
		return err
	}

	defer f.Close()

	header, data, err := tsvToSheet(f, c.area)
	if err != nil {
		return err
	} else if header == nil {
		return fmt.Errorf("Invalid TSV file (%v)", err)
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{header, data},
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	info(fmt.Sprintf("Uploaded TSV file %v to Google Sheets %v", c.file, c.area))

	return nil
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
	fmt.Printf("  Usage: %s [options] put --credentials <credentials> --url <URL> --range <range> --file <file>\n", APP)
	fmt.Println()
	fmt.Println("  Uploads a TSV file to a Google Sheets worksheet")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-12s %s\n", f.Name, f.Usage)
	})

	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println()
	fmt.Println("    --debug Displays internal information for diagnosing errors")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug put --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println(`                                   --file "example.tsv"`)
	fmt.Println()
}
