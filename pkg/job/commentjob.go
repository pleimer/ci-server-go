package job

import (
	"context"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

// CommentJob works on a comment webhook event.
type CommentJob struct {
	//This job runs a regular core job when it descovers the _trigger keyword in a comment message.
	// Any trailing character sequence after the _trigger keyword will be treated as a commit sha
	// on which to run the new core job sequence. This sha must be contained in the current branch
	// of the PR request on which the comment is made.

	// A message containing only the _trigger keyword, or a subsequent character sequence that does
	// not match any of the commits contained within the branch, will trigger a core job sequence
	// on the commit at the head of the branch

}

//SetLogger implements Job interface
func (cj *CommentJob) SetLogger(logger *logging.Logger) {
	return
}

//Run implements Job interface
func (cj *CommentJob) Run(ctx context.Context) {
	return
}

//Compare implements Job interface
func (cj *CommentJob) Compare(queue.Item) int {
	return 0
}

//GetRefName implements Job interface
func (cj *CommentJob) GetRefName() string {
	return ""
}

//GetRepoName implements Job interface
func (cj *CommentJob) GetRepoName() string {
	return ""
}
