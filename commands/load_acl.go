package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	api "github.com/uhppoted/uhppoted-api/acl"
	"github.com/uhppoted/uhppoted-api/config"
	"github.com/uhppoted/uhppoted-app-sheets/acl"
)

var LoadACLCmd = LoadACL{
	credentials: "",
	url:         "",
	region:      "",
	logsheet:    "Log!A2:G",
	dryrun:      false,
	config:      config.DefaultConfig,
	debug:       false,
}

type LoadACL struct {
	credentials string
	url         string
	region      string
	logsheet    string
	dryrun      bool
	config      string
	debug       bool
}

func (l *LoadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("load-acl", flag.ExitOnError)

	flagset.StringVar(&l.credentials, "credentials", l.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&l.url, "url", l.url, "Spreadsheet URL")
	flagset.StringVar(&l.region, "range", l.region, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flagset.StringVar(&l.logsheet, "log", l.logsheet, fmt.Sprintf("Spreadsheet range for logging result. Defaults to %s", l.logsheet))
	flagset.StringVar(&l.config, "config", l.config, "Configuration file path")
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
		fmt.Errorf("--range is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(l.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheet := match[1]
	region := l.region

	if l.debug {
		log.Printf("DEBUG  Spreadsheet - ID:%s  range:%s  log:%s", spreadsheet, region, l.logsheet)
	}

	client, err := authorize(l.credentials)
	if err != nil {
		return fmt.Errorf("Authentication/authorization error (%v)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		return fmt.Errorf("Unable to create new Sheets client (%v)", err)
	}

	list, err := l.getACL(google, spreadsheet, region, devices, ctx)
	if err != nil {
		return err
	}

	for k, l := range *list {
		log.Printf("%v  Retrieved %v records", k, len(l))
	}

	rpt, err := api.PutACL(&u, *list)
	for k, v := range rpt {
		log.Printf("%v  SUMMARY  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v", k, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed)
	}

	err = l.logToWorksheet(google, spreadsheet, rpt, ctx)
	if err != nil {
		return err
	}

	return nil
}

func (l *LoadACL) getACL(google *sheets.Service, spreadsheet, region string, devices []*uhppote.Device, ctx context.Context) (*api.ACL, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet, region).Do()
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

func (l *LoadACL) logToWorksheet(google *sheets.Service, spreadsheet string, report map[uint32]api.Report, ctx context.Context) error {
	var rows = sheets.ValueRange{
		Values: [][]interface{}{},
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	for k, v := range report {
		rows.Values = append(rows.Values, []interface{}{timestamp, k, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed})
	}

	_, err := google.Spreadsheets.Values.Append(spreadsheet, l.logsheet, &rows).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("Error writing log to Google Sheets (%w)", err)
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
