package job

import (
	"context"
	"testing"

	"github.com/pleimer/ci-server-go/pkg/ghclient"
)

// TestPushJob simulate push event coming from github, test running the job
func TestPushJob(t *testing.T) {
	t.Run("regular run", func(t *testing.T) {
		_, github, repo, ref, _, log, _ := genTestEnvironment([]string{"echo $OCP_PROJECT"}, []string{"echo Done"})
		path := "/tmp"
		deleteFiles(path)

		ev := ghclient.Push{
			Repo:    *repo,
			RefName: "refs/head/master",
			Ref:     *ref,
		}

		pj := PushJob{
			event:    &ev,
			client:   github,
			BasePath: path,
			Log:      log,
		}

		pj.Run(context.Background(), nil)
		// expGistStr := formatGistOutput(repo.Name, commit.Sha, "t0", "Done")
		// assert.Equals(t, expGistStr, gistString)
	})
}

// 	t.Run("script fail", func(t *testing.T) {
// 		_, github, repo, ref, commit, log, _ := genTestEnvironment([]string{"./ci.sh"}, []string{"echo Done"})
// 		path := "/tmp"
// 		deleteFiles(path)

// 		ev := ghclient.Push{
// 			Repo:    *repo,
// 			RefName: "refs/head/master",
// 			Ref:     *ref,
// 		}

// 		pj := PushJob{
// 			event:    &ev,
// 			client:   github,
// 			BasePath: path,
// 			Log:      log,
// 		}

// 		pj.Run(context.Background())
// 		expGistStr := formatGistOutput(repo.Name, commit.Sha, "\nerror(exit status 1) ", "Done")
// 		assert.Equals(t, expGistStr, gistString)
// 	})

// 	t.Run("send cancel signal", func(t *testing.T) {
// 		// in this case, the after script should still run to cleanup resources
// 		path := "/tmp"
// 		deleteFiles(path)

// 		_, client, repo, ref, commit, log, _ := genTestEnvironment([]string{"sleep 2", "echo Script Done"}, []string{"echo Done"})
// 		ev := ghclient.Push{
// 			Repo:    *repo,
// 			RefName: "refs/head/master",
// 			Ref:     *ref,
// 		}
// 		pj := PushJob{
// 			event:    &ev,
// 			client:   client,
// 			BasePath: path,
// 			Log:      log,
// 		}

// 		ctx, cancel := context.WithCancel(context.Background())
// 		cancel()
// 		pj.Run(ctx)
// 		expGistStr := formatGistOutput(repo.Name, commit.Sha, "error: context canceled", "Done")
// 		assert.Equals(t, expGistStr, gistString)
// 	})
// }
