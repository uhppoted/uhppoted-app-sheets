package acl

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

func MakeTSV(f io.Writer, data *sheets.ValueRange) error {
	if len(data.Values) == 0 {
		return fmt.Errorf("Empty sheet")
	}

	// .. build index
	index := map[string]int{}
	row := data.Values[0]
	for i, v := range row {
		k := normalise(v.(string))
		if _, ok := index[k]; ok {
			return fmt.Errorf("Duplicate column name '%s'", v.(string))
		}

		index[k] = i
	}

	// ... header
	row = data.Values[0]
	header := []string{}

	if ix, ok := index["cardnumber"]; ok {
		header = append(header, clean(row[ix].(string)))
	}

	if ix, ok := index["from"]; ok {
		header = append(header, clean(row[ix].(string)))
	}

	if ix, ok := index["to"]; ok {
		header = append(header, clean(row[ix].(string)))
	}

	for _, v := range row {
		k := normalise(v.(string))
		if k != "cardnumber" && k != "from" && k != "to" {
			header = append(header, clean(v.(string)))
		}
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
		if cardnumber, ok := row[index["cardnumber"]].(string); !ok {
			continue
		} else if ok, err := regexp.Match(`^\s*[0-9]+\s*$`, []byte(cardnumber)); !ok || err != nil {
			continue
		}

		if from, ok := row[index["from"]].(string); !ok {
			continue
		} else if _, err := time.ParseInLocation("2006-01-02", from, time.Local); err != nil {
			continue
		}

		if to, ok := row[index["to"]].(string); !ok {
			continue
		} else if _, err := time.ParseInLocation("2006-01-02", to, time.Local); err != nil {
			continue
		}

		record := []string{}
		for _, h := range header {
			k := normalise(h)
			v := ""
			if ix, ok := index[k]; ok {
				v = row[ix].(string)
			}

			record = append(record, clean(v))
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
