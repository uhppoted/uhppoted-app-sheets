## v0.6.x

## IN PROGRESS

- [x] get
- [x] Make Sheets/TSV column order less fixed
- [x] load-acl
      - change report format to be more log-like 
      - add duplicate cards to returned/logged errors
      - check that compare is the right way around i.e. comparing source and destination correctly
- [x] compare-acl
- [x] upload-acl
- [x] put 
- [x] update other ACL parsers to ignore duplicates
- [x] Rename 'release' target to 'build-all' throughout
- [ ] SystemDiff consolidation for e.g. added + updated ?
- [ ] Move --debug flag before command
- [ ] Move --conf flag before command
- [ ] Remove 'run' argument from uhppoted-api:command.Parse
- [ ] Clean up reporting code
      - Move templates to report/diff
      - ReportSummary.String() (?)
      - Update CLI, etc to use report.Summarize and report.Consolidate

## TODO

- [ ] TLA+ model
- [ ] Templates
      - Named ranges
      - Spreadsheet version/modified fields
