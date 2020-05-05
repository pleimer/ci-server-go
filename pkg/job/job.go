package job

import (
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
	event  *ghclient.Push
	client *ghclient.Client

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

	// run script with timeout
	out, err := spec.ScriptCmd().Output()
	if err != nil {
		errChan <- err
		return
	}

	fmt.Print(string(out))

	// run after_script
	out, err = spec.AfterScriptCmd().Output()
	if err != nil {
		errChan <- err
		return
	}
	fmt.Println(out)

	//post gist

}

// helper functions
func (p *PushJob) yamlPath(tree *ghclient.Tree) string {
	return strings.Join([]string{p.basePath, tree.Path, "ci.yml"}, "/")
}
