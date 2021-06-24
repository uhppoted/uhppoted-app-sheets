package commands

import (
	"reflect"
	"testing"

	api "github.com/uhppoted/uhppoted-lib/acl"
)

func TestMakeTable(t *testing.T) {
	expected := api.Table{
		Header: []string{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		Records: [][]string{
			{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	var data = [][]interface{}{
		[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
		[]interface{}{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
	}

	table, err := makeTable(data)
	if err != nil {
		t.Fatalf("Unexpected error returned from makeTable (%v)", err)
	}

	if table == nil {
		t.Fatalf("makeTable returend %v", table)
	}

	if !reflect.DeepEqual(*table, expected) {
		t.Errorf("Incorrect table\n   expected: %v\n   got:      %v\n", expected, *table)
	}
}

func TestMakeTableWithOutOfOrderColumns(t *testing.T) {
	expected := api.Table{
		Header: []string{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		Records: [][]string{
			{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		},
	}

	var data = [][]interface{}{
		[]interface{}{"Gate", "Card Number", "Tower", "To", "From", "Dungeon", "Lair"},
		[]interface{}{"Y", "6001001", "N", "2020-12-31", "2020-01-01", "N", "Y"},
		[]interface{}{"Y", "6001002", "Y", "2020-11-30", "2020-02-03", "N", "N"},
	}

	table, err := makeTable(data)
	if err != nil {
		t.Fatalf("Unexpected error returned from makeTable (%v)", err)
	}

	if table == nil {
		t.Fatalf("makeTable returend %v", table)
	}

	if !reflect.DeepEqual(*table, expected) {
		t.Errorf("Incorrect table\n   expected: %v\n   got:      %v\n", expected, *table)
	}
}

func TestMakeTableWithEmptySheet(t *testing.T) {
	var data = [][]interface{}{}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for empty sheet, got %v", err)
	}
}

func TestMakeTableWithoutHeaders(t *testing.T) {
	data := [][]interface{}{
		[]interface{}{},
	}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for missing headers, got %v", err)
	}
}

func TestMakeTableWithMissingCardNumber(t *testing.T) {
	data := [][]interface{}{
		[]interface{}{"Card Number X"},
	}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'card number' column, got %v", err)
	}
}

func TestMakeTableWithMissingFromDate(t *testing.T) {
	data := [][]interface{}{
		[]interface{}{"Card Number"},
	}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'from' column, got %v", err)
	}
}

func TestMakeTableWithMissingToDate(t *testing.T) {
	data := [][]interface{}{
		[]interface{}{"Card Number", "From"},
	}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for missing 'to' column, got %v", err)
	}
}

func TestMakeTableWithDuplicatedColumn(t *testing.T) {
	var data = [][]interface{}{
		[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Gate"},
		[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
		[]interface{}{"6001002", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
	}

	_, err := makeTable(data)
	if err == nil {
		t.Fatalf("Expected error return for duplicated column, got %v", err)
	}
}

func TestMakeTableWithInvalidCardNumber(t *testing.T) {
	expected := api.Table{
		Header: []string{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		Records: [][]string{
			{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	var data = [][]interface{}{
		[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
		[]interface{}{"600100X", "2020-02-03", "2020-11-30", "Y", "Y", "N", "N"},
		[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
	}

	table, err := makeTable(data)
	if err != nil {
		t.Fatalf("Unexpected error returned from makeTable (%v)", err)
	}

	if table == nil {
		t.Fatalf("makeTable returend %v", table)
	}

	if !reflect.DeepEqual(*table, expected) {
		t.Errorf("Incorrect table\n   expected: %v\n   got:      %v\n", expected, *table)
	}
}

func TestMakeTableWithInvalidFromDate(t *testing.T) {
	expected := api.Table{
		Header: []string{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		Records: [][]string{
			{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	var data = [][]interface{}{
		[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
		[]interface{}{"6001002", "2020-02-0X", "2020-11-30", "Y", "Y", "N", "N"},
		[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
	}

	table, err := makeTable(data)
	if err != nil {
		t.Fatalf("Unexpected error returned from makeTable (%v)", err)
	}

	if table == nil {
		t.Fatalf("makeTable returend %v", table)
	}

	if !reflect.DeepEqual(*table, expected) {
		t.Errorf("Incorrect table\n   expected: %v\n   got:      %v\n", expected, *table)
	}
}

func TestMakeTableWithInvalidToDate(t *testing.T) {
	expected := api.Table{
		Header: []string{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		Records: [][]string{
			{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
			{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
		},
	}

	var data = [][]interface{}{
		[]interface{}{"Card Number", "From", "To", "Gate", "Tower", "Dungeon", "Lair"},
		[]interface{}{"6001001", "2020-01-01", "2020-12-31", "Y", "N", "N", "Y"},
		[]interface{}{"6001002", "2020-02-03", "2020-11-3X", "Y", "Y", "N", "N"},
		[]interface{}{"6001003", "2020-01-01", "2020-12-31", "Y", "N", "Y", "N"},
	}

	table, err := makeTable(data)
	if err != nil {
		t.Fatalf("Unexpected error returned from makeTable (%v)", err)
	}

	if table == nil {
		t.Fatalf("makeTable returend %v", table)
	}

	if !reflect.DeepEqual(*table, expected) {
		t.Errorf("Incorrect table\n   expected: %v\n   got:      %v\n", expected, *table)
	}
}
