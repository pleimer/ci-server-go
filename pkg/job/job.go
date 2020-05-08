package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
	"github.com/infrawatch/ci-server-go/pkg/parser"
)

// Job type the main server runs
type Job interface {
	Run(context.Context, *sync.WaitGroup)
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
func (p *PushJob) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	commit := p.event.Ref.GetHead()
	cj := newCoreJob(p.client, p.event.Repo, *commit, p.Log)
	cj.BasePath = "/tmp/"

	cj.getResources()
	ctxTimeoutScript, cancelScript := context.WithTimeout(ctx, time.Second*300)
	defer cancelScript()
	cj.runScript(ctxTimeoutScript)

	// It is highly NOT recommended to create top level contexts in lower functions
	// 'After script' is responsible for cleaning up resources, so it must run even when a cancel signal
	// has been sent by the main server goroutine. This still garauntees an exit after timeout
	// so it isn't too terrible
	ctxTimeoutAfterScrip, cancelAfterScript := context.WithTimeout(context.Background(), time.Second)
	defer cancelAfterScript()
	cj.runAfterScript(ctxTimeoutAfterScrip)

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

func (cj *coreJob) setCommitStatus(state ghclient.CommitState, message, targetURL string) {
	cj.commit.SetContext("ci-server-go")
	cj.commit.SetStatus(state, message, targetURL)
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		cj.Log.Error(err.Error())
	}
}

func (cj *coreJob) runScript(ctx context.Context) error {
	var err error

	// run script with timeout
	cj.scriptOutput, err = cj.spec.ScriptCmd(ctx, cj.BasePath).Output()
	if ctx.Err() != nil {
		cj.Log.Error(ctx.Err().Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("main script failed: %s", ctx.Err().Error()), "")
		return ctx.Err()
	}

	if err != nil {
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		cj.Log.Error(err.Error())
		return err
	}
	return nil
}

func (cj *coreJob) runAfterScript(ctx context.Context) error {
	var err error
	cj.afterScriptOutput, err = cj.spec.AfterScriptCmd(ctx, cj.BasePath).Output()

	if ctx.Err() != nil {
		cj.Log.Error(ctx.Err().Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("after_script failed: %s", ctx.Err().Error()), "")
		return ctx.Err()
	}

	if err != nil {
		cj.Log.Error(err.Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		return err
	}
	return nil
}

func (cj *coreJob) postResults() string {
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

	return resMap["url"].(string)
}

func (cj *coreJob) buildReport() []byte {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```")
	sb.Write(cj.scriptOutput)
	if len(cj.scriptOutput) == 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("```\n## After Script Results\n```\n")
	sb.Write(cj.afterScriptOutput)
	if len(cj.afterScriptOutput) == 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("```")
	return []byte(sb.String())
}

// helper functions
func (cj *coreJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{cj.BasePath, "ci.yml"}, "/")
}
