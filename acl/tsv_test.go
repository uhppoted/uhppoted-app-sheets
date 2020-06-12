package acl

import (
	"strings"
	"testing"

	"google.golang.org/api/sheets/v4"
)

func TestMakeTSV(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001002	2020-02-03	2020-11-30	Y	Y	N	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	err := MakeTSV(&f, &data)
	if err != nil {
		t.Fatalf("Unexpected error returned from MakeTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestMakeTSVWithOutOfOrderColumns(t *testing.T) {
	expected := `Card Number	From	To	Gate	Tower	Dungeon	Lair
6001001	2020-01-01	2020-12-31	Y	N	N	Y
6001002	2020-02-03	2020-11-30	Y	Y	N	N
`

	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Gate", "Card Number", "Tower", "To", "From", "Dungeon", "Lair"},
			[]interface{}{"Y", "6001001", "N", "2020-12-31", "2020-01-01", "N", "Y"},
			[]interface{}{"Y", "6001002", "Y", "2020-11-30", "2020-02-03", "N", "N"},
		},
	}

	err := MakeTSV(&f, &data)
	if err != nil {
		t.Fatalf("Unexpected error returned from MakeTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestMakeTSVWithEmptySheet(t *testing.T) {
	var f strings.Builder
	var data = sheets.ValueRange{}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for empty sheet, got %v", err)
	}
}

func TestMakeTSVWithoutHeaders(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{},
		},
	}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for missing headers, got %v", err)
	}
}

func TestMakeTSVWithMissingCardNumber(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number X"},
		},
	}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'card number' column, got %v", err)
	}
}

func TestMakeTSVWithMissingFromDate(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number"},
		},
	}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'from' column, got %v", err)
	}
}

func TestMakeTSVWithMissingToDate(t *testing.T) {
	var f strings.Builder

	data := sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From"},
		},
	}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'to' column, got %v", err)
	}
}

func TestMakeTSVWithDuplicatedColumn(t *testing.T) {
	var f strings.Builder
	var data = sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Gate"},
			[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			[]interface{}{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	err := MakeTSV(&f, &data)
	if err == nil {
		t.Fatalf("Expected error return for duplicated column, got %v", err)
	}
}

func TestMakeTSVWithInvalidCardNumber(t *testing.T) {
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

	err := MakeTSV(&f, &data)
	if err != nil {
		t.Fatalf("Unexpected error returned from MakeTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestMakeTSVWithInvalidFromDate(t *testing.T) {
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

	err := MakeTSV(&f, &data)
	if err != nil {
		t.Fatalf("Unexpected error returned from MakeTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}

func TestMakeTSVWithInvalidToDate(t *testing.T) {
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

	err := MakeTSV(&f, &data)
	if err != nil {
		t.Fatalf("Unexpected error returned from MakeTSV (%v)", err)
	}

	if f.String() != expected {
		t.Errorf("Incorrect TSV\n   expected: %s\n   got:      %s\n", expected, f.String())
	}
}
