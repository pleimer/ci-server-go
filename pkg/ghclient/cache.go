package ghclient

//Cache one way cache for tracking github trees
type Cache struct {
	trees       map[string]*Tree
	blobs       map[string]*Blob
	commitIndex map[string]*Commit
}

//NewCache create cache
func NewCache() Cache {
	return Cache{
		trees:       make(map[string]*Tree),
		blobs:       make(map[string]*Blob),
		commitIndex: make(map[string]*Commit),
	}
}

func (c *Cache) GetBlob(sha string) *Blob {
	return c.blobs[sha]
}

func (c *Cache) GetTree(sha string) *Tree {
	return c.trees[sha]
}

func (c *Cache) GetCommit(sha string) *Commit {
	return c.commitIndex[sha]
}

func (c *Cache) WriteBlob(b *Blob) {
	c.blobs[b.Sha] = b
}

func (c *Cache) WriteTree(t *Tree) {
	c.trees[t.Sha] = t
}

func (c *Cache) WriteCommits(head *Commit) {
	for p := head; p != nil; p = p.parent {
		if c.commitIndex[p.Sha] == nil {
			c.commitIndex[p.Sha] = p
		}
	}
}
