package commands

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	"github.com/uhppoted/uhppoted-api/config"
)

const APP = "uhppoted-app-sheets"

type Options struct {
	Config string
	Debug  bool
}

type report struct {
	top     int64
	left    string
	title   string
	headers string
	data    string
	columns map[string]string
	xref    map[int]int
}

func helpOptions(flagset *flag.FlagSet) {
	count := 0
	flag.VisitAll(func(f *flag.Flag) {
		count++
	})

	flagset.VisitAll(func(f *flag.Flag) {
		fmt.Printf("    --%-13s %s\n", f.Name, f.Usage)
	})

	if count > 0 {
		fmt.Println()
		fmt.Println("  Options:")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Printf("    --%-13s %s\n", f.Name, f.Usage)
		})
	}
}

func getDevices(conf *config.Config, debug bool) (uhppote.IUHPPOTE, []uhppote.Device) {
	bind, broadcast, listen := config.DefaultIpAddresses()

	if conf.BindAddress != nil {
		bind = *conf.BindAddress
	}

	if conf.BroadcastAddress != nil {
		broadcast = *conf.BroadcastAddress
	}

	if conf.ListenAddress != nil {
		listen = *conf.ListenAddress
	}

	devices := []uhppote.Device{}
	for s, d := range conf.Devices {
		// ... because d is *Device and all devices end up with the same info if you don't make a manual copy
		name := d.Name
		deviceID := s
		address := d.Address
		rollover := d.Rollover
		doors := d.Doors

		if device := uhppote.NewDevice(name, deviceID, address, rollover, doors); device != nil {
			devices = append(devices, *device)
		}
	}

	u := uhppote.NewUHPPOTE(bind, broadcast, listen, 5*time.Second, devices, debug)

	return u, devices
}

func getSpreadsheet(google *sheets.Service, id string) (*sheets.Spreadsheet, error) {
	spreadsheet, err := google.Spreadsheets.Get(id).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch spreadsheet (%v)", err)
	}

	return spreadsheet, nil
}

func getSheet(spreadsheet *sheets.Spreadsheet, area string) (*sheets.Sheet, error) {
	name := regexp.MustCompile(`(.+?)!.*`).FindStringSubmatch(area)[1]
	for _, sheet := range spreadsheet.Sheets {
		if strings.ToLower(strings.TrimSpace(sheet.Properties.Title)) == strings.ToLower(strings.TrimSpace(name)) {
			return sheet, nil
		}
	}

	return nil, fmt.Errorf("Unable to identify worksheet for '%s'", area)
}

func buildIndex(rows [][]interface{}, fields []string) (map[string]int, int) {
	index := map[string]int{}

	for ix, col := range fields {
		index[col] = ix
	}

	if len(rows) > 0 {
		header := rows[0]
		index = map[string]int{}

		for i, v := range header {
			k := normalise(v.(string))
			for _, f := range fields {
				if k == f {
					index[f] = i
				}
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

func iToCol(index int) string {
	columns := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	N := len(columns)

	col := string(columns[index%N])
	index = index / N
	for ; index > 0; index = index / N {
		col = col + string(columns[index%N])
	}

	return col
}

func colToI(column string) int {
	columns := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	N := len(columns)
	col := []rune(strings.ToUpper(column))
	ix := 0

	for _, c := range col {
		for i, r := range columns {
			if r == c {
				ix = ix*N + i
				break
			}
		}
	}

	return ix
}
func normalise(v string) string {
	return strings.ToLower(strings.ReplaceAll(v, " ", ""))
}

func clean(v string) string {
	return strings.TrimSpace(v)
}

func debug(msg string) {
	log.Printf("%-5s %s", "DEBUG", msg)
}

func info(msg string) {
	log.Printf("%-5s %s", "INFO", msg)
}

func warn(msg string) {
	log.Printf("%-5s %s", "WARN", msg)
}

func fatal(msg string) {
	log.Printf("%-5s %s", "ERROR", msg)
}
