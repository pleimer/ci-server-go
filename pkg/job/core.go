package job

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
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

	Log *logging.Logger

	BasePath string
}

func newCoreJob(client *ghclient.Client, repo ghclient.Repository, commit ghclient.Commit, logger *logging.Logger) *coreJob {
	cj := coreJob{
		client: client,
		repo:   repo,
		commit: commit,
		Log:    logger,
	}
	cj.commit.SetContext("ci-server-go")
	return &cj
}

// gather resources needed to run any of the core functions
// specifically, download the tree and gather the spec from
// the yaml file
func (cj *coreJob) getResources() error {
	tree, err := cj.client.GetTree(cj.commit.Sha, cj.repo)
	if err != nil {
		cj.Log.Info(err.Error())
		return err
	}

	err = ghclient.WriteTreeToDirectory(tree, cj.BasePath)
	if err != nil {
		cj.Log.Info(err.Error())
		return err
	}
	cj.BasePath = cj.BasePath + tree.Path

	// Read in spec
	f, err := os.Open(cj.yamlPath(tree))
	defer f.Close()
	if err != nil {
		cj.Log.Info(err.Error())
		return err
	}
	cj.spec, err = parser.NewSpecFromYAML(f)
	if err != nil {
		cj.Log.Info(err.Error())
		return err
	}
	return nil
}

// runs spec.Script
func (cj *coreJob) runScript(ctx context.Context) error {
	var err error

	cj.commit.SetStatus(ghclient.PENDING, "pending", "")
	cj.postCommitStatus()

	// run script with timeout
	cj.scriptOutput, err = cj.spec.ScriptCmd(ctx, cj.BasePath).Output()
	if ctx.Err() != nil {
		cj.Log.Info(ctx.Err().Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("main script failed: %s", ctx.Err().Error()), "")
		return ctx.Err()
	}

	if err != nil {
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		cj.Log.Info(err.Error())
		return err
	}
	cj.commit.SetStatus(ghclient.SUCCESS, "all tests passed", "")
	return nil
}

// runs spec.AfterScript
func (cj *coreJob) runAfterScript(ctx context.Context) error {
	var err error
	cj.afterScriptOutput, err = cj.spec.AfterScriptCmd(ctx, cj.BasePath).Output()

	if ctx.Err() != nil {
		cj.Log.Info(ctx.Err().Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("after_script failed: %s", ctx.Err().Error()), "")
		return ctx.Err()
	}

	if err != nil {
		cj.Log.Info(err.Error())
		cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		return err
	}
	return nil
}

// post commit status and gist report to github client
func (cj *coreJob) postResults() {
	//post gist
	report := string(cj.buildReport())
	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", cj.repo.Name, cj.commit.Sha)
	gist.AddFile(fmt.Sprintf("%s_%s.md", cj.repo.Name, cj.commit.Sha), report)
	gJSON, err := json.Marshal(gist)
	if err != nil {
		cj.Log.Info(err.Error())
	}

	res, err := cj.client.Api.PostGists(gJSON)
	if err != nil {
		cj.Log.Info(err.Error())
	}

	// get gist target url
	resMap := make(map[string]interface{})
	err = json.Unmarshal(res, &resMap)
	if err != nil {
		cj.Log.Info(fmt.Sprintf("while unmarshalling gist response: %s", err))
	}

	targetURL := resMap["url"].(string)
	cj.commit.Status.TargetURL = targetURL
	cj.postCommitStatus()
}

// ----------- helper functions ---------------
func (cj *coreJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{cj.BasePath, "ci.yml"}, "/")
}

// build report in markdown format
func (cj *coreJob) buildReport() []byte {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```")
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
func (cj *coreJob) postCommitStatus() error {
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		cj.Log.Info(err.Error())
		return err
	}
	return nil
}
