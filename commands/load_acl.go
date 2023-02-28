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

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	lib "github.com/uhppoted/uhppoted-lib/acl"
	"github.com/uhppoted/uhppoted-lib/config"
	"github.com/uhppoted/uhppoted-lib/lockfile"
)

var LoadACLCmd = LoadACL{
	command: command{
		workdir:     DEFAULT_WORKDIR,
		credentials: DEFAULT_CREDENTIALS,
		tokens:      "",
		url:         "",
		debug:       false,
	},

	config: config.DefaultConfig,
	area:   "",

	nolog:           false,
	logRange:        "Log!A1:H",
	reportRetention: 7,
	logRetention:    30,

	noreport:    false,
	reportRange: "Report!A1:E",

	force:     false,
	strict:    false,
	dryrun:    false,
	delay:     15 * time.Minute,
	revisions: filepath.Join(DEFAULT_WORKDIR, ".google", "uhppoted-app-sheets.revision"),
}

type LoadACL struct {
	command

	config          string
	withPIN         bool
	area            string
	nolog           bool
	logRange        string
	logRetention    int
	noreport        bool
	reportRange     string
	reportRetention int
	force           bool
	strict          bool
	dryrun          bool
	delay           time.Duration
	revisions       string
}

func (cmd *LoadACL) Name() string {
	return "load-acl"
}

func (cmd *LoadACL) Description() string {
	return "Updates a set of configured UHPPOTE access controllers from a Google Sheets worksheet"
}

func (cmd *LoadACL) Usage() string {
	return "--credentials <file> --url <url>"
}

