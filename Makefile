DIST       ?= development
CLI         = ./bin/uhppoted-app-sheets
CREDENTIALS = /usr/local/etc/com.github.uhppoted/sheets/.google/credentials.json
CONFIG      = /usr/local/etc/com.github.uhppoted/uhppoted.conf
URL         = https://docs.google.com/spreadsheets/d/1_erZMyFmO6PM0PrAfEqdsiH9haiw-2UqY0kLwo_WTO8
URL_WITH_PIN = https://docs.google.com/spreadsheets/d/1OztvzkTlCpa_OBK4u6reckKAB4d7VbDRrGrXNCXgEMQ/edit#gid=640947601

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
	go get -u github.com/uhppoted/uhppote-core@main
	go get -u github.com/uhppoted/uhppoted-lib@main
	# go get -u golang.org/x/net
	# go get -u golang.org/x/oauth2
	# go get -u golang.org/x/sys
	# go get -u google.golang.org/api
	go mod tidy

update-release:
	go get -u github.com/uhppoted/uhppote-core
	go get -u github.com/uhppoted/uhppoted-lib
	go mod tidy

update-all:
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

benchmark: build
	go test -bench ./...

coverage: build
	go test -cover ./...

vet: build
	go vet ./...

lint: build
	env GOOS=darwin  GOARCH=amd64 staticcheck ./...
	env GOOS=linux   GOARCH=amd64 staticcheck ./...
	env GOOS=windows GOARCH=amd64 staticcheck ./...

vuln:
	govulncheck ./...

build-all: build test vet lint
	mkdir -p dist/$(DIST)/linux
	mkdir -p dist/$(DIST)/arm
	mkdir -p dist/$(DIST)/arm7
	mkdir -p dist/$(DIST)/darwin-x64
	mkdir -p dist/$(DIST)/darwin-arm64
	mkdir -p dist/$(DIST)/windows
	env GOOS=linux   GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/linux        ./...
	env GOOS=linux   GOARCH=arm64         GOWORK=off go build -trimpath -o dist/$(DIST)/arm          ./...
	env GOOS=linux   GOARCH=arm   GOARM=7 GOWORK=off go build -trimpath -o dist/$(DIST)/arm7         ./...
	env GOOS=darwin  GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/darwin-x64   ./...
	env GOOS=darwin  GOARCH=arm64         GOWORK=off go build -trimpath -o dist/$(DIST)/darwin-arm64 ./...
	env GOOS=windows GOARCH=amd64         GOWORK=off go build -trimpath -o dist/$(DIST)/windows      ./...

release: update-release build-all
	find . -name ".DS_Store" -delete
	tar --directory=dist/$(DIST)/linux        --exclude=".DS_Store" -cvzf dist/$(DIST)-linux-x64.tar.gz    .
	tar --directory=dist/$(DIST)/arm          --exclude=".DS_Store" -cvzf dist/$(DIST)-arm-x64.tar.gz      .
	tar --directory=dist/$(DIST)/arm7         --exclude=".DS_Store" -cvzf dist/$(DIST)-arm7.tar.gz         .
	tar --directory=dist/$(DIST)/darwin-x64   --exclude=".DS_Store" -cvzf dist/$(DIST)-darwin-x64.tar.gz   .
	tar --directory=dist/$(DIST)/darwin-arm64 --exclude=".DS_Store" -cvzf dist/$(DIST)-darwin-arm64.tar.gz .
	cd dist/$(DIST)/windows && zip --recurse-paths ../../$(DIST)-windows-x64.zip . -x ".DS_Store"

publish: release
	echo "Releasing version $(VERSION)"
	gh release create "$(VERSION)" "./dist/$(DIST)-arm-x64.tar.gz"      \
	                               "./dist/$(DIST)-arm7.tar.gz"         \
	                               "./dist/$(DIST)-darwin-arm64.tar.gz" \
	                               "./dist/$(DIST)-darwin-x64.tar.gz"   \
	                               "./dist/$(DIST)-linux-x64.tar.gz"    \
	                               "./dist/$(DIST)-windows-x64.zip"     \
	                               --draft --prerelease --title "$(VERSION)-beta" --notes-file release-notes.md

debug: build
	$(CLI) get --url $(URL_WITH_PIN) \
	           --credentials $(CREDENTIALS) \
	           --tokens ../runtime/sheets/.debug \
	           --range "ACL!A2:K" \
	           --file "../runtime/sheets/debug.acl"
	# env GOOS=windows GOARCH=amd64 go build -trimpath -o dist/$(DIST)/windows ./...
	# $(CLI) authorise --url $(URL) \
	#                  --tokens ../runtime/sheets/.google
	# $(CLI) get --url $(URL) \
	#            --credentials $(CREDENTIALS) \
	#            --tokens ../runtime/sheets/.google \
	#            --range "ACL!A2:K" \
	#            --file "../runtime/sheets/debug.acl"

godoc:
	godoc -http=:80	-index_interval=60s

# GENERAL COMMANDS

usage: build
	$(CLI)

help: build
	$(CLI) help
	$(CLI) help authorise
	$(CLI) help authorize
	$(CLI) help get
	$(CLI) help put
	$(CLI) help load-acl
	$(CLI) help compare-acl
	$(CLI) help upload-acl

version: build
	$(CLI) version

# ACL COMMANDS

auth: build
	$(CLI) authorize --credentials ${CREDENTIALS} --url $(URL)

auth-with-pin: build
	$(CLI) authorize --credentials ${CREDENTIALS} --url $(URL_WITH_PIN)

get: build
	$(CLI) get --url $(URL_WITH_PIN) \
	           --credentials $(CREDENTIALS) \
	           --range "ACL!A2:K" \
	           --file "../runtime/sheets/debug.acl"

get-with-pin: build
	$(CLI) get --url $(URL_WITH_PIN) \
	           --with-pin \
	           --credentials $(CREDENTIALS) \
	           --range "ACL!A2:M" \
	           --file "../runtime/sheets/debug-with-pin.acl"

put: build
	$(CLI) put --url $(URL) \
               --range "AsIs!A2:K"      \
               --credentials $(CREDENTIALS) \
               --file ../runtime/sheets/debug.acl

put-with-pin: build
	$(CLI) put --url $(URL_WITH_PIN) \
	           --with-pin \
               --range "AsIs!A2:M"      \
               --credentials $(CREDENTIALS) \
               --file ../runtime/sheets/debug-with-pin.acl

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

load-acl-with-pin: build
	$(CLI) --config $(CONFIG) load-acl \
           --with-pin \
           --url $(URL_WITH_PIN) \
	       --range "ACL!A2:M" \
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

compare-acl-with-pin: build
	$(CLI) --config $(CONFIG) compare-acl \
           --with-pin \
           --url $(URL_WITH_PIN) \
           --range "ACL!A2:M" \
           --credentials $(CREDENTIALS) \
           --report-range "Audit!A1:E"

upload-acl: build
	$(CLI) --config $(CONFIG) upload-acl \
           --url $(URL) \
           --range "Uploaded!A1:K"      \
           --credentials $(CREDENTIALS)
                       
upload-acl-with-pin: build
	$(CLI) --config $(CONFIG) upload-acl \
           --with-pin \
           --url $(URL_WITH_PIN) \
           --range "Uploaded!A1:M"      \
           --credentials $(CREDENTIALS)
                       
