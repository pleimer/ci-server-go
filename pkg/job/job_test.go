package job

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/infrawatch/ci-server-go/pkg/assert"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
	"github.com/infrawatch/ci-server-go/pkg/parser"
	"gopkg.in/yaml.v2"
)

// Helper function that creates preliminary objects for running tests with
func genTestEnvironment(client *http.Client) (*parser.Spec, *ghclient.Client, *ghclient.Repository, *ghclient.Reference, ghclient.Commit, *logging.Logger, *ghclient.Tree) {
	// default specification - a.k.a ci.yml
	spec := &parser.Spec{
		Global: &parser.Global{
			Timeout: 300,
			Env: map[string]interface{}{
				"OCP_PROJECT": "stf",
			},
		},
		Script: []string{
			"echo $OCP_PROJECT",
		},
		AfterScript: []string{
			"echo Done",
		},
	}

	// default repository
	repo := &ghclient.Repository{
		Name: "example",
		Owner: struct {
			Login string `json:"login"`
		}{
			Login: "owner",
		},
	}

	commit := ghclient.Commit{
		Sha: "t0",
	}

	// github client with custom http client
	api := ghclient.NewAPI()
	api.Client = client

	gh := ghclient.NewClient(nil, nil)
	gh.Api = api

	ref := ghclient.Reference{}
	ref.Register(&commit)
	gh.Cache.WriteCommits(&commit)

	// generate tree with ci.yml in it
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

	log, err := logging.NewLogger(logging.INFO, "console")
	if err != nil {
		panic(err)
	}

	return spec, &gh, repo, &ref, commit, log, t0
}

// TestPushJob simulate push event coming from github, test running the job
func TestPushJob(t *testing.T) {
	var serverResponses map[string]func(*http.Request) (*http.Response, error)
	httpClient := ghclient.NewTestClient(func(req *http.Request) *http.Response {
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

	_, client, repo, ref, commit, _, tree := genTestEnvironment(httpClient)

	serverResponses, err := ghclient.TreeServer(tree, repo)
	assert.Ok(t, err)

	// add server response to status Post requests
	serverResponses[client.Api.StatusURL(repo.Owner.Login, repo.Name, commit.Sha)] = func(req *http.Request) (*http.Response, error) {
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
	var gist strings.Builder
	serverResponses[client.Api.GistURL()] = func(req *http.Request) (*http.Response, error) {
		body := `{"url":"` + client.Api.GistURL() + `/testgist"}`
		res, _ := ioutil.ReadAll(req.Body)
		gist.Write(res)
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	ev := ghclient.Push{
		Repo: *repo,
		Ref:  *ref,
	}

	// Create PushJob
	path := "/tmp/"
	pj := PushJob{
		event:    &ev,
		client:   client,
		BasePath: path,
	}

	if _, err := os.Stat("/tmp/t0"); err == nil {
		removeContents("/tmp/t0")
		os.RemoveAll("/tmp/t0")
	}

	pj.Run(context.Background())

	// gist to compare reply against
	cGist := ghclient.Gist{
		Description: fmt.Sprintf("CI Results for repository '%s' commit '%s'", repo.Name, commit.Sha),
		Public:      true,
		Files: map[string]ghclient.File{
			fmt.Sprintf("%s_%s.md", repo.Name, commit.Sha): {
				Content: formatGistOutput("stf", "Done"),
			},
		},
	}

	cGistBytes, _ := json.Marshal(cGist)
	assert.Equals(t, string(cGistBytes), gist.String())
}

func formatGistOutput(scriptOutput, afterScriptOutput string) string {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```")
	sb.WriteString(scriptOutput)
	sb.WriteString("\n```\n## After Script Results\n```\n")
	sb.WriteString(afterScriptOutput)
	sb.WriteString("\n```")
	return sb.String()

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

func TestTimeoutScript(t *testing.T) {
	httpClient := ghclient.NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(req.URL.String())),
			Header:     make(http.Header),
		}
		// return &http.Response{}
	})

	spec, github, repo, _, commit, log, _ := genTestEnvironment(httpClient)

	spec.Script = []string{"sleep 10"}

	cjUT := newCoreJob(github, *repo, commit, log)
	cjUT.spec = spec

	ctx := context.Background()
	err := cjUT.runScript(ctx)
	assert.Equals(t, context.DeadlineExceeded, err)
}
