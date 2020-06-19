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
	credentials:  "",
	url:          "",
	region:       "",
	logRange:     "Log!A1:G",
	logRetention: 30,
	dryrun:       false,
	config:       config.DefaultConfig,
	nolog:        false,
	debug:        false,
}

type LoadACL struct {
	credentials  string
	url          string
	region       string
	logRange     string
	logRetention uint
	dryrun       bool
	config       string
	nolog        bool
	debug        bool
}

func (l *LoadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("load-acl", flag.ExitOnError)

	flagset.StringVar(&l.credentials, "credentials", l.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&l.url, "url", l.url, "Spreadsheet URL")
	flagset.StringVar(&l.region, "range", l.region, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flagset.StringVar(&l.logRange, "log-range", l.logRange, fmt.Sprintf("Spreadsheet range for logging result. Defaults to %s", l.logRange))
	flagset.UintVar(&l.logRetention, "log-retention", l.logRetention, fmt.Sprintf("Log sheet records older than 'log-retention' days are automatically pruned. Defaults to %v", l.logRetention))
	flagset.StringVar(&l.config, "config", l.config, "Configuration file path")
	flagset.BoolVar(&l.nolog, "no-log", l.nolog, "Disables writing a summary to the 'log' worksheet")
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

	if strings.TrimSpace(l.region) == "" {
		return fmt.Errorf("--range is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(l.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]
	region := l.region
	if l.debug {
		log.Printf("DEBUG  Spreadsheet - ID:%s  range:%s  log:%s", spreadsheetId, region, l.logRange)
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

	list, err := l.getACL(google, spreadsheetId, region, devices, ctx)
	if err != nil {
		return err
	}

	for k, l := range *list {
		log.Printf("%v  Retrieved %v records", k, len(l))
	}

	rpt, err := api.PutACL(&u, *list, l.dryrun)
	for k, v := range rpt {
		log.Printf("%v  SUMMARY  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v", k, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed)
	}

	if !l.nolog {
		err = l.updateLogSheet(google, spreadsheet, rpt, ctx)
		if err != nil {
			return err
		}

		err = l.pruneLogSheet(google, spreadsheet, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LoadACL) getACL(google *sheets.Service, spreadsheet, area string, devices []*uhppote.Device, ctx context.Context) (*api.ACL, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet, area).Do()
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
		rows.Values = append(rows.Values, []interface{}{timestamp, k, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed})
	}

	_, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, l.logRange, &rows).ValueInputOption("RAW").InsertDataOption("OVERWRITE").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("Error writing log to Google Sheets (%w)", err)
	}

	return nil
}

func (l *LoadACL) pruneLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ctx context.Context) error {
	logSheetID, err := getSheetID(spreadsheet, l.logRange)
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
							SheetId:    logSheetID,
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