func (cmd *LoadACL) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] [--config <configuration>] load-acl [options] --url <URL> --range <range>\n", APP)
	fmt.Println()
	fmt.Println("  Updates the cards on a set of configured controllers from a Google Sheets worksheet access control list. Unless the --force option")
	fmt.Println("  is specified updates will be silently ignored (no log and no report) if the spreadsheet revision has not changed or the updated")
	fmt.Println("  spreadsheet contains no relevant changes.")
	fmt.Println()
	fmt.Println("  Duplicate card numbers are automatically deleted across the system unless the --strict option is provided to fail the load.")
	fmt.Println()

	helpOptions(cmd.FlagSet())

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println(`    uhppote-app-sheets load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                --range "ACL!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                            --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                            --range "ACL!A2:E" \`)
	fmt.Println()
}

func (cmd *LoadACL) FlagSet() *flag.FlagSet {
	flagset := cmd.flagset("load-acl")

	flagset.StringVar(&cmd.area, "range", cmd.area, "Spreadsheet range e.g. 'ACL!A2:E'")
	flagset.BoolVar(&cmd.withPIN, "with-pin", cmd.withPIN, "Updates card keypad PIN codes when loading an ACL")
	flagset.BoolVar(&cmd.force, "force", cmd.force, "Forces an update, overriding the spreadsheet version and compare logic")
	flagset.BoolVar(&cmd.strict, "strict", cmd.strict, "Fails with an error if the spreadsheet contains duplicate card numbers")
	flagset.BoolVar(&cmd.dryrun, "dry-run", cmd.dryrun, "Simulates a load-acl without making any changes to the access controllers")
	flagset.DurationVar(&cmd.delay, "delay", cmd.delay, "Sets the delay between when a spreadsheet is modified and when it is regarded as sufficiently stable to use")

	flagset.BoolVar(&cmd.nolog, "no-log", cmd.nolog, "Disables writing a summary to the 'log' worksheet")
	flagset.StringVar(&cmd.logRange, "log-range", cmd.logRange, "Spreadsheet range for logging result")
	flagset.IntVar(&cmd.logRetention, "log-retention", cmd.logRetention, "Log sheet records older than 'log-retention' days are automatically pruned")

	flagset.BoolVar(&cmd.noreport, "no-report", cmd.noreport, "Disables writing a report to the 'report' worksheet")
	flagset.StringVar(&cmd.reportRange, "report-range", cmd.reportRange, "Spreadsheet range for load report")
	flagset.IntVar(&cmd.reportRetention, "report-retention", cmd.reportRetention, "Report sheet records older than 'report-retention' days are automatically pruned")

	return flagset
}

func (cmd *LoadACL) Execute(args ...interface{}) error {
	options := args[0].(*Options)

	cmd.config = options.Config
	cmd.debug = options.Debug

	// ... check parameters
	if err := cmd.validate(); err != nil {
		return err
	}

	// ... locked?
	lockFile := config.Lockfile{
		File:   filepath.Join(cmd.workdir, ".google", "uhppoted-app-sheets.lock"),
		Remove: lockfile.RemoveLockfile,
	}

	if kraken, err := lockfile.MakeLockFile(lockFile); err != nil {
		return err
	} else {
		defer func() {
			infof("Removing lockfile '%v'", lockFile.File)
			kraken.Release()
		}()
	}

	// ... good to go!
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
	cmd.revisions = filepath.Join(cmd.workdir, ".google", fmt.Sprintf("%s.revision", spreadsheetId))

	if cmd.debug {
		debugf("Spreadsheet - ID:%s  range:%s  log:%s", spreadsheetId, cmd.area, cmd.logRange)
	}

	version, err := cmd.getRevision(spreadsheetId)
	if err != nil {
		errorf("%v", err)
	}

	if !cmd.force && !cmd.revised(version) {
		infof("Nothing to do")
		return nil
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

	list, warnings, err := cmd.getACL(google, spreadsheet, devices)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		warnf("%v", w.Error())
	}

	for k, l := range *list {
		infof("%v  Downloaded %v records", k, len(l))
	}

	updated, err := cmd.compare(u, devices, list)
	if err != nil {
		return err
	}

	if cmd.force || updated {
		f := func(u uhppote.IUHPPOTE, list lib.ACL) (map[uint32]lib.Report, []error) {
			if cmd.withPIN {
				return lib.PutACLWithPIN(u, list, cmd.dryrun)
			} else {
				return lib.PutACL(u, list, cmd.dryrun)
			}
		}

		rpt, errors := f(u, *list)
		if len(errors) > 0 {
			return fmt.Errorf("%v", errors)
		}

		for _, w := range warnings {
			if duplicate, ok := w.(*lib.DuplicateCardError); ok {
				for k, v := range rpt {
					v.Errored = append(v.Errored, duplicate.CardNumber)
					rpt[k] = v
				}
			}
		}

		summary := lib.Summarize(rpt)
		format := "%v  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v  errors:%v"
		for _, v := range summary {
			infof(format, v.DeviceID, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed, v.Errored+len(warnings))
		}

		for k, v := range rpt {
			for _, err := range v.Errors {
				errorf("%v  %v", k, err)
			}
		}

		if !cmd.nolog {
			if err := cmd.updateLogSheet(google, spreadsheet, rpt); err != nil {
				return err
			}

			if err := pruneSheet(google, spreadsheet, cmd.logRange, cmd.logRetention); err != nil {
				return err
			}
		}

		if !cmd.noreport {
			if err := cmd.updateReportSheet(google, spreadsheet, rpt); err != nil {
				return err
			}
		}
	} else {
		infof("No changes - Nothing to do")
	}

	if version != nil {
		version.store(cmd.revisions)
	}

	return nil
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
		return fmt.Errorf("invalid range '%s' - expected something like 'ACL!A2:K", l.area)
	}

	if !l.nolog {
		if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+[0-9]+):([a-zA-Z]+(?:[0-9]+)?)`).FindStringSubmatch(l.logRange); len(match) < 4 {
			return fmt.Errorf("invalid log-range '%s' - expected something like 'Log!A1:H", l.logRange)
		}
	}

	if !l.noreport {
		if match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(l.reportRange); len(match) < 5 {
			return fmt.Errorf("invalid report-range '%s' - expected something like 'Report!A1:E", l.reportRange)
		}
	}

	return nil
}

func (cmd *LoadACL) getRevision(spreadsheetId string) (*revision, error) {
	tokens := cmd.tokens
	if tokens == "" {
		tokens = filepath.Join(cmd.workdir, ".google")
	}

	client, err := authorize(cmd.credentials, drive.DriveMetadataReadonlyScope, tokens)
	if err != nil {
		//lint:ignore ST1005 Google should be capitalized
		return nil, fmt.Errorf("Google Drive authentication/authorization error (%w)", err)
	}

	gdrive, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create new Google Drive client (%w)", err)
	}

	version, err := getRevision(gdrive, spreadsheetId)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve spreadsheet revision (%v)", err)
	}

	return version, nil
}

