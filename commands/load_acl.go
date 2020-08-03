package commands

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/device"
	"github.com/uhppoted/uhppote-core/uhppote"
	api "github.com/uhppoted/uhppoted-api/acl"
	"github.com/uhppoted/uhppoted-api/config"
)

var LoadACLCmd = LoadACL{
	workdir:     DEFAULT_WORKDIR,
	config:      config.DefaultConfig,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	area:        "",

	nolog:           false,
	logRange:        "Log!A1:H",
	reportRetention: 7,
	logRetention:    30,

	noreport:    false,
	reportRange: "Report!A1:E",

	force:     false,
	strict:    false,
	dryrun:    false,
	debug:     false,
	delay:     delay(15 * time.Minute),
	revisions: filepath.Join(DEFAULT_WORKDIR, ".google", "uhppoted-app-sheets.revision"),
}

type LoadACL struct {
	workdir     string
	config      string
	credentials string
	url         string
	area        string

	nolog        bool
	logRange     string
	logRetention int

	noreport        bool
	reportRange     string
	reportRetention int

	force     bool
	strict    bool
	dryrun    bool
	debug     bool
	delay     delay
	revisions string
}

type delay time.Duration

func (d delay) String() string {
	return time.Duration(d).String()
}

func (d *delay) Set(s string) error {
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = delay(duration)

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
	fmt.Printf("  Usage: %s [--debug] [--config <configuration>] load-acl [options] --credentials <credentials> --url <URL> --range <range>\n", APP)
	fmt.Println()
	fmt.Println("  Updates the cards on a set of configured controllers from a Google Sheets worksheet access control list. Unless the --force option")
	fmt.Println("  is specified updates will be silently ignored (no log and no report) if the spreadsheet revision has not changed or the updated")
	fmt.Println("  spreadsheet contains no relevant changes.")
	fmt.Println()
	fmt.Println("  Duplicate card numbers are automatically deleted across the system unless the --strict option is provided to fail the load.")
	fmt.Println()

	c.FlagSet().VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-13s %s\n", f.Name, f.Usage)
	})

	fmt.Println(helpOptions())

	fmt.Println("  Examples:")
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                --range "ACL!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                            --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                            --range "ACL!A2:E" \`)
	fmt.Println()
}

