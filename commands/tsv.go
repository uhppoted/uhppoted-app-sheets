package commands

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"google.golang.org/api/sheets/v4"
)

func sheetToTSV(f io.Writer, data *sheets.ValueRange, withPIN bool) error {
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

	if ix, ok := index["pin"]; ok && withPIN {
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
		if k != "cardnumber" && k != "pin" && k != "from" && k != "to" {
			header = append(header, clean(v.(string)))
		}
	}

	if len(header) == 0 {
		return fmt.Errorf("Missing/invalid header row")
	}

	if len(header) < 1 || normalise(header[0]) != "cardnumber" {
		return fmt.Errorf("Missing 'card number' column")
	}

	if withPIN {
		if len(header) < 2 || normalise(header[1]) != "pin" {
			return fmt.Errorf("Missing 'PIN' column")
		}

		if len(header) < 3 || normalise(header[2]) != "from" {
			return fmt.Errorf("Missing 'from' column")
		}

		if len(header) < 4 || normalise(header[3]) != "to" {
			return fmt.Errorf("Missing 'to' column")
		}
	} else {
		if len(header) < 2 || normalise(header[1]) != "from" {
			return fmt.Errorf("Missing 'from' column")
		}

		if len(header) < 3 || normalise(header[2]) != "to" {
			return fmt.Errorf("Missing 'to' column")
		}
	}

	// ... records
	records := [][]string{}
	for _, row := range data.Values[1:] {
		if cardnumber, ok := row[index["cardnumber"]].(string); !ok {
			continue
		} else if ok, err := regexp.Match(`^\s*[0-9]+\s*$`, []byte(cardnumber)); !ok || err != nil {
			continue
		}

		if withPIN {
			if PIN, ok := row[index["pin"]].(string); !ok {
				continue
			} else if ok, err := regexp.Match(`^\s*[0-9]*\s*$`, []byte(PIN)); !ok || err != nil {
				continue
			}
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

func tsvToSheet(f io.Reader, area string) (*sheets.ValueRange, *sheets.ValueRange, error) {
	match := regexp.MustCompile(`(.+?)!([a-zA-Z]+)([0-9]+):([a-zA-Z]+)([0-9]+)?`).FindStringSubmatch(area)
	if len(match) < 5 {
		return nil, nil, fmt.Errorf("Invalid spreadsheet range '%s'", area)
	}

	name := match[1]
	left := match[2]
	top, _ := strconv.Atoi(match[3])
	right := match[4]

	r := csv.NewReader(f)
	r.Comma = '\t'

	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("TSV file is empty")
	}

	// header
	if len(records) < 1 {
		return nil, nil, fmt.Errorf("TSV file missing header")
	}

	h := make([]interface{}, len(records[0]))

	for i, v := range records[0] {
		h[i] = fmt.Sprintf("%v", v)
	}

	header := sheets.ValueRange{
		Range:  fmt.Sprintf("%s!%s%v:%s%v", name, left, top, right, top),
		Values: [][]interface{}{h},
	}

	// data
	rows := make([][]interface{}, 0)

	for _, record := range records[1:] {
		row := make([]interface{}, len(record))

		for i, v := range record {
			row[i] = fmt.Sprintf("%v", v)
		}

		rows = append(rows, row)
	}

	data := sheets.ValueRange{
		Range:  fmt.Sprintf("%s!%s%v:%s", name, left, top+1, right),
		Values: rows,
	}

	return &header, &data, nil
}
