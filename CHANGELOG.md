# CHANGELOG

## Unreleased


## [0.8.4](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.8.4) - 2023-03-17

### Added
1. `doc.go` package overview documentation.
2. Added `--with-pin` option for card keypad PIN to `get`, `put`, `load-acl`, 
   `upload-acl` and `compare-acl` commands

### Updated
1. Fixed initial round of _staticcheck_ lint errors and added _staticcheck_ to CI build.
2. Correct auth flow to suggest `authorise` command on fail


## [0.8.3](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.8.3) - 2022-12-16

### Added
1. Added HOWTO for Google Sheets and Google Drive authentication.
2. Added --tokens command line argument for (optional) custom _tokens_ folder.
3. Added ARM64 to release build artifacts

### Changed
1. Reworked Google Sheets and Google Drive authentication for [OOB Migration](https://developers.google.com/identity/protocols/oauth2/resources/oob-migration).
2. Restricted Google Drive authorisation scope to `drive.metadata.readonly`.
3. Added section to READ clarifying _uhppoted.conf_ `controllers` section.
4. Migrated `git` default branch to `main`.
5. Reworked lockfile to use `flock` _syscall_.
6. Removed _zip_ files from release artifacts (no longer necessary)

## [0.8.2](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.8.2) - 2022-10-14

### Changed
1. Updated for compatibility with [uhppoted-lib](https://github.com/uhppoted/uhppoted-lib) v0.8.2

## [0.8.1](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.8.1) - 2022-08-01

### Changed
1. Updated for compatibility with [uhppoted-lib](https://github.com/uhppoted/uhppoted-lib) v0.8.1


## [0.8.0](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.8.0) - 2022-07-01

### Changed
1. Updated for compatibility with [uhppoted-lib](https://github.com/uhppoted/uhppoted-lib) v0.8.0


## [0.7.3](https://github.com/uhppoted/uhppoted-app-sheets/releases/tag/v0.7.3) - 2022-06-01

### Changed
1. Updated for compatibility with [uhppoted-lib](https://github.com/uhppoted/uhppoted-lib) v0.7.3

