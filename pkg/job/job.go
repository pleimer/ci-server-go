package job

import (
	"context"
	"fmt"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// Job type contains sequence of actions for different scenarios
type Job interface {
	GetRefName() string
	Run(context.Context)
	Cancel()
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
