package job

import (
	"context"
	"fmt"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/logging"
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
	p.Status = RUNNING
	commit := p.event.Ref.GetHead()
	cj := newCoreJob(p.client, p.event.Repo, *commit)
	cj.BasePath = "/tmp"

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("downloading git tree")
	err := cj.getTree()
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("failed to get resources")
		err = cj.postResults(p.user)
		if err != nil {
			p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
			p.Log.Error("failed to post results")
		}
		return
	}

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("loading test specifications")
	err = cj.loadSpec()
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Error("failed to load spec")
		return
	}

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("running main script")
	err = cj.runScript(ctx)
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Info("script failed")
	} else {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
		p.Log.Info("main script completed successfully")
	}
	p.handleContextError(err)

	// It is highly NOT recommended to create top level contexts in lower functions
	// 'After script' is responsible for cleaning up resources, so it must run even when a cancel signal
	// has been sent by the main server goroutine. This still garauntees an exit after timeout
	// so it isn't too terrible
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info("running after script")
	err = cj.runAfterScript(context.Background())
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Info("after_script failed")
	} else {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
		p.Log.Info("after_script completed successfully")
	}
	p.handleContextError(err)

	err = cj.postResults(p.user)
	if err != nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": err})
		p.Log.Info("failed to post results")
	} else {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
		repoName := cj.repo.Name
		p.Log.Info(fmt.Sprintf("posted '%s' status to '%s|%s|%s'", cj.commit.Status.State, repoName, p.event.RefName, commit.Sha))
	}
	p.Status = COMPLETE
}

func (p *PushJob) handleContextError(err error) {
	switch err {
	case context.DeadlineExceeded:
		p.Status = TIMEDOUT
	case context.Canceled:
		p.Status = CANCELED
	}
}
