package ghclient

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	//ErrInvalidResp occurs when server reply does not contain all required fields
	ErrInvalidResp error = errors.New("did not receive required fields")
)

// GistWriter implements io.Writer type for writing to
// a single file in a github gists. Gists last the lifetime of this object.
// That is, first calls to write will create a new gist,
// and subsequent calls will update the existing gist
// until this object is destroyed.
//
// This should be used in conjustion
// with a buffered writer to avoid frequent API calls
type GistWriter struct {
	API        *API
	serverGist *serverGist
	gist       Gist
	filename   string
}

//NewGistWriter GistWriter constructor
func NewGistWriter(api *API, g Gist, filename string) (*GistWriter, error) {
	gw := &GistWriter{
		API:      api,
		gist:     g,
		filename: filename,
	}
	gw.gist.WriteFile(filename, "pending...")
	data, err := json.Marshal(gw.gist)
	if err != nil {
		return nil, err
	}

	gw.gist.DeleteFile(filename)

	fmt.Println(string(data))
	resp, err := api.PostGists(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &gw.serverGist)
	if gw.serverGist.ID == "" {
		return nil, ErrInvalidResp
	}
	return gw, nil
}

// Write implements io.Writer
func (gw *GistWriter) Write(p []byte) (int, error) {
	gw.gist.WriteFile(gw.filename, string(p))
	data, err := json.Marshal(gw.gist)

	if err != nil {
		return 0, err
	}

	_, err = gw.API.UpdateGist(data, gw.serverGist.ID)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

//GetServerGistID returns ID of gist on github server
func (gw *GistWriter) GetServerGistID() string {
	if gw.serverGist == nil {
		return ""
	}
	return gw.serverGist.ID
}

type serverGist struct {
	ID string `json:"id"`
}
