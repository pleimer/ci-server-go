package ghclient

import "fmt"

// GithubClientError all returned errors from this package are of this type
type GithubClientError struct {
	err    string
	module string
}

func (e *GithubClientError) withMessage(message string) *GithubClientError {
	e.err = message
	return e
}

func (e *GithubClientError) Error() string {
	return fmt.Sprintf("module: %s, error: %s ", e.module, e.err)
}
