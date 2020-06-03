package acl

import (
	"encoding/csv"
	"google.golang.org/api/sheets/v4"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func MakeTSV(filename string, data *sheets.ValueRange) error {
	// ... header
	row := data.Values[0]
	header := make([]string, len(row))
	for i, v := range row {
		header[i] = strings.TrimSpace(v.(string))
	}

	// ... records
	records := [][]string{}
	for _, row := range data.Values[1:] {
		if cardnumber, ok := row[0].(string); !ok {
			continue
		} else if ok, err := regexp.Match(`^\s*[0-9]+\s*$`, []byte(cardnumber)); !ok || err != nil {
			continue
		}

		record := make([]string, len(row))
		for i, v := range row {
			record[i] = strings.TrimSpace(v.(string))
		}
		records = append(records, record)
	}

	// ... write to file
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0660); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = '\t'

	w.Write(header)
	for _, record := range records {
		w.Write(record)
	}

	w.Flush()

	return nil
}
