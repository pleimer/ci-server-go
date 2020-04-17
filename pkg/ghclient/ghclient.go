package ghclient

import (
	"errors"
	"net/http"
)

type github struct {
	apiURL string
}

// Client github client object
type Client struct {
	oauth  string
	github github
	client *http.Client
}

// NewClient create a new github Client
func NewClient() Client {
	return Client{
		github: github{
			apiURL: "https://api.github.com",
		},
		client: &http.Client{},
	}
}

// Authenticate set and test authentication with Oauth token
func (c *Client) Authenticate(oauth string) error {
	c.oauth = oauth
	req, err := http.NewRequest("GET", c.github.apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+oauth)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case 200:
		break
	default:
		return errors.New("Failed to authenticate. Received status " + res.Status)
	}
	return nil
}

// UpdateStatus update status of commit in remote repository
func (c *Client) UpdateStatus(commit Commit) error {

	return nil
}
