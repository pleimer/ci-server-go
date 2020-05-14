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
// incoming jobs.
type JobManager struct {
	// once a job begins running, it is subject to cancellation on the following events:
	// 1) if a job targeted at the same github Ref arrives (in this case, it is replaced)
	// 2) if the parent context cancel function is called
	runningJobs map[string]job.Job
	jobQueue    *queue.PriorityQueue
	log         *logging.Logger
	numWorkers  int
}

// NewJobManager job manager factory
func NewJobManager(numWorkers int, log *logging.Logger) *JobManager {
	return &JobManager{
		runningJobs: make(map[string]job.Job),
		jobQueue:    queue.NewPriorityQueue(1000),
		log:         log,
		numWorkers:  numWorkers,
	}
}

// Run main job manager process
func (jb *JobManager) Run(ctx context.Context, wg *sync.WaitGroup, jobChan <-chan job.Job) {
	defer wg.Done()

	workChan := make(chan job.Job)
	for w := 0; w < jb.numWorkers; w++ {
		jb.log.Info(fmt.Sprintf("created worker #%d", w))
		wg.Add(1)
		go jb.worker(ctx, wg, w, workChan)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			items, err := jb.jobQueue.Get(1)
			if err != nil {
				jb.log.Info("job queue disposed") //only one type of error can happen here
				return
			}
			job, ok := items[0].(job.Job)
			if !ok {
				jb.log.Error(fmt.Sprintf("while retrieving job from job queue: got data of type %T but wanted type job.Job", job))
				continue
			}
			workChan <- job
		}
	}()

	select {
	case j := <-jobChan:
		if runningJob, ok := jb.runningJobs[j.GetRefName()]; ok {
			runningJob.Cancel()
		}
		jb.jobQueue.Put(j)
	case <-ctx.Done():
		jb.jobQueue.Dispose()
		jb.log.Info("job manager exited")
		return
	}
}

func (jb *JobManager) worker(ctx context.Context, wg *sync.WaitGroup, num int, job <-chan job.Job) {
	// worker will not immediately exit when context is cancelled until the job completes its cancel sequence
	// this is to enforce worker number limits
	defer wg.Done()
	select {
	case <-ctx.Done():
		jb.log.Info(fmt.Sprintf("worker #%d exited", num))
		return
	case j := <-job:
		jb.runningJobs[j.GetRefName()] = j
		j.Run(ctx)
		delete(jb.runningJobs, j.GetRefName())
	}
}
