package ghclient

import (
	"encoding/json"
	"errors"
)

// Repository object for tracking remote repository
type Repository struct {
	refs  map[int]*Reference
	Name  string `json:"name"`
	Fork  bool   `json:"fork"`
	Owner struct {
		Login string `json:"login"`
	}
}

// NewRepositoryFromJSON create new repository from json byte slice
func NewRepositoryFromJSON(repoJSON []byte) (*Repository, error) {
	repo := Repository{}
	json.Unmarshal(repoJSON, &repo)
	if repo.Name == "" {
		return nil, errors.New("Failed parsing repository JSON")
	}
	return &repo, nil
}

// GetReference retrieve git reference by ID
func (r *Repository) GetReference(refID int) Reference {
	return *r.refs[refID]
}

// AddReference add git reference to repository
func (r *Repository) AddReference(refID int, ref *Reference) {
	r.refs[refID] = ref
}

func (r *Repository) String() string {
	return "Method Repository.String() not implemented"
}
