package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	api "github.com/uhppoted/uhppoted-api/acl"
	"github.com/uhppoted/uhppoted-api/config"
	"github.com/uhppoted/uhppoted-app-sheets/acl"
)

var LoadACLCmd = LoadACL{
	config:      config.DefaultConfig,
	credentials: "",
	url:         "",
	area:        "",

	nolog:        false,
	logRange:     "Log!A1:H",
	logRetention: 30,

	noreport:    false,
	reportRange: "Report!A1:H",

	dryrun: false,
	debug:  false,
}

type LoadACL struct {
	config      string
	credentials string
	url         string
	area        string

	nolog        bool
	logRange     string
	logRetention uint

	noreport    bool
	reportRange string

	dryrun bool
	debug  bool
}

func (l *LoadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("load-acl", flag.ExitOnError)

	flagset.StringVar(&l.credentials, "credentials", l.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&l.url, "url", l.url, "Spreadsheet URL")
	flagset.StringVar(&l.area, "range", l.area, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flagset.StringVar(&l.reportRange, "report-range", l.reportRange, fmt.Sprintf("Spreadsheet range for load report. Defaults to %s", l.reportRange))
	flagset.StringVar(&l.logRange, "log-range", l.logRange, fmt.Sprintf("Spreadsheet range for logging result. Defaults to %s", l.logRange))
	flagset.UintVar(&l.logRetention, "log-retention", l.logRetention, fmt.Sprintf("Log sheet records older than 'log-retention' days are automatically pruned. Defaults to %v", l.logRetention))
	flagset.StringVar(&l.config, "config", l.config, "Configuration file path")
	flagset.BoolVar(&l.nolog, "no-log", l.nolog, "Disables writing a summary to the 'log' worksheet")
	flagset.BoolVar(&l.noreport, "no-report", l.noreport, "Disables writing a report to the 'report' worksheet")
	flagset.BoolVar(&l.dryrun, "dryrun", l.dryrun, "Simulates a load-acl without making any changes to the access controllers")

	return flagset
}

func (l *LoadACL) Execute(ctx context.Context) error {
	conf := config.NewConfig()
	if err := conf.Load(l.config); err != nil {
		return fmt.Errorf("WARN  Could not load configuration (%v)", err)
	}

	u, devices := getDevices(conf, l.debug)

	if strings.TrimSpace(l.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(l.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(l.area) == "" {
		return fmt.Errorf("--range is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(l.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]
	if l.debug {
		log.Printf("DEBUG  Spreadsheet - ID:%s  range:%s  log:%s", spreadsheetId, l.area, l.logRange)
	}

	client, err := authorize(l.credentials)
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

	list, err := l.getACL(google, spreadsheet, devices, ctx)
	if err != nil {
		return err
	}

	for k, l := range *list {
		log.Printf("%v  Retrieved %v records", k, len(l))
	}

	rpt, err := api.PutACL(&u, *list, l.dryrun)
	for k, v := range rpt {
		log.Printf("%v  SUMMARY  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v  errors:%v",
			k,
			len(v.Unchanged),
			len(v.Updated),
			len(v.Added),
			len(v.Deleted),
			len(v.Failed),
			len(v.Errors))
	}

	if !l.nolog {
		if err := l.updateLogSheet(google, spreadsheet, rpt, ctx); err != nil {
			return err
		}

		if err := l.pruneLogSheet(google, spreadsheet, ctx); err != nil {
			return err
		}
	}

	if !l.noreport {
		if err := l.updateReportSheet(google, spreadsheet, rpt, ctx); err != nil {
			return err
		}
	}

	return nil
}

func (l *LoadACL) getACL(google *sheets.Service, spreadsheet *sheets.Spreadsheet, devices []*uhppote.Device, ctx context.Context) (*api.ACL, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.area).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return nil, fmt.Errorf("No data in spreadsheet/range")
	}

	table, err := acl.MakeTable(response)
	if err != nil {
		return nil, fmt.Errorf("Error creating table from worksheet (%v)", err)
	}

	list, err := api.ParseTable(table, devices)
	if err != nil {
		return nil, err
	} else if list == nil {
		return nil, fmt.Errorf("Error creating ACL from worksheet (%v)", list)
	}

	return list, nil
}

func (l *LoadACL) updateLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, report map[uint32]api.Report, ctx context.Context) error {
	var rows = sheets.ValueRange{
		Values: [][]interface{}{},
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	for k, v := range report {
		rows.Values = append(rows.Values, []interface{}{
			timestamp,
			k,
			len(v.Unchanged),
			len(v.Updated),
			len(v.Added),
			len(v.Deleted),
			len(v.Failed),
			len(v.Errors),
		})
	}

	_, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, l.logRange, &rows).ValueInputOption("RAW").InsertDataOption("OVERWRITE").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("Error writing log to Google Sheets (%w)", err)
	}

	return nil
}

func (l *LoadACL) pruneLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ctx context.Context) error {
	sheet, err := getSheet(spreadsheet, l.logRange)
	if err != nil {
		return err
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.logRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve data from Log sheet (%v)", err)
	}

	before := time.Now().
		In(time.Local).
		Add(time.Hour * time.Duration(-24*(int(l.logRetention)-1))).
		Truncate(24 * time.Hour)

	cutoff := time.Date(before.Year(), before.Month(), before.Day(), 0, 0, 0, 0, before.Location())
	list := []int{}
	deleted := 0

	log.Printf("Pruning log records from before %v", cutoff.Format("2006-01-02"))

	for row, record := range response.Values {
		timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", record[0].(string), time.Local)
		if err == nil && timestamp.Before(cutoff) {
			list = append(list, row)
		}
	}

	if len(list) > 0 {
		sort.Ints(list[:])

		ranges := map[int]int{}
		start := list[0]
		last := list[0]
		for _, row := range list[1:] {
			if row != last+1 {
				ranges[start] = last
				start = row
			}

			last = row
		}
		ranges[start] = last

		if len(ranges) > 0 {
			rq := sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{},
			}

			for start, end := range ranges {
				rq.Requests = append(rq.Requests, &sheets.Request{
					DeleteDimension: &sheets.DeleteDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheet.Properties.SheetId,
							Dimension:  "ROWS",
							StartIndex: int64(start - deleted),
							EndIndex:   int64(end - deleted + 1),
						},
					},
				})

				deleted += end - start + 1
			}

			if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
				return err
			}
		}
	}

	log.Printf("Pruned %d log records from log sheet", deleted)

	return nil
}

