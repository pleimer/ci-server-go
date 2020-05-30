package job

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/parser"
	"github.com/pleimer/ci-server-go/pkg/report"
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
	cj.BasePath = filepath.Join(cj.BasePath, tree.Path)

	f, err := os.Open(filepath.Join(cj.BasePath, "ci.yml"))
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

//runs a script and writes buffered output to file and gist writer
func (cj *coreJob) runScript(ctx context.Context, script *exec.Cmd, title string) error {
	var err error

	stdout, err := script.StdoutPipe()
	if err != nil {
		return err
	}

	script.Stderr = script.Stdout //want stderr in same pipe as stdout

	if err := script.Start(); err != nil {
		return err
	}

	//reader := bufio.NewReader(stdout)
	scanner := bufio.NewScanner(stdout)
	logPath := filepath.Join(cj.BasePath, fmt.Sprintf("%s.log", cj.commit.Sha))

	f, err := os.Create(logPath)
	if err != nil {
		return err
	}

	defer f.Close()

	//TODO: add new writer for gists here
	bwFile := report.NewWriter(f)
	bwFile.AddTitle(title)
	if bwFile.Err() != nil {
		return bwFile.Err()
	}
	bwFile.OpenBlock()

	scriptDone := make(chan struct{})
	go func() {
		script.Wait()
	}()

	select {
	case <-scriptDone:
		break
	default:
		for scanner.Scan() {
			bwFile.Write(scanner.Text())
		}
	}
	err = scanner.Err()

	if ctx.Err() != nil {
		bwFile.Write(fmt.Sprintf("\nerror: %s", ctx.Err()))
		bwFile.CloseBlock()
		bwFile.Flush()
		return ctx.Err()
	}

	if err != nil {
		bwFile.Write(fmt.Sprintf("\nerror: %s", err))
		bwFile.CloseBlock()
		bwFile.Flush()
		return err
	}

	bwFile.CloseBlock()
	bwFile.Flush()
	if bwFile.Err() != nil {
		return bwFile.Err()
	}

	return nil
}

// runs spec.Script
func (cj *coreJob) RunMainScript(ctx context.Context) error {
	cj.commit.SetStatus(ghclient.PENDING, "pending", "")
	cj.postCommitStatus()
	cj.commit.SetStatus(ghclient.SUCCESS, "all jobs passed", "")

	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()

	cmd := cj.spec.ScriptCmd(scriptCtx, cj.BasePath)
	err := cj.runScript(scriptCtx, cmd, "main script")
	if err != nil {
		return err
	}
	return nil
}

// runs spec.AfterScript
func (cj *coreJob) runAfterScript(ctx context.Context) error {
	var err error

	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()

	cj.afterScriptOutput, err = cj.spec.AfterScriptCmd(scriptCtx, cj.BasePath).Output()
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
func (cj *coreJob) postResults(user string) error {
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

	targetURL := cj.getGistPublishedURL(resMap["id"].(string), user)
	cj.commit.Status.TargetURL = targetURL
	err = cj.postCommitStatus()
	if err != nil {
		return err
	}
	return nil
}

// ----------- helper functions ---------------
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

// post commit status to gh client
func (cj *coreJob) getGistPublishedURL(id, user string) string {
	return fmt.Sprintf("https://gist.github.com/%s/%s", user, id)
}

func (cj *coreJob) postCommitStatus() error {
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		return err
	}
	return nil
}
