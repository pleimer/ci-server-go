package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/parser"
)

// coreJob contains processes for the stages of running a script in a repository, generating and posting reports
// getResources() must be run before both runScript() and runAfterScript()
type coreJob struct {
	client *ghclient.Client
	repo   ghclient.Repository
	commit ghclient.Commit

	spec              *parser.Spec
	scriptOutput      []byte
	afterScriptOutput []byte

	BasePath string
}

func newCoreJob(client *ghclient.Client, repo ghclient.Repository, commit ghclient.Commit) *coreJob {
	cj := coreJob{
		client: client,
		repo:   repo,
		commit: commit,
	}
	cj.commit.SetContext("ci-server-go")
	return &cj
}

func (cj *coreJob) getTree() error {
	tree, err := cj.client.GetTree(cj.commit.Sha, cj.repo)
	if err != nil {
		return err
	}

	err = ghclient.WriteTreeToDirectory(tree, cj.BasePath)
	if err != nil {
		return err
	}
	return nil
}

func (cj *coreJob) loadSpec() error {
	tree, err := cj.client.GetTree(cj.commit.Sha, cj.repo)
	if err != nil {
		return err
	}
	cj.BasePath = cj.BasePath + tree.Path

	f, err := os.Open(cj.yamlPath(tree))
	defer f.Close()
	if err != nil {
		return err
	}
	cj.spec, err = parser.NewSpecFromYAML(f)
	cj.spec.SetMetaVar("__commit__", cj.commit.Sha)
	if err != nil {
		return err
	}
	return nil
}

// runs spec.Script
func (cj *coreJob) runScript(ctx context.Context) error {
	var err error

	cj.commit.SetStatus(ghclient.PENDING, "pending", "")
	cj.postCommitStatus()
	cj.commit.SetStatus(ghclient.SUCCESS, "all jobs passed", "")

	// run script with timeout
	cj.scriptOutput, err = cj.spec.ScriptCmd(ctx, cj.BasePath).Output()
	if ctx.Err() != nil {
		if ctx.Err() == context.Canceled {
			cj.commit.SetStatus(ghclient.ERROR, "script canceled", "")
		}
		if ctx.Err() == context.DeadlineExceeded {
			cj.commit.SetStatus(ghclient.FAILURE, "main script timed out", "")
		}
		cj.scriptOutput = []byte(fmt.Sprintf("%serror: %s\n", cj.scriptOutput, err))
		return ctx.Err()
	}

	if err != nil {
		cj.commit.SetStatus(ghclient.FAILURE, fmt.Sprintf("script failed: %s", err), "")
		cj.scriptOutput = []byte(fmt.Sprintf("%s\nerror(%s) %s\n", cj.scriptOutput, err, err.(*exec.ExitError).Stderr))
		return err
	}
	return nil
}

// runs spec.AfterScript
func (cj *coreJob) runAfterScript(ctx context.Context) error {
	var err error
	cj.afterScriptOutput, err = cj.spec.AfterScriptCmd(ctx, cj.BasePath).Output()

	if ctx.Err() != nil {
		if ctx.Err() == context.Canceled {
			cj.commit.SetStatus(ghclient.ERROR, "after_script canceled", "")
		}
		if ctx.Err() == context.DeadlineExceeded {
			cj.commit.SetStatus(ghclient.FAILURE, "after_script timed out", "")
		}
		cj.afterScriptOutput = []byte(fmt.Sprintf("%s\nerror: %s\n", cj.afterScriptOutput, err))
		return ctx.Err()
	}

	if err != nil {
		cj.commit.SetStatus(ghclient.FAILURE, fmt.Sprintf("after_script failed: %s", err), "")
		cj.afterScriptOutput = []byte(fmt.Sprintf("%s\nerror(%s) %s\n", cj.afterScriptOutput, err, err.(*exec.ExitError).Stderr))
		return err
	}
	return nil
}

// post commit status and gist report to github client
func (cj *coreJob) postResults() error {
	//post gist
	report := string(cj.buildReport())
	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", cj.repo.Name, cj.commit.Sha)
	gist.AddFile(fmt.Sprintf("%s_%s.md", cj.repo.Name, cj.commit.Sha), report)
	gJSON, err := json.Marshal(gist)
	if err != nil {
		return err
	}

	res, err := cj.client.Api.PostGists(gJSON)
	if err != nil {
		return err
	}

	// get gist target url
	resMap := make(map[string]interface{})
	err = json.Unmarshal(res, &resMap)
	if err != nil {
		return err
	}

	if _, ok := resMap["id"]; !ok {
		return fmt.Errorf("failed to retrieve gist ID from github api response")
	}

	targetURL := cj.getGistPublishedURL(resMap["id"].(string))
	cj.commit.Status.TargetURL = targetURL
	err = cj.postCommitStatus()
	if err != nil {
		return err
	}
	return nil
}

// ----------- helper functions ---------------
func (cj *coreJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{cj.BasePath, "ci.yml"}, "/")
}

// build report in markdown format
func (cj *coreJob) buildReport() []byte {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```\n")
	sb.Write(cj.scriptOutput)
	if len(cj.scriptOutput) == 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("```\n## After Script Results\n```\n")
	sb.Write(cj.afterScriptOutput)
	if len(cj.afterScriptOutput) == 0 {
		sb.WriteString("\n")
	}
	sb.WriteString("```")
	return []byte(sb.String())
}

// helper function post commit status to gh client
func (cj *coreJob) getGistPublishedURL(id string) string {
	return fmt.Sprintf("https://gist.github.com/%s/%s", cj.repo.Owner.Login, id)
}

func (cj *coreJob) postCommitStatus() error {
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		return err
	}
	return nil
}
