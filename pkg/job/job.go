package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
	"github.com/infrawatch/ci-server-go/pkg/parser"
)

// Job type the main server runs
type Job interface {
	Run(ctx context.Context)
	SetLogger(*logging.Logger)
}

// Factory generate jobs based on event type
func Factory(event ghclient.Event, client *ghclient.Client) (Job, error) {
	switch e := event.(type) {
	case *ghclient.Push:
		l, err := logging.NewLogger(logging.ERROR, "console")
		if err != nil {
			return nil, err
		}
		return &PushJob{
			event:  e,
			client: client,
			Log:    l,
		}, nil
	}
	return nil, fmt.Errorf("could not determine github event type")
}

// PushJob contains logic for dealing with github push events
type PushJob struct {
	event             *ghclient.Push
	client            *ghclient.Client
	scriptOutput      []byte
	afterScriptOutput []byte

	BasePath string
	Log      *logging.Logger
}

// SetLogger impelements type job.Job
func (p *PushJob) SetLogger(l *logging.Logger) {
	p.Log = l
}

// Run ...
func (p *PushJob) Run(ctx context.Context) {
	commit := p.event.Ref.GetHead()
	cj := newCoreJob(p.client, p.event.Repo, *commit, p.Log)
	cj.BasePath = "/tmp/"

	cj.getResources()
	cj.runScript(ctx)
	cj.runAfterScript(ctx)
	cj.postResults()
}

// coreJob contains processes for the stages of running a script in a repository, generating and posting reports
// getResources() must be run before both runScript() and runAfterScript()
type coreJob struct {
	client *ghclient.Client
	repo   ghclient.Repository
	commit ghclient.Commit

	spec              *parser.Spec
	scriptOutput      []byte
	afterScriptOutput []byte

	Log *logging.Logger

	BasePath string
}

func newCoreJob(client *ghclient.Client, repo ghclient.Repository, commit ghclient.Commit, logger *logging.Logger) *coreJob {
	return &coreJob{
		client: client,
		repo:   repo,
		commit: commit,
		Log:    logger,
	}
}

func (cj *coreJob) getResources() {
	tree, err := cj.client.GetTree(cj.commit.Sha, cj.repo)
	if err != nil {
		cj.Log.Error(err.Error())
		return
	}

	err = ghclient.WriteTreeToDirectory(tree, cj.BasePath)
	if err != nil {
		cj.Log.Error(err.Error())
		return
	}
	cj.BasePath = cj.BasePath + tree.Path

	// Read in spec
	f, err := os.Open(cj.yamlPath(tree))
	defer f.Close()
	if err != nil {
		cj.Log.Error(err.Error())
		return
	}
	cj.spec, err = parser.NewSpecFromYAML(f)
	if err != nil {
		cj.Log.Error(err.Error())
		return
	}
}

func (cj *coreJob) setCommitPending() {
	cj.commit.SetContext("ci-server-go")
	cj.commit.SetStatus(ghclient.PENDING, "pending", "")
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		cj.Log.Error(err.Error())
	}
}

func (cj *coreJob) runScript(ctx context.Context) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	var err error

	// run script with timeout
	cj.scriptOutput, err = cj.spec.ScriptCmd(ctxTimeout, cj.BasePath).Output()
	if ctxTimeout.Err() != nil {
		cj.Log.Error(ctxTimeout.Err().Error())
		cj.commit.SetStatus(ghclient.ERROR, "main script timed out", "")
		return ctxTimeout.Err()
	}

	if err != nil {
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		cj.Log.Error(err.Error())
		return err
	}
	return nil
}

func (cj *coreJob) runAfterScript(ctx context.Context) {
	var err error
	cj.afterScriptOutput, err = cj.spec.AfterScriptCmd(ctx, cj.BasePath).Output()
	if err != nil {
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("after_script failed with: %s", err), "")
		cj.Log.Error(err.Error())
	}
}

func (cj *coreJob) postResults() {
	//post gist
	report := string(cj.buildReport())
	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", cj.repo.Name, cj.commit.Sha)
	gist.AddFile(fmt.Sprintf("%s_%s.md", cj.repo.Name, cj.commit.Sha), report)
	gJSON, err := json.Marshal(gist)
	if err != nil {
		cj.Log.Error(err.Error())
	}

	res, err := cj.client.Api.PostGists(gJSON)
	if err != nil {
		cj.Log.Error(err.Error())
	}

	// get gist target url
	resMap := make(map[string]interface{})
	err = json.Unmarshal(res, &resMap)
	if err != nil {
		cj.Log.Error(fmt.Sprintf("while unmarshalling gist response: %s", err))
	}

	targetURL := resMap["url"].(string)

	// update status of commit
	cj.commit.SetStatus(ghclient.SUCCESS, "all jobs passed", targetURL)
	err = cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		cj.Log.Error(err.Error())
	}
}

func (cj *coreJob) buildReport() []byte {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```")
	sb.Write(cj.scriptOutput)
	sb.WriteString("```\n## After Script Results\n```\n")
	sb.Write(cj.afterScriptOutput)
	sb.WriteString("```")
	return []byte(sb.String())
}

// helper functions
func (cj *coreJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{cj.BasePath, "ci.yml"}, "/")
}
