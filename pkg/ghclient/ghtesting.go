package ghclient

import (
	"encoding/json"
	"net/http"
)

// RoundTripFunc mock http Transport
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

//TreeServer generates mock server body data based on a given tree structure. Use this when
// function being tested makes API calls to retrieve a github tree. Use in conjunction with
// NewTestClient()
func TreeServer(tree *Tree, repo *Repository) (map[string]func() ([]byte, error), error) {
	serverQuerries := make(map[string]func() ([]byte, error))

	api := NewAPI()

	err := recurseTreeAction(tree,
		func(t *Tree) error {
			serverQuerries[api.treeURL(repo.Owner.Login, repo.Name, t.Sha)] = func() ([]byte, error) {
				tm := treeMarshalFromTree(t)
				respBody, err := json.Marshal(tm)
				if err != nil {
					return nil, err
				}
				return respBody, nil
			}
			return nil
		},
		func(b *Blob) error {
			serverQuerries[api.blobURL(repo.Owner.Login, repo.Name, b.Sha)] = func() ([]byte, error) {
				respBody, err := json.Marshal(b)
				if err != nil {
					return nil, err
				}
				return respBody, nil
			}
			return nil
		},
	)
	return serverQuerries, err
}

func treeMarshalFromTree(tree *Tree) TreeMarshal {
	tm := TreeMarshal{
		Sha: tree.Sha,
	}

	for _, child := range tree.children {
		cRef := ChildRef{}
		switch v := child.(type) {
		case *Blob:
			cRef.Sha = v.Sha
			cRef.Path = v.Path
			cRef.Type = "blob"
		case *Tree:
			cRef.Sha = v.Sha
			cRef.Path = v.Path
			cRef.Type = "tree"
		}
		tm.Tree = append(tm.Tree, cRef)
	}
	return tm
}
