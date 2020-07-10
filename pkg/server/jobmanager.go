package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/pleimer/ci-server-go/pkg/job"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

// job wrapper that pairs cancel function with running job
type jobContext struct {
	job    job.Job
	cancel context.CancelFunc
}

type tracker struct {
	jobContexts cmap.ConcurrentMap
}

func (t *tracker) Get(repo string, ref string) (*jobContext, bool) {
	ret, ok := t.jobContexts.Get(fmt.Sprintf("%s.%s", repo, ref))
	if ret == nil {
		return nil, ok
	}
	return ret.(*jobContext), ok
}

func (t *tracker) Set(repo string, ref string, jobContext *jobContext) {
	t.jobContexts.Set(fmt.Sprintf("%s.%s", repo, ref), jobContext)
}

func (t *tracker) Remove(repo string, ref string) {
	t.jobContexts.Remove(fmt.Sprintf("%s.%s", repo, ref))
}

// JobManager manages set number of parallel running jobs and queues up extra
// incoming jobs.
type JobManager struct {
	// once a job begins running, it is subject to cancellation on the following events:
	// 1) if a job targeted at the same github Ref arrives (in this case, it is replaced)
	// 2) if the parent context cancel function is called
	runningJobs tracker
	jobQueue    chan job.Job
	log         *logging.Logger
	numWorkers  int
	jobTime     time.Duration
}

// NewJobManager job manager factory
func NewJobManager(numWorkers int, log *logging.Logger) *JobManager {
	return &JobManager{
		runningJobs: tracker{jobContexts: cmap.New()},
		jobQueue:    make(chan job.Job, 100),
		log:         log,
		numWorkers:  numWorkers,
	}
}

// Run main job manager process
func (jb *JobManager) Run(ctx context.Context, wg *sync.WaitGroup, jobChan <-chan job.Job, authUsers []string) {
	defer wg.Done()

	workChan := make(chan func())
	for w := 0; w < jb.numWorkers; w++ {
		jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
		jb.log.Debug(fmt.Sprintf("created worker #%d", w))
		wg.Add(1)
		go jb.worker(ctx, wg, w, workChan)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case j, ok := <-jb.jobQueue:
				if !ok {
					jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
					jb.log.Debug("job queue disposed")
					return
				}

				jCtx, jCancel := context.WithCancel(ctx)
				jb.runningJobs.Set(j.GetRepoName(), j.GetRefName(), &jobContext{
					job:    j,
					cancel: jCancel,
				})
				workChan <- func() {
					j.Run(jCtx, authUsers)
					jb.runningJobs.Remove(j.GetRepoName(), j.GetRefName())
				}
			}
		}
	}()

	for {
		select {
		case j := <-jobChan:
			if runningJob, ok := jb.runningJobs.Get(j.GetRepoName(), j.GetRefName()); ok {
				runningJob.cancel()
				jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
				jb.log.Info(fmt.Sprintf("conflicting job for repository %s, ref %s - cancelled running job", j.GetRepoName(), j.GetRefName()))
			}
			jb.jobQueue <- j
		case <-ctx.Done():
			close(jb.jobQueue)
			jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
			jb.log.Info("exited")
			return
		}
	}
}

func (jb *JobManager) worker(ctx context.Context, wg *sync.WaitGroup, num int, workerChan <-chan func()) {
	// worker will not immediately exit when context is cancelled until the job completes its cancel sequence
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
			jb.log.Debug(fmt.Sprintf("worker #%d exited", num))
			return
		case j := <-workerChan:
			jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
			jb.log.Info(fmt.Sprintf("worker #%d running job", num))
			j()
			jb.log.Metadata(map[string]interface{}{"process": "JobManager"})
			jb.log.Info(fmt.Sprintf("worker #%d completed job", num))
		}
	}
}
