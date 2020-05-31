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
	"time"

	"github.com/pleimer/ci-server-go/pkg/assert"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/logging"
	"github.com/pleimer/ci-server-go/pkg/parser"
	"github.com/pleimer/ci-server-go/pkg/report"
	"gopkg.in/yaml.v2"
)

var (
	// stores result of post to api gist endpoint for evaluation in tests
	gistString string
)

// TestPushJob simulate push event coming from github, test running the job
func TestPushJob(t *testing.T) {
	t.Run("regular run", func(t *testing.T) {
		_, github, repo, ref, _, log, _ := genTestEnvironment([]string{"echo $OCP_PROJECT"}, []string{"echo Done"})
		path := "/tmp"
		deleteFiles(path)

		ev := ghclient.Push{
			Repo:    *repo,
			RefName: "refs/head/master",
			Ref:     *ref,
		}

		pj := PushJob{
			event:    &ev,
			client:   github,
			BasePath: path,
			Log:      log,
		}

		pj.Run(context.Background())
		// expGistStr := formatGistOutput(repo.Name, commit.Sha, "t0", "Done")
		// assert.Equals(t, expGistStr, gistString)
	})
}

// 	t.Run("script fail", func(t *testing.T) {
// 		_, github, repo, ref, commit, log, _ := genTestEnvironment([]string{"./ci.sh"}, []string{"echo Done"})
// 		path := "/tmp"
// 		deleteFiles(path)

// 		ev := ghclient.Push{
// 			Repo:    *repo,
// 			RefName: "refs/head/master",
// 			Ref:     *ref,
// 		}

// 		pj := PushJob{
// 			event:    &ev,
// 			client:   github,
// 			BasePath: path,
// 			Log:      log,
// 		}

// 		pj.Run(context.Background())
// 		expGistStr := formatGistOutput(repo.Name, commit.Sha, "\nerror(exit status 1) ", "Done")
// 		assert.Equals(t, expGistStr, gistString)
// 	})

// 	t.Run("send cancel signal", func(t *testing.T) {
// 		// in this case, the after script should still run to cleanup resources
// 		path := "/tmp"
// 		deleteFiles(path)

// 		_, client, repo, ref, commit, log, _ := genTestEnvironment([]string{"sleep 2", "echo Script Done"}, []string{"echo Done"})
// 		ev := ghclient.Push{
// 			Repo:    *repo,
// 			RefName: "refs/head/master",
// 			Ref:     *ref,
// 		}
// 		pj := PushJob{
// 			event:    &ev,
// 			client:   client,
// 			BasePath: path,
// 			Log:      log,
// 		}

// 		ctx, cancel := context.WithCancel(context.Background())
// 		cancel()
// 		pj.Run(ctx)
// 		expGistStr := formatGistOutput(repo.Name, commit.Sha, "error: context canceled", "Done")
// 		assert.Equals(t, expGistStr, gistString)
// 	})
// }

func TestFailedScript(t *testing.T) {
	spec, github, repo, _, commit, _, _ := genTestEnvironment([]string{"asdfasdf"}, []string{""})
	var sb strings.Builder
	writer := report.NewWriter(&sb)

	cjUT := newCoreJob(github, *repo, commit)
	cjUT.spec = spec

	err := cjUT.RunMainScript(context.Background(), writer, "")
	t.Log(sb.String())
	t.Log(err.Error())

}

func TestCancel(t *testing.T) {
	spec, github, repo, _, commit, _, _ := genTestEnvironment([]string{"echo starting", "sleep 5", "echo ending"}, []string{"sleep 5"})
	var sb strings.Builder
	writer := report.NewWriter(&sb)

	cjUT := newCoreJob(github, *repo, commit)
	cjUT.spec = spec

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)

	t.Run("script timeout", func(t *testing.T) {
		err := cjUT.RunMainScript(ctx, writer, "newgist")
		assert.Equals(t, context.DeadlineExceeded, err)
	})
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
	t.Run("after script timeout", func(t *testing.T) {
		err := cjUT.RunAfterScript(ctx, writer, "newgist")
		assert.Equals(t, context.DeadlineExceeded, err)
	})
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*10)
	t.Run("script cancel", func(t *testing.T) {
		cancel()
		err := cjUT.RunMainScript(ctx, writer, "newgist")
		assert.Equals(t, context.Canceled, err)
	})
}