func (l *LoadACL) updateReportSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, report map[uint32]api.Report, ctx context.Context) error {
	// ... consolidate report

	consolidated := map[uint32]*struct {
		updated bool
		added   bool
		deleted bool
		failed  bool
	}{}

	for _, r := range report {
		lists := [][]uint32{r.Updated, r.Added, r.Deleted, r.Failed}
		for _, l := range lists {
			for _, card := range l {
				consolidated[card] = &struct {
					updated bool
					added   bool
					deleted bool
					failed  bool
				}{}
			}
		}
	}

	for _, r := range report {
		for _, card := range r.Updated {
			consolidated[card].updated = true
		}

		for _, card := range r.Added {
			consolidated[card].added = true
		}

		for _, card := range r.Deleted {
			consolidated[card].deleted = true
		}

		for _, card := range r.Failed {
			consolidated[card].failed = true
		}
	}

	updated := []uint32{}
	added := []uint32{}
	deleted := []uint32{}
	failed := []uint32{}

	for card, s := range consolidated {
		if s.updated {
			updated = append(updated, card)
		}

		if s.added {
			added = append(added, card)
		}

		if s.deleted {
			deleted = append(deleted, card)
		}
		if s.failed {
			failed = append(failed, card)
		}
	}

	sort.Slice(updated, func(i, j int) bool { return updated[i] < updated[j] })
	sort.Slice(added, func(i, j int) bool { return added[i] < added[j] })
	sort.Slice(deleted, func(i, j int) bool { return deleted[i] < deleted[j] })
	sort.Slice(failed, func(i, j int) bool { return failed[i] < failed[j] })

	// ... clear existing report

	sheet, err := getSheet(spreadsheet, l.reportRange)
	if err != nil {
		return err
	}

	start := sheet.Properties.GridProperties.FrozenRowCount
	end := sheet.Properties.GridProperties.RowCount

	log.Printf("Clearing old report data from worksheet")

	if end > start {
		title := fmt.Sprintf("Report!A1:D1")
		data := fmt.Sprintf("Report!A3:D")

		rq := sheets.BatchClearValuesRequest{
			Ranges: []string{title, data},
		}

		if _, err := google.Spreadsheets.Values.BatchClear(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
			return err
		}
	}

	// ... write report

	log.Printf("Writing report to worksheet")

	var title = sheets.ValueRange{
		Range: "Report!A1:A1",
		Values: [][]interface{}{
			[]interface{}{
				time.Now().Format("ACL Update Report: 2006-01-02 15:04:05"),
			},
		},
	}

	var A = sheets.ValueRange{
		Range:  "Report!A3:A",
		Values: [][]interface{}{},
	}
	var B = sheets.ValueRange{
		Range:  "Report!B3:B",
		Values: [][]interface{}{},
	}

	var C = sheets.ValueRange{
		Range:  "Report!C3:C",
		Values: [][]interface{}{},
	}

	var D = sheets.ValueRange{
		Range:  "Report!D3:D",
		Values: [][]interface{}{},
	}

	for _, card := range updated {
		A.Values = append(A.Values, []interface{}{fmt.Sprintf("%v", card)})
	}

	for _, card := range added {
		B.Values = append(B.Values, []interface{}{fmt.Sprintf("%v", card)})
	}

	for _, card := range deleted {
		C.Values = append(C.Values, []interface{}{fmt.Sprintf("%v", card)})
	}

	for _, card := range failed {
		D.Values = append(D.Values, []interface{}{fmt.Sprintf("%v", card)})
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "RAW",
		Data:             []*sheets.ValueRange{&title, &A, &B, &C, &D},
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	return nil
}

func (c *LoadACL) Name() string {
	return "load-acl"
}

func (c *LoadACL) Description() string {
	return "Updates a set of configured UHPPOTE access controllers from a Google Sheets worksheet"
}

func (c *LoadACL) Usage() string {
	return "--credentials <file> --url <url>"
}

func (c *LoadACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [options] load-acl --credentials <credentials> --url <URL> --range <range> --dryrun\n", APP)
	fmt.Println()
	fmt.Println("  Retrieves an access control list from a Google Sheets worksheet and writes it to a file in TSV format")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-12s %s\n", f.Name, f.Usage)
	})

	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println()
	fmt.Println("    --config <file>  Path to controllers configuration file")
	fmt.Println("    --debug          Displays vaguely useful internal information")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                        --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                        --range "Class Data!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --conf example.conf load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                    --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                    --range "Class Data!A2:E" \`)
	fmt.Println()
}
