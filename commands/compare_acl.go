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

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	lib "github.com/uhppoted/uhppoted-lib/acl"
	"github.com/uhppoted/uhppoted-lib/config"
)

var CompareACLCmd = CompareACL{
	command: command{
		workdir:     DEFAULT_WORKDIR,
		credentials: DEFAULT_CREDENTIALS,
		tokens:      "",
		url:         "",
		debug:       false,
	},

	config: config.DefaultConfig,
	acl:    "",
	report: "Audit!A1:D",
}

type CompareACL struct {
	command
	config  string
	acl     string
	report  string
	withPIN bool
}

func (cmd *CompareACL) Name() string {
	return "compare-acl"
}

func (cmd *CompareACL) Description() string {
	return "Compare the access permission of a set of configured UHPPOTE access controllers to a Google Sheets worksheet"
}

func (cmd *CompareACL) Usage() string {
	return "--credentials <file> --url <url>"
}

func (cmd *CompareACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] [--config <configuration file>] compare-acl [options] --url <URL> --range <range> --report-range <range>\n", APP)
	fmt.Println()
	fmt.Println("  Compares the access permissions of a set of configured controllers to a Google Sheets worksheet access control list")
	fmt.Println()

	helpOptions(cmd.FlagSet())

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println(`    uhppote-app-sheets compare-acl --credentials "credentials.json" \`)
	fmt.Println(`                                   --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                   --range "ACL!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf compare-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                               --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                               --range "ACL!A2:E" \`)
	fmt.Println()
}

func (cmd *CompareACL) FlagSet() *flag.FlagSet {
	flagset := cmd.flagset("compare-acl")

	flagset.StringVar(&cmd.acl, "range", cmd.acl, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.StringVar(&cmd.report, "report-range", cmd.report, "Spreadsheet range for compare report e.g. 'Audit!A1:D'")
	flagset.BoolVar(&cmd.withPIN, "with-pin", cmd.withPIN, "Includes the card keypad PIN codes when comparing ACLs")

	return flagset
}

func (cmd *CompareACL) Execute(args ...interface{}) error {
	options := args[0].(*Options)

	cmd.config = options.Config
	cmd.debug = options.Debug

	// ... check parameters
	if err := cmd.validate(); err != nil {
		return err
	}

	conf := config.NewConfig()
	if err := conf.Load(cmd.config); err != nil {
		return fmt.Errorf("could not load configuration (%v)", err)
	}

	u, devices := getDevices(conf, cmd.debug)

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(strings.TrimSpace(cmd.url))
	if len(match) < 2 {
		return fmt.Errorf("invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]

	if cmd.debug {
		debugf("Spreadsheet - ID:%s  range:%s  audit:%s", spreadsheetId, cmd.acl, cmd.report)
	}

	// ... authorise
	tokens := cmd.tokens
	if tokens == "" {
		tokens = filepath.Join(cmd.workdir, ".google")
	}

	client, err := authorize(cmd.credentials, SHEETS, tokens)
	if err != nil {
		//lint:ignore ST1005 Google should be capitalized
		return fmt.Errorf("Google Sheets authentication/authorization error (%w)", err)
	}

	google, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to create new Google Sheets client (%w)", err)
	}

	spreadsheet, err := getSpreadsheet(google, spreadsheetId)
	if err != nil {
		return err
	}

	list, err := cmd.getACL(google, spreadsheet, devices)
	if err != nil {
		return err
	}

	for k, l := range *list {
		infof("%v  Downloaded %v records", k, len(l))
	}

	diff, err := cmd.compare(u, devices, list)
	if err != nil {
		return err
	}

	if err := cmd.write(google, spreadsheet, diff); err != nil {
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
		return fmt.Errorf("invalid range '%s' - expected something like 'ACL!A2:K", c.acl)
	}

	if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(c.report); len(match) < 5 {
		return fmt.Errorf("invalid report-range '%s' - expected something like 'Audit!A1:E", c.report)
	}

	return nil
}

func (cmd *CompareACL) compare(u uhppote.IUHPPOTE, devices []uhppote.Device, list *lib.ACL) (*lib.SystemDiff, error) {
	current, errors := lib.GetACL(u, devices)
	if len(errors) > 0 {
		return nil, fmt.Errorf("%v", errors)
	}

	f := func(current lib.ACL, list lib.ACL) (map[uint32]lib.Diff, error) {
		if cmd.withPIN {
			return lib.CompareWithPIN(current, list)
		} else {
			return lib.Compare(current, list)
		}
	}

	if d, err := f(current, *list); err != nil {
		return nil, err
	} else {
		diff := lib.SystemDiff(d)

		return &diff, nil
	}
}

func (cmd *CompareACL) getACL(google *sheets.Service, spreadsheet *sheets.Spreadsheet, devices []uhppote.Device) (*lib.ACL, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, cmd.acl).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return nil, fmt.Errorf("no data in spreadsheet/range")
	}

	table, err := makeTable(response.Values)
	if err != nil {
		return nil, fmt.Errorf("error creating table from worksheet (%v)", err)
	}

	f := func(table *lib.Table, devices []uhppote.Device) (*lib.ACL, []error, error) {
		if cmd.withPIN {
			return lib.ParseTable(table, devices, false)
		} else {
			return lib.ParseTable(table, devices, false)
		}
	}

	if list, warnings, err := f(table, devices); err != nil {
		return nil, err
	} else if list == nil {
		return nil, fmt.Errorf("error creating ACL from worksheet (%v)", list)
	} else {
		for _, w := range warnings {
			warnf("%v", w.Error())
		}

		return list, nil
	}
}

func (c *CompareACL) write(google *sheets.Service, spreadsheet *sheets.Spreadsheet, diff *lib.SystemDiff) error {
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
	infof("Clearing existing report from worksheet")
	if err := clear(google, spreadsheet, []string{format.title, format.data}); err != nil {
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

		if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &prune).Do(); err != nil {
			return fmt.Errorf("error pruning report worksheet (%w)", err)
		}
	}

	// ... write report
	infof("Writing report to worksheet")

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
	for k := range *diff {
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

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Do(); err != nil {
		return err
	}

	// ... pad

	var pad = sheets.ValueRange{
		Values: [][]interface{}{[]interface{}{""}},
	}

	if _, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, c.report, &pad).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("OVERWRITE").
		Do(); err != nil {
		return fmt.Errorf("error padding report worksheet (%w)", err)
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
