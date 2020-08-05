![build](https://github.com/uhppoted/uhppoted-app-sheets/workflows/build/badge.svg)

# uhppoted-app-sheets

```cron```'able command line utility to transfer access control lists and events between a Google Sheets worksheet and a set of UHPPOTE UTO311-L0x access controllers.
access controller boards. 

It originated out of a need for a cheap-and-nasty access management system for a non-profit after budget constraints caused by the Covid-19 pandemic precluded a commercial offering. And turned out to be surprisingly usable!

Supported operating systems:
- Linux
- MacOS
- Windows
- ARM7

## Releases

| *Version* | *Description*                                                            |
| --------- | ------------------------------------------------------------------------ |
| v0.6.4    | Initial release                                                          |

## Installation

Executables for all the supported operating systems are packaged in the [releases](https://github.com/uhppoted/uhppoted-app-sheets/releases). The provided archives contain the executables for all the operating systems - OS specific tarballs can be found in the [uhppoted](https://github.com/uhppoted/uhppoted/releases) releases.

Installation is straightforward - download the archive and extract it to a directory of your choice and then place the executable in a directory in your PATH. The `uhppoted-app-sheets` utility requires the following additional 
files:

- `uhppoted.conf`

On the first invocation of any of the commands you will be prompted to grant read/write access to a worksheet:

1. Open the URL provided at the prompt
2. Grant access to the application
3. Copy-and-paste the code provided by Google back into the command prompt
4. You're good to go.

`load-acl` additionally requires read access to Google Drive to retrieve the worksheet version (the same procedure applies).

**NOTE:** 

*`uhppoted-app-sheets` requires read permission for Google Drive and read/write for Google Sheets. Google access permissions are granted for the whole account, not just the worksheet in use. It is **highly** recommended that a dedicated account be created for use with `uhppoted-app-sheets`, with access only to the worksheets required for maintaining the access controllers.*


### `uhppoted.conf`

`uhppoted.conf` is the communal configuration file shared by all the `uhppoted` project modules and is (or will 
eventually be) documented in [uhppoted](https://github.com/uhppoted/uhppoted). `uhppoted-app-sheets` requires the 
_devices_ section to resolve non-local controller IP addresses and door to controller door identities.

A sample _[uhppoted.conf](https://github.com/uhppoted/uhppoted/blob/master/app-notes/google-sheets/uhppoted.conf)_ file is included in the `uhppoted` distribution.

### Building from source

Assuming you have `Go` and `make` installed:

```
git clone https://github.com/uhppoted/uhppoted-app-sheets.git
cd uhppoted-app-sheets
make build
```

If you prefer not to use `make`:
```
git clone https://github.com/uhppoted/uhppoted-app-sheets.git
cd uhppoted-app-sheets
mkdir bin
go build -o bin ./...
```

The above commands build the `uhppoted-app-sheets` executable to the `bin` directory.

#### Dependencies

| *Dependency*                                                                 | *Description*                              |
| ---------------------------------------------------------------------------- | ------------------------------------------ |
| [com.github/uhppoted/uhppote-core](https://github.com/uhppoted/uhppote-core) | Device level API implementation            |
| [com.github/uhppoted/uhppoted-api](https://github.com/uhppoted/uhppoted-api) | Common API for external applications       |
| [google.golang.org/api](https://google.golang.org/api)                       | Google Sheets API v4 Go library            |
| golang.org/x/net                                                             | Google Sheets API library dependency              |
| golang.org/x/oauth2                                                          | Google Sheets API library dependency              |
| golang.org/x/sys                                                             | Google Sheets API library dependency              |
| golang.org/x/lint/golint                                                     | Additional *lint* check for release builds |

## uhppoted-app-sheets

Usage: ```uhppoted-app-sheets [--debug] [--config <configuration file>] <command> [options]```

Supported commands:

- `help`
- `version`
- `get`
- `put`
- `load-acl`
- `upload-acl`
- `compare-acl`

### `help`

Displays a summary of the command usage and options.

Command line:

- ```uhppoted-app-sheets help``` displays a short summary of the command and a list of the available commands

- ```uhppoted-app-sheet help <command>``` displays the command specific information.

### `version`

Displays the current version of the command.

Command line:

```uhppoted-app-sheets version```

### `get`

Fetches tabular data from a Google Sheets worksheet and stores it as a TSV file. Intended for use in a `cron` task that routinely transfers information from the worksheet for scripts on the local host managing the access control system. 

The range retrieved from the worksheet is expected to have column headings in the first row.

Command line:

```uhppoted-app-sheets get --url <url> --range <range>``` 

```uhppoted-app-sheets [--debug] get --url <url> --range <range> [--file <TSV>] [--workdir <dir>] [--credentials <file>]```

```
  --url         Google Sheets worksheet URL from which to fetch the data 
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the data (e.g. Names!A2:K)
  --file        File path for the destination TSV file. Defaults to <yyyy-mm-dd HHmmss>.tsv
  
  --workdir     Directory for working files, in particular the tokens, revisions, etc
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/.google/credentials.json
  
  --debug       Displays verbose debugging information, in particular the communications
                with the UHPPOTE controllers
```

### `put`

Uploads a TSV file as tabular data to a Google Sheets worksheet. Intended for use in a `cron` task that routinely transfers information to the worksheet from scripts on the local host (e.g. consolidated daily event reports).

The first row of the TSV file is interpreted as column headers.

Command line:

```uhppoted-app-sheets put --file <TSV> --url <url> --range <range>``` 

```uhppoted-app-sheets [--debug] put --file <TSV> --url <url> --range <range> [--workdir <dir>] [--credentials <file>]```

```
  --file        File path for the TSV file to be uploaded
  --url         Google Sheets worksheet URL to which to upload the data 
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the data (e.g. Summary!A1:D)
  
  --workdir     Directory for working files, in particular the tokens, revisions, etc
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/.google/credentials.json
  
  --debug       Displays verbose debugging information, in particular the communications
                with the UHPPOTE controllers
```

### `load-acl`

Fetches an ACL file from a Google Sheets worksheet and downloads it to the configured UHPPOTE controllers. Intended for use in a `cron` task that routinely updates the controllers from an access management system based on Google Sheets.

Optionally, the command writes an operation summary to a _Log_ worksheet and a summary of changes to a _Report_ worksheet.

Unless the `--force` option is specified, the command will not download and update the access controllers if the Google Sheets worksheet revision has not changed. 

Command line:

```uhppoted-app-sheets load-acl --url <url> --range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] load-acl --url <url> --range <range> [--force] [--delay <duration>] [--strict] [--dry-run] [--workdir <dir>] [--credentials <file>] [--no-log] [--log-range <range>] [--log-retention <days>] [--no-report] [--report-range <range>] [--report-retention <days>] ```

```
  --url              Google Sheets worksheet URL from which to fetch the ACL
                     e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range            Worksheet range of the ACL (e.g. ACL!A2:K)
  --delay            'Settling' delay after an edit before a worksheet is regarded as stable.
                     Specified in as a Go 'duration' e.g. 10m15s and defaults to 15m
  --force            Ignores the worksheet revision and retrieves and updates the access
                     control lists. 
  --strict           Fails with an error if the worksheet contains errors e.g. duplicate 
                     card numbers
  --dry-run          Executes the load-acl command but does not update the access
                     control lists on the controllers. Used primarily for testing 
                     scripts, crontab entries and debugging. 

  --workdir          Directory for working files, in particular the tokens, revisions,
                     etc, that provide access to Google Sheets. Defaults to:
                     - /var/uhppoted on Linux
                     - /usr/local/var/com.github.uhppoted on MacOS
                     - ./uhppoted on Microsoft Windows
  --credentials      Path for the Google Docs credentials file. 
                     Defaults to <workdir>/.google/credentials.json

  --no-log           Disables the creation of log entries on the 'log' worksheet
  --log-range        Worksheet range (e.g. Log!A2:H) for log entries. Defaults to Log!A1:H
  --log-retention    Number of days to retain log entries. Rows in the 'log' worksheet
                     with a timestamp before the retention date are deleted.
  
  --no-report        Disables the creation of report entries on the 'report' worksheet
  --report-range     Worksheet range (e.g. Report!B2:F) for report entries. Defaults to Report!A1:E
  --report-retention Number of days to retain report entries. Rows in the 'report'
                     worksheet with a timestamp before the retention date are deleted.
    
  --config           File path to the uhppoted.conf file containing the access
                     controller configuration information. Defaults to:
                     - /etc/uhppoted/uhppoted.conf (Linux)
                     - /usr/local/etc/com.github.twystd.uhppoted/uhppoted.conf (MacOS)
                     - ./uhppoted.conf (Windows)

  --debug            Displays verbose debugging information, in particular the 
                     communications with the UHPPOTE controllers
```

### `upload-acl`

Fetches the cards stored in the configured UHPPOTE controllers, creates a matching ACL from the controller configuration and uploads it to a Google Sheets worksheet. Intended for use in a `cron` task that facilitates audits of the cards stored on the controllers against an authoritative source. 

The destination worksheet is expected to the column names in the first row of the range, and the following column names are hardcoded (column names are case- and space-insensitive):

- card number
- from
- to

Command line:

```uhppoted-app-sheets upload-acl --url <url> --range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] upload-acl --url <url> [--workdir <dir>] [--credentials <file>]```

```
  --url         Google Sheets worksheet URL to which to upload the ACL
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the ACL (e.g. ACL!A2:K)
  
  --workdir     Directory for working files, in particular the tokens, revisions, etc, 
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/.google/credentials.json

  --config      File path to the uhppoted.conf file containing the access controller 
                configuration information. Defaults to:
                - /etc/uhppoted/uhppoted.conf (Linux)
                - /usr/local/etc/com.github.twystd.uhppoted/uhppoted.conf (MacOS)
                - ./uhppoted.conf (Windows)

  --debug       Displays verbose debugging information, in particular the communications 
                with the UHPPOTE controllers
```

### `compare-acl`

Fetches an ACL from a Google Sheets worksheet and compares it to the cards stored in the configured access controllers. Intended for use in a `cron` task that routinely audits the controllers against an authoritative source.


Command line:

```uhppoted-app-sheets compare-acl --url <url> --range <range> --report-range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] compare-acl --acl <url> --report-range <range> [--workdir <dir>] [--credentials <file>]```
```
  --url           Google Sheets worksheet URL from which to retrieve the ACL and to which
                  to upload the report
                  e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range         Worksheet range of the ACL (e.g. ACL!A2:K)
  --report-range  Worksheet range (e.g. Audit!A1:D) for the compare report. Defaults to 
                  Audit!A1:D
  
  --workdir       Directory for working files, in particular the tokens, revisions, etc, 
                  that provide access to Google Sheets. Defaults to:
                  - /var/uhppoted on Linux
                  - /usr/local/var/com.github.uhppoted on MacOS
                  - ./uhppoted on Microsoft Windows
  --credentials   Path for the Google Docs credentials file. 
                  Defaults to <workdir>/.google/credentials.json

  --config        File path to the uhppoted.conf file containing the access controller 
                  configuration information. Defaults to:
                  - /etc/uhppoted/uhppoted.conf (Linux)
                  - /usr/local/etc/com.github.twystd.uhppoted/uhppoted.conf (MacOS)
                  - ./uhppoted.conf (Windows)

  --debug         Displays verbose debugging information, in particular the 
                  communications with the UHPPOTE controllers

```
```
