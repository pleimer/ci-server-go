package ghclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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

//TreeServer generates mock server tree responses based on input tree structure. Use this when
// function being tested makes API calls to retrieve a github tree. Use in conjunction with
// NewTestClient()
func TreeServer(tree *Tree, repo *Repository) (map[string]func(*http.Request) (*http.Response, error), error) {
	serverQuerries := make(map[string]func(req *http.Request) (*http.Response, error))

	api := NewAPI()

	err := recurseTreeAction(tree,
		func(t *Tree) error {
			serverQuerries[api.TreeURL(repo.Owner.Login, repo.Name, t.Sha)] = func(req *http.Request) (*http.Response, error) {
				tm := treeMarshalFromTree(t)
				respBody, err := json.Marshal(tm)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
					Header:     make(http.Header),
				}, nil
			}
			return nil
		},
		func(b *Blob) error {
			serverQuerries[api.BlobURL(repo.Owner.Login, repo.Name, b.Sha)] = func(req *http.Request) (*http.Response, error) {
				respBody, err := json.Marshal(b)
				if err != nil {
					return nil, err
				}
				return &http.Response{
					StatusCode: 200,
					Status:     "200 OK",
					Body:       ioutil.NopCloser(bytes.NewReader(respBody)),
					Header:     make(http.Header),
				}, nil
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
