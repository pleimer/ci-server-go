package ghclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/infrawatch/ci-server-go/pkg/assert"
)

// RoundTripFunc mock http Transport
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestAuthenticate(t *testing.T) {
	oauth := strings.NewReader("oauthstring")
	t.Run("valid request", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			expectedHeader := make(http.Header)
			expectedHeader.Set("Authorization", "token oauthstring")

			assert.Equals(t, "https://api.github.com", req.URL.String())
			assert.Equals(t, expectedHeader, req.Header)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
				Header:     make(http.Header),
			}
		})

		api := NewAPI()
		api.client = client
		err := api.Authenticate(oauth)
		assert.Ok(t, err)
	})

	t.Run("invalid request", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 403,
				Status:     "403 Unauthorized",
				Body:       ioutil.NopCloser(bytes.NewBufferString(`Unauthorized`)),
				Header:     make(http.Header),
			}
		})

		api := NewAPI()
		api.client = client

		err := api.Authenticate(oauth)
		assert.Assert(t, (err != nil), "Should have been an error")
	})
}

func TestUpdateStatus(t *testing.T) {

	status := Status{
		State:       "success",
		TargetURL:   "example.url",
		Description: "description",
		Context:     "context",
	}

	commit := Commit{
		Sha:    "sha",
		Status: status,
	}

	repo := Repository{
		Name: "example",
		Owner: struct {
			Login string `json:"login"`
		}{
			Login: "owner",
		},
	}

	statusCmpObj, _ := json.Marshal(status)

	client := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equals(t, "https://api.github.com/repos/owner/example/statuses/sha", req.URL.String())
		assert.Equals(t, "POST", req.Method)
		assert.Equals(t, ioutil.NopCloser(bytes.NewReader(statusCmpObj)), req.Body)

		return &http.Response{
			StatusCode: 201,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
	})

	api := NewAPI()
	api.client = client

	gh := Client{
		api: api,
	}

	err := gh.UpdateStatus(repo, commit)
	assert.Ok(t, err)
}

func TestListen(t *testing.T) {
	eventChan := make(chan Event, 10)
	errChan := make(chan error, 10)
	gh := NewClient(eventChan, errChan)

	go gh.Listen(":8888")
	req, _ := http.NewRequest("POST", "http://127.0.0.1:8888/webhook", strings.NewReader(`{payload:"payload"}`))

	time.Sleep(time.Second)
	req.Header.Set("X-Github-Event", "push")

	srv := http.Client{}
	_, err := srv.Do(req)
	assert.Ok(t, err)

	select {
	case err := <-gh.ErrorChan:
		assert.Ok(t, err)
	case <-gh.EventChan:
	default:
		t.Errorf("Did not receive event or error")
	}
}
