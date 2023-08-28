![build](https://github.com/uhppoted/uhppoted-app-sheets/workflows/build/badge.svg)

# uhppoted-app-sheets

```cron```'able command line utility to transfer access control lists and events between a Google Sheets worksheet and a set of UHPPOTE UTO311-L0x access controllers.

It originated out of a need for a cheap-and-nasty access management system for a non-profit after budget constraints caused by the Covid-19 pandemic precluded a commercial offering. And turned out to be surprisingly usable.

Supported operating systems:
- Linux
- MacOS
- Windows
- ARM7

_Example Google Sheets worksheet: [uhppoted-app-sheets-demo](https://docs.google.com/spreadsheets/d/1_erZMyFmO6PM0PrAfEqdsiH9haiw-2UqY0kLwo_WTO8/edit?usp=sharing)_

## Releases

| *Version* | *Description*                                                                   |
| --------- | ------------------------------------------------------------------------------- |
| v0.8.6    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.8.5    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.8.4    | Added support for card keypad PINs                                              |
| v0.8.3    | Updated authorisation for new Google Docs OAuth2 flow                           |
| v0.8.2    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.8.1    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.8.0    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.7.3    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.7.2    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.7.1    | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-lib` |
| v0.7.0    | Added support for time profiles from the extended API                           |
| v0.6.12   | Maintenance release to update dependencies on `uhppote-core` and `uhppoted-api` |
| v0.6.10   | Maintenance release for version compatibility with `uhppoted-app-wild-apricot`  |
| v0.6.8    | Maintenance release for version compatibility with `uhppote-core` `v.0.6.8`     |
| v0.6.7    | Maintenance release for version compatibility with `uhppoted-api` `v.0.6.7`     |
| v0.6.5    | Maintenance release for version compatibility with `node-red-contrib-uhppoted`  |
| v0.6.4    | Initial release                                                                 |

## Installation

Executables for all the supported operating systems are packaged in the [releases](https://github.com/uhppoted/uhppoted-app-sheets/releases). The provided archives contain the executables for all the operating systems - OS specific tarballs can be found in the [uhppoted](https://github.com/uhppoted/uhppoted/releases) releases.

Installation is straightforward - download the archive and extract it to a directory of your choice and then place the executable in a directory in your PATH. The `uhppoted-app-sheets` utility requires the following additional 
files:

- `uhppoted.conf`
- `credentials.json` (see [HOWTO: Authorisation](https://github.com/uhppoted/uhppoted-app-sheets/blob/main/documentation/authorisation.md))

As of [3 October 2022](https://developers.google.com/identity/protocols/oauth2/resources/oob-migration), 
read and write authorisation for _uhppoted-app-sheets_ to access the _Google Sheet_ spreadsheet now requires
a fairly involved setup process, which is documented in more detail in [HOWTO: Authorisation](https://github.com/uhppoted/uhppoted-app-sheets/blob/main/documentation/authorisation.md).

**NOTE:** 

*`uhppoted-app-sheets` requires read permission for _Google Drive_ and read/write for _Google Sheets_.*

*The access permissions are granted for the whole account, not just the worksheet in use. It is **highly** recommended that a dedicated account be created for use with `uhppoted-app-sheets`, with access only to the spreadsheets required for maintaining the access controllers.*


### `uhppoted.conf`

`uhppoted.conf` is the communal configuration file shared by all the `uhppoted` project modules and is (or will 
eventually be) documented in [uhppoted](https://github.com/uhppoted/uhppoted). A sample [_uhppoted.conf_](https://github.com/uhppoted/uhppoted-app-sheets/blob/main/documentation/uhppoted.conf) file is included in the _documentation_ folder.

`uhppoted-app-sheets` requires the _controllers_ section to resolve non-local controller IP addresses and the mapping from
ACL door names to controller doors e.g.:

 _ACL_

| Card Number | PIN  | From       | To         | Great Hall | Gryffindor | Hufflepuff | Ravenclaw | Slytherin | Dungeon | Kitchen | Hogsmeade |
|-------------|------|------------|------------|------------|------------|------------|-----------|-----------|---------|---------|-----------|
| 8112345     | 1234 | 2023-01-01 | 2022-12-31 | Y          | Y          | N          | N         | N         | N       | Y       | Y         |
| 8154321     | 4321 | 2023-01-01 | 2022-12-31 | Y          | N          | Y          | N         | N         | Y       | N       | Y         |

(the PIN field is optional)

_uhppoted.conf_

```
...
# CONTROLLERS
UT0311-L0x.405419896.door.1 = Great Hall
UT0311-L0x.405419896.door.2 = Kitchen
UT0311-L0x.405419896.door.3 = Dungeon
UT0311-L0x.405419896.door.4 = Hogsmeade

UT0311-L0x.303986753.door.1 = Gryffindor
UT0311-L0x.303986753.door.2 = Hufflepuff
UT0311-L0x.303986753.door.3 = Ravenclaw
UT0311-L0x.303986753.door.4 = Slytherin
...
```
Permissions granted to the _Great Hall_, _Kitchen_, _Dungeon_ and _Hogsmeade_ are mapped to doors 1-4 on the controller with
serial number 405419896 and permissions granted to _Gryffindor_, _Hufflepuff_, _Ravenclaw_ and _Slytherin_ are mapped to doors
1-4 on the controller with serial number 303986753.

Controller doors that are not in use should also be mapped to name e.g. Unused1, Unused2, etc.s

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
go build -trimpath -o bin ./...
```

The above commands build the `uhppoted-app-sheets` executable to the `bin` directory.

#### Dependencies

| *Dependency*                                                                 | *Description*                              |
| ---------------------------------------------------------------------------- | ------------------------------------------ |
| [com.github/uhppoted/uhppote-core](https://github.com/uhppoted/uhppote-core) | Device level API implementation            |
| [com.github/uhppoted/uhppoted-lib](https://github.com/uhppoted/uhppoted-lib) | Shared application library                 |
| [google.golang.org/api](https://google.golang.org/api)                       | Google Sheets API v4 Go library            |
| golang.org/x/net                                                             | Google Sheets API library dependency       |
| golang.org/x/oauth2                                                          | Google Sheets API library dependency       |
| golang.org/x/sys                                                             | Google Sheets API library dependency       |
| golang.org/x/lint/golint                                                     | Additional *lint* check for release builds |

## uhppoted-app-sheets

Usage: ```uhppoted-app-sheets [--debug] [--config <configuration file>] <command> [options]```

Supported commands:

- `help`
- `version`
- `authorise`
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


### `authorise`

Opens a web page with the links required to authorise read and write access to the spreadsheet.. 

Command line:

```uhppoted-app-sheets authorise --url <url>``` 

```uhppoted-app-sheets [--debug] authorise [--workdir <dir>] [--credentials <file>] --url <url>```

```
  --url         Google Sheets worksheet URL from which to fetch the data 
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  
  --workdir     Directory for working files, in particular the tokens, revisions, etc
                that provide access to Google Sheets. Defaults to:
                - `/var/uhppoted` on Linux
                - `/usr/local/var/com.github.uhppoted` on MacOS
                - `./uhppoted` or `\Program Data\uhppoted` on Microsoft Windows

  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/sheets/.google/credentials.json
  
  --debug       Displays verbose debugging information, in particular the communications
                with the UHPPOTE controllers
```


### `get`

Fetches tabular data from a Google Sheets worksheet and stores it as a TSV file. Intended for use in a `cron` task that routinely transfers information from the worksheet for scripts on the local host managing the access control system. 

The range retrieved from the worksheet is expected to have column headings in the first row.

Command line:

```uhppoted-app-sheets get --url <url> --range <range>``` 

```uhppoted-app-sheets [--debug] get --url <url> --range <range> [--with-pin] [--file <TSV>] [--workdir <dir>] [--credentials <file>]```

```
  --url         Google Sheets worksheet URL from which to fetch the data 
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the data (e.g. Names!A2:K)
  --with-pin    Includes the card keypad PIN code in the retrieved file
  --file        File path for the destination TSV file. Defaults to <yyyy-mm-dd HHmmss>.tsv
  
  --workdir     Directory for working files, in particular the tokens, revisions, etc
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/sheets/.google/credentials.json
  
  --debug       Displays verbose debugging information, in particular the communications
                with the UHPPOTE controllers
```

### `put`

Uploads a TSV file as tabular data to a Google Sheets worksheet. Intended for use in a `cron` task that routinely transfers information to the worksheet from scripts on the local host (e.g. consolidated daily event reports).

The first row of the TSV file is interpreted as column headers.

Command line:

```uhppoted-app-sheets put --file <TSV> --url <url> --range <range>``` 

```uhppoted-app-sheets [--debug] put --file <TSV> --url <url> --range <range> [--with-pin] [--workdir <dir>] [--credentials <file>]```

```
  --file        File path for the TSV file to be uploaded
  --url         Google Sheets worksheet URL to which to upload the data 
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the data (e.g. Summary!A1:D)
  --with-pin    Includes the card keypad PIN code in the uploaded data
  --workdir     Directory for working files, in particular the tokens, revisions, etc
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/sheets/.google/credentials.json
  
  --debug       Displays verbose debugging information, in particular the communications
                with the UHPPOTE controllers
```

### `load-acl`

Fetches an ACL file from a Google Sheets worksheet and downloads it to the configured UHPPOTE controllers. Intended for use in a `cron` task that routinely updates the controllers from an access management system based on Google Sheets.

Optionally, the command writes an operation summary to a _Log_ worksheet and a summary of changes to a _Report_ worksheet.

Unless the `--force` option is specified, the command will not download and update the access controllers if the Google Sheets worksheet revision has not changed. 

Command line:

```uhppoted-app-sheets load-acl --url <url> --range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] load-acl --url <url> --range <range> [--with-pin] [--force] [--delay <duration>] [--strict] [--dry-run] [--workdir <dir>] [--credentials <file>] [--no-log] [--log-range <range>] [--log-retention <days>] [--no-report] [--report-range <range>] [--report-retention <days>] ```

```
  --url              Google Sheets worksheet URL from which to fetch the ACL
                     e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range            Worksheet range of the ACL (e.g. ACL!A2:K)
  --with-pin         Updated the card keypad PIN codes on the controllers
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
                     Defaults to <workdir>/sheets/.google/credentials.json

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
                     - /usr/local/etc/com.github.uhppoted/uhppoted.conf (MacOS)
                     - ./uhppoted.conf (Windows)

  --debug            Displays verbose debugging information, in particular the 
                     communications with the UHPPOTE controllers
```

### `upload-acl`

Fetches the cards stored in the configured UHPPOTE controllers, creates a matching ACL from the controller configuration and uploads it to a Google Sheets worksheet. Intended for use in a `cron` task that facilitates audits of the cards stored on the controllers against an authoritative source. 

The destination worksheet is expected to the column names in the first row of the range, and the following column names are hardcoded (column names are case- and space-insensitive):

- card number
- PIN
- from
- to

Command line:

```uhppoted-app-sheets upload-acl --url <url> --range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] upload-acl --url <url> [--with-pin] [--workdir <dir>] [--credentials <file>]```

```
  --url         Google Sheets worksheet URL to which to upload the ACL
                e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range       Worksheet range of the ACL (e.g. ACL!A2:K)
  --with-pin    Includes the card keypad PIN codes in the uploaded ACL
  --workdir     Directory for working files, in particular the tokens, revisions, etc, 
                that provide access to Google Sheets. Defaults to:
                - /var/uhppoted on Linux
                - /usr/local/var/com.github.uhppoted on MacOS
                - ./uhppoted on Microsoft Windows
  --credentials Path for the Google Docs credentials file. 
                Defaults to <workdir>/sheets/.google/credentials.json

  --config      File path to the uhppoted.conf file containing the access controller 
                configuration information. Defaults to:
                - /etc/uhppoted/uhppoted.conf (Linux)
                - /usr/local/etc/com.github.uhppoted/uhppoted.conf (MacOS)
                - ./uhppoted.conf (Windows)

  --debug       Displays verbose debugging information, in particular the communications 
                with the UHPPOTE controllers
```

### `compare-acl`

Fetches an ACL from a Google Sheets worksheet and compares it to the cards stored in the configured access controllers. Intended for use in a `cron` task that routinely audits the controllers against an authoritative source.


Command line:

```uhppoted-app-sheets compare-acl --url <url> --range <range>--report-range <range>```

```uhppoted-app-sheets [--debug] [--config <file>] compare-acl --acl <url> --report-range <range> [--with-pin] [--workdir <dir>] [--credentials <file>]```
```
  --url           Google Sheets worksheet URL from which to retrieve the ACL and to which
                  to upload the report
                  e.g. https://docs.google.com/spreadsheets/d/1iSZzHlrXsl3-mipIq0uuEqDNlPWGdamSPJrPe9OBD0k
  --range         Worksheet range of the ACL (e.g. ACL!A2:K)
  --report-range  Worksheet range (e.g. Audit!A1:D) for the compare report. Defaults to 
                  Audit!A1:D
  --with-pin      Includes the card keypad PIN code when comparing records
  --workdir       Directory for working files, in particular the tokens, revisions, etc, 
                  that provide access to Google Sheets. Defaults to:
                  - /var/uhppoted on Linux
                  - /usr/local/var/com.github.uhppoted on MacOS
                  - ./uhppoted on Microsoft Windows
  --credentials   Path for the Google Docs credentials file. 
                  Defaults to <workdir>/sheets/.google/credentials.json

  --config        File path to the uhppoted.conf file containing the access controller 
                  configuration information. Defaults to:
                  - /etc/uhppoted/uhppoted.conf (Linux)
                  - /usr/local/etc/com.github.uhppoted/uhppoted.conf (MacOS)
                  - ./uhppoted.conf (Windows)

  --debug         Displays verbose debugging information, in particular the 
                  communications with the UHPPOTE controllers

```
```
