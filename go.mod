module github.com/uhppoted/uhppoted-app-sheets

go 1.14

require (
	github.com/uhppoted/uhppote-core v0.6.4
	github.com/uhppoted/uhppoted-api v0.6.4
	golang.org/x/net v0.0.0-20200813134508-3edf25e44fcc
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200812155832-6a926be9bd1d
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.30.0
)

replace (
	github.com/uhppoted/uhppote-core => ../uhppote-core
	github.com/uhppoted/uhppoted-api => ../uhppoted-api
)
