package commands

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/device"
	"github.com/uhppoted/uhppote-core/uhppote"
	api "github.com/uhppoted/uhppoted-api/acl"
	"github.com/uhppoted/uhppoted-api/config"
)

var UploadACLCmd = UploadACL{
	workdir:     DEFAULT_WORKDIR,
	config:      config.DefaultConfig,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	acl:         "",
	debug:       false,
}

type UploadACL struct {
	workdir     string
	config      string
	credentials string
	url         string
	acl         string
	debug       bool
}

func (c *UploadACL) Name() string {
	return "upload-acl"
}

func (c *UploadACL) Description() string {
	return "Uploads the access permissions from a set of configured UHPPOTE access controllers to a Google Sheets worksheet"
}

func (c *UploadACL) Usage() string {
	return "--credentials <file> --url <url>"
}

func (c *UploadACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] [--config <configuration>] upload-acl [options] --credentials <credentials> --url <URL> --range <range>\n", APP)
	fmt.Println()
	fmt.Println("  Uploads the access permissions from a set of configured controllers to a Google Sheets worksheet access control list")
	fmt.Println()
	fmt.Println("    --config <file>  Path to controllers configuration file")
	fmt.Println("    --debug          Displays internal information for diagnosing errors")
	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-13s %s\n", f.Name, f.Usage)
	})

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets upload-acl --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "Uploaded!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf upload-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                               --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                               --range "Uploaded!A2:E" \`)
	fmt.Println()
}

func (c *UploadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("upload-acl", flag.ExitOnError)

	flagset.StringVar(&c.credentials, "credentials", c.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&c.url, "url", c.url, "Spreadsheet URL")
	flagset.StringVar(&c.acl, "range", c.acl, "Spreadsheet range e.g. 'Uploaded!A2:E'")
	flagset.StringVar(&c.workdir, "workdir", c.workdir, "Directory for working files (tokens, revisions, etc)")
	flagset.StringVar(&c.config, "config", c.config, "Configuration file path")

	return flagset
}

func (c *UploadACL) Execute(ctx context.Context, options ...interface{}) error {
	if len(options) > 0 {
		if debug, ok := options[0].(bool); ok {
			c.debug = debug
		}
	}

	// ... check parameters
	if err := c.validate(); err != nil {
		return err
	}

	conf := config.NewConfig()
	if err := conf.Load(c.config); err != nil {
		return fmt.Errorf("WARN  Could not load configuration (%v)", err)
	}

	u, devices := getDevices(conf, c.debug)

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(strings.TrimSpace(c.url))
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]

	if c.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s", spreadsheetId, c.acl))
	}

	client, err := authorize(c.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(c.workdir, ".google"))
	if err != nil {
		return fmt.Errorf("Google Sheets authentication/authorization error (%w)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		return fmt.Errorf("Unable to create new Google Sheets client (%w)", err)
	}

	spreadsheet, err := getSpreadsheet(google, spreadsheetId)
	if err != nil {
		return err
	}

	acl, err := c.get(&u, devices)
	if err != nil {
		return err
	}

	table, err := api.MakeTable(acl, devices)
	if err != nil {
		return err
	}

	if err := c.upload(google, spreadsheet, table, ctx); err != nil {
		return err
	}

	return nil
}

func (c *UploadACL) validate() error {
	if strings.TrimSpace(c.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(c.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(c.acl) == "" {
		return fmt.Errorf("--range is a required option")
	}

	if match := regexp.MustCompile(`(.+?)!.*`).FindStringSubmatch(strings.TrimSpace(c.acl)); len(match) < 2 {
		return fmt.Errorf("Invalid range '%s' - expected something like 'Current!A2:K", c.acl)
	}

	return nil
}

func (c *UploadACL) get(u device.IDevice, devices []*uhppote.Device) (api.ACL, error) {
	current, err := api.GetACL(u, devices)
	if err != nil {
		return nil, err
	}

	return current, nil
}

func (c *UploadACL) upload(google *sheets.Service, spreadsheet *sheets.Spreadsheet, table *api.Table, ctx context.Context) error {
	sheet, err := getSheet(spreadsheet, c.acl)
	if err != nil {
		return err
	}

	format, err := c.buildFormat(google, spreadsheet, table)
	if err != nil {
		return err
	}

	// ... clear existing ACL
	info("Clearing existing ACL from worksheet")
	if err := clear(google, spreadsheet, []string{format.title, format.data}, ctx); err != nil {
		return err
	}

	if sheet.Properties.GridProperties.RowCount > format.top+24 {
		prune := sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				&sheets.Request{
					DeleteDimension: &sheets.DeleteDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheet.Properties.SheetId,
							Dimension:  "ROWS",
							StartIndex: int64(format.top + 24),
						},
					},
				},
			},
		}

		if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &prune).Context(ctx).Do(); err != nil {
			return fmt.Errorf("Error pruning report worksheet (%w)", err)
		}
	}

	// ... upload ACL
	info("Uploading ACL to worksheet")

	var timestamp = sheets.ValueRange{
		Range: format.title,
		Values: [][]interface{}{
			[]interface{}{
				time.Now().Format("2006-01-02 15:04:05"),
			},
		},
	}

	var values = sheets.ValueRange{
		Range:  format.data,
		Values: [][]interface{}{},
	}

	cols := 0
	for _, v := range format.xref {
		if v >= cols {
			cols = v + 1
		}
	}

	for _, record := range table.Records {
		row := make([]interface{}, cols)
		for i, _ := range row {
			row[i] = ""
		}

		for i, v := range record {
			if ix, ok := format.xref[i]; ok {
				row[ix] = fmt.Sprintf("%v", v)
			}
		}

		values.Values = append(values.Values, row)
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{&timestamp, &values},
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	// ... pad

	var pad = sheets.ValueRange{
		Values: [][]interface{}{[]interface{}{""}},
	}

	if _, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, c.acl, &pad).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("OVERWRITE").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("Error padding report worksheet (%w)", err)
	}

	return nil
}

func (c *UploadACL) buildFormat(google *sheets.Service, spreadsheet *sheets.Spreadsheet, table *api.Table) (*report, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, c.acl).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data from upload sheet (%v)", err)
	}

	columns := map[int]int{}
	rows := response.Values
	if len(rows) > 1 {
		header := rows[1]
		for i, col := range table.Header {
			p := normalise(col)
			for j, h := range header {
				if q, ok := h.(string); ok {
					if p == normalise(q) {
						columns[i] = j
					}
				}
			}
		}
	}

	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(c.acl)
	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	format := report{
		top:     int64(top),
		left:    left,
		title:   fmt.Sprintf("%v!%v%v:%v%v", name, left, top, left, top),
		headers: fmt.Sprintf("%v!%v%v:%v%v", name, left, top+1, right, top+1),
		data:    fmt.Sprintf("%v!%v%v:%v", name, left, top+2, right),
		columns: map[string]string{},
		xref:    columns,
	}

	return &format, nil
}
