package job

import (
	"encoding/base64"
	"fmt"
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

	serverResponses, err := ghclient.TreeServer(t0, repo)
	assert.Ok(t, err)
	client := ghclient.NewTestClient(func(req *http.Request) *http.Response {
		respFn := serverResponses[req.URL.String()]
		// t.Log(req.URL.String())

		if respFn == nil {
			return &http.Response{
				StatusCode: 404,
				Status:     fmt.Sprintf("404 Not Found. Location %s does not exist", req.URL.String()),
				Body:       ioutil.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}
		}

		resp, err := respFn(req)
		assert.Ok(t, err)
		return resp
	})

	api := ghclient.NewAPI()
	api.Client = client

	gh := ghclient.NewClient(nil, nil)
	gh.Api = api

	// simulated push event
	commit := ghclient.Commit{
		Sha: "t0",
	}

	// add server response to status Post requests
	serverResponses[api.StatusURL(repo.Owner.Login, repo.Name, commit.Sha)] = func(req *http.Request) (*http.Response, error) {
		// res, _ := ioutil.ReadAll(req.Body)
		// t.Log(string(res))
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(req.URL.String())),
			Header:     make(http.Header),
		}, nil
	}

	// add server response to gist Post
	serverResponses[api.GistURL()] = func(req *http.Request) (*http.Response, error) {
		body := `{"url":"` + api.GistURL() + `/testgist"}`
		res, _ := ioutil.ReadAll(req.Body)
		t.Log(string(res))
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	ref := ghclient.Reference{}
	ref.Register(&commit)
	gh.Cache.WriteCommits(&commit)

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
