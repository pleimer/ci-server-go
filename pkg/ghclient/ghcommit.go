package ghclient

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Status github status object
type Status struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context`
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

// GetParent returns copy of parent commit
func (c *Commit) GetParent() Commit {
	return *(c.parent)
}

func (c *Commit) setChild(child *Commit) {
	child.parent = c
	c.child = child
}

func (c *Commit) setParent(parent *Commit) {
	c.parent = parent
	parent.child = c
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
