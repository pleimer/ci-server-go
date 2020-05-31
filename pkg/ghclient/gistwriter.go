package ghclient

import (
	"encoding/json"
	"errors"
)

var (
	//ErrInvalidResp occurs when server reply does not contain all required fields
	ErrInvalidResp error = errors.New("did not receive required fields")
)

// GistWriter implements io.Writer type for writing to
// github gists. Gists last the lifetime of this object.
// That is, first calls to write will create a new gist,
// and subsequent calls will update the existing gist
// until this object is destroyed.
//
// This should be used in conjustion
// with a buffered writer to avoid frequent API calls
type GistWriter struct {
	API     *API
	desc    gistDesc
	created bool
}

//NewGistWriter GistWriter constructor
func NewGistWriter(api *API) *GistWriter {
	return &GistWriter{
		API: api,
	}
}

// Write implements io.Writer
func (gw *GistWriter) Write(p []byte) (int, error) {
	if !gw.created {
		resp, err := gw.API.PostGists(p)
		if err != nil {
			return 0, err
		}

		err = json.Unmarshal(resp, &gw.desc)
		if gw.desc.ID == "" {
			return 0, ErrInvalidResp
		}
		gw.created = true
		return len(p), nil
	}

	_, err := gw.API.UpdateGist(p, gw.desc.ID)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type gistDesc struct {
	ID string `json:"id"`
}
