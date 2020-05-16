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
	return nil, fmt.Errorf("received error %s does not exist", incoming)
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
		return fmt.Errorf("failed parsing push event json: %s", err)
	}

	repo, err := NewRepositoryFromJSON(eventMap["repository"])
	if err != nil {
		return err
	}

	if _, ok := client.repositories[repo.Name]; !ok {
		client.repositories[repo.Name] = repo
	}

	var cSliceJSON []json.RawMessage
	err = json.Unmarshal(eventMap["commits"], &cSliceJSON)
	if err != nil {
		return fmt.Errorf("failed parsing list of commits: %s", err)
	}

	// build up commit slive to create list from
	var cSlice []Commit
	if len(cSliceJSON) == 0 {
		headCommit := eventMap["head_commit"]
		if headCommit == nil {
			return fmt.Errorf("no commits arrived with event")
		}
		cSliceJSON = append(cSliceJSON, headCommit)
	}

	for _, cJSON := range cSliceJSON {
		c, err := NewCommitFromJSON(cJSON)
		if err != nil {
			return fmt.Errorf("failed creating commit object from JSON: %s", err)
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
	p.RefName = refName
	client.repositories[repo.Name].registerCommits(head, refName)
	client.Cache.WriteCommits(head)

	p.Repo = *repo
	p.Ref = repo.GetReference(refName)
	return nil
}
