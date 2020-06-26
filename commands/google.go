package commands

import (
	"context"

	"google.golang.org/api/sheets/v4"
)

func clear(google *sheets.Service, spreadsheet *sheets.Spreadsheet, ranges []string, ctx context.Context) error {
	rq := sheets.BatchClearValuesRequest{
		Ranges: ranges,
	}

	if _, err := google.Spreadsheets.Values.BatchClear(spreadsheet.SpreadsheetId, &rq).Context(ctx).Do(); err != nil {
		return err
	}

	return nil
}
