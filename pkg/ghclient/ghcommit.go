package ghclient

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type CommitState int

const (
	ERROR CommitState = iota
	FAILURE
	PENDING
	SUCCESS
	NONE
)

func (cs CommitState) String() string {
	return [...]string{"error", "failure", "pending", "success", ""}[cs]
}

// Status github status object
type Status struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// Commit resource tracking a github commit
type Commit struct {
	Sha     string `json:"sha"`
	Message string `json:"message"`
	URL     string `json:"url"`
	ID      string `json:"id"`
	Status  Status
	Author  struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	}
	parent *Commit
	child  *Commit
}

// SetContext set context of commit status
func (c *Commit) SetContext(context string) {
	c.Status.Context = context
}

// SetStatus sets status of commit with a message
func (c *Commit) SetStatus(state CommitState, message string, targetURL string) {
	c.Status.State = state.String()
	c.Status.Description = message
	c.Status.TargetURL = targetURL
}

// GetParent returns copy of parent commit
func (c *Commit) GetParent() *Commit {
	return c.parent
}

func (c *Commit) setChild(child *Commit) {
	if child != nil {
		child.parent = c
	}
	c.child = child
}

func (c *Commit) setParent(parent *Commit) {
	c.parent = parent
	if parent != nil {
		parent.child = c
	}
}

func (c *Commit) String() string {
	ws := regexp.MustCompile(`\s+`)
	return ws.ReplaceAllString(fmt.Sprintf("[%s %s %s %s]\n", c.Sha, c.Author.Name, c.Author.Email, c.Message), " ")
}

// NewCommitFromJSON build commit from json byte slice
func NewCommitFromJSON(commitJSON []byte) (*Commit, error) {
	commit := &Commit{}
	err := json.Unmarshal(commitJSON, &commit)
	if err != nil {
		return nil, err
	}
	if commit.Sha == "" {
		commit.Sha = commit.ID
	}
	return commit, nil
}

func commitError(msg string) error {
	return &GithubClientError{
		module: "Commit",
		err:    msg,
	}
}
