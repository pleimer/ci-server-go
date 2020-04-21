package ghclient

import "strings"

// Reference analogous to a git reference. Points to a commit list
type Reference struct {
	head *Commit
}

func (r *Reference) String() string {
	var sb strings.Builder
	for c := r.head; c != nil; c = c.parent {
		sb.WriteString(c.String())
	}
	return sb.String()
}
