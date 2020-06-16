module github.com/uhppoted/uhppoted-app-sheets

go 1.14

require (
	github.com/uhppoted/uhppote-core v0.6.3
	github.com/uhppoted/uhppoted-api v0.6.3
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	google.golang.org/api v0.26.0
)

replace (
	github.com/uhppoted/uhppote-core => github.com/uhppoted/uhppote-core v0.6.4-0.20200616052722-1d3ab43ea21e
	github.com/uhppoted/uhppoted-api => github.com/uhppoted/uhppoted-api v0.6.4-0.20200616053657-974bf27cd459
)
