package ghclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Commit resource tracking a github commit
type Commit struct {
	Sha     string `json:"sha"`
	Message string `json:"message"`
	URL     string `json:"url"`
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

func (c *Commit) linkChild(child *Commit) {
	child.parent = c
	c.child = child
}

func (c *Commit) String() string {
	ws := regexp.MustCompile(`\s+`)
	return ws.ReplaceAllString(fmt.Sprintf("[%s %s %s]\n", c.Sha, c.Author.Name, c.Author.Email), " ")
}

// CreateCommitFromJSON build commit from json byte slice
func CreateCommitFromJSON(commitJSON []byte) (*Commit, error) {
	commit := &Commit{}
	json.Unmarshal(commitJSON, &commit)
	if commit.Sha == "" {
		return nil, errors.New("Failed parsing commit json")
	}
	return commit, nil
}
