package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/infrawatch/ci-server-go/pkg/job"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// JobManager manages set number of parallel running jobs and queues up extra
// incoming jobs
type JobManager struct {
	runningJobs map[string]job.Job
	jobQueue    *queue.PriorityQueue
	log         *logging.Logger
	numWorkers  int
}

// NewJobManager job manager factory
func NewJobManager(numWorkers int, log *logging.Logger) *JobManager {
	return &JobManager{
		runningJobs: make(map[string]job.Job),
		jobQueue:    queue.NewPriorityQueue(0),
		log:         log,
		numWorkers:  numWorkers,
	}
}

// Run main job manager process
func (jb *JobManager) Run(ctx context.Context, wg *sync.WaitGroup, jobChan <-chan job.Job) {
	defer wg.Done()

	workChan := make(chan job.Job)
	for w := 0; w < jb.numWorkers; w++ {
		wg.Add(1)
		go jb.worker(ctx, wg, w, workChan)
	}

	select {
	case j := <-jobChan:
		if runningJob, ok := jb.runningJobs[j.GetRefName()]; ok {
			runningJob.Cancel()
		}
		workChan <- j
	}
}

func (jb *JobManager) worker(ctx context.Context, wg *sync.WaitGroup, num int, job <-chan job.Job) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		jb.log.Info(fmt.Sprintf("worker #%s exited", num))
		return
	case j := <-job:
		jb.runningJobs[j.GetRefName()] = j
		j.Run(ctx)
		delete(jb.runningJobs, j.GetRefName())
	}
}
