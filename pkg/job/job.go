package job

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/parser"
)

type Job interface {
	Run(chan error)
}

type PushJob struct {
	event             *ghclient.Push
	client            *ghclient.Client
	scriptOutput      []byte
	afterScriptOutput []byte

	basePath string
}

func (p *PushJob) Run(errChan chan error) {
	defer close(errChan)
	// Get tree into filesystem
	commit := p.event.Ref.GetHead()
	tree, err := p.client.GetTree(commit.Sha, p.event.Repo)
	if err != nil {
		errChan <- err
		return
	}

	err = ghclient.WriteTreeToDirectory(tree, p.basePath)
	if err != nil {
		errChan <- err
		return
	}
	p.basePath = p.basePath + tree.Path

	// Read in spec
	f, err := os.Open(p.yamlPath(tree))
	defer f.Close()
	if err != nil {
		errChan <- err
		return
	}
	spec, err := parser.NewSpecFromYAML(f)
	if err != nil {
		errChan <- err
		return
	}

	// update status to pending
	commit.SetContext("ci-server-go")
	commit.SetStatus(ghclient.PENDING, "pending", "")
	err = p.client.UpdateCommitStatus(p.event.Repo, *commit)
	if err != nil {
		errChan <- err
	}

	// run script with timeout
	p.scriptOutput, err = spec.ScriptCmd(p.basePath).Output()
	if err != nil {
		commit.SetStatus(ghclient.ERROR, fmt.Sprintf("job failed with: %s", err), "")
		errChan <- err
	}

	// run after_script
	p.afterScriptOutput, err = spec.AfterScriptCmd(p.basePath).Output()
	if err != nil {
		commit.SetStatus(ghclient.ERROR, fmt.Sprintf("after_script failed with: %s", err), "")
		errChan <- err
	}

	//post gist
	report := string(p.buildReport())
	gist := ghclient.NewGist()
	gist.Description = fmt.Sprintf("CI Results for repository '%s' commit '%s'", p.event.Repo.Name, commit.Sha)
	gist.AddFile(fmt.Sprintf("%s_%s.md", p.event.Repo.Name, commit.Sha), report)
	gJSON, err := json.Marshal(gist)
	if err != nil {
		errChan <- err
	}

	res, err := p.client.Api.PostGists(gJSON)
	if err != nil {
		errChan <- err
	}

	// get gist target url
	resMap := make(map[string]interface{})
	err = json.Unmarshal(res, &resMap)
	if err != nil {
		errChan <- fmt.Errorf("while unmarshalling gist response: %s", err)
	}

	targetURL := resMap["url"].(string)

	// update status of commit
	commit.SetStatus(ghclient.SUCCESS, "all jobs passed", targetURL)
	err = p.client.UpdateCommitStatus(p.event.Repo, *commit)
	if err != nil {
		errChan <- err
	}

}

func (p *PushJob) buildReport() []byte {
	var sb strings.Builder
	sb.WriteString("## Script Results\n```")
	sb.Write(p.scriptOutput)
	sb.WriteString("```\n## After Script Results\n```\n")
	sb.Write(p.afterScriptOutput)
	sb.WriteString("```")
	return []byte(sb.String())
}

// helper functions
func (p *PushJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{p.basePath, "ci.yml"}, "/")
}
