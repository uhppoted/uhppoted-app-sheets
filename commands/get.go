package commands

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

var GetCmd = Get{
	workdir:     DEFAULT_WORKDIR,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	area:        "",
	file:        time.Now().Format("2006-01-02T150405.acl"),
	debug:       false,
}

type Get struct {
	workdir     string
	credentials string
	url         string
	area        string
	file        string
	debug       bool
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
	fmt.Printf("  Usage: %s [--debug] get --credentials <credentials> --url <URL> --range <range> --file <file>\n", APP)
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
	flagset := flag.NewFlagSet("get", flag.ExitOnError)

	flagset.StringVar(&cmd.workdir, "workdir", cmd.workdir, "Directory for working files (tokens, revisions, etc)'")
	flagset.StringVar(&cmd.credentials, "credentials", cmd.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&cmd.url, "url", cmd.url, "Spreadsheet URL")
	flagset.StringVar(&cmd.area, "range", cmd.area, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.StringVar(&cmd.file, "file", cmd.file, "TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")

	return flagset
}

func (cmd *Get) Execute(ctx context.Context, options ...interface{}) error {
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

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(cmd.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheet := match[1]
	area := cmd.area

	if cmd.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s", spreadsheet, area))
	}

	client, err := authorize(cmd.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(cmd.workdir, ".google"))
	if err != nil {
		return fmt.Errorf("Authentication/authorization error (%v)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		return fmt.Errorf("Unable to create new Sheets client (%v)", err)
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet, area).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return fmt.Errorf("No data in spreadsheet/range")
	}

	tmp, err := ioutil.TempFile(os.TempDir(), "ACL")
	if err != nil {
		return err
	}

	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if err := sheetToTSV(tmp, response); err != nil {
		return fmt.Errorf("Error creating TSV file (%v)", err)
	}

	tmp.Close()

	dir := filepath.Dir(cmd.file)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), cmd.file); err != nil {
		return err
	}

	info(fmt.Sprintf("Retrieved ACL to file %s\n", cmd.file))

	return nil
}
