package job

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

// CommentJob works on a comment webhook event.
type CommentJob struct {
	// This job runs a regular core job when it descovers the _trigger keyword in a comment message.
	// Any trailing character sequence after the _trigger keyword will be treated as a commit sha
	// on which to run the new core job sequence. This sha must be contained in the current branch
	// of the PR request on which the comment is made.

	// A message containing only the _trigger keyword, or a subsequent character sequence that does
	// not match any of the commits contained within the branch, will trigger a core job sequence
	// on the commit at the head of the branch

	Log    *logging.Logger
	client *ghclient.Client
	event  *ghclient.Comment

	execute bool
}

//SetLogger implements Job interface
func (cj *CommentJob) SetLogger(l *logging.Logger) {
	cj.Log = l
}

// Setup posts a comment about queing the job
func (cj *CommentJob) Setup(ctx context.Context, authUsers []string) {
	commit := cj.event.Ref.GetHead()
	commit.SetContext("ci-server-go")

	cj.Log.Metadata(map[string]interface{}{"process": "CommentJob"})
	cj.Log.Info(fmt.Sprintf("received comment: %s", cj.event.Body))

	//parse message body
	tokens := strings.Split(cj.event.Body, " ")
	if sliceContainsString(tokens, "/runtest") {
		if sliceContainsString(authUsers, cj.event.User) {
			cj.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
			cj.Log.Info(fmt.Sprintf("authorized user '%s' requested '/runtest' for commit '%s' in repository '%s', ref '%s'",
				cj.event.User, commit.Sha, cj.event.Repo.Name, cj.event.RefName))

			commit.SetStatus(ghclient.PENDING, "queued", "")
			err := cj.client.UpdateCommitStatus(cj.event.Repo, *commit)
			if err != nil {
				cj.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup", "error": err.Error()})
				cj.Log.Error("failed to update commit status to 'queued'")
			}
			cj.execute = true
			return
		}
		cj.Log.Metadata(map[string]interface{}{"process": "CommentJob", "stage": "setup"})
		cj.Log.Info(fmt.Sprintf("user '%s' not authorized to run jobs, ignoring", cj.event.User))
	}
}

//Run implements Job interface
func (cj *CommentJob) Run(ctx context.Context) {
	commit := cj.event.Ref.GetHead()
	if cj.execute {
		RunCoreJob(ctx, cj.client, cj.event.Repo, cj.GetRefName(), *commit, cj.Log)
	}
}

//Compare implements queue.Item
func (cj *CommentJob) Compare(queue.Item) int {
	return 0
}

//GetRefName implements Job interface
func (cj *CommentJob) GetRefName() string {
	return cj.event.RefName
}

//GetRepoName implements Job interface
func (cj *CommentJob) GetRepoName() string {
	return ""
}
