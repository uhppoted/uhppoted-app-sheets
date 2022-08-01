DIST       ?= development
CLI         = ./bin/uhppoted-app-sheets
CREDENTIALS = ../runtime/sheets/.google.json
CONFIG      = /usr/local/etc/com.github.uhppoted/uhppoted.conf
URL         = https://docs.google.com/spreadsheets/d/1_erZMyFmO6PM0PrAfEqdsiH9haiw-2UqY0kLwo_WTO8

DATETIME  = $(shell date "+%Y-%m-%d %H:%M:%S")
DEBUG    ?= --debug

.PHONY: clean
.PHONY: update
.PHONY: update-release

all: test      \
	 benchmark \
     coverage

clean:
	go clean
	rm -rf bin

update:
	go get -u github.com/uhppoted/uhppote-core@master
	go get -u github.com/uhppoted/uhppoted-lib@master
	go get -u golang.org/x/net
	go get -u golang.org/x/oauth2
	go get -u golang.org/x/sys
	go get -u google.golang.org/api
	go mod tidy

update-release:
	go get -u github.com/uhppoted/uhppote-core
	go get -u github.com/uhppoted/uhppoted-lib
	go get -u golang.org/x/net
	go get -u golang.org/x/oauth2
	go get -u golang.org/x/sys
	go get -u google.golang.org/api
	go mod tidy

format: 
	go fmt ./...

build: format
	mkdir -p bin
	go build -trimpath -o bin ./...

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
	env GOOS=linux   GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/linux   ./...
	env GOOS=linux   GOARCH=arm   GOARM=7 GOWORK=off go build -trimpath -o dist/$(DIST)/arm7    ./...
	env GOOS=darwin  GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/darwin  ./...
	env GOOS=windows GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/windows ./...

release: update-release build-all
	find . -name ".DS_Store" -delete
	tar --directory=dist --exclude=".DS_Store" -cvzf dist/$(DIST).tar.gz $(DIST)
	cd dist; zip --recurse-paths $(DIST).zip $(DIST)

bump:
	go get -u github.com/uhppoted/uhppote-core
	go get -u github.com/uhppoted/uhppoted-lib
	go get -u golang.org/x/net
	go get -u golang.org/x/oauth2
	go get -u golang.org/x/sys
	go get -u google.golang.org/api

debug: build
	$(CLI) --config $(CONFIG) load-acl \
	       --url $(URL) \
	       --range "ACL!A2:K" \
	       --credentials $(CREDENTIALS) \
	       --report-range "Report!A1:C" \
	       --report-retention 1 \
	       --log-range "Log!A1:H" \
	       --log-retention 1 \
	       --dry-run \
	       --force \
	       --delay 5m


# GENERAL COMMANDS

usage: build
	$(CLI)

help: build
	$(CLI) help
	$(CLI) help get
	$(CLI) help put
	$(CLI) help load-acl
	$(CLI) help compare-acl
	$(CLI) help upload-acl

version: build
	$(CLI) version

# ACL COMMANDS

get: build
	$(CLI) get --url $(URL) \
               --range "ACL!A2:K" \
               --credentials $(CREDENTIALS) \
               --file "../runtime/sheets/debug.acl"

put: build
	$(CLI) put --url $(URL) \
               --range "AsIs!A2:K"      \
               --credentials $(CREDENTIALS) \
               --file ../runtime/sheets/debug.acl

load-acl: build
	$(CLI) --config $(CONFIG) load-acl \
           --url $(URL) \
	       --range "ACL!A2:K" \
	       --credentials $(CREDENTIALS) \
	       --report-range "Report!A1:C" \
	       --report-retention 1 \
	       --log-range "Log!A1:H" \
	       --log-retention 1 \
	       --force \
	       --delay 5m

compare-acl: build
	$(CLI) --config $(CONFIG) compare-acl \
           --url $(URL) \
           --range "ACL!A2:K" \
           --credentials $(CREDENTIALS) \
           --report-range "Audit!A1:E"

upload-acl: build
	$(CLI) --config $(CONFIG) upload-acl \
           --url $(URL) \
           --range "Uploaded!A1:K"      \
           --credentials $(CREDENTIALS)
                       
