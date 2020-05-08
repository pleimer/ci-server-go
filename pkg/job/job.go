package job

import (
	"context"
	"sync"

	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// Job type contains sequence of actions for different scenarios
type Job interface {
	Run(context.Context, *sync.WaitGroup)
	SetLogger(*logging.Logger)
}
