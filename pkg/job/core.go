package job

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/logging"
	"github.com/pleimer/ci-server-go/pkg/parser"
	"github.com/pleimer/ci-server-go/pkg/report"
)

// RunCoreJob executes the main sequence of steps that a CI job contains.
func RunCoreJob(ctx context.Context, client *ghclient.Client, repo ghclient.Repository, refName string, commit ghclient.Commit, log *logging.Logger) {
	// This function downloads the git tree, loads in the ci.yml, creates writers
	// to log test output to both a file and github gist, and runs the script and
	// after_script sections of ci.yml

	cj := newCoreJob(client, repo, commit)
	cj.BasePath = "/tmp"

	log.Metadata(map[string]interface{}{"process": "Core"})
	log.Info("downloading git tree")
	err := cj.GetTree()
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Error("retrieving resources")
		return
	}

	log.Metadata(map[string]interface{}{"process": "Core"})
	log.Info("loading test specifications")
	err = cj.LoadSpec(refName)
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Error("failed to load spec")
		return
	}

	// initialize writers
	logPath := filepath.Join(cj.BasePath, fmt.Sprintf("%s.log", commit.Sha))
	f, err := os.Create(logPath)
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Error("opening log path")
		return
	}
	defer f.Close()

	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", cj.repo.Name, cj.commit.Sha)

	gw, err := ghclient.NewGistWriter(&client.Api, gist, fmt.Sprintf("%s_%s.md", cj.repo.Name, cj.commit.Sha))
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Error("creating gist writer")
		return
	}
	writer := report.NewWriter(f, gw)

	// run scripts
	log.Metadata(map[string]interface{}{"process": "Core"})
	log.Info("running main script")
	err = cj.RunMainScript(ctx, writer, gw.GetServerGistID())
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Info("script failed")
	} else {
		log.Metadata(map[string]interface{}{"process": "Core"})
		log.Info("main script completed successfully")
	}
	if err := cj.postCommitStatus(); err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err.Error()})
		log.Error("posting commit status")
	}

	// It is highly NOT recommended to create top level contexts in lower functions
	// 'After script' is responsible for cleaning up resources, so it must run even when a cancel signal
	// has been sent by the main server goroutine. This still garauntees an exit after timeout
	// so it isn't too terrible
	log.Metadata(map[string]interface{}{"process": "Core"})
	log.Info("running after script")
	err = cj.RunAfterScript(context.Background(), writer, gw.GetServerGistID())
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err})
		log.Info("after_script failed")
	} else {
		log.Metadata(map[string]interface{}{"process": "Core"})
		log.Info("after_script completed successfully")
	}
	if err := cj.postCommitStatus(); err != nil {
		log.Metadata(map[string]interface{}{"process": "Core", "error": err.Error()})
		log.Error("posting commit status")
	}
}

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

func (cj *coreJob) GetTree() error {
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

func (cj *coreJob) LoadSpec(refName string) error {
	tree, err := cj.client.GetTree(cj.commit.Sha, cj.repo)
	if err != nil {
		return err
	}
	cj.BasePath = filepath.Join(cj.BasePath, tree.Path)

	f, err := os.Open(filepath.Join(cj.BasePath, "ci.yml"))
	if err != nil {
		return err
	}

	defer f.Close()

	cj.spec, err = parser.NewSpecFromYAML(f)
	if err != nil {
		return err
	}
	cj.spec.SetMetaVar("__commit__", cj.commit.Sha)
	refName = strings.ReplaceAll(refName, "\"", "")
	cj.spec.SetMetaVar("__ref__", refName)

	refComponents := strings.Split(refName, "/")
	branchName := refComponents[len(refComponents)-1]
	cj.spec.SetMetaVar("__branch__", branchName)

	return nil
}

//runs a script and writes buffered output to file and gist writer
func (cj *coreJob) runScript(ctx context.Context, script *exec.Cmd, writer *report.Writer) error {
	var err error
	var scriptErr error

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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scriptErr = script.Wait()
		scriptDone <- struct{}{}
	}()

	ticker := time.NewTicker(60 * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-scriptDone:
				ticker.Stop()
				return
			case <-ticker.C:
				writer.Flush()
			}
		}
	}()

	for scanner.Scan() {
		writer.Write(scanner.Text())
	}

	wg.Wait()

	if ctx.Err() != nil {
		writer.Write(fmt.Sprintf("\nerror: %s", ctx.Err()))
		writer.CloseBlock()
		writer.Flush()
		return ctx.Err()
	}

	if scriptErr != nil {
		writer.Write(fmt.Sprintf("\n[ci-server] script error: %s", scriptErr))
		writer.CloseBlock()
		writer.Flush()
		return scriptErr
	}

	err = scanner.Err()

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
func (cj *coreJob) RunMainScript(ctx context.Context, writer *report.Writer, gistID string) error {
	gistURL := cj.client.Api.PublishedGistURL(gistID, cj.client.User)

	cj.commit.SetStatus(ghclient.PENDING, "running main script", gistURL)
	cj.postCommitStatus()
	cj.commit.SetStatus(ghclient.SUCCESS, "main script successful", gistURL)

	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()

	cmd := cj.spec.ScriptCmd(scriptCtx, cj.BasePath)
	writer.AddTitle("Main Script")
	err := cj.runScript(scriptCtx, cmd, writer)

	if err != nil {
		switch err {
		case context.Canceled:
			cj.commit.SetStatus(ghclient.ERROR, "main script canceled", gistURL)
		case context.DeadlineExceeded:
			cj.commit.SetStatus(ghclient.FAILURE, "main script timed out", gistURL)
		case ghclient.ErrInvalidResp:
			cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("error logging: %s", err), gistURL)
		default:
			cj.commit.SetStatus(ghclient.FAILURE, "main script failed", gistURL)
		}
		return err
	}
	return nil
}

// runs spec.AfterScript
func (cj *coreJob) RunAfterScript(ctx context.Context, writer *report.Writer, gistID string) error {
	gistURL := cj.client.Api.PublishedGistURL(gistID, cj.client.User)

	scriptCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(cj.spec.Global.Timeout))
	defer cancel()

	cmd := cj.spec.AfterScriptCmd(scriptCtx, cj.BasePath)
	writer.AddTitle("After Script")
	err := cj.runScript(scriptCtx, cmd, writer)
	if err != nil {
		switch err {
		case context.Canceled:
			cj.commit.SetStatus(ghclient.ERROR, "after_script canceled", gistURL)
		case context.DeadlineExceeded:
			cj.commit.SetStatus(ghclient.ERROR, "after_script timed out", gistURL)
		case ghclient.ErrInvalidResp:
			cj.commit.SetStatus(ghclient.ERROR, fmt.Sprintf("error logging: %s", err), gistURL)
		default:
			cj.commit.SetStatus(ghclient.ERROR, "after_script failed", gistURL)
		}
		return err
	}
	return nil
}

// ----------- helper functions ---------------
func (cj *coreJob) postCommitStatus() error {
	err := cj.client.UpdateCommitStatus(cj.repo, cj.commit)
	if err != nil {
		return err
	}
	return nil
}
