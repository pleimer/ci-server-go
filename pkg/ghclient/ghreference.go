package ghclient

// Reference analogous to a git reference
type Reference []*Commit

// GetHead get head commit of reference
func (r *Reference) GetHead() Commit {
	return *(*r)[len(*r)-1]
}

func (r *Reference) String() string {
	return "Method Reference.String() not implemented"
}
