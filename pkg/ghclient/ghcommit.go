package ghclient

// Commit resource tracking a github commit
type Commit struct {
	Sha     string
	Message string
	URL     string
	Author  struct {
		Name     string
		Email    string
		Username string
	}
}
