package ghclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

//API generates github URLS
type API struct {
	baseURL string
	client  *http.Client
	oauth   string
}

// NewAPI creates new api type
func NewAPI() API {
	return API{
		client:  &http.Client{},
		baseURL: "https://api.github.com",
	}
}

// Authenticate set and test authentication with Oauth token
func (a *API) Authenticate(oauth io.Reader) error {
	obytes, err := ioutil.ReadAll(oauth)
	if err != nil {
		return fmt.Errorf("reading oauth token failed: %s", err)
	}
	a.oauth = string(obytes)
	res, err := a.get(a.baseURL)
	if err != nil {
		return err
	}
	return assertCode(res, 200, "failed to authenticate")
}

// PostStatus sends post request to status
func (a *API) PostStatus(owner, repo, commitSha string, body []byte) error {
	res, err := a.post(a.statusURL(owner, repo, commitSha), body)
	if err != nil {
		return err
	}

	return assertCode(res, 201, "failed to update github status")
}

func (a *API) get(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+a.oauth)
	return a.client.Do(req)
}

func (a *API) post(URL string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+a.oauth)
	return a.client.Do(req)
}

func (a *API) statusURL(owner, repo, sha string) string {
	return a.makeURL([]string{"repos", owner, repo, "statuses", sha})
}

func (a *API) treeURL(owner, repo, treeSha string) string {
	return a.makeURL([]string{"repos", owner, repo, "git", "trees", treeSha}, "recursive=1")
}

func (a *API) blobURL(owner, repo, fileSha string) string {
	return a.makeURL([]string{"repos", owner, repo, "git", "blobs", fileSha})
}

func (a *API) makeURL(items []string, params ...string) string {
	var sb strings.Builder
	sb.WriteString(a.baseURL)
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
		return fmt.Errorf(premsg+" Received status %s", res.Status)
	}
	return nil
}

// Client github client object
type Client struct {
	EventChan chan Event
	ErrorChan chan error

	api          API
	cache        Cache
	repositories map[string]*Repository
}

// NewClient create a new github Client
func NewClient(eventChan chan Event, errorChan chan error) Client {
	return Client{
		api:          NewAPI(),
		repositories: make(map[string]*Repository),
		cache:        NewCache(),
		EventChan:    eventChan,
		ErrorChan:    errorChan,
	}
}

// UpdateStatus update status of commit in repository
func (c *Client) UpdateStatus(repo Repository, commit Commit) error {
	body, err := json.Marshal(commit.Status)
	if err != nil {
		return err
	}
	return c.api.PostStatus(repo.Owner.Login, repo.Name, commit.Sha, body)
}

// Listen listen on address for webhooks
func (c *Client) Listen(address string) error {
	http.HandleFunc("/webhook", c.webhookHandler)
	return http.ListenAndServe(address, nil)
}

func (c *Client) webhookHandler(w http.ResponseWriter, req *http.Request) {
	ev, err := EventFactory(req.Header.Get("X-Github-Event"))
	if err != nil {
		c.ErrorChan <- err
		return
	}

	json, err := ioutil.ReadAll(req.Body)
	if err != nil {
		c.ErrorChan <- fmt.Errorf("Error in event payload: %s", err)
	}
	ev.handle(c, json)
	c.EventChan <- ev
}

// Helper functions
