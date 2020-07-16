package ghclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// add pull request and comment

// Event definitions from github
type Event interface {
	Handle(*Client, []byte) error
}

var events map[string]Event = map[string]Event{
	"push":          &Push{},
	"issue_comment": &Comment{},
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

	Action    string
	RefName   string
	Body      string
	CommitSHA string
	User      string
}

// Handle parses the contents of a github comment
func (c *Comment) Handle(client *Client, commentJSON []byte) error {
	// There are four types of comment webhooks:
	// 1. commit_comment
	// 2. pull_request_review
	// 3. pull_request_review_comment
	// 4. issue_comment

	// This event handles only #4

	var ok bool

	eventMap := make(map[string]interface{})
	err := json.Unmarshal(commentJSON, &eventMap)
	if err != nil {
		return commentEventError(fmt.Sprintf("failed parsing comment event json: %s", err))
	}

	// first level attributes
	c.Action, ok = eventMap["action"].(string)
	if !ok {
		return commentEventError("failed parsing comment event json: event data did not contain 'action' attribute")
	}

	issue, ok := eventMap["issue"].(map[string]interface{})
	if !ok {
		return commentEventError("failed parsing comment event json: event data did not contain 'issue' attribute")
	}

	comment, ok := eventMap["comment"].(map[string]interface{})
	if !ok {
		return commentEventError("failed parsing comment event json: event data did not contain 'comment' attribute")
	}

	repository, ok := eventMap["repository"].(map[string]interface{})
	if !ok {
		return commentEventError("failed parsing comment event json: event data did not contain 'repository' attribute")
	}

	// second level attributes
	pullRequest, ok := issue["pull_request"].(map[string]interface{})
	if !ok {
		return commentEventError("failed parsing pull request data from event message")
	}

	user, ok := comment["user"].(map[string]interface{})
	if !ok {
		return commentEventError("failed parsing user data from event message")
	}

	c.Body, ok = comment["body"].(string)
	if !ok {
		return commentEventError("failed parsing comment body from event message")
	}

	repoName, ok := repository["name"].(string)
	if !ok {
		return commentEventError("failed parsing repository name from event message")
	}

	// third level attributes
	prURL, ok := pullRequest["url"].(string)
	if !ok {
		return commentEventError("failed parsing pull request url from event message")
	}

	c.User, ok = user["login"].(string)
	if !ok {
		return commentEventError("failed parsing user information from event message")
	}

	// find resources based on parsed values
	if repo := client.Repositories[repoName]; repo == nil {
		return commentEventError(fmt.Sprintf("could not find '%s' repository", repoName))
	}
	c.Repo = *client.Repositories[repoName]

	// retrieve pull request data for issue comment
	prData := make(map[string]interface{})
	prDataBytes, err := client.Api.GetURL(prURL)
	if err != nil {
		return err
	}

	err = json.Unmarshal(prDataBytes, &prData)
	if err != nil {
		return errors.Wrap(err, "failed to parse pull request json")
	}

	head, ok := prData["head"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find pull request head commit data")
	}

	c.CommitSHA, ok = head["sha"].(string)
	if !ok {
		return fmt.Errorf("failed to find commit sha from pull request data")
	}

	c.RefName, ok = head["ref"].(string)
	if !ok {
		return fmt.Errorf("failed to find reference data from pull request data")
	}

	//TODO: query this from commit URL instead of this hack
	c.RefName = strings.Join([]string{"refs", "heads", c.RefName}, "/")
	c.RefName = "\"" + c.RefName + "\""

	if ref := client.Repositories[repoName].GetReference(c.RefName); ref == nil {
		return fmt.Errorf("could not find '%s' reference in repository '%s'", c.RefName, repoName)
	}

	c.Ref = *client.Repositories[repoName].GetReference(c.RefName)

	return nil
}

// Push implements github Event interface
type Push struct {
	Ref     Reference
	RefName string
	Repo    Repository
	User    string
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

	// build up commit slice to create list from
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

	// get user information
	sender := map[string]interface{}{}
	err = json.Unmarshal(eventMap["sender"], &sender)
	if err != nil {
		return pushEventError(fmt.Sprintf("failed retrieving sender information from message: %s", err))
	}

	var ok bool
	if p.User, ok = sender["login"].(string); !ok {
		return pushEventError("failed retrieving user credentials from push message")
	}

	// get reference and repository information
	refName := string(eventMap["ref"])
	if refName == "" {
		return pushEventError("no reference found in event message")
	}
	p.RefName = refName
	client.Repositories[repo.Name].registerCommits(head, refName)
	client.Cache.WriteCommits(head)

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
