package commands

import (
	"strings"
	"testing"

	"google.golang.org/api/sheets/v4"
)

func TestSheetToTSV(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001002	2020-02-03	2020-11-30	Y	Y	N	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]any{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]any{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error returned fromsheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithPIN(t *testing.T) {
	expected := `Card Number	PIN	From	To	Gate	Tower	Dungeon	Lair
6001001	7531	2023-01-01	2023-12-31	Y	N	N	Y
6001002	1357	2023-02-03	2023-11-30	Y	Y	N	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number", "PIN", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]any{"6001001", "7531", "2023-01-01", "2023-12-31", "Y", "N", "N", "Y"},
			[]any{"6001002", "1357", "2023-02-03", "2023-11-30", "Y", "Y", "N", "N"},
		},
	}

	err := sheetToTSV(&f, &data, true)
	if err != nil {
		t.Fatalf("Unexpected error returned fromsheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithOutOfOrderColumns(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001002	2020-02-03	2020-11-30	Y	Y	N	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]any{
			[]any{"Gate", "Card Number", "Tower", "To", "From", "Dungeon", "Lair"},
			[]any{"Y", "6001001", "N", "2020-12-31", "2020-01-01", "N", "Y"},
			[]any{"Y", "6001002", "Y", "2020-11-30", "2020-02-03", "N", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error returned fromsheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithEmptySheet(t *testing.T) {
	var f strings.Builder
	var data = sheets.ValueRange{}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for empty sheet, got %v", err)
	}
}

func TestSheetToTSVWithoutHeaders(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]any{
			[]any{},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for missing headers, got %v", err)
	}
}

func TestSheetToTSVWithMissingCardNumber(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number X"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for missing 'card number' column, got %v", err)
	}
}

func TestSheetToTSVWithMissingPIN(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number", "PINX", "From", "To"},
		},
	}

	err := sheetToTSV(&f, &data, true)
	if err == nil {
		t.Fatalf("Expected error return for missing 'PIN' column, got %v", err)
	}
}

func TestSheetToTSVWithMissingPIN2(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number", "PINX", "From", "To"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error return for missing 'PIN' column (%v)", err)
	}
}

func TestSheetToTSVWithMissingFromDate(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for missing 'from' column, got %v", err)
	}
}

func TestSheetToTSVWithMissingToDate(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for missing 'to' column, got %v", err)
	}
}

func TestSheetToTSVWithDuplicatedColumn(t *testing.T) {
	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Gate"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err == nil {
		t.Fatalf("Expected error return for duplicated column, got %v", err)
	}
}

func TestSheetToTSVWithInvalidCardNumber(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001003	2020-01-01	2020-12-31	Y	N	Y	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"600100X", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
			[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error returned from sheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithInvalidPIN(t *testing.T) {
	expected := `Card Number	PIN	From	To	Gate	Tower	Dungeon	Lair
6001001	7531	2023-01-01	2023-12-31	Y	N	N	Y
6001003	1357	2023-01-01	2023-12-31	Y	N	Y	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]any{
			[]any{"Card Number", "PIN", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]any{"6001001", "7531", "2023-01-01", "2023-12-31", "Y", "N", "N", "Y"},
			[]any{"6001002", "qwerty", "2023-02-03", "2023-11-30", "Y", "Y", "N", "N"},
			[]any{"6001003", "1357", "2023-01-01", "2023-12-31", "Y", "N", "Y", "N"},
		},
	}

	err := sheetToTSV(&f, &data, true)
	if err != nil {
		t.Fatalf("Unexpected error returned from sheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithInvalidFromDate(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001003	2020-01-01	2020-12-31	Y	N	Y	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"6001002", "2020-02-0X", "2020-11-30", "Y", "Y", "N", "N"},
			[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error returned fromsheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestSheetToTSVWithInvalidToDate(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001003	2020-01-01	2020-12-31	Y	N	Y	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"6001002", "2020-02-03", "2020-11-3X", "Y", "Y", "N", "N"},
			[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	err := sheetToTSV(&f, &data, false)
	if err != nil {
		t.Fatalf("Unexpected error returned fromsheetToTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}
