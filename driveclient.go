package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

var client_secret []byte = []byte(`{"installed":{"client_id":"384278056379-s5bl01mk5oa738r2s1ttmsbkptbrnbra.apps.googleusercontent.com","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://accounts.google.com/o/oauth2/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"LoMMwPvFwNEXL0G7-FT9b94h","redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]}}`)

func getToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	usr, _ := user.Current()
	cache_directory := filepath.Join(usr.HomeDir, ".cache", "drive_client")
	os.MkdirAll(cache_directory, 0700)

	path := filepath.Join(cache_directory, "token.json")

	tokenfile, err := os.Open(path)
	if err != nil {
		tk, err := createNewToken(cfg)

		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(tk)
		// Cache only, doesn't need to succeed
		ioutil.WriteFile(path, buf.Bytes(), 0600)

		return tk, err
	}
	defer tokenfile.Close()

	tk := new(oauth2.Token)
	err = json.NewDecoder(tokenfile).Decode(tk)

	return tk, err
}

func createNewToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	url := cfg.AuthCodeURL("mystatetoken", oauth2.AccessTypeOffline)

	fmt.Println("Please paste the token obtained from", url, "here:")

	var code string
	_, err := fmt.Scan(&code)
	if err != nil {
		return nil, err
	}

	token, err := cfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func createClient(cfg *oauth2.Config) (*http.Client, error) {
	tok, err := getToken(cfg)

	if err != nil {
		return nil, err
	}
	return cfg.Client(context.Background(), tok), nil
}

func getDriveClient() (*drive.Service, error) {

	config, err := google.ConfigFromJSON(client_secret, drive.DriveScope)
	if err != nil {
		return nil, err
	}
	client, err := createClient(config)

	if err != nil {
		return nil, err
	}

	driveclient, err := drive.New(client)
	if err != nil {
		return nil, err
	}

	return driveclient, nil
}