func (l *LoadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("load-acl", flag.ExitOnError)

	flagset.StringVar(&l.credentials, "credentials", l.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&l.url, "url", l.url, "Spreadsheet URL")
	flagset.StringVar(&l.area, "range", l.area, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.BoolVar(&l.force, "force", l.force, "Forces an update, overriding the spreadsheet version and compare logic")
	flagset.BoolVar(&l.strict, "strict", l.strict, "Fails with an error if the spreadsheet contains duplicate card numbers")
	flagset.BoolVar(&l.dryrun, "dry-run", l.dryrun, "Simulates a load-acl without making any changes to the access controllers")
	flagset.Var(&l.delay, "delay", "Sets the delay between when a spreadsheet is modified and when it is regarded as sufficiently stable to use")

	flagset.BoolVar(&l.nolog, "no-log", l.nolog, "Disables writing a summary to the 'log' worksheet")
	flagset.StringVar(&l.logRange, "log-range", l.logRange, "Spreadsheet range for logging result")
	flagset.IntVar(&l.logRetention, "log-retention", l.logRetention, "Log sheet records older than 'log-retention' days are automatically pruned")

	flagset.BoolVar(&l.noreport, "no-report", l.noreport, "Disables writing a report to the 'report' worksheet")
	flagset.StringVar(&l.reportRange, "report-range", l.reportRange, "Spreadsheet range for load report")
	flagset.IntVar(&l.reportRetention, "report-retention", l.reportRetention, "Report sheet records older than 'report-retention' days are automatically pruned")

	flagset.StringVar(&l.workdir, "workdir", l.workdir, "Directory for working files (tokens, revisions, etc)")

	return flagset
}

func (cmd *LoadACL) Execute(ctx context.Context, options ...interface{}) error {
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

	// ... locked?
	lockfile, err := cmd.lock()
	if err != nil {
		return err
	} else {
		defer func() {
			info(fmt.Sprintf("Removing lockfile '%v'", lockfile))
			os.Remove(lockfile)
		}()
	}

	// ... good to go!
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
	cmd.revisions = filepath.Join(cmd.workdir, ".google", fmt.Sprintf("%s.revision", spreadsheetId))

	if cmd.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s  log:%s", spreadsheetId, cmd.area, cmd.logRange))
	}

	version, err := cmd.getRevision(spreadsheetId, ctx)
	if err != nil {
		fatal(err.Error())
	}

	if !cmd.force && !cmd.revised(version) {
		info("Nothing to do")
		return nil
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

	list, warnings, err := cmd.getACL(google, spreadsheet, devices, ctx)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		warn(w.Error())
	}

	for k, l := range *list {
		info(fmt.Sprintf("%v  Downloaded %v records", k, len(l)))
	}

	updated, err := cmd.compare(&u, devices, list)
	if err != nil {
		return err
	}

	if cmd.force || updated {
		rpt, err := api.PutACL(&u, *list, cmd.dryrun)
		if err != nil {
			return err
		}

		for _, w := range warnings {
			if duplicate, ok := w.(*api.DuplicateCardError); ok {
				for k, v := range rpt {
					v.Errored = append(v.Errored, duplicate.CardNumber)
					rpt[k] = v
				}
			}
		}

		summary := api.Summarize(rpt)
		format := "%v  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v  errors:%v"
		for _, v := range summary {
			info(fmt.Sprintf(format, v.DeviceID, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed, v.Errored+len(warnings)))
		}

		for k, v := range rpt {
			for _, err := range v.Errors {
				fatal(fmt.Sprintf("%v  %v", k, err))
			}
		}

		if !cmd.nolog {
			if err := cmd.updateLogSheet(google, spreadsheet, rpt, ctx); err != nil {
				return err
			}

			if err := pruneSheet(google, spreadsheet, cmd.logRange, cmd.logRetention, ctx); err != nil {
				return err
			}
		}

		if !cmd.noreport {
			if err := cmd.updateReportSheet(google, spreadsheet, rpt, ctx); err != nil {
				return err
			}
		}
	} else {
		info("No changes - Nothing to do")
	}

	if version != nil {
		version.store(cmd.revisions)
	}

	return nil
}

func (l *LoadACL) lock() (string, error) {
	lockfile := filepath.Join(l.workdir, ".google", "uhppoted-app-sheets.lock")
	pid := fmt.Sprintf("%d\n", os.Getpid())

	if err := os.MkdirAll(filepath.Dir(lockfile), 0770); err != nil {
		return "", fmt.Errorf("Unable to create directory '%v' for lockfile (%v)", lockfile, err)
	}

	if _, err := os.Stat(lockfile); err == nil {
		return "", fmt.Errorf("Locked by '%v'", lockfile)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("Error checking PID lockfile '%v' (%v)", lockfile, err)
	}

	if err := ioutil.WriteFile(lockfile, []byte(pid), 0660); err != nil {
		return "", fmt.Errorf("Unable to create lockfile '%v' (%v)", lockfile, err)
	}

	return lockfile, nil
}

