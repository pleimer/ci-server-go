package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
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
