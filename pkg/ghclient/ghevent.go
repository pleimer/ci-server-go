package ghclient

import (
	"encoding/json"
	"fmt"
)

// add pull request and comment

// Event definitions from github
type Event interface {
	handle(*Client, []byte) error
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
	// repo
	// branch
	// commits
}

func (p *Push) handle(client *Client, pushJSON []byte) error {
	// push handle
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

	var cSlice []Commit
	for _, cJSON := range cSliceJSON {
		c, err := NewCommitFromJSON(cJSON)
		if err != nil {
			return err
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
	client.repositories[repo.Name].registerCommits(head, refName)
	return nil
}
