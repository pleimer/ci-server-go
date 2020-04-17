package ghclient

import (
	"fmt"
	"regexp"
)

func (c Commit) String() string {
	ws := regexp.MustCompile(`\s+`)
	return ws.ReplaceAllString(fmt.Sprintf("%s\n%s\n%s\n%s\n", c.ref, c.sha, c.authorName, c.authorEmail), " ")
}

//CommitCache cached commits tracked by priority and expired when priority gets too low
type CommitCache struct {
	commits []Commit
}

// Register take in slice of commits and register new ones to cache
// TODO: use priority queue data structure, expire commits
// TODO: optimize this
func (cc *CommitCache) Register(commits []Commit) {
	lastSha := ""
	if len(cc.commits) > 0 {
		lastSha = cc.getLastSha()
	}

	var tmpsC []Commit
	for i := len(commits) - 1; i >= 0; i-- {
		if commits[i].sha == lastSha {
			break
		}
		tmpsC = append([]Commit{commits[i]}, tmpsC...)
	}
	cc.commits = append(cc.commits, tmpsC...)
}

func (cc *CommitCache) getLastSha() string {
	return cc.commits[len(cc.commits)-1].sha
}

// Cache stores and expires github object states
type Cache struct {
	Repositories map[string]*Repository
}
