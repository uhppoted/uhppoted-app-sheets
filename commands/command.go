package commands

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppote-core/uhppote"
	"github.com/uhppoted/uhppoted-api/config"
)

const APP = "uhppoted-app-sheets"

func getVersion(gdrive *drive.Service, fileId string, ctx context.Context) (*version, error) {
	page := ""
	latest := version{
		revision: "",
		modified: time.Time{},
	}

	for {
		call := drive.NewRevisionsService(gdrive).List(fileId)
		if page != "" {
			call.PageToken(page)
		}

		revisions, err := call.Do()
		if err != nil {
			return nil, err
		}

		for _, revision := range revisions.Revisions {
			datetime, err := time.Parse("2006-01-02T15:04:05.999Z", revision.ModifiedTime)
			if err != nil {
				return nil, err
			}

			if latest.modified.Before(datetime) {
				latest.revision = revision.Id
				latest.modified = datetime
			}
		}

		if page = revisions.NextPageToken; page == "" {
			break
		}
	}

	if latest.modified.IsZero() {
		return nil, fmt.Errorf("Unable to identify latest revision for file ID %s", fileId)
	}

	return &latest, nil
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
	log.Printf("%-5s %s", "ERROR", msg)
}
