package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/pleimer/ci-server-go/pkg/assert"
	"github.com/pleimer/ci-server-go/pkg/job"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

type TestJob struct {
	Status job.Status
	Ref    string
	Repo   string

	cancel context.CancelFunc
}

func (tj *TestJob) GetRefName() string {
	return tj.Ref
}

func (tj *TestJob) GetRepoName() string {
	return tj.Repo
}

func (tj *TestJob) Setup(ctx context.Context, authUsers []string) {}

func (tj *TestJob) Run(ctx context.Context) {
	tj.Status = job.RUNNING
	<-ctx.Done()
	if ctx.Err() == context.Canceled {
		tj.Status = job.CANCELED
		return
	}
	tj.Status = job.COMPLETE
}

func (tj *TestJob) Compare(q queue.Item) int {
	return 0
}

func (tj *TestJob) SetLogger(l *logging.Logger) {

}

func TestJobManager(t *testing.T) {
	l, err := logging.NewLogger(logging.DEBUG, "console")
	assert.Ok(t, err)

	t.Run("interfering jobs", func(t *testing.T) {

		jmUT := NewJobManager(1, l)

		var wg sync.WaitGroup
		jobChan := make(chan job.Job)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		wg.Add(1)
		go jmUT.Run(ctx, &wg, jobChan, nil)

		original := &TestJob{
			Ref:  "refs/head/master",
			Repo: "github.com/haha/exa",
		}
		jobChan <- original
		jobChan <- &TestJob{
			Ref:  "refs/head/master",
			Repo: "github.com/haha/exa",
		}
		wg.Wait()
		assert.Equals(t, job.CANCELED, original.Status)
	})

	t.Run("more jobs than workers", func(t *testing.T) {

		jmUT := NewJobManager(2, l)
		jmUT.jobTime = time.Millisecond * 4

		var wg sync.WaitGroup
		jobChan := make(chan job.Job)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		wg.Add(1)
		go jmUT.Run(ctx, &wg, jobChan, nil)

		a := &TestJob{
			Ref:  "refs/head/master",
			Repo: "github.com/haha/exa",
		}
		b := &TestJob{
			Ref:  "refs/head/bee",
			Repo: "github.com/haha/exa",
		}
		c := &TestJob{
			Ref:  "refs/head/cee",
			Repo: "github.com/haha/exa",
		}
		jobChan <- a
		jobChan <- b
		jobChan <- c
		wg.Wait()
		assert.Equals(t, job.COMPLETE, a.Status)
		assert.Equals(t, job.COMPLETE, b.Status)
		assert.Equals(t, job.COMPLETE, c.Status)
	})
}
