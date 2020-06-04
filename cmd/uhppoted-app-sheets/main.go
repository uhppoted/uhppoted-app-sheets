package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/uhppoted/uhppoted-app-sheets/acl"
)

var options = struct {
	credentials string
	url         string
	region      string
	file        string
	debug       bool
}{
	credentials: "",
	url:         "",
	region:      "",
	file:        time.Now().Format("2006-01-02T150405.acl"),
	debug:       false,
}

func main() {
	flag.StringVar(&options.credentials, "credentials", options.credentials, "Path for the 'credentials.json' file")
	flag.StringVar(&options.url, "url", options.url, "Spreadsheet URL")
	flag.StringVar(&options.region, "range", options.region, "Spreadsheet range e.g. 'Class Data!A2:E'")
	flag.StringVar(&options.file, "file", options.file, "TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")
	flag.BoolVar(&options.debug, "debug", options.debug, "Enable debugging information")
	flag.Parse()

	if strings.TrimSpace(options.credentials) == "" {
		fmt.Printf("--credentials is a required option")
		usage()
		os.Exit(1)
	}

	if strings.TrimSpace(options.url) == "" {
		fmt.Printf("--url is a required option")
		usage()
		os.Exit(1)
	}

	if strings.TrimSpace(options.region) == "" {
		fmt.Printf("--range is a required option")
		usage()
		os.Exit(1)
	}

	re := regexp.MustCompile("https://docs.google.com/spreadsheets/d/([A-Za-z0-9_]+)")
	match := re.FindStringSubmatch(options.url)

	if len(match) < 2 {
		log.Fatalf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	spreadsheet := match[1]
	region := options.region

	if options.debug {
		log.Printf("DEBUG  Spreadsheet - ID:%s  range:%s", spreadsheet, region)
	}

	client, err := authorize(options.credentials)
	if err != nil {
		log.Fatalf("Authentication/authorization error (%w)", err)
	}

	google, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to create new Sheets client (%w)", err)
	}

	response, err := google.Spreadsheets.Values.Get(spreadsheet, region).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet (%w)", err)
	}

	if len(response.Values) == 0 {
		log.Fatalf("No data in spreadsheet/range")
	}

	if err := acl.MakeTSV(options.file, response); err != nil {
		log.Fatalf("Error creating TSV file (%w)", err)
	}

	//		w := csv.NewWriter(f)
	//		w.Comma = '\t'
	//
	//		// ... header
	//		row := resp.Values[0]
	//		record := make([]string, len(row))
	//		for i, v := range row {
	//			record[i] = strings.TrimSpace(v.(string))
	//		}
	//
	//		w.Write(record)
	//
	//		// ... data
	//		for _, row := range resp.Values[1:] {
	//			if cardnumber, ok := row[0].(string); !ok {
	//				continue
	//			} else if ok, err := regexp.Match(`^\s*[0-9]+\s*$`, []byte(cardnumber)); !ok || err != nil {
	//				continue
	//			}
	//
	//			record := make([]string, len(row))
	//			for i, v := range row {
	//				record[i] = strings.TrimSpace(v.(string))
	//			}
	//
	//			w.Write(record)
	//		}
	//
	//		w.Flush()

	log.Printf("Retrieved ACL to file %s\n", options.file)
}

func usage() {
	fmt.Println()
	fmt.Println("  Usage: uhppoted-app-sheets <command> [options]")
	fmt.Println()
	fmt.Println("  Commands:")
	fmt.Println()
	fmt.Println("    get-acl          Retrieves an ACL from a Google Sheets worksheet as a TSV file")
	fmt.Println("    help             Displays this message")
	fmt.Println("                     For help on a specific command use 'uhppoted-app-sheets help <command>'")
	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println()
	fmt.Println("    --credentials    Path to 'credentials.json'")
	fmt.Println("    --url            Spreadsheet URL")
	fmt.Println("    --range          Worksheet range containing ACL e.g. 'Class Data!A2:E'")
	fmt.Println("    --file           TSV file name. Defaults to 'ACL - <yyyy-mm-dd HHmmss>.tsv'")
	fmt.Println()
}
