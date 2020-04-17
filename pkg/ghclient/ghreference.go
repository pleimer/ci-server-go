package ghclient

// Reference analogous to a git reference
type Reference struct {
	commits map[string]*Commit
}
