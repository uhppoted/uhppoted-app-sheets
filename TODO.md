## v0.6.x

## IN PROGRESS

- [x] get-acl
- [x] Make Sheets/TSV column order less fixed
- [ ] load-acl
      - Make report sheet structure less hardcoded and less fragile e.g. reliant on frozen rows
      - Use named ranges (?)
      - get-version/is-updated/somesuch (https://stackoverflow.com/questions/18321050/google-docs-spreadsheet-get-revision-id)
      - use version that has been stable for e.g. 30 minutes
      - --force
      - TLA+ model
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

