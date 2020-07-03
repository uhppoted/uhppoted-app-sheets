## v0.6.x

## IN PROGRESS

- [x] get-acl
- [x] Make Sheets/TSV column order less fixed
- [ ] load-acl
      - add --revision to options (or rather --workdir or ???)
      - lockfile
      - Report duplicates as card errors/ignored rather than failing the whole load (--strict (?))
      - Use named ranges (?)
      - Add version/modified to the report (named ranges only)
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
