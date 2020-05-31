package ghclient

import "sync"

//Cache one way cache for tracking github trees
type Cache struct {
	// trees       map[string]*Tree
	// blobs       map[string]*Blob
	// commitIndex map[string]*Commit
	// rw          sync.RWMutex
	trees       sync.Map
	blobs       sync.Map
	commitIndex sync.Map
}

//NewCache create cache
func NewCache() Cache {
	return Cache{
		// trees:       make(map[string]*Tree),
		// blobs:       make(map[string]*Blob),
		// commitIndex: make(map[string]*Commit),
		trees:       sync.Map{},
		blobs:       sync.Map{},
		commitIndex: sync.Map{},
	}
}

func (c *Cache) GetBlob(sha string) *Blob {
	if res, found := c.blobs.Load(sha); found {
		return res.(*Blob)
	}
	return nil
}

func (c *Cache) GetTree(sha string) *Tree {
	if res, found := c.trees.Load(sha); found {
		return res.(*Tree)
	}
	return nil
}

func (c *Cache) GetCommit(sha string) *Commit {
	if res, found := c.commitIndex.Load(sha); found {
		return res.(*Commit)
	}
	return nil
}

func (c *Cache) WriteBlob(b *Blob) {
	c.blobs.Store(b.Sha, b)
}

func (c *Cache) WriteTree(t *Tree) {
	c.trees.Store(t.Sha, t)
}

func (c *Cache) WriteCommits(head *Commit) {
	for p := head; p != nil; p = p.parent {
		if _, found := c.commitIndex.Load(p.Sha); !found {
			c.commitIndex.Store(p.Sha, p)
		}
	}
}
