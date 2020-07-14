package commands

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	"github.com/uhppoted/uhppoted-api/config"
)

const APP = "uhppoted-app-sheets"

type report struct {
	top     int64
	left    string
	title   string
	headers string
	data    string
	columns map[string]string
}

func getDevices(conf *config.Config, debug bool) (uhppote.UHPPOTE, []*uhppote.Device) {
	keys := []uint32{}
	for id, _ := range conf.Devices {
		keys = append(keys, id)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	u := uhppote.UHPPOTE{
		BindAddress:      conf.BindAddress,
		BroadcastAddress: conf.BroadcastAddress,
		ListenAddress:    conf.ListenAddress,
		Devices:          make(map[uint32]*uhppote.Device),
		Debug:            debug,
	}

	devices := []*uhppote.Device{}
	for _, id := range keys {
		d := conf.Devices[id]
		u.Devices[id] = uhppote.NewDevice(id, d.Address, d.Rollover, d.Doors)
		devices = append(devices, uhppote.NewDevice(id, d.Address, d.Rollover, d.Doors))
	}

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
