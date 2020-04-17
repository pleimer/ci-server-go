package ghclient

// Commit resource tracking a github commit
type Commit struct {
	// full ref
	// head - sha of most recent commit
	// size - number of commits in the push
	// commits[] - list of commits in push
	// commits[][sha]
	// commits[][author][name]
	// commits[][author][email]
	// commits[][url]
	// commits[][distinct]
	ref         string
	sha         string
	authorName  string
	authorEmail string
}
