module github.com/uhppoted/uhppoted-app-sheets

go 1.14

require (
	github.com/uhppoted/uhppote-core v0.6.2
	github.com/uhppoted/uhppoted-api v0.6.2
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.26.0
)

replace (
	github.com/uhppoted/uhppote-core => ../uhppote-core
	github.com/uhppoted/uhppoted-api => ../uhppoted-api
)
