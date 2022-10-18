package commands

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/uhppoted/uhppoted-app-sheets/commands/html"
)

var AuthoriseCmd = Authorise{
	workdir:     DEFAULT_WORKDIR,
	credentials: filepath.Join(DEFAULT_WORKDIR, ".google", "credentials.json"),
	url:         "",
	debug:       false,
}

type Authorise struct {
	workdir     string
	credentials string
	url         string
	debug       bool
}

func (cmd *Authorise) Name() string {
	return "authorise"
}

func (cmd *Authorise) Description() string {
	return "Authorises uhppoted-app-sheets to access a Google Sheets worksheet"
}

func (cmd *Authorise) Usage() string {
	return "--credentials <file> --url <url>"
}

func (cmd *Authorise) Help() {
	fmt.Println()
	fmt.Printf("  Usage: %s [--debug] authorise [options] --url <URL>\n", APP)
	fmt.Println()
	fmt.Println("  Authorises uhppoted-app-sheets to access a Google Sheets worksheet")
	fmt.Println()

	helpOptions(cmd.FlagSet())

	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println(`    uhppote-app-sheets authorise --credentials "credentials.json" --url "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"`)
	fmt.Println()
}

func (cmd *Authorise) FlagSet() *flag.FlagSet {
	flagset := flag.NewFlagSet("get", flag.ExitOnError)

	flagset.StringVar(&cmd.workdir, "workdir", cmd.workdir, "Directory for working files (tokens, revisions, etc)'")
	flagset.StringVar(&cmd.credentials, "credentials", cmd.credentials, "Path for the 'credentials.json' file")
	flagset.StringVar(&cmd.url, "url", cmd.url, "Spreadsheet URL")

	return flagset
}

func (cmd *Authorise) Execute(args ...any) error {
	options := args[0].(*Options)

	cmd.debug = options.Debug

	// ... check parameters
	if strings.TrimSpace(cmd.credentials) == "" {
		return fmt.Errorf("--credentials is a required option")
	}

	if strings.TrimSpace(cmd.url) == "" {
		return fmt.Errorf("--url is a required option")
	}

	match := regexp.MustCompile(`^https://docs.google.com/spreadsheets/d/(.*?)(?:/.*)?$`).FindStringSubmatch(cmd.url)
	if len(match) < 2 {
		return fmt.Errorf("Invalid spreadsheet URL - expected something like 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms'")
	}

	if err := authenticate(cmd.credentials, SHEETS, cmd.workdir); err != nil {
		return fmt.Errorf("Authorisation error (%v)", err)
	}

	return fmt.Errorf("NOT IMPLEMENTED")
}

func authenticate(credentials, scope, workdir string) (err error) {
	// ... get OAuth2 configuration
	var config *oauth2.Config
	var b []byte

	if b, err = ioutil.ReadFile(credentials); err != nil {
		return
	} else if config, err = google.ConfigFromJSON(b, scope); err != nil {
		return
	}

	// ... get tokens file
	_, file := filepath.Split(credentials)
	name := strings.TrimSuffix(file, filepath.Ext(file))
	tokens := ""

	switch {
	case strings.HasPrefix(scope, SHEETS):
		tokens = filepath.Join(workdir, fmt.Sprintf("%s.sheets", name))

	case strings.HasPrefix(scope, DRIVE):
		tokens = filepath.Join(workdir, fmt.Sprintf("%s.drive", name))

	default:
		tokens = filepath.Join(workdir, fmt.Sprintf("%s.tokens", name))
	}

	// ... start HTTP server on localhost

	fs := filesystem{
		FileSystem: http.FS(html.HTML),
	}

	authorised := make(chan string)
	mux := http.NewServeMux()
	sheets := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	mux.Handle("/css/", http.FileServer(fs))
	mux.Handle("/images/", http.FileServer(fs))
	mux.Handle("/fonts/", http.FileServer(fs))
	mux.Handle("/manifest.json", http.FileServer(fs))
	mux.Handle("/favicon.ico", http.FileServer(fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, rq *http.Request) {
		fmt.Printf("RQ:  %+v\n", rq.URL)

		state := rq.FormValue("state")
		code := rq.FormValue("code")
		scope := rq.FormValue("scope")

		fmt.Printf("state: %v\n", state)
		fmt.Printf("code:  %v\n", code)
		fmt.Printf("scope: %v\n", scope)

		if state == "state-token" && code != "" && (scope == SHEETS || scope == DRIVE) {
			authorised <- code
		}
	})

	mux.HandleFunc("/auth.html", func(w http.ResponseWriter, rq *http.Request) {
		page := map[string]any{
			"sheets": sheets,
			"drive":  "<coming soon>",
		}

		t, err := template.New("auth.html").ParseFS(html.HTML, "auth.html")
		if err != nil {
			http.Error(w, "Internal error formatting page", http.StatusInternalServerError)
			return
		}

		var b bytes.Buffer
		if err := t.Execute(&b, page); err != nil {
			http.Error(w, "Error formatting page", http.StatusInternalServerError)
			return
		}

		w.Write(b.Bytes())
	})

	srv := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			panic(fmt.Sprintf("ERROR: %v", err))
		}
	}()

	// ... CTRL-C handler
	interrupt := make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt)

	// ... open OAuth2 URL in browser
	command := exec.Command("open", "http://localhost/auth.html")
	if _, err := command.CombinedOutput(); err != nil {
		fmt.Println("Could not open authorisation page in your browser - please open http://localhost manually")
	}

	// ... wait for authorisation
	select {
	case <-interrupt:
		fmt.Printf("\n.. cancelled\n\n")

	case code := <-authorised:
		if token, err := config.Exchange(context.TODO(), code); err != nil {
			panic(fmt.Sprintf("Unable to retrieve token from web: %v", err))
		} else {
			saveToken(tokens, token)
		}
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		warn(fmt.Sprintf("%v", err))
	}

	return nil
}
