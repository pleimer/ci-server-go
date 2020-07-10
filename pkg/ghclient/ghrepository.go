package ghclient

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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
	//repo := Repository{}
	var repo *Repository
	json.Unmarshal(repoJSON, &repo)
	if repo == nil {
		return nil, repositoryError("failed parsing repository from JSON")
	}
	repo.refs = make(map[string]*Reference)
	return repo, nil
}

//NewRepositoryFromMap create new repository from map
func NewRepositoryFromMap(m map[string]interface{}) (*Repository, error) {
	rt := reflect.TypeOf(Repository{})
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		key := field.Tag.Get("json")
		fmt.Println(key)

		if val, ok := m[key]; ok {
			if vt := reflect.ValueOf(val); vt.Kind() != field.Type.Kind() {
				return nil, fmt.Errorf("attribute '%s' is wrong type - expected '%s' but got '%s'", key, field.Type.Kind().String(), vt.Kind().String())
			}
			continue
		}
		return nil, fmt.Errorf("missing value: %s", key)
	}

	repo := Repository{
		Name: m["name"].(string),
		Fork: m["fork"].(bool),
	}
	return &repo, nil
}

// GetReference retrieve git reference by ID
func (r *Repository) GetReference(refName string) *Reference {
	return r.refs[refName]
}

func (r *Repository) registerCommits(incHead *Commit, refName string) {
	if _, ok := r.refs[refName]; !ok {
		r.refs[refName] = &Reference{}
	}

	r.refs[refName].Register(incHead)
}

func (r *Repository) String() string {
	var sb strings.Builder
	for key, val := range r.refs {
		sb.WriteString(key)
		sb.WriteString(":")
		sb.WriteString(val.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

func repositoryError(msg string) error {
	return &GithubClientError{
		err:    msg,
		module: "Repository",
	}
}
