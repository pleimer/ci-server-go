package ghclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Interim object for holding tree data from api calls in expected format
type ChildRef struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Sha  string `json:"sha"`
}
type TreeMarshal struct {
	Sha  string     `json:"sha"`
	Tree []ChildRef `json:"tree"`
}

// Node represents node in file tree
type Node interface {
	GetParent() Node
	GetChildren() []Node
	SetChild(Node)

	setParent(Node)
}

// Tree github tree type
type Tree struct {
	Sha  string `json:"sha"`
	Path string `json:"path"`

	parent   Node
	children []Node
}

// NewTreeFromJSON tree factory
func NewTreeFromJSON(tJSON []byte) (*Tree, error) {
	tree := &Tree{}
	err := json.Unmarshal(tJSON, tree)
	if err != nil {
		return nil, fmt.Errorf("error building tree from json: %s", err)
	}
	return tree, nil
}

//GetParent implements type Node
func (t *Tree) GetParent() Node {
	return t.parent
}

func (t *Tree) GetChildren() []Node {
	return t.children
}

func (t *Tree) setParent(parent Node) {
	t.parent = parent
}

func (t *Tree) SetChild(child Node) {
	child.setParent(t)
	t.children = append(t.children, child)
}

// helper function for printing out nodes in tree order
func recursivePrint(t Node) (string, error) {
	var sb strings.Builder
	sb.WriteString("\n")
	tJ, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	sb.Write(tJ)

	for _, child := range t.GetChildren() {
		res, err := recursivePrint(child)
		if err != nil {
			return "", err
		}
		sb.WriteString(" ")
		sb.WriteString(res)
	}

	return sb.String(), nil
}

//Blob encoded contents of a file
type Blob struct {
	Sha      string `json:"sha"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`

	Path   string `json:"Path"`
	parent Node
}

// NewBlobFromJSON blob factory
func NewBlobFromJSON(bJSON []byte) (*Blob, error) {
	blob := &Blob{}
	err := json.Unmarshal(bJSON, blob)
	if err != nil {
		return nil, fmt.Errorf("error building blob from json: %s", err)
	}
	return blob, nil
}

func (b *Blob) GetParent() Node {
	return b.parent
}

func (b *Blob) GetChildren() []Node {
	return nil
}

func (b *Blob) setParent(parent Node) {
	b.parent = parent
}

func (b *Blob) SetChild(child Node) {
	return
}

func (b *Blob) Write(w io.Writer) error {
	decoded, err := base64.StdEncoding.DecodeString(b.Content)
	if err != nil {
		return fmt.Errorf("when decoding blob %s content: %s", b.Sha, err)
	}
	_, err = w.Write(decoded)
	return fmt.Errorf("when writing blob %s to writer: %s", b.Sha, err)
}

//Cache one way cache for tracking github trees
type Cache struct {
	trees map[string]*Tree
	blobs map[string]*Blob
}

//NewCache create cache
func NewCache() Cache {
	return Cache{
		trees: make(map[string]*Tree),
		blobs: make(map[string]*Blob),
	}
}

func (c *Cache) GetBlob(sha string) *Blob {
	return c.blobs[sha]
}

func (c *Cache) GetTree(sha string) *Tree {
	return c.trees[sha]
}

func (c *Cache) WriteBlob(b *Blob) {
	c.blobs[b.Sha] = b
}

func (c *Cache) WriteTree(t *Tree) {
	c.trees[t.Sha] = t
}

func (c *Client) buildTree(parent Node, treeMarsh TreeMarshal, repo Repository) error {
	if parent == nil {
		return nil
	}

	for _, cRef := range treeMarsh.Tree {
		switch cRef.Type {
		case "blob":
			child := c.cache.GetBlob(cRef.Sha)
			if child != nil {
				parent.SetChild(child)
				continue
			}

			blobJSON, err := c.Api.GetBlob(repo.Owner.Login, repo.Name, cRef.Sha)
			if err != nil {
				return err
			}

			child, err = NewBlobFromJSON(blobJSON)
			if err != nil {
				return err
			}

			child.Path = cRef.Path

			c.cache.WriteBlob(child)
			parent.SetChild(child)

		case "tree":
			child := c.cache.GetTree(cRef.Sha)
			if child != nil {
				parent.SetChild(child)
				continue
			}

			treeJSON, err := c.Api.GetTree(repo.Owner.Login, repo.Name, cRef.Sha)
			if err != nil {
				return err
			}

			child, err = NewTreeFromJSON(treeJSON)
			if err != nil {
				return err
			}

			child.Path = cRef.Path

			c.cache.WriteTree(child)
			parent.SetChild(child)

			err = json.Unmarshal(treeJSON, &treeMarsh)
			if err != nil {
				return fmt.Errorf("error while parsing tree json: %s", err)
			}

			err = c.buildTree(child, treeMarsh, repo)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetTree checks cache for tree, else pulls tree from github
func (c *Client) GetTree(sha string, repo Repository) (*Tree, error) {
	t := c.cache.GetTree(sha)

	if t != nil {
		return t, nil
	}

	treeJSON, err := c.Api.GetTree(repo.Owner.Login, repo.Name, sha)
	if err != nil {
		return nil, err
	}

	treeMarsh := TreeMarshal{}
	err = json.Unmarshal(treeJSON, &treeMarsh)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling tree json: %s", err)
	}

	top := &Tree{
		Sha:      treeMarsh.Sha,
		Path:     treeMarsh.Sha,
		parent:   nil,
		children: []Node{},
	}

	c.cache.WriteTree(top)
	err = c.buildTree(top, treeMarsh, repo)
	if err != nil {
		return nil, err
	}
	return top, nil
}

// WriteTreeToDirectory write tree to specified directory path
func WriteTreeToDirectory(top Node, path string) error {
	return recurseTreeAction(top,
		func(t *Tree) error {
			err := os.Mkdir(path+t.Path, 0777)
			if err != nil {
				return err
			}
			return nil
		},
		func(b *Blob) error {
			f, err := os.Create(path + b.Path)
			if err != nil {
				return err
			}
			defer f.Close()
			b.Write(f)
			return nil
		})
}

func recurseTreeAction(top Node, treeAction func(*Tree) error, blobAction func(*Blob) error) error {
	switch v := top.(type) {
	case *Tree:
		err := treeAction(v)
		if err != nil {
			return err
		}
	case *Blob:
		err := blobAction(v)
		if err != nil {
			return err
		}
	}
	for _, child := range top.GetChildren() {
		err := recurseTreeAction(child, treeAction, blobAction)
		if err != nil {
			return err
		}
	}
	return nil
}