func (l *LoadACL) revised(version *revision) bool {
	if version != nil {
		infof("Latest revision %v, %s", version.ID, version.Modified.Local().Format("2006-01-02 15:04:05 MST"))

		var last revision

		if err := last.load(l.revisions); err != nil {
			errorf("Error reading last revision from %s", l.revisions)
			errorf("%v", err)
		} else {
			infof("Last revision   %v, %s", last.ID, last.Modified.Local().Format("2006-01-02 15:04:05 MST"))

			if version.sameAs(&last) {
				return false
			}
		}

		cutoff := time.Now().Add(-time.Duration(l.delay))
		if cutoff.Before(version.Modified) {
			infof("Latest revision modified less than %s ago (%s)", l.delay, version.Modified.Local().Format("2006-01-02 15:04:05 MST"))
			return false
		}
	}

	return true
}

func (cmd *LoadACL) compare(u uhppote.IUHPPOTE, devices []uhppote.Device, list *lib.ACL) (bool, error) {
	current, errors := lib.GetACL(u, devices)
	if len(errors) > 0 {
		return false, fmt.Errorf("%v", errors)
	}

	f := func(current lib.ACL, list lib.ACL) (map[uint32]lib.Diff, error) {
		if cmd.withPIN {
			return lib.CompareWithPIN(current, list)
		} else {
			return lib.Compare(current, list)
		}
	}

	if diff, err := f(current, *list); err != nil {
		return false, err
	} else {

		for _, v := range diff {
			if v.HasChanges() {
				return true, nil
			}
		}

		return false, nil
	}
}

func (l *LoadACL) getACL(google *sheets.Service, spreadsheet *sheets.Spreadsheet, devices []uhppote.Device) (*lib.ACL, []error, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.area).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve data from sheet (%v)", err)
	}

	if len(response.Values) == 0 {
		return nil, nil, fmt.Errorf("no data in spreadsheet/range")
	}

	table, err := makeTable(response.Values)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating table from worksheet (%v)", err)
	}

	list, warnings, err := lib.ParseTable(table, devices, l.strict)
	if err != nil {
		return nil, nil, err
	}

	if list == nil {
		return nil, nil, fmt.Errorf("error creating ACL from worksheet (%v)", list)
	}

	return list, warnings, nil
}

func (l *LoadACL) updateLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]lib.Report) error {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.logRange).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve column headers from log sheet (%v)", err)
	}

	fields := []string{"timestamp", "deviceid", "unchanged", "updated", "added", "deleted", "failed", "errors"}

	index, columns := buildIndex(response.Values, fields)

	summary := lib.Summarize(rpt)
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
		Do(); err != nil {
		return fmt.Errorf("error writing log to Google Sheets (%w)", err)
	}

	return nil
}

func (l *LoadACL) updateReportSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]lib.Report) error {
	infof("Appending report to worksheet")

	// ... include 'after cutoff' rows from existing report
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(l.reportRange)
	if len(match) < 5 {
		return fmt.Errorf("invalid report range '%s'", l.reportRange)
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
		return fmt.Errorf("unable to retrieve column headers from report sheet (%v)", err)
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
	consolidated := lib.Consolidate(rpt)
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

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Do(); err != nil {
		return err
	}

	if err := clear(google, spreadsheet, []string{below}); err != nil {
		return err
	}

	return nil
}

func pruneSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, area string, retention int) error {
	sheet, err := getSheet(spreadsheet, area)
	if err != nil {
		return err
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, area).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve data from %s (%v)", area, err)
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

	infof("Pruning records before %v from '%s' worksheet ", cutoff.Format("2006-01-02"), sheet.Properties.Title)

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

			if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Do(); err != nil {
				return err
			}
		}
	}

	infof("Pruned %d records from '%s' worksheet", deleted, sheet.Properties.Title)

	return nil
}
