package ghclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

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
