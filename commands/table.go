package commands

import (
	"fmt"
	"regexp"
	"time"

	api "github.com/uhppoted/uhppoted-api/acl"
)

func makeTable(rows [][]interface{}) (*api.Table, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("Empty sheet")
	}

	// .. build index
	index := map[string]int{}
	record := rows[0]
	for i, v := range record {
		k := normalise(v.(string))
		if _, ok := index[k]; ok {
			return nil, fmt.Errorf("Duplicate column name '%s'", v.(string))
		}

		index[k] = i
	}

	// ... header
	record = rows[0]
	header := []string{}

	if ix, ok := index["cardnumber"]; ok {
		header = append(header, clean(record[ix].(string)))
	}

	if ix, ok := index["from"]; ok {
		header = append(header, clean(record[ix].(string)))
	}

	if ix, ok := index["to"]; ok {
		header = append(header, clean(record[ix].(string)))
	}

	for _, v := range record {
		k := normalise(v.(string))
		if k != "cardnumber" && k != "from" && k != "to" {
			header = append(header, clean(v.(string)))
		}
	}

	if len(header) == 0 {
		return nil, fmt.Errorf("Missing/invalid header row")
	}

	if len(header) < 1 || normalise(header[0]) != "cardnumber" {
		return nil, fmt.Errorf("Missing 'card number' column")
	}

	if len(header) < 2 || normalise(header[1]) != "from" {
		return nil, fmt.Errorf("Missing 'from' column")
	}

	if len(header) < 3 || normalise(header[2]) != "to" {
		return nil, fmt.Errorf("Missing 'to' column")
	}

	// ... records
	records := [][]string{}
	for _, row := range rows[1:] {
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

	return &api.Table{
		Header:  header,
		Records: records,
	}, nil
}
