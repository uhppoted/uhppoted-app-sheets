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

	"github.com/uhppoted/uhppoted-app-sheets/acl"
)

var GetACLCmd = GetACL{
	credentials: "",
	url:         "",
	region:      "",
	file:        time.Now().Format("2006-01-02T150405.acl"),
	debug:       false,
}

type GetACL struct {
	credentials string
	url         string
	region      string
	file        string
	debug       bool
}

func (c *GetACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("get-acl", flag.ExitOnError)

	flagset.StringVar(&c.credentials, "credentials", c.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&c.url, "url", c.url, "Spreadsheet URL")
	flagset.StringVar(&c.region, "range", c.region, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flagset.StringVar(&c.file, "file", c.file, "TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")

	return flagset
}

func (c *GetACL) Execute(ctx context.Context) error {
	if strings.TrimSpace(c.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(c.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(c.region) == "" {
		return fmt.Errorf("--range is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(c.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheet := match[1]
	region := c.region

	if c.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s", spreadsheet, region))
	}

	client, err := authorize(c.credentials)
	if err != nil {
		return fmt.Errorf("Authentication/authorization error (%v)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		return fmt.Errorf("Unable to create new Sheets client (%v)", err)
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet, region).Do()
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

	if err := acl.MakeTSV(tmp, response); err != nil {
		return fmt.Errorf("Error creating TSV file (%v)", err)
	}

	tmp.Close()

	dir := filepath.Dir(c.file)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), c.file); err != nil {
		return err
	}

	info(fmt.Sprintf("Retrieved ACL to file %s\n", c.file))

	return nil
}

func (c *GetACL) Name() string {
	return "get-acl"
}

func (c *GetACL) Description() string {
	return "Retrieves an access control list from a Google Sheets worksheet and stores it to a local file"
}

func (c *GetACL) Usage() string {
	return "--credentials <file> --url <url> --file <file>"
}

func (c *GetACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [options] get-acl --credentials <credentials> --url <URL> --range <range> --file <file>\n", APP)
	fmt.Println()
	fmt.Println("  Retrieves an access control list from a Google Sheets worksheet and writes it to a file in TSV format")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-12s %s\n", f.Name, f.Usage)
	})

	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println()
	fmt.Println("    --debug   Displays vaguely useful internal information")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug get-acl --credentials "credentials.json" \`)
	fmt.Println(`                                       --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                       --range "Class Data!A2:E" \`)
	fmt.Println(`                                       --file "example.acl"`)
	fmt.Println()
}
