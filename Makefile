VERSION     = v0.6.x
LDFLAGS     = -ldflags "-X uhppote.VERSION=$(VERSION)" 
DIST       ?= development
CLI         = ./bin/uhppoted-app-sheets
CREDENTIALS = ../runtime/.uhppoted-test.json

DATETIME  = $(shell date "+%Y-%m-%d %H:%M:%S")
DEBUG    ?= --debug

all: test      \
	 benchmark \
     coverage

clean:
	go clean
	rm -rf bin

format: 
	go fmt ./...

build: format
	mkdir -p bin
	go build -o bin ./...

test: build
	go test ./...

vet: build
	go vet ./...

lint: build
	golint ./...

benchmark: build
	go test -bench ./...

coverage: build
	go test -cover ./...

build-all: test vet
	mkdir -p dist/$(DIST)/windows
	mkdir -p dist/$(DIST)/darwin
	mkdir -p dist/$(DIST)/linux
	mkdir -p dist/$(DIST)/arm7
	env GOOS=linux   GOARCH=amd64         go build -o dist/$(DIST)/linux   ./...
	env GOOS=linux   GOARCH=arm   GOARM=7 go build -o dist/$(DIST)/arm7    ./...
	env GOOS=darwin  GOARCH=amd64         go build -o dist/$(DIST)/darwin  ./...
	env GOOS=windows GOARCH=amd64         go build -o dist/$(DIST)/windows ./...

release: build-all
	find . -name ".DS_Store" -delete
	tar --directory=dist --exclude=".DS_Store" -cvzf dist/$(DIST).tar.gz $(DIST)
	cd dist; zip --recurse-paths $(DIST).zip $(DIST)

debug: build
	$(CLI) help
	$(CLI) help upload-acl
	$(CLI) upload-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
                       --range "Uploaded!A1:K"      \
                       --credentials $(CREDENTIALS) \
                       --config ../runtime/sheets/uhppoted.conf

usage: build
	$(CLI)

help: build
	$(CLI) help
	$(CLI) help get-acl
	$(CLI) help load-acl
	$(CLI) help compare-acl
	$(CLI) help upload-acl

version: build
	$(CLI) version

# SHEETS COMMANDS

get-acl: build
#	$(CLI) get-acl --credentials $(CREDENTIALS) --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" --range "ACL!A2:K" --file "../runtime/sheets/debug.acl"
	$(CLI) get-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
	               --range "ACL!A2:K" \
	               --file "../runtime/sheets/debug.acl"

load-acl: build
#	$(CLI) load-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
#	                --range "ACL!A2:K" \
#	                --credentials $(CREDENTIALS) \
#	                --dry-run \
#	                --force   \
#	                --config ../runtime/sheets/uhppoted.conf \
#	                --report-range "Report!B2:G" \
#	                --log-range "Log!A1:I" \
#	                --log-retention 1 \
#	                --delay 5m \
#	                --report-always
	$(CLI) load-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
	                --range "ACL!A2:K" \
	                --credentials $(CREDENTIALS) \
	                --config ../runtime/sheets/uhppoted.conf \
	                --report-range "Report!B2:G" \
	                --log-range "Log!A1:I" \
	                --log-retention 1 \
	                --delay 5m

compare-acl: build
	$(CLI) compare-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
                       --range "ACL!A2:K" \
                       --credentials $(CREDENTIALS) \
                       --config ../runtime/sheets/uhppoted.conf \
                       --report-range "Audit!B2:E"

upload-acl: build
	$(CLI) upload-acl --url "https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k" \
                       --range "Uploaded!A1:K"      \
                       --credentials $(CREDENTIALS) \
                       --config ../runtime/sheets/uhppoted.conf
