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
		api.Client = client
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
		api.Client = client

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
	api.Client = client

	gh := Client{
		Api: api,
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

func TestGetTree(t *testing.T) {
	// Test inputs
	t.Run("no cache hits", func(t *testing.T) {

		t0Input := TreeMarshal{
			Sha: "t0",
			Tree: []ChildRef{
				{
					Sha:  "b1",
					Type: "blob",
					Path: "t0/b1",
				},
				{
					Sha:  "t1",
					Type: "tree",
					Path: "t0/t1",
				},
			},
		}

		t1Input := TreeMarshal{
			Sha: "t1",
			Tree: []ChildRef{
				{
					Sha:  "b2",
					Type: "blob",
					Path: "t0/t1/b2",
				},
				{
					Sha:  "b22",
					Type: "blob",
					Path: "t0/t1/b22",
				},
			},
		}

		// Tree to compare to
		t0 := &Tree{
			Sha:  "t0",
			Path: "t0",
		}

		b1 := &Blob{
			Sha:     "b1",
			Content: "dGhpcyBpcyBteSBsaWZl",
			Path:    "t0/b1",
		}
		t0.SetChild(b1)

		t1 := &Tree{
			Sha:  "t1",
			Path: "t0/t1",
		}

		b2 := &Blob{
			Sha:     "b2",
			Content: "dGhpcyBpcyBteSBsaWZl",
			Path:    "t0/t1/b2",
		}
		t1.SetChild(b2)

		b22 := &Blob{
			Sha:     "b22",
			Content: "dGhpcyBpcyBteSBsaWZl",
			Path:    "t0/t1/b22",
		}
		t1.SetChild(b22)
		t0.SetChild(t1)

		client := NewTestClient(func(req *http.Request) *http.Response {
			var respBody []byte
			var err error

			switch req.URL.String() {
			case "https://api.github.com/repos/owner/example/git/trees/t0":
				respBody, err = json.Marshal(t0Input)
				assert.Ok(t, err)

			case "https://api.github.com/repos/owner/example/git/blobs/b1":
				respBody, err = json.Marshal(b1)
				assert.Ok(t, err)
			case "https://api.github.com/repos/owner/example/git/trees/t1":
				respBody, err = json.Marshal(t1Input)
				assert.Ok(t, err)
			case "https://api.github.com/repos/owner/example/git/blobs/b2":
				respBody, err = json.Marshal(b2)
				assert.Ok(t, err)
			case "https://api.github.com/repos/owner/example/git/blobs/b22":
				respBody, err = json.Marshal(b22)
				assert.Ok(t, err)
			default:
				return &http.Response{
					StatusCode: 404,
					Status:     "404 Not Found",
					Body:       ioutil.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}
			}
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
				Header:     make(http.Header),
			}
		})

		repo := Repository{
			Name: "example",
			Owner: struct {
				Login string `json:"login"`
			}{
				Login: "owner",
			},
		}

		api := NewAPI()
		api.Client = client

		gh := Client{
			Api:   api,
			cache: NewCache(),
		}

		tree, err := gh.GetTree("t0", repo)
		assert.Ok(t, err)
		assert.Equals(t, t0, tree)

		err = WriteTreeToDirectory(tree, "")
		assert.Ok(t, err)
	})

	t.Run("cache hits", func(t *testing.T) {
		t0Input := TreeMarshal{
			Sha: "t0",
		}

		var respBody []byte
		var err error

		misses := 0
		client := NewTestClient(func(req *http.Request) *http.Response {
			assert.Assert(t, (misses <= 1), "two api calls when there only should have been one")
			respBody, err = json.Marshal(t0Input)
			assert.Ok(t, err)
			misses++

			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
				Header:     make(http.Header),
			}
		})

		repo := Repository{
			Name: "example",
			Owner: struct {
				Login string `json:"login"`
			}{
				Login: "owner",
			},
		}

		api := NewAPI()
		api.Client = client

		gh := Client{
			Api:   api,
			cache: NewCache(),
		}

		_, err = gh.GetTree("t0", repo)
		assert.Ok(t, err)
		_, err = gh.GetTree("t0", repo)
		assert.Ok(t, err)
	})
}
