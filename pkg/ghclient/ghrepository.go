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

func (r *Repository) registerCommits(incHead *Commit, refName string) {
	if _, ok := r.refs[refName]; !ok {
		r.refs[refName] = &Reference{}
	}

	r.refs[refName].register(incHead)
}

func (r *Repository) String() string {
	return "Method Repository.String() not implemented"
}
