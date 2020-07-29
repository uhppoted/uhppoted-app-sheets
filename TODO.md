## v0.6.x

## IN PROGRESS

- [x] get
- [x] Make Sheets/TSV column order less fixed
- [x] load-acl
      - replace report rows to leave formulae intact 
      - fetch all rows i.e. check response for more rows
      - check that compare is the right way around i.e. comparing source and destination correctly
      - add duplicate cards to returned/logged errors
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


