## v0.6.x

## IN PROGRESS

- [x] get-acl
- [x] Make Sheets/TSV column order less fixed
- [ ] load-acl
      - Pad report with 1 extra row and prune remaining rows
      - Report duplicates as card errors/ignored rather than failing the whole load (--strict (?))
      - Clean up reporting code
      - Use named ranges (?)
      - get-version/is-updated/somesuch (https://stackoverflow.com/questions/18321050/google-docs-spreadsheet-get-revision-id)
      - use version that has been stable for e.g. 30 minutes
      - --force
- [ ] Rename 'release' target to 'build-all' throughout
- [ ] Move --debug flag before command
- [ ] Move --conf flag before command
- [ ] compare-acl
- [ ] store-acl
- [ ] Update the DEFAULT values - they all refer to twystd :-(
- [ ] Remove 'run' argument from uhppoted-api:command.Parse
- [ ] Clean up reporting code
      - Move templates to report/diff
      - ReportSummary.String() (?)
      - Update CLI, etc to use report.Summarize and report.Consolidate
- [ ] GMail API notifications (?)

## TODO

- [ ] TLA+ model
