package job

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/logging"
	"github.com/pleimer/ci-server-go/pkg/report"
)

// PushJob contains logic for dealing with github push events
type PushJob struct {
	event             *ghclient.Push
	client            *ghclient.Client
	scriptOutput      []byte
	afterScriptOutput []byte
	user              string

	BasePath string
	Log      *logging.Logger
	Status   Status
}

// SetLogger impelements type job.Job
func (p *PushJob) SetLogger(l *logging.Logger) {
	p.Log = l
}

// GetRefName get reference name from event that triggered job
func (p *PushJob) GetRefName() string {
	return p.event.RefName
}

// GetRepoName get reference name from event that triggered job
func (p *PushJob) GetRepoName() string {
	return p.event.Repo.Name
}

// Compare implements queue.Item
func (p *PushJob) Compare(other queue.Item) int {
	return 0
}

// Run ...
func (p *PushJob) Run(ctx context.Context) {
	commit := p.event.Ref.GetHead()
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Debug(fmt.Sprintf("running push job for %s", commit.Sha))

	if commit == nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": "commit does not exist"})
		p.Log.Error("retrieving resources")
		return
	}
	cj := newCoreJob(p.client, p.event.Repo, *commit)
	cj.BasePath = "/tmp"

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("downloading git tree")
	err := cj.GetTree()
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("retrieving resources")
		return
	}

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("loading test specifications")
	err = cj.LoadSpec(p.GetRefName())
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("failed to load spec")
		return
	}

	// initialize writers
	logPath := filepath.Join(cj.BasePath, fmt.Sprintf("%s.log", commit.Sha))
	f, err := os.Create(logPath)
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("opening log path")
		return
	}
	defer f.Close()

	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", cj.repo.Name, cj.commit.Sha)

	gw, err := ghclient.NewGistWriter(&p.client.Api, gist, fmt.Sprintf("%s_%s.md", cj.repo.Name, cj.commit.Sha))
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("creating gist writer")
		return
	}
	writer := report.NewWriter(f, gw)

	// run scripts
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("running main script")
	err = cj.RunMainScript(ctx, writer, gw.GetServerGistID())
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Info("script failed")
	} else {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
		p.Log.Info("main script completed successfully")
	}
	if err := cj.postCommitStatus(); err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err.Error()})
		p.Log.Error("posting commit status")
	}

	// It is highly NOT recommended to create top level contexts in lower functions
	// 'After script' is responsible for cleaning up resources, so it must run even when a cancel signal
	// has been sent by the main server goroutine. This still garauntees an exit after timeout
	// so it isn't too terrible
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("running after script")
	err = cj.RunAfterScript(context.Background(), writer, gw.GetServerGistID())
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Info("after_script failed")
	} else {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
		p.Log.Info("after_script completed successfully")
	}
	if err := cj.postCommitStatus(); err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err.Error()})
		p.Log.Error("posting commit status")
	}
}
