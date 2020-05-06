package ghclient

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

//API generates github URLS
type API struct {
	Client  *http.Client
	BaseURL string
	oauth   string
}

// NewAPI creates new api type
func NewAPI() API {
	return API{
		Client:  &http.Client{},
		BaseURL: "https://api.github.com",
	}
}

// Authenticate set and test authentication with Oauth token
func (a *API) Authenticate(oauth io.Reader) error {
	obytes, err := ioutil.ReadAll(oauth)
	if err != nil {
		return fmt.Errorf("reading oauth token failed: %s", err)
	}
	a.oauth = string(obytes)
	res, err := a.get(a.BaseURL)
	if err != nil {
		return err
	}
	return assertCode(res, 200, "failed to authenticate")
}

// PostStatus sends post request to status
func (a *API) PostStatus(owner, repo, commitSha string, body []byte) error {
	res, err := a.post(a.StatusURL(owner, repo, commitSha), body)
	if err != nil {
		return err
	}
	return assertCode(res, 201, "failed to update github status")
}

// PostGists sends post request to status
func (a *API) PostGists(body []byte) ([]byte, error) {
	res, err := a.post(a.GistURL(), body)
	if err != nil {
		return nil, err
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("while reading response to gist POST: %s", err)
	}

	err = assertCode(res, 201, "failed to post gist to github")
	if err != nil {
		return nil, err
	}

	return info, nil
}

// GetTree retrieve github tree
func (a *API) GetTree(owner, repo, sha string) ([]byte, error) {
	res, err := a.get(a.TreeURL(owner, repo, sha))
	if err != nil {
		return nil, err
	}

	if err := assertCode(res, 200, "failed to retrieve github tree"); err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

// GetBlob retrieve github tree
func (a *API) GetBlob(owner, repo, sha string) ([]byte, error) {
	res, err := a.get(a.BlobURL(owner, repo, sha))
	if err != nil {
		return nil, err
	}

	if err := assertCode(res, 200, "failed to retrieve github blob"); err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func (a *API) get(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+a.oauth)
	return a.Client.Do(req)
}

func (a *API) post(URL string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+a.oauth)
	return a.Client.Do(req)
}

// StatusURL generates url for querying status
func (a *API) StatusURL(owner, repo, sha string) string {
	return a.makeURL([]string{"repos", owner, repo, "statuses", sha})
}

func (a *API) TreeURL(owner, repo, treeSha string) string {
	return a.makeURL([]string{"repos", owner, repo, "git", "trees", treeSha})
}

func (a *API) BlobURL(owner, repo, fileSha string) string {
	return a.makeURL([]string{"repos", owner, repo, "git", "blobs", fileSha})
}

func (a *API) GistURL() string {
	return a.makeURL([]string{"gists"})
}

func (a *API) makeURL(items []string, params ...string) string {
	var sb strings.Builder
	sb.WriteString(a.BaseURL)
	for _, item := range items {
		sb.WriteString("/")
		sb.WriteString(item)
	}

	if len(params) > 0 {
		sb.WriteString("?")
		joiner := ""
		for _, p := range params {
			sb.WriteString(joiner)
			joiner = "&"
			sb.WriteString(p)
		}
	}
	return sb.String()
}

// API object helper functions
func assertCode(res *http.Response, status int, premsg string) error {
	defer res.Body.Close()

	if res.StatusCode != status {
		return fmt.Errorf(premsg+". Received status: %s", res.Status)
	}
	return nil
}
