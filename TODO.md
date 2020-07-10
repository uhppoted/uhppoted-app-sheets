## v0.6.x

## IN PROGRESS

- [x] get-acl
- [x] Make Sheets/TSV column order less fixed
- [x] load-acl
      - add duplicate cards to returned/logged errors
      - check that compare is the right way around i.e. comparing source and destination correctly
- [x] compare-acl
- [ ] upload-acl
- [ ] put-acl (? for initializing a spreadsheet from an ACL file)
- [ ] update other ACL parsers to ignore duplicates
- [ ] SystemDiff consolidation for e.g. added + updated ?
- [x] Rename 'release' target to 'build-all' throughout
- [ ] Move --debug flag before command
- [ ] Move --conf flag before command
- [ ] Update the DEFAULT values - they all refer to twystd :-(
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
