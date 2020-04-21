package ghclient

import (
	"encoding/json"
	"errors"
)

// Repository object for tracking remote repository
type Repository struct {
	Name  string `json:"name"`
	Fork  bool   `json:"fork"`
	Owner struct {
		Login string `json:"login"`
	}

	refs map[string]*Reference
}

// NewRepositoryFromJSON create new repository from json byte slice
func NewRepositoryFromJSON(repoJSON []byte) (*Repository, error) {
	repo := Repository{}
	json.Unmarshal(repoJSON, &repo)
	if repo.Name == "" {
		return nil, errors.New("Failed parsing repository JSON")
	}

	repo.refs = make(map[string]*Reference)
	return &repo, nil
}

// GetReference retrieve git reference by ID
func (r *Repository) GetReference(refName string) Reference {
	return *r.refs[refName]
}

// registerCommits register commits to reference
func (r *Repository) registerCommits(incHead *Commit, refName string) {
	// check for crossover between already registered commits and incoming commits.
	// If none found, existing commits pointed to by ref are deleted and ref set to point
	// to the incoming head. If duplicate parent found, children of the parent set to
	// to the children of the incoming head

	if r.refs[refName] == nil {
		r.refs[refName] = &Reference{
			head: incHead,
		}
		return
	}

	var incTail *Commit
	for c := incHead; c != nil; c = c.parent {
		incTail = c
	}

	for c := r.refs[refName].head; c != nil; c = c.parent {
		if c.Sha == incTail.Sha {
			c.linkChild(incTail.child)
			continue
		}
		c.child = nil
	}

	r.refs[refName].head = incHead
}

func (r *Repository) String() string {
	return "Method Repository.String() not implemented"
}
