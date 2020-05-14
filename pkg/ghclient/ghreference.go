package ghclient

import "strings"

// Reference analogous to a git reference. Points to a commit list
type Reference struct {
	head *Commit
	tail *Commit
}

// GetHead get head commit pointed at by ref
func (r *Reference) GetHead() *Commit {
	return r.head
}

// Register register commits to reference
func (r *Reference) Register(incHead *Commit) {
	// check for crossover between already registered commits and incoming commits.
	// If none found, existing commits pointed to by ref are deleted and ref set to point
	// to the incoming head. If duplicate parent found, children of the existing parent set to
	// to root parent of incoming head

	var incTail *Commit
	for c := incHead; c != nil; c = c.parent {
		incTail = c
	}

	if r.head == nil {
		r.head = incHead
		r.tail = incTail
		return
	}

	for c := r.head; c != nil; c = c.parent {
		if c.Sha == incTail.Sha {
			c.setChild(incTail.child)
			break
		}
		c.child = nil
	}

	r.head = incHead
}

func (r *Reference) String() string {
	var sb strings.Builder
	for c := r.head; c != nil; c = c.parent {
		sb.WriteString(c.String())
	}
	return sb.String()
}
