package ghclient

import "encoding/json"

// Repository object for tracking remote repository
type Repository struct {
	Refs  map[string]*Reference
	Name  string `json:"name"`
	Fork  bool   `json:"fork"`
	Owner struct {
		Login string `json:"login"`
	}
}

// NewRepositoryFromJSON create new repository from json byte slice
func NewRepositoryFromJSON(repoJSON []byte) *Repository {
	repo := Repository{}
	json.Unmarshal(repoJSON, &repo)
	return &repo
}

func (r *Repository) String() string {
	return r.Name
}
