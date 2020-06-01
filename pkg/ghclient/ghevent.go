package ghclient

import (
	"encoding/json"
	"fmt"
)

// add pull request and comment

// Event definitions from github
type Event interface {
	Handle(*Client, []byte) error
}

var events map[string]Event = map[string]Event{
	"push": &Push{},
}

// EventFactory create github event based on string name
func EventFactory(incoming string) (Event, error) {
	if e := events[incoming]; e != nil {
		return e, nil
	}
	return nil, fmt.Errorf("unknown event type '%s'", incoming)
}

// Push implements github Event interface
type Push struct {
	Ref     Reference
	RefName string
	Repo    Repository
}

func (p *Push) Handle(client *Client, pushJSON []byte) error {
	// updates client repositories with pushed commits

	eventMap := make(map[string]json.RawMessage)
	err := json.Unmarshal(pushJSON, &eventMap)
	if err != nil {
		return pushEventError(fmt.Sprintf("failed parsing push event json: %s", err))
	}

	repo, err := NewRepositoryFromJSON(eventMap["repository"])
	if err != nil {
		return err
	}

	if _, ok := client.Repositories[repo.Name]; !ok {
		client.Repositories[repo.Name] = repo
	}

	var cSliceJSON []json.RawMessage
	err = json.Unmarshal(eventMap["commits"], &cSliceJSON)
	if err != nil {
		return pushEventError(fmt.Sprintf("failed parsing list of commits: %s", err))
	}

	// build up commit slive to create list from
	var cSlice []Commit
	if len(cSliceJSON) == 0 {
		headCommit := eventMap["head_commit"]
		if headCommit == nil {
			return pushEventError("no commits arrived with event")
		}
		cSliceJSON = append(cSliceJSON, headCommit)
	}

	for _, cJSON := range cSliceJSON {
		c, err := NewCommitFromJSON(cJSON)
		if err != nil {
			return pushEventError(fmt.Sprintf("failed creating commit object from JSON: %s", err))
		}
		cSlice = append(cSlice, *c)
	}

	// create ordered list of parents
	for i, c := range cSlice {
		if i < len(cSlice)-1 {
			c.setParent(&cSlice[i+1])
		}
	}

	head := &cSlice[0]
	cSlice = nil

	refName := string(eventMap["ref"])
	if refName == "" {
		return pushEventError("no reference found in event message")
	}
	p.RefName = refName
	client.Repositories[repo.Name].registerCommits(head, refName)
	client.Cache.WriteCommits(head)

	// p.Repo = *repo
	p.Repo = *client.Repositories[repo.Name]
	if p.Repo.GetReference(refName) == nil {
		return pushEventError("failed to retrieve reference from repository")
	}

	p.Ref = *p.Repo.GetReference(refName)
	return nil
}

func pushEventError(msg string) error {
	return &GithubClientError{
		module: "Push",
		err:    msg,
	}
}
