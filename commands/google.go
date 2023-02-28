package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

type revision struct {
	FileID   string    `json:"file-id"`
	ID       string    `json:"id"`
	Modified time.Time `json:"modified"`
}

func (r *revision) load(file string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	object := revision{}
	if err := json.Unmarshal(bytes, &object); err != nil {
		return err
	}

	if r != nil {
		r.FileID = object.FileID
		r.ID = object.ID
		r.Modified = object.Modified
	}

	return nil
}

func (r *revision) store(file string) error {
	dir := filepath.Dir(file)

	if err := os.MkdirAll(dir, 0770); err != nil {
		return err
	}

	if bytes, err := json.Marshal(r); err != nil {
		return err
	} else if err := os.WriteFile(file, bytes, 0660); err != nil {
		return err
	}

	return nil
}

func (r *revision) sameAs(v *revision) bool {
	return reflect.DeepEqual(r, v)
}

func getRevision(gdrive *drive.Service, fileId string) (*revision, error) {
	page := ""
	latest := revision{
		FileID:   fileId,
		ID:       "",
		Modified: time.Time{},
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

		for _, r := range revisions.Revisions {
			datetime, err := time.Parse("2006-01-02T15:04:05.999Z", r.ModifiedTime)
			if err != nil {
				return nil, err
			}

			if latest.Modified.Before(datetime) {
				latest.ID = r.Id
				latest.Modified = datetime
			}
		}

		if page = revisions.NextPageToken; page == "" {
			break
		}
	}

	if latest.Modified.IsZero() {
		return nil, fmt.Errorf("unable to identify latest revision for file ID %s", fileId)
	}

	return &latest, nil
}

func clear(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ranges []string) error {
	rq := sheets.BatchClearValuesRequest{
		Ranges: ranges,
	}

	if _, err := google.Spreadsheets.Values.BatchClear(spreadsheet.SpreadsheetId, &rq).Do(); err != nil {
		return err
	}

	return nil
}
