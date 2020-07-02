module github.com/uhppoted/uhppoted-app-sheets

go 1.14

require (
	github.com/uhppoted/uhppote-core v0.6.3
	github.com/uhppoted/uhppoted-api v0.6.3
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200331124033-c3d80250170d
	google.golang.org/api v0.26.0
)

replace (
	github.com/uhppoted/uhppote-core => ../uhppote-core
	github.com/uhppoted/uhppoted-api => ../uhppoted-api
)
