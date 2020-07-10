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
	"push":                        &Push{},
	"pull_request_review_comment": &Comment{},
}

// EventFactory create github event based on string name
func EventFactory(incoming string) (Event, error) {
	if e := events[incoming]; e != nil {
		return e, nil
	}
	return nil, fmt.Errorf("unknown event type '%s'", incoming)
}

// Comment implements Event interface. Represents a github comment webhook
type Comment struct {
	Ref  Reference
	Repo Repository

	Action   string
	RefName  string
	Body     string
	CommitID string
}

// Handle parses the contents of a github comment
func (c *Comment) Handle(client *Client, commentJSON []byte) error {
	// There are four types of comment webhooks:
	// 1. commit_comment
	// 2. pull_request_review
	// 3. pull_request_review_comment
	// 4. issue_comment

	// This event handles only 3

	var ok bool

	eventMap := make(map[string]interface{})
	err := json.Unmarshal(commentJSON, &eventMap)
	if err != nil {
		return commentEventError(fmt.Sprintf("failed parsing comment event json: %s", err))
	}

	c.Action, ok = eventMap["action"].(string)
	if !ok {
		return commentEventError("failed parsing comment event json: event data did not contain 'action' attribute")
	}

	commentMap, ok := eventMap["comment"].(map[string]interface{})
	if !ok {
		return commentEventError("failed retrieving comment data. Data did not exist or was wrong type")
	}

	c.Body, ok = commentMap["body"].(string)
	if !ok {
		return commentEventError("failed retrieving comment body. Body did not exist or was wrong type")
	}

	prMap, ok := eventMap["pull_request"].(map[string]interface{})
	if !ok {
		return commentEventError("failed to retrieve pull request data from comment json")
	}

	head, ok := prMap["head"].(map[string]interface{})
	if !ok {
		return commentEventError("failed to retrieve head commit data for comment")
	}

	c.CommitID, ok = head["sha"].(string)
	if !ok {
		return commentEventError("failed retrieving associated commit ID. Item did not exist or was wrong type")
	}

	c.RefName, ok = head["ref"].(string)
	if !ok {
		return commentEventError("failed retrieving reference name. Reference name did not exist or was wrong type")
	}

	repoName, ok := head["repo"].(map[string]interface{})["name"].(string)
	if !ok {
		return commentEventError("failed retrieving repository name. Repository name did not exist or was wrong type")
	}

	var repo *Repository
	if repo, ok = client.Repositories[repoName]; !ok {
		return commentEventError("could not find repository in client registry - it must be loaded before comment")
	}
	c.Repo = *repo

	ref := repo.GetReference(c.RefName)
	if ref == nil {
		return commentEventError(fmt.Sprintf("could not find reference %s in %s repository", c.RefName, repoName))
	}
	c.Ref = *ref
	return nil
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
		cSlice = append([]Commit{*c}, cSlice...)
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

func commentEventError(msg string) error {
	return &GithubClientError{
		module: "Comment",
		err:    msg,
	}
}
