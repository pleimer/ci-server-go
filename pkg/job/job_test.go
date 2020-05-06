package job

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infrawatch/ci-server-go/pkg/assert"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/parser"
	"gopkg.in/yaml.v2"
)

// TestPushJob simulate push event coming from github, test running the job
func TestPushJob(t *testing.T) {
	spec := &parser.Spec{
		Global: &parser.Global{
			Timeout: 300,
			Env: map[string]interface{}{
				"OCP_PROJECT": "stf",
			},
		},
		Script: []string{
			"echo $OCP_PROJECT",
			"pwd",
		},
		AfterScript: []string{
			"cat ci.yml",
		},
	}
	content, _ := yaml.Marshal(spec)

	t0 := &ghclient.Tree{
		Sha:  "t0",
		Path: "t0",
	}

	b1 := &ghclient.Blob{
		Sha:     "b1",
		Content: base64.StdEncoding.EncodeToString(content),
		Path:    "t0/ci.yml",
	}
	t0.SetChild(b1)

	repo := &ghclient.Repository{
		Name: "example",
		Owner: struct {
			Login string `json:"login"`
		}{
			Login: "owner",
		},
	}

	serverRespBodies, err := ghclient.TreeServer(t0, repo)
	assert.Ok(t, err)
	client := ghclient.NewTestClient(func(req *http.Request) *http.Response {
		respFn := serverRespBodies[req.URL.String()]

		if respFn == nil {
			return &http.Response{
				StatusCode: 404,
				Status:     "404 Not Found",
				Body:       ioutil.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}
		}

		respBody, err := respFn()
		assert.Ok(t, err)
		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
			Header:     make(http.Header),
		}
	})

	api := ghclient.NewAPI()
	api.Client = client

	gh := ghclient.NewClient(nil, nil)
	gh.Api = api

	// simulated push event
	commit := ghclient.Commit{
		Sha: "t0",
	}
	ref := ghclient.Reference{}
	ref.Register(&commit)

	ev := ghclient.Push{
		Repo: *repo,
		Ref:  ref,
	}

	// Create PushJob
	path := "/tmp/"
	pj := PushJob{
		event:    &ev,
		client:   &gh,
		basePath: path,
	}

	if _, err := os.Stat("/tmp/t0"); err == nil {
		removeContents("/tmp/t0")
		os.RemoveAll("/tmp/t0")
	}

	errChan := make(chan error)
	go pj.Run(errChan)

	assert.Ok(t, <-errChan)
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