func (l *LoadACL) validate() error {
	if strings.TrimSpace(l.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(l.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	if strings.TrimSpace(l.area) == "" {
		return fmt.Errorf("--range is a required option")
	}

	if match := regexp.MustCompile(`(.+?)!.*`).FindStringSubmatch(strings.TrimSpace(l.area)); len(match) < 2 {
		return fmt.Errorf("Invalid range '%s' - expected something like 'ACL!A2:K", l.area)
	}

	if !l.nolog {
		if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+[0-9]+):([a-zA-Z]+(?:[0-9]+)?)`).FindStringSubmatch(l.logRange); len(match) < 4 {
			return fmt.Errorf("Invalid log-range '%s' - expected something like 'Log!A1:H", l.logRange)
		}
	}

	if !l.noreport {
		if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(l.reportRange); len(match) < 5 {
			return fmt.Errorf("Invalid report-range '%s' - expected something like 'Report!A1:E", l.reportRange)
		}
	}

	return nil
}

func (l *LoadACL) getRevision(spreadsheetId string, ctx context.Context) (*revision, error) {
	client, err := authorize(l.credentials, drive.DriveMetadataReadonlyScope, filepath.Join(l.workdir, ".google"))
	if err != nil {
		return nil, fmt.Errorf("Google Drive authentication/authorization error (%w)", err)
	}

	gdrive, err := drive.New(client)
	if err != nil {
		return nil, fmt.Errorf("Unable to create new Google Drive client (%w)", err)
	}

	version, err := getRevision(gdrive, spreadsheetId, ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve spreadsheet revision (%v)", err)
	}

	return version, nil
}

func (l *LoadACL) revised(version *revision) bool {
	if version != nil {
		info(fmt.Sprintf("Latest revision %v, %s", version.ID, version.Modified.Local().Format("2006-01-02 15:04:05 MST")))

		var last revision

		if err := last.load(l.revisions); err != nil {
			fatal(fmt.Sprintf("Error reading last revision from %s", l.revisions))
			fatal(fmt.Sprintf("%v", err))
		} else {
			info(fmt.Sprintf("Last revision   %v, %s", last.ID, last.Modified.Local().Format("2006-01-02 15:04:05 MST")))

			if version.sameAs(&last) {
				return false
			}
		}

		cutoff := time.Now().Add(-time.Duration(l.delay))
		if cutoff.Before(version.Modified) {
			info(fmt.Sprintf("Latest revision modified less than %s ago (%s)", l.delay, version.Modified.Local().Format("2006-01-02 15:04:05 MST")))
			return false
		}
	}

	return true
}

func (l *LoadACL) compare(u device.IDevice, devices []*uhppote.Device, list *api.ACL) (bool, error) {
	current, err := api.GetACL(u, devices)
	if err != nil {
		return false, err
	}

	diff, err := api.Compare(current, *list)
	if err != nil {
		return false, err
	}

	for _, v := range diff {
		if v.HasChanges() {
			return true, nil
		}
	}

	return false, nil
}

func (l *LoadACL) getACL(google *sheets.Service, spreadsheet *sheets.Spreadsheet, devices []*uhppote.Device, ctx context.Context) (*api.ACL, []error, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.area).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return nil, nil, fmt.Errorf("No data in spreadsheet/range")
	}

	table, err := makeTable(response.Values)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating table from worksheet (%v)", err)
	}

	list, warnings, err := api.ParseTable(table, devices, l.strict)
	if err != nil {
		return nil, nil, err
	}

	if list == nil {
		return nil, nil, fmt.Errorf("Error creating ACL from worksheet (%v)", list)
	}

	return list, warnings, nil
}

func (l *LoadACL) updateLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]api.Report, ctx context.Context) error {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.logRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve column headers from log sheet (%v)", err)
	}

	fields := []string{"timestamp", "deviceid", "unchanged", "updated", "added", "deleted", "failed", "errors"}

	index, columns := buildIndex(response.Values, fields)

	summary := api.Summarize(rpt)
	var rows = sheets.ValueRange{
		Values: [][]interface{}{},
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	for _, v := range summary {
		row := make([]interface{}, columns)

		for i := 0; i < columns; i++ {
			row[i] = ""
		}

		if ix, ok := index["timestamp"]; ok {
			row[ix] = timestamp
		}

		if ix, ok := index["deviceid"]; ok {
			row[ix] = fmt.Sprintf("'%v", v.DeviceID)
		}

		if ix, ok := index["unchanged"]; ok {
			row[ix] = v.Unchanged
		}

		if ix, ok := index["updated"]; ok {
			row[ix] = v.Updated
		}

		if ix, ok := index["added"]; ok {
			row[ix] = v.Added
		}

		if ix, ok := index["deleted"]; ok {
			row[ix] = v.Deleted
		}

		if ix, ok := index["failed"]; ok {
			row[ix] = v.Failed
		}

		if ix, ok := index["errors"]; ok {
			row[ix] = v.Errored
		}

		rows.Values = append(rows.Values, row)
	}

	if _, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, l.logRange, &rows).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("Error writing log to Google Sheets (%w)", err)
	}

	return nil
}

func (l *LoadACL) updateReportSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]api.Report, ctx context.Context) error {
	info("Appending report to worksheet")

	// ... include 'after cutoff' rows from existing report
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(l.reportRange)
	if len(match) < 5 {
		return fmt.Errorf("Invalid report range '%s'", l.reportRange)
	}

	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	var rows = sheets.ValueRange{
		Range:  fmt.Sprintf("%s!%s%v:%s", name, left, top+1, right),
		Values: [][]interface{}{},
	}

	before := time.Now().
		In(time.Local).
		Add(time.Hour * time.Duration(-24*(l.reportRetention-1))).
		Truncate(24 * time.Hour)

	cutoff := time.Date(before.Year(), before.Month(), before.Day(), 0, 0, 0, 0, before.Location())

	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.reportRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve column headers from report sheet (%v)", err)
	}

	fields := []string{"timestamp", "action", "cardnumber"}
	index, columns := buildIndex(response.Values, fields)

	for _, record := range response.Values[1:] {
		row := make([]interface{}, columns)
		for i := 0; i < columns; i++ {
			row[i] = ""
		}

		if ix, ok := index["timestamp"]; ok && ix < len(record) {
			timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", record[ix].(string), time.Local)
			if err == nil && !timestamp.Before(cutoff) {
				for _, f := range fields {
					if ix, ok := index[f]; ok && ix < len(record) {
						row[ix] = strings.TrimSpace(fmt.Sprintf("%v", record[ix]))
					}
				}

				rows.Values = append(rows.Values, row)
			}
		}
	}

	// ... append new report

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	consolidated := api.Consolidate(rpt)

	format := []struct {
		Cards  []uint32
		Action string
	}{
		{consolidated.Updated, "Updated"},
		{consolidated.Added, "Added"},
		{consolidated.Deleted, "Deleted"},
		{consolidated.Failed, "Failed"},
		{consolidated.Errored, "Error"},
	}

	for _, f := range format {
		for _, card := range f.Cards {
			row := make([]interface{}, columns)

			for i := 0; i < columns; i++ {
				row[i] = ""
			}

			if ix, ok := index["timestamp"]; ok {
				row[ix] = timestamp
			}

			if ix, ok := index["action"]; ok {
				row[ix] = f.Action
			}

			if ix, ok := index["cardnumber"]; ok {
				row[ix] = card
			}

			rows.Values = append(rows.Values, row)
		}
	}

	// ... pad below
	for i := 0; i < 2; i++ {
		row := make([]interface{}, columns)

		for i := 0; i < columns; i++ {
			row[i] = ""
		}

		rows.Values = append(rows.Values, row)
	}

	// ... update worksheet
	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{&rows},
	}

	// TEENSY LITTLE HACK - top+len(rows.Values) relies on the padding below the report to avoid
	//                      an error because the 'below' range is out of range
	below := fmt.Sprintf("%s!%s%v:%s", name, left, top+len(rows.Values), right)

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	if err := clear(google, spreadsheet, []string{below}, ctx); err != nil {
		return err
	}

	return nil
}

func pruneSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, area string, retention int, ctx context.Context) error {
	sheet, err := getSheet(spreadsheet, area)
	if err != nil {
		return err
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, area).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve data from %s (%v)", area, err)
	}

	fields := []string{"timestamp"}
	index, _ := buildIndex(response.Values, fields)

	before := time.Now().
		In(time.Local).
		Add(time.Hour * time.Duration(-24*(retention-1))).
		Truncate(24 * time.Hour)

	cutoff := time.Date(before.Year(), before.Month(), before.Day(), 0, 0, 0, 0, before.Location())
	list := []int{}
	deleted := 0

	info(fmt.Sprintf("Pruning records before %v from '%s' worksheet ", cutoff.Format("2006-01-02"), sheet.Properties.Title))

	for row, record := range response.Values {
		if ix, ok := index["timestamp"]; ok && ix < len(record) {
			timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", record[ix].(string), time.Local)
			if err == nil && timestamp.Before(cutoff) {
				list = append(list, row)
			}
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

	info(fmt.Sprintf("Pruned %d records from '%s' worksheet", deleted, sheet.Properties.Title))

	return nil
}
