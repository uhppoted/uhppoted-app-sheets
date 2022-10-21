package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const SHEETS = "https://www.googleapis.com/auth/spreadsheets"
const DRIVE = "https://www.googleapis.com/auth/drive"

func authorize(credentials, scope, dir string) (*http.Client, error) {
	b, err := ioutil.ReadFile(credentials)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		return nil, err
	}

	_, filename := filepath.Split(credentials)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	file := ""

	switch {
	case strings.HasPrefix(scope, DRIVE):
		file = filepath.Join(dir, fmt.Sprintf("%s.drive", name))

	case strings.HasPrefix(scope, SHEETS):
		file = filepath.Join(dir, fmt.Sprintf("%s.sheets", name))

	default:
		file = filepath.Join(dir, fmt.Sprintf("%s.tokens", name))
	}

	return getClient(file, config), nil
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(tokens string, config *oauth2.Config) *http.Client {
	token, err := tokenFromFile(tokens)
	if err != nil {
		token = getTokenFromWeb(config)
		saveToken(tokens, token)
	}

	return config.Client(context.Background(), token)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)

	err := os.MkdirAll(filepath.Dir(path), 0770)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}

	defer f.Close()

	json.NewEncoder(f).Encode(token)
}
