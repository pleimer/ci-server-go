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

}

// Run ...
func (p *PushJob) Run(ctx context.Context) {
	commit := p.event.Ref.GetHead()
	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	p.Log.Info(fmt.Sprintf("running push job for %s", commit.Sha))

	if commit == nil {
		p.Log.Metadata(map[string]interface{}{"process": "PushJob", "error": "commit does not exist"})
		p.Log.Error("retrieving resources")
		return
	}

	//TODO - make automatic running of job configureable
	// if sliceContainsString(authUsers, p.event.User) {
	// 	p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	// 	p.Log.Info(fmt.Sprintf("authorized user '%s' received - proceeding with core job", p.event.User))
	// 	RunCoreJob(ctx, p.client, p.event.Repo, p.GetRefName(), *commit, p.Log)
	// 	return
	// }

	// p.Log.Metadata(map[string]interface{}{"process": "PushJob"})
	// p.Log.Info(fmt.Sprintf("user '%s' not authorized to run jobs - ignoring", p.event.User))
}
