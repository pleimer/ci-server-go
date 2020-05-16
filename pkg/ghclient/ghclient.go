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

	Api          API
	Cache        Cache
	repositories map[string]*Repository

	err GithubClientError
}

// NewClient create a new github Client
func NewClient(eventChan chan Event) Client {
	return Client{
		Api:          NewAPI(),
		repositories: make(map[string]*Repository),
		Cache:        NewCache(),
		EventChan:    eventChan,
		err: GithubClientError{
			module: "ghclient",
		},
	}
}

// UpdateCommitStatus update status of commit in repository
func (c *Client) UpdateCommitStatus(repo Repository, commit Commit) error {
	// update internally
	cIn := c.Cache.GetCommit(commit.Sha)
	if cIn == nil {
		return c.err.withMessage("commit has not been indexed or previously initialized")
	}

	cIn.Status = commit.Status

	// update remote
	body, err := json.Marshal(cIn.Status)
	if err != nil {
		return c.err.withMessage(fmt.Sprintf("failed to commit json: %s", err))
	}
	return c.Api.PostStatus(repo.Owner.Login, repo.Name, cIn.Sha, body)
}

// Listen listen on address for webhooks
func (c *Client) Listen(wg *sync.WaitGroup, address string, log *logging.Logger) *http.Server {
	srv := &http.Server{Addr: address}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Hello!"))
	})
	http.HandleFunc("/webhook", func(w http.ResponseWriter, req *http.Request) {
		ev, err := EventFactory(req.Header.Get("X-Github-Event"))
		if err != nil {
			log.Metadata(map[string]interface{}{"module": "ghclient", "endpoint": "/webhook", "error": err})
			log.Error("failed parsing incoming event")
			return
		}
		json, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Metadata(map[string]interface{}{"module": "ghclient", "endpoint": "/webhook"})
			log.Error(fmt.Sprintf("error in event payload: %s", err))
		}
		log.Metadata(map[string]interface{}{"module": "ghclient", "endpoint": "/webhook"})
		log.Debug(fmt.Sprintf("received payload: %s", json))
		ev.Handle(c, json)
		c.EventChan <- ev
	})

	go func() {
		defer wg.Done()
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Metadata(map[string]interface{}{"module": "ghclient", "address": address, "error": err})
			log.Error("webhook server failed")
		}
		log.Metadata(map[string]interface{}{"module": "ghclient", "info": err})
		log.Info("closed webhook server")
	}()
	return srv
}
