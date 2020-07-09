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
	"github.com/uhppoted/uhppoted-app-sheets/acl"
)

var LoadACLCmd = LoadACL{
	workdir:     DEFAULT_WORKDIR,
	config:      config.DefaultConfig,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	area:        "",

	nolog:        false,
	logRange:     "Log!A1:H",
	logRetention: 30,

	noreport:     false,
	reportRange:  "Report!A1:E",
	reportAlways: false,

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
	logRetention uint

	noreport     bool
	reportAlways bool
	reportRange  string

	force     bool
	strict    bool
	dryrun    bool
	debug     bool
	delay     delay
	revisions string
}

type report struct {
	top       int64
	left      string
	timestamp string
	data      string
	columns   map[string]string
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
	fmt.Println(`    uhppote-app-sheets load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                --range "Class ACL!A2:E" \`)
	fmt.Println()
	fmt.Println(`    uhppote-app-sheets --debug --conf example.conf load-acl --credentials "credentials.json" \`)
	fmt.Println(`                                                            --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \`)
	fmt.Println(`                                                            --range "Class ACL!A2:E" \`)
	fmt.Println()
}

func (l *LoadACL) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("load-acl", flag.ExitOnError)

	flagset.StringVar(&l.credentials, "credentials", l.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&l.url, "url", l.url, "Spreadsheet URL")
	flagset.StringVar(&l.area, "range", l.area, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flagset.BoolVar(&l.force, "force", l.force, "Forces an update, overriding the spreadsheet version and compare logic")
	flagset.BoolVar(&l.strict, "strict", l.strict, "Fails with an error if the spreadsheet contains duplicate card numbers")
	flagset.BoolVar(&l.dryrun, "dry-run", l.dryrun, "Simulates a load-acl without making any changes to the access controllers")
	flagset.Var(&l.delay, "delay", "Sets the delay between when a spreadsheet is modified and when it is regarded as sufficiently stable to use")

	flagset.BoolVar(&l.nolog, "no-log", l.nolog, "Disables writing a summary to the 'log' worksheet")
	flagset.StringVar(&l.logRange, "log-range", l.logRange, "Spreadsheet range for logging result")
	flagset.UintVar(&l.logRetention, "log-retention", l.logRetention, "Log sheet records older than 'log-retention' days are automatically pruned")

	flagset.BoolVar(&l.noreport, "no-report", l.noreport, "Disables writing a report to the 'report' worksheet")
	flagset.BoolVar(&l.reportAlways, "report-always", l.reportAlways, "Writes a report even if there were no changes or errors")
	flagset.StringVar(&l.reportRange, "report-range", l.reportRange, "Spreadsheet range for load report")

	flagset.StringVar(&l.workdir, "workdir", l.workdir, "Directory for working files (tokens, revisions, etc)")
	flagset.StringVar(&l.config, "config", l.config, "Configuration file path")

	return flagset
}

func (l *LoadACL) Execute(ctx context.Context) error {
	// ... check parameters
	if err := l.validate(); err != nil {
		return err
	}

	// ... locked?
	lockfile, err := l.lock()
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
	if err := conf.Load(l.config); err != nil {
		return fmt.Errorf("WARN  Could not load configuration (%v)", err)
	}

	u, devices := getDevices(conf, l.debug)

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(strings.TrimSpace(l.url))
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheetId := match[1]
	l.revisions = filepath.Join(l.workdir, ".google", fmt.Sprintf("%s.revision", spreadsheetId))

	if l.debug {
		debug(fmt.Sprintf("Spreadsheet - ID:%s  range:%s  log:%s", spreadsheetId, l.area, l.logRange))
	}

	version, err := l.getRevision(spreadsheetId, ctx)
	if err != nil {
		fatal(err.Error())
	}

	if !l.force && !l.revised(version) {
		info("Nothing to do")
		return nil
	}

	client, err := authorize(l.credentials, "https://www.googleapis.com/auth/spreadsheets", filepath.Join(l.workdir, ".google"))
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

	list, err := l.getACL(google, spreadsheet, devices, ctx)
	if err != nil {
		return err
	}

	for k, l := range *list {
		info(fmt.Sprintf("%v  Downloaded %v records", k, len(l)))
	}

	updated, err := l.compare(&u, devices, list)
	if err != nil {
		return err
	}

	if l.force || updated {
		rpt, err := api.PutACL(&u, *list, l.dryrun)
		if err != nil {
			return err
		}

		summary := api.Summarize(rpt)
		format := "%v  unchanged:%v  updated:%v  added:%v  deleted:%v  failed:%v  errors:%v"
		for _, v := range summary {
			info(fmt.Sprintf(format, v.DeviceID, v.Unchanged, v.Updated, v.Added, v.Deleted, v.Failed, v.Errored))
		}

		for k, v := range rpt {
			for _, err := range v.Errors {
				fatal(fmt.Sprintf("%v  %v", k, err))
			}
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
	} else {
		info("No changes - Nothing to do")
	}

	if version != nil {
		version.store(l.revisions)
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

	diff, err := api.Compare(*list, current)
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

	list, warnings, err := api.ParseTable(table, devices, l.strict)
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

func (l *LoadACL) updateLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]api.Report, ctx context.Context) error {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.logRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve column headers from Log sheet (%v)", err)
	}

	index, columns := buildLogIndex(response.Values)

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

func (l *LoadACL) pruneLogSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ctx context.Context) error {
	sheet, err := getSheet(spreadsheet, l.logRange)
	if err != nil {
		return err
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.logRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve data from Log sheet (%v)", err)
	}

	index, _ := buildLogIndex(response.Values)

	before := time.Now().
		In(time.Local).
		Add(time.Hour * time.Duration(-24*(int(l.logRetention)-1))).
		Truncate(24 * time.Hour)

	cutoff := time.Date(before.Year(), before.Month(), before.Day(), 0, 0, 0, 0, before.Location())
	list := []int{}
	deleted := 0

	info(fmt.Sprintf("Pruning log records from before %v", cutoff.Format("2006-01-02")))

	for row, record := range response.Values {
		if ix, ok := index["timestamp"]; ok {
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

	info(fmt.Sprintf("Pruned %d log records from log sheet", deleted))

	return nil
}

func (l *LoadACL) updateReportSheet(google *sheets.Service, spreadsheet *sheets.Spreadsheet, rpt map[uint32]api.Report, ctx context.Context) error {
	// ... anything interesting?
	if !l.reportAlways {
		interesting := false
		for _, v := range rpt {
			if len(v.Updated) > 0 || len(v.Added) > 0 || len(v.Deleted) > 0 || len(v.Failed) > 0 || len(v.Errored) > 0 {
				interesting = true
			}
		}

		if !interesting {
			info("No interesting information in report - leaving existing report 'as is'")
			return nil
		}
	}

	// ... create report format
	sheet, err := getSheet(spreadsheet, l.reportRange)
	if err != nil {
		return err
	}

	format, err := l.buildReportFormat(google, spreadsheet)
	if err != nil {
		return err
	}

	// ... clear existing report
	info("Clearing existing report from worksheet")
	if err := clear(google, spreadsheet, []string{format.timestamp, format.data}, ctx); err != nil {
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

	consolidated := api.Consolidate(rpt)

	columns := map[string][]uint32{
		"updated": consolidated.Updated,
		"added":   consolidated.Added,
		"deleted": consolidated.Deleted,
		"failed":  consolidated.Failed,
		"errors":  consolidated.Errored,
	}

	var timestamp = sheets.ValueRange{
		Range: format.timestamp,
		Values: [][]interface{}{
			[]interface{}{
				time.Now().Format("2006-01-02 15:04:05"),
			},
		},
	}

	rq := sheets.BatchUpdateValuesRequest{
		ValueInputOption: "USER_ENTERED",
		Data:             []*sheets.ValueRange{&timestamp},
	}

	for k, v := range columns {
		if r, ok := format.columns[k]; ok {
			var values = sheets.ValueRange{
				Range:  r,
				Values: [][]interface{}{},
			}

			for _, card := range v {
				values.Values = append(values.Values, []interface{}{fmt.Sprintf("%v", card)})
			}

			rq.Data = append(rq.Data, &values)
		}
	}

	if _, err := google.Spreadsheets.Values.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	// ... pad

	var pad = sheets.ValueRange{
		Values: [][]interface{}{[]interface{}{""}},
	}

	if _, err := google.Spreadsheets.Values.Append(spreadsheet.SpreadsheetId, l.reportRange, &pad).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("OVERWRITE").
		Context(ctx).
		Do(); err != nil {
		return fmt.Errorf("Error padding report worksheet (%w)", err)
	}

	//	// ... (experimental)
	//
	//	updated := time.Now().Format("2006-01-02 15:04:05")
	//	rq := sheets.BatchUpdateSpreadsheetRequest{
	//		Requests: []*sheets.Request{
	//			&sheets.Request{
	//				UpdateCells: &sheets.UpdateCellsRequest{
	//					Fields: "*",
	//					Range: &sheets.GridRange{
	//						SheetId:          sheet.Properties.SheetId,
	//						StartColumnIndex: 7,
	//						StartRowIndex:    1,
	//						EndColumnIndex:   8,
	//						EndRowIndex:      2,
	//					},
	//					Rows: []*sheets.RowData{
	//						&sheets.RowData{
	//							Values: []*sheets.CellData{
	//								&sheets.CellData{
	//									UserEnteredValue: &sheets.ExtendedValue{
	//										StringValue: &updated,
	//									},
	//								},
	//							},
	//						},
	//					},
	//				},
	//			},
	//		},
	//	}
	//
	//	if _, err := google.Spreadsheets.BatchUpdate(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
	//		return err
	//	}

	return nil
}

func buildLogIndex(rows [][]interface{}) (map[string]int, int) {
	index := map[string]int{
		"timestamp": 0,
		"deviceid":  1,
		"unchanged": 2,
		"updated":   3,
		"added":     4,
		"deleted":   5,
		"failed":    6,
		"errors":    7,
	}

	if len(rows) > 0 {
		header := rows[0]
		index = map[string]int{}

		for i, v := range header {
			k := normalise(v.(string))
			switch k {
			case "timestamp":
				index["timestamp"] = i
			case "deviceid":
				index["deviceid"] = i
			case "unchanged":
				index["unchanged"] = i
			case "updated":
				index["updated"] = i
			case "added":
				index["added"] = i
			case "deleted":
				index["deleted"] = i
			case "failed":
				index["failed"] = i
			case "errors":
				index["errors"] = i
			}
		}
	}

	columns := 0
	for _, v := range index {
		if v >= columns {
			columns = v + 1
		}
	}

	return index, columns
}

func (l *LoadACL) buildReportFormat(google *sheets.Service, spreadsheet *sheets.Spreadsheet) (*report, error) {
	response, err := google.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, l.reportRange).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data from report sheet (%v)", err)
	}

	rows := response.Values
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(l.reportRange)
	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	index := map[string]string{
		"unchanged": "A",
		"added":     "B",
		"deleted":   "C",
		"failed":    "D",
		"errors":    "E",
	}

	if len(rows) > 1 {
		header := rows[1]
		offset := colToI(left)
		index = map[string]string{}

		for i, v := range header {
			k := normalise(v.(string))
			switch k {
			case "updated":
				index["updated"] = iToCol(offset + i)
			case "added":
				index["added"] = iToCol(offset + i)
			case "deleted":
				index["deleted"] = iToCol(offset + i)
			case "failed":
				index["failed"] = iToCol(offset + i)
			case "errors":
				index["errors"] = iToCol(offset + i)
			}
		}
	}

	format := report{
		top:       int64(top),
		left:      left,
		timestamp: fmt.Sprintf("%v!%v%v:%v%v", name, left, top, left, top),
		data:      fmt.Sprintf("%v!%v%v:%v", name, left, top+2, right),
		columns:   map[string]string{},
	}

	columns := []string{"updated", "added", "deleted", "failed", "errors"}
	for _, column := range columns {
		if ix, ok := index[column]; ok {
			format.columns[column] = fmt.Sprintf("%v!%v%v:%v", name, ix, top+2, ix)
		}
	}

	return &format, nil
}
