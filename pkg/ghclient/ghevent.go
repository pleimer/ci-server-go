package ghclient

import (
	"encoding/json"
	"errors"
)

// add pull request and comment

// Event definitions from github
type Event interface {
	Handle(JSON []byte) (GhObject, error)
}

var events map[string]Event = map[string]Event{
	"push": &Push{},
}

// EventFactory create github event based on string name
func EventFactory(incoming string) (Event, error) {
	if e := events[incoming]; e != nil {
		return e, nil
	}
	return nil, errors.New("Received error " + incoming + "does not exist")
}

// Push implements github Event interface
type Push struct {
	// repo
	// branch
	// commits
}

// Handle implements github Event.Handle
func (p *Push) Handle(pushJSON []byte) (GhObject, error) {
	// Handle incoming webhook byte slice
	// Retreive repo, branch, and commits and build/update
	// internal structures

	// check cache if repo exists. If not, create new one
	eventMap := make(map[string]json.RawMessage)
	json.Unmarshal(pushJSON, &eventMap)
	if len(eventMap) == 0 {
		return nil, errors.New("Failed parsing incoming event map")
	}
	return NewRepositoryFromJSON(eventMap["repository"])
}
