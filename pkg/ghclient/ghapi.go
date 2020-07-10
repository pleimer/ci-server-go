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

	err GithubClientError
}

// NewAPI creates new api type
func NewAPI() API {
	return API{
		Client:  &http.Client{},
		BaseURL: "https://api.github.com",
		err: GithubClientError{
			module: "API",
		},
	}
}

// Authenticate set and test authentication with Oauth token
func (a *API) Authenticate(oauth io.Reader) error {
	obytes, err := ioutil.ReadAll(oauth)
	if err != nil {
		return a.err.withMessage(fmt.Sprintf("reading oauth token failed: %s", err))
	}
	a.oauth = string(obytes)
	res, err := a.get(a.BaseURL)
	if err != nil {
		return a.err.withMessage(err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return a.err.withMessage(fmt.Sprintf("received status %s", res.Status))
	}
	return nil
}

// PostStatus sends post request to status
func (a *API) PostStatus(owner, repo, commitSha string, body []byte) error {
	res, err := a.post(a.StatusURL(owner, repo, commitSha), body)
	if err != nil {
		return a.err.withMessage(err.Error())
	}
	defer res.Body.Close()

	cCode := 201
	if res.StatusCode != cCode {
		return a.err.withMessage(fmt.Sprintf("expected status code %d, received %s", cCode, res.Status))
	}

	return nil
}

// PostGists sends post request to status
func (a *API) PostGists(body []byte) ([]byte, error) {
	res, err := a.post(a.NewGistURL(), body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, a.err.withMessage(fmt.Sprintf("failed reading server response: %s", err))
	}

	cCode := 201
	if res.StatusCode != cCode {
		return nil, a.err.withMessage(fmt.Sprintf("expected status code %d, received %s", cCode, res.Status))
	}
	return info, nil
}

//UpdateGist updates gist with ID
func (a *API) UpdateGist(body []byte, ID string) ([]byte, error) {
	res, err := a.post(a.UpdateGistURL(ID), body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, a.err.withMessage(fmt.Sprintf("failed reading server response: %s", err))
	}

	cCode := 200
	if res.StatusCode != cCode {
		return nil, a.err.withMessage(fmt.Sprintf("expected status code %d, received %s", cCode, res.Status))
	}
	return info, nil
}

// GetTree retrieve github tree
func (a *API) GetTree(owner, repo, sha string) ([]byte, error) {
	res, err := a.get(a.TreeURL(owner, repo, sha))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	cCode := 200
	if res.StatusCode != cCode {
		return nil, a.err.withMessage(fmt.Sprintf("expected status code %d, received %s", cCode, res.Status))
	}

	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, a.err.withMessage(fmt.Sprintf("failed reading server response: %s", err))
	}
	return info, nil
}

// GetBlob retrieve github tree
func (a *API) GetBlob(owner, repo, sha string) ([]byte, error) {
	res, err := a.get(a.BlobURL(owner, repo, sha))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	cCode := 200
	if res.StatusCode != cCode {
		return nil, a.err.withMessage(fmt.Sprintf("expected status code %d, received %s", cCode, res.Status))
	}

	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, a.err.withMessage(fmt.Sprintf("failed reading server response: %s", err))
	}
	return info, nil
}

// GetURL generic function for querying a preconcieved URL
func (a *API) GetURL(url string) ([]byte, error) {
	res, err := a.get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
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

func (a *API) patch(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
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

func (a *API) NewGistURL() string {
	return a.makeURL([]string{"gists"})
}

func (a *API) UpdateGistURL(ID string) string {
	return a.makeURL([]string{"gists", ID})
}

func (a *API) PublishedGistURL(id, user string) string {
	return fmt.Sprintf("https://gist.github.com/%s/%s", user, id)
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
