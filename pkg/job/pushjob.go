package job

import (
	"context"
	"fmt"
	"strings"

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
	execute           bool

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

//Setup ..
func (p *PushJob) Setup(ctx context.Context, authUsers []string) {
	commit := p.event.Ref.GetHead()
	commit.SetContext("ci-server-go")

	//parse message body

	p.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
	p.Log.Info("initializing push job sequence")
	refName := strings.Trim(p.event.RefName, "\"")
	if strings.Compare(refName, "refs/heads/master") == 0 {
		if sliceContainsString(authUsers, p.event.User) {
			p.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
			p.Log.Info(fmt.Sprintf("authorized user '%s' requested job for '%s' in '%s' master branch - proceeding",
				p.event.User, commit.Sha, p.event.Repo.Name))

			commit.SetStatus(ghclient.PENDING, "queued", "")
			err := p.client.UpdateCommitStatus(p.event.Repo, *commit)
			if err != nil {
				p.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup", "error": err.Error()})
				p.Log.Error("failed to update commit status to 'queued'")
			}
			p.execute = true
			return
		}

		p.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
		p.Log.Info(fmt.Sprintf("user '%s' not authorized to run jobs, ignoring", p.event.User))
	}
	p.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
	p.Log.Info(fmt.Sprintf("push job received for ref '%s', not 'refs/heads/master' - skipping", refName))
}

// Run ...
func (p *PushJob) Run(ctx context.Context) {
	if !p.execute {
		return
	}

	commit := p.event.Ref.GetHead()
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info(fmt.Sprintf("running push job for %s", commit.Sha))

	if commit == nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": "commit does not exist"})
		p.Log.Error("retrieving resources")
		return
	}

	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info(fmt.Sprintf("proceeding with job sequence on master branch for commit %s", commit.Sha))
	RunCoreJob(ctx, p.client, p.event.Repo, p.GetRefName(), *commit, p.Log)
}