// test helper functions
func formatGistOutput(repoName, commitSha, scriptOutput, afterScriptOutput string) string {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```\n")
	sb.WriteString(scriptOutput)
	sb.WriteString("\n```\n## After Script Results\n```\n")
	sb.WriteString(afterScriptOutput)
	sb.WriteString("\n```")

	cGist := ghclient.Gist{
		Description: fmt.Sprintf("CI Results for repository '%s' commit '%s'", repoName, commitSha),
		Public:      true,
		Files: map[string]*ghclient.File{
			fmt.Sprintf("%s_%s.md", repoName, commitSha): {
				Content: sb.String(),
			},
		},
	}

	cGistBytes, _ := json.Marshal(cGist)
	return string(cGistBytes)

}

// Expects path in format: `/tmp/`. Looks for t0 directory
func deleteFiles(path string) {
	path = path + "t0"
	if _, err := os.Stat(path); err == nil {
		removeContents(path)
		os.RemoveAll(path)
	}
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

// function that creates fully integrated objects for job testing
func genTestEnvironment(script, afterScript []string) (*parser.Spec, *ghclient.Client, *ghclient.Repository, *ghclient.Reference, ghclient.Commit, *logging.Logger, *ghclient.Tree) {
	// default specification - a.k.a ci.yml
	spec := &parser.Spec{
		Global: &parser.Global{
			Timeout: 300,
			Env: map[string]interface{}{
				"OCP_PROJECT": "__commit__",
			},
		},
		Script:      script,
		AfterScript: afterScript,
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

	// generate tree with ci.yml in it
	content, _ := yaml.Marshal(spec)
	t0 := &ghclient.Tree{
		Sha:  "t0",
		Path: "t0",
	}

	b1 := &ghclient.Blob{
		Sha:     "b1",
		Content: base64.StdEncoding.EncodeToString(content),
		Path:    "ci.yml",
	}

	b2 := &ghclient.Blob{
		Sha:     "b2",
		Content: base64.StdEncoding.EncodeToString([]byte("exit 1")),
		Path:    "ci.sh",
	}
	t0.SetChild(b2)
	t0.SetChild(b1)

	log, err := logging.NewLogger(logging.DEBUG, "console")
	if err != nil {
		panic(err)
	}

	//var serverResponses map[string]func(*http.Request) (*http.Response, error)
	serverResponses, err := ghclient.TreeServer(t0, repo)
	if err != nil {
		panic(err)
	}
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
		if err != nil {
			panic(err)
		}
		return resp
	})

	// github client with custom http client
	api := ghclient.NewAPI()
	api.Client = client

	gh := ghclient.NewClient(nil)
	gh.Api = api

	ref := ghclient.Reference{}
	ref.Register(&commit)
	gh.Cache.WriteCommits(&commit)

	// add server response to status Post requests
	serverResponses[gh.Api.StatusURL(repo.Owner.Login, repo.Name, commit.Sha)] = func(req *http.Request) (*http.Response, error) {
		// res, _ := ioutil.ReadAll(req.Body)
		// fmt.Println(string(res))
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(req.URL.String())),
			Header:     make(http.Header),
		}, nil
	}

	serverResponses[gh.Api.NewGistURL()] = func(req *http.Request) (*http.Response, error) {
		body := `{"url":"` + gh.Api.NewGistURL() + `/testgist", "id":"newgist"}`
		res, _ := ioutil.ReadAll(req.Body)
		gistString = string(res)
		return &http.Response{
			StatusCode: 201,
			Status:     "201 Created",
			Body:       ioutil.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	serverResponses[gh.Api.UpdateGistURL("newgist")] = func(req *http.Request) (*http.Response, error) {
		body := `{"url":"` + gh.Api.NewGistURL() + `/testgist", "id":"newgist"}`
		res, _ := ioutil.ReadAll(req.Body)
		gistString = string(res)
		return &http.Response{
			StatusCode: 200,
			Status:     "200 Ok",
			Body:       ioutil.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}

	return spec, &gh, repo, &ref, commit, log, t0
}
