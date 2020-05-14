package job

import (
	"context"
	"fmt"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

//Status indicates current status of job
type Status int

// job status types
const (
	RUNNING Status = iota
	COMPLETE
	CANCELED
	TIMEDOUT
)

func (js Status) String() string {
	return [...]string{"COMPLETE", "CANCELED", "TIMEDOUT"}[js]
}

// Job type contains sequence of actions for different scenarios
type Job interface {
	GetRefName() string
	Run(context.Context)
	Cancel()
	Compare(queue.Item) int
	SetLogger(*logging.Logger)
}

// Factory generate jobs based on event type
func Factory(event ghclient.Event, client *ghclient.Client, log *logging.Logger) (Job, error) {
	switch e := event.(type) {
	case *ghclient.Push:
		return &PushJob{
			event:  e,
			client: client,
			Log:    log,
		}, nil
	}
	return nil, fmt.Errorf("failed creating job: could not determine github event type")
}
