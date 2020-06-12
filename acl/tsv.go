package acl

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"

	"google.golang.org/api/sheets/v4"
)

func MakeTSV(f io.Writer, data *sheets.ValueRange) error {
	if len(data.Values) == 0 {
		return fmt.Errorf("Empty sheet")
	}

	// ... header
	row := data.Values[0]
	header := make([]string, len(row))
	for i, v := range row {
		header[i] = clean(v.(string))
	}

	if len(header) == 0 {
		return fmt.Errorf("Missing/invalid header row")
	}

	if len(header) < 1 || normalise(header[0]) != "cardnumber" {
		return fmt.Errorf("Missing 'card number' column")
	}

	if len(header) < 2 || normalise(header[1]) != "from" {
		return fmt.Errorf("Missing 'from' column")
	}

	if len(header) < 3 || normalise(header[2]) != "to" {
		return fmt.Errorf("Missing 'to' column")
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
	w := csv.NewWriter(f)
	w.Comma = '\t'

	w.Write(header)
	for _, record := range records {
		w.Write(record)
	}

	w.Flush()

	return nil
}

func clean(v string) string {
	return strings.TrimSpace(v)
}

func normalise(v string) string {
	return strings.ToLower(strings.ReplaceAll(v, " ", ""))
}
