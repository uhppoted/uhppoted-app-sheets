package commands

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/device"
	"github.com/uhppoted/uhppote-core/uhppote"
	api "github.com/uhppoted/uhppoted-api/acl"
	"github.com/uhppoted/uhppoted-api/config"
)

var CompareACLCmd = CompareACL{
	workdir:     DEFAULT_WORKDIR,
	config:      config.DefaultConfig,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	acl:         "",
	report:      "Audit!A1:D",
	debug:       false,
}

type CompareACL struct {
	workdir     string
	config      string
	credentials string
	url         string
	acl         string
	report      string
	debug       bool
}

func (c *CompareACL) Name() string {
	return "compare-acl"
}

func (c *CompareACL) Description() string {
	return "Compare the access permission of a set of configured UHPPOTE access controllers to a Google Sheets worksheet"
}

func (c *CompareACL) Usage() string {
	return "--credentials <file> --url <url>"
}

func (c *CompareACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] [--config <configuration file>] compare-acl [options] --credentials <credentials> --url <URL> --range <range>\n", APP)
	fmt.Println()
	fmt.Println("  Compares the access permissions of a set of configured controllers to a Google Sheets worksheet access control list")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-13s %s\n", f.Name, f.Usage)
	})

	fmt.Println(helpOptions())

	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets compare-acl --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf compare-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                               --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                               --range "ACL!A2:E" \`)
	fmt.Println()
}

func (c *CompareACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("compare-acl", flag.ExitOnError)

	flagset.StringVar(&c.credentials, "credentials", c.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&c.url, "url", c.url, "Spreadsheet URL")
	flagset.StringVar(&c.acl, "range", c.acl, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.StringVar(&c.report, "report-range", c.report, "Spreadsheet range for compare report")
	flagset.StringVar(&c.workdir, "workdir", c.workdir, "Directory for working files (tokens, revisions, etc)")

	return flagset
}

func (cmd *CompareACL) Execute(ctx context.Context, options ...interface{}) error {
	if len(options) > 0 {
		if opt, ok := options[0].(*Options); ok {
			cmd.config = opt.Config
			cmd.debug = opt.Debug
		}
	}

	// ... check parameters
	if err := cmd.validate(); err != nil {
		return err
	}

	conf := config.NewConfig()
	if err := conf.Load(cmd.config); err != nil {
		return fmt.Errorf("WARN  Could not load configuration (%v)", err)
	}

	u, devices := getDevices(conf, cmd.debug)

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(strings.TrimSpace(cmd.url))
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]

	if cmd.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s  audit:%s", spreadsheetId, cmd.acl, cmd.report))
	}

	client, err := authorize(cmd.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(cmd.workdir, ".google"))
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

	list, err := cmd.getACL(google, spreadsheet, devices, ctx)
	if err != nil {
		return err
	}

	for k, l := range *list {
		info(fmt.Sprintf("%v  Downloaded %v records", k, len(l)))
	}

	diff, err := cmd.compare(&u, devices, list)
	if err != nil {
		return err
	}

	if err := cmd.write(google, spreadsheet, diff, ctx); err != nil {
		return err
	}

	return nil
}

func (c *CompareACL) validate() error {
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
		return fmt.Errorf("Invalid range '%s' - expected something like 'ACL!A2:K", c.acl)
	}

	if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(c.report); len(match) < 5 {
		return fmt.Errorf("Invalid report-range '%s' - expected something like 'Audit!A1:E", c.report)
	}

	return nil
}

func (c *CompareACL) compare(u device.IDevice, devices []*uhppote.Device, list *api.ACL) (*api.SystemDiff, error) {
	current, err := api.GetACL(u, devices)
	if err != nil {
		return nil, err
	}

	d, err := api.Compare(current, *list)
	if err != nil {
		return nil, err
	}

	diff := api.SystemDiff(d)

	return &diff, nil
}

func (c *CompareACL) getACL(google *sheets.Service, spreadsheet *sheets.Spreadsheet, devices []*uhppote.Device, ctx context.Context) (*api.ACL, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, c.acl).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return nil, fmt.Errorf("No data in spreadsheet/range")
	}

	table, err := makeTable(response.Values)
	if err != nil {
		return nil, fmt.Errorf("Error creating table from worksheet (%v)", err)
	}

	list, warnings, err := api.ParseTable(table, devices, false)
	if err != nil {
		return nil, err
	}

	if list == nil {
		return nil, fmt.Errorf("Error creating ACL from worksheet (%v)", list)
	}

	for _, w := range warnings {
		warn(w.Error())
	}

	return list, nil
}

func (c *CompareACL) write(google *sheets.Service, spreadsheet *sheets.Spreadsheet, diff *api.SystemDiff, ctx context.Context) error {
	// ... create report format
	sheet, err := getSheet(spreadsheet, c.report)
	if err != nil {
		return err
	}

	format, err := c.buildReportFormat(google, spreadsheet)
	if err != nil {
		return err
	}

	// ... clear existing report
	info("Clearing existing report from worksheet")
	if err := clear(google, spreadsheet, []string{format.title, format.data}, ctx); err != nil {
		return err
	}

	if sheet.Properties.GridProperties.RowCount > format.top+16 {
		prune := sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				&sheets.Request{
					DeleteDimension: &sheets.DeleteDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheet.Properties.SheetId,
							Dimension:  "ROWS",
							StartIndex: int64(format.top + 2),
						},
					},
				},
			},
		}

		if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &prune).Context(ctx).Do(); err != nil {
			return fmt.Errorf("Error pruning report worksheet (%w)", err)
		}
	}

	// ... write report
	info("Writing report to worksheet")

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

	keys := []uint32{}
	for k, _ := range *diff {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, k := range keys {
		if v, ok := (*diff)[k]; ok {
			top := len(values.Values)
			values.Values = append(values.Values, []interface{}{fmt.Sprintf("%v", k), "'-", "'-", "'-"})

			rows := len(v.Updated)
			if len(v.Added) > rows {
				rows = len(v.Added)
			}
			if len(v.Deleted) > rows {
				rows = len(v.Deleted)
			}

			for i := 1; i <= rows; i++ {
				values.Values = append(values.Values, []interface{}{"", "", "", ""})
			}

			for i, c := range v.Updated {
				values.Values[top+i][1] = fmt.Sprintf("%v", c.CardNumber)
			}

			for i, c := range v.Added {
				values.Values[top+i][2] = fmt.Sprintf("%v", c.CardNumber)
			}

			for i, c := range v.Deleted {
				values.Values[top+i][3] = fmt.Sprintf("%v", c.CardNumber)
			}
		}
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

	if _, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, c.report, &pad).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("OVERWRITE").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("Error padding report worksheet (%w)", err)
	}

	return nil
}

func (c *CompareACL) buildReportFormat(google *sheets.Service, spreadsheet *sheets.Spreadsheet) (*report, error) {
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(c.report)
	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	format := report{
		top:     int64(top),
		left:    left,
		title:   fmt.Sprintf("%v!%v%v:%v%v", name, left, top, left, top),
		data:    fmt.Sprintf("%v!%v%v:%v", name, left, top+2, right),
		columns: map[string]string{},
	}

	return &format, nil
}
