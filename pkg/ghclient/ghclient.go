package ghclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/infrawatch/ci-server-go/pkg/logging"
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
func NewClient(eventChan chan Event) Client {
	return Client{
		Api:          NewAPI(),
		repositories: make(map[string]*Repository),
		Cache:        NewCache(),
		EventChan:    eventChan,
	}
}

// UpdateCommitStatus update status of commit in repository
func (c *Client) UpdateCommitStatus(repo Repository, commit Commit) error {
	// update internally
	cIn := c.Cache.GetCommit(commit.Sha)
	if cIn == nil {
		return fmt.Errorf("ghclient - commit has not been indexed or previously initialized")
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
func (c *Client) Listen(wg *sync.WaitGroup, address string, log *logging.Logger) *http.Server {
	srv := &http.Server{Addr: address}
	http.HandleFunc("/webhook", func(w http.ResponseWriter, req *http.Request) {
		ev, err := EventFactory(req.Header.Get("X-Github-Event"))
		if err != nil {
			c.ErrorChan <- err
			return
		}
		json, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Error(fmt.Sprintf("error in event payload: %s", err))
		}
		ev.Handle(c, json)
		c.EventChan <- ev
	})

	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("while listening on address %s: %s", address, err))
		}
	}()
	return srv
}
