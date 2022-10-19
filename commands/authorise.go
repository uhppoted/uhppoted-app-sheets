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

	if err := authenticate(cmd.credentials, cmd.workdir); err != nil {
		return fmt.Errorf("Authorisation error (%v)", err)
	}

	return fmt.Errorf("NOT IMPLEMENTED")
}

func authenticate(credentials, workdir string) error {
	// ... token file
	save := func(scope string, token *oauth2.Token) bool {
		_, file := filepath.Split(credentials)
		basename := strings.TrimSuffix(file, filepath.Ext(file))

		switch scope {
		case SHEETS:
			saveToken(filepath.Join(workdir, basename+".sheets"), token)
			return true

		case DRIVE:
			saveToken(filepath.Join(workdir, basename+".drive"), token)
			return true

		default:
			return false
		}
	}

	// ... get OAuth2 configuration
	var sheets *oauth2.Config
	var drive *oauth2.Config
	var buffer []byte

	if b, err := ioutil.ReadFile(credentials); err != nil {
		return err
	} else {
		buffer = b
	}

	if config, err := google.ConfigFromJSON(buffer, SHEETS); err != nil {
		return err
	} else {
		sheets = config
	}

	if config, err := google.ConfigFromJSON(buffer, DRIVE); err != nil {
		return err
	} else {
		drive = config
	}

	// ... start HTTP server on localhost

	fs := filesystem{
		FileSystem: http.FS(html.HTML),
	}

	authorised := make(chan struct {
		scope string
		code  string
	})

	mux := http.NewServeMux()

	mux.Handle("/css/", http.FileServer(fs))
	mux.Handle("/images/", http.FileServer(fs))
	mux.Handle("/fonts/", http.FileServer(fs))
	mux.Handle("/manifest.json", http.FileServer(fs))
	mux.Handle("/favicon.ico", http.FileServer(fs))

	mux.HandleFunc("/", func(w http.ResponseWriter, rq *http.Request) {
		state := rq.FormValue("state")
		code := rq.FormValue("code")
		scope := rq.FormValue("scope")

		if state == "state-token" && code != "" && (scope == SHEETS || scope == DRIVE) {
			authorised <- struct {
				scope string
				code  string
			}{
				scope: scope,
				code:  code,
			}
		}
	})

	mux.HandleFunc("/auth.html", func(w http.ResponseWriter, rq *http.Request) {
		page := map[string]any{
			"sheets": sheets.AuthCodeURL("state-token", oauth2.AccessTypeOffline),
			"drive":  drive.AuthCodeURL("state-token", oauth2.AccessTypeOffline),
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

	// ... open auth.html URL in browser
	command := exec.Command("open", "http://localhost/auth.html")
	if _, err := command.CombinedOutput(); err != nil {
		fmt.Println("Could not open authorisation page in your browser - please open http://localhost manually")
	}

	// ... wait for authorisation

	received := map[string]bool{}

loop:
	for {
		select {
		case <-interrupt:
			fmt.Printf("\n.. cancelled\n\n")

		case auth := <-authorised:
			if token, err := sheets.Exchange(context.TODO(), auth.code); err != nil {
				panic(fmt.Sprintf("Unable to retrieve token from web: %v", err))
			} else {
				received[auth.scope] = save(auth.scope, token)
			}

			if received[SHEETS] && received[DRIVE] {
				break loop
			}
		}
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		warn(fmt.Sprintf("%v", err))
	}

	return nil
}
