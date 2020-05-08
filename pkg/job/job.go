package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// Job type contains sequence of actions for different scenarios
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
