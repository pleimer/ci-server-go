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

	Api          API
	Cache        Cache
	repositories map[string]*Repository
}

// NewClient create a new github Client
func NewClient(eventChan chan Event, errorChan chan error) Client {
	return Client{
		Api:          NewAPI(),
		repositories: make(map[string]*Repository),
		Cache:        NewCache(),
		EventChan:    eventChan,
		ErrorChan:    errorChan,
	}
}

// UpdateCommitStatus update status of commit in repository
func (c *Client) UpdateCommitStatus(repo Repository, commit Commit) error {
	// update internally
	cIn := c.Cache.GetCommit(commit.Sha)
	if cIn == nil {
		return fmt.Errorf("commit has not been indexed or previously initialized")
	}

	cIn.Status = commit.Status

	// update remote
	body, err := json.Marshal(cIn.Status)
	if err != nil {
		return err
	}
	return c.Api.PostStatus(repo.Owner.Login, repo.Name, cIn.Sha, body)
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
	ev.Handle(c, json)
	c.EventChan <- ev
}
