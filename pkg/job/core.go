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
func (cj *coreJob) runScript(ctx context.Context, script *exec.Cmd, writer *report.Writer) error {
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

	if writer.Err() != nil {
		return writer.Err()
	}
	writer.OpenBlock()

	scriptDone := make(chan struct{})
	go func() {
		script.Wait()
	}()

	ticker := time.NewTicker(60 * time.Second)

	select {
	case <-scriptDone:
		break
	case <-ticker.C:
		writer.Flush()
	default:
		for scanner.Scan() {
			writer.Write(scanner.Text())
		}
	}
	err = scanner.Err()

	if ctx.Err() != nil {
		writer.Write(fmt.Sprintf("\nerror: %s", ctx.Err()))
		writer.CloseBlock()
		writer.Flush()
		return ctx.Err()
	}

	if err != nil {
		writer.Write(fmt.Sprintf("\nerror: %s", err))
		writer.CloseBlock()
		writer.Flush()
		return err
	}

	writer.CloseBlock()
	writer.Flush()
	if writer.Err() != nil {
		return writer.Err()
	}

	return nil
}

// runs spec.Script
func (cj *coreJob) RunMainScript(ctx context.Context, writer *report.Writer) error {
	cj.commit.SetStatus(ghclient.PENDING, "pending", "")
	cj.postCommitStatus()
	cj.commit.SetStatus(ghclient.SUCCESS, "all jobs passed", "")

	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()
	cmd := cj.spec.ScriptCmd(scriptCtx, cj.BasePath)
	writer.AddTitle("Main Script")
	err := cj.runScript(scriptCtx, cmd, writer)
	if err != nil {
		return err
	}
	return nil
}

// runs spec.AfterScript
func (cj *coreJob) RunAfterScript(ctx context.Context, writer *report.Writer) error {
	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()
	cmd := cj.spec.AfterScriptCmd(scriptCtx, cj.BasePath)
	writer.AddTitle("After Script")
	err := cj.runScript(scriptCtx, cmd, writer)
	if err != nil {
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
