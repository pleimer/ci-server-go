package ghclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

//API generates github URLS
type API struct {
	baseURL string
}

// StatusURL  url for get/post statuses
func (a *API) StatusURL(owner, repo, sha string) string {
	return a.makeURL([]string{"repos", owner, repo, "statuses", sha})
}

// AuthURL authorization URL
func (a *API) AuthURL() string {
	return a.baseURL
}

func (a *API) makeURL(items []string) string {
	var sb strings.Builder
	sb.WriteString(a.baseURL)
	for _, item := range items {
		sb.WriteString("/")
		sb.WriteString(item)
	}
	return sb.String()
}

// NewAPI creates new api type
func NewAPI() API {
	return API{
		baseURL: "https://api.github.com",
	}
}

// Client github client object
type Client struct {
	EventChan chan Event
	ErrorChan chan error

	oauth        string
	api          API
	client       *http.Client
	repositories map[string]*Repository
}

// NewClient create a new github Client
func NewClient() Client {
	return Client{
		api:          NewAPI(),
		client:       &http.Client{},
		repositories: make(map[string]*Repository),
		EventChan:    make(chan Event, 10),
		ErrorChan:    make(chan error, 10),
	}
}

// Authenticate set and test authentication with Oauth token
func (c *Client) Authenticate(oauth string) error {
	c.oauth = oauth
	res, err := c.get(c.api.AuthURL())
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("Failed to authenticate. Received status %s", res.Status)
	}
	return nil
}

// UpdateStatus update status of commit in remote repository
func (c *Client) UpdateStatus(repo Repository, commit Commit) error {
	body, err := json.Marshal(commit.Status)
	if err != nil {
		return err
	}
	res, err := c.post(c.api.StatusURL(repo.Owner.Login, repo.Name, commit.Sha), body)

	if err != nil {
		return err
	}
	if res.StatusCode != 201 {
		return fmt.Errorf("Failed to update github status. Received status %s", res.Status)
	}
	return nil
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
	c.EventChan <- ev
}

// Helper functions

func (c *Client) get(URL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.oauth)
	return c.client.Do(req)
}

func (c *Client) post(URL string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.oauth)
	return c.client.Do(req)
}
