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
	GetPath() string
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

func (t *Tree) GetPath() string {
	return t.Path
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

// RecursivePrint helper function for printing out nodes in tree order
func RecursivePrint(t Node) (string, error) {
	var sb strings.Builder
	// sb.WriteString("\n")
	// tJ, err := json.Marshal(t)
	// if err != nil {
	// 	return "", err
	// }
	//sb.Write(tJ)
	sb.WriteString(t.GetPath())

	for _, child := range t.GetChildren() {
		res, err := RecursivePrint(child)
		if err != nil {
			return "", err
		}
		sb.WriteString("\n")
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

func (b *Blob) GetPath() string {
	return b.Path
}

func (b *Blob) Write(w io.Writer) error {
	decoded, err := base64.StdEncoding.DecodeString(b.Content)
	if err != nil {
		return fmt.Errorf("when decoding blob %s content: %s", b.Sha, err)
	}
	_, err = w.Write(decoded)
	return fmt.Errorf("when writing blob %s to writer: %s", b.Sha, err)
}

func (c *Client) buildTree(parent Node, treeMarsh TreeMarshal, repo Repository) error {
	for _, cRef := range treeMarsh.Tree {
		switch cRef.Type {
		case "blob":
			child := c.Cache.GetBlob(cRef.Sha)
			if child != nil {
				parent.SetChild(child)
				continue
			}

			blobJSON, err := c.Api.GetBlob(repo.Owner.Login, repo.Name, cRef.Sha)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to get blob: %s", err))
			}

			child, err = NewBlobFromJSON(blobJSON)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to create blob object from JSON: %s", err))
			}

			child.Path = cRef.Path

			c.Cache.WriteBlob(child)
			parent.SetChild(child)

		case "tree":

			child := c.Cache.GetTree(cRef.Sha)
			if child != nil {
				parent.SetChild(child)
				continue
			}

			treeJSON, err := c.Api.GetTree(repo.Owner.Login, repo.Name, cRef.Sha)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to get tree: %s", err))
			}

			child, err = NewTreeFromJSON(treeJSON)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to create tree object from JSON: %s", err))
			}

			child.Path = cRef.Path

			c.Cache.WriteTree(child)
			parent.SetChild(child)

			var newTreeMarsh TreeMarshal
			err = json.Unmarshal(treeJSON, &newTreeMarsh)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to parse tree JSON: %s", err))
			}

			err = c.buildTree(child, newTreeMarsh, repo)
			if err != nil {
				return c.err.withMessage(fmt.Sprintf("failed to build tree: %s", err))
			}
		}
	}
	return nil
}

// GetTree checks cache for tree, else pulls tree from github
func (c *Client) GetTree(sha string, repo Repository) (*Tree, error) {
	t := c.Cache.GetTree(sha)

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

	c.Cache.WriteTree(top)
	err = c.buildTree(top, treeMarsh, repo)
	if err != nil {
		return nil, err
	}
	return top, nil
}

// WriteTreeToDirectory write tree to specified directory path
func WriteTreeToDirectory(top Node, basePath string) error {
	return recurseTreeAction(top,
		[]string{basePath},
		func(t *Tree, path []string) ([]string, error) {
			path = append(path, t.Path)
			pathStr := strings.Join(path, "/")
			err := os.MkdirAll(pathStr, 0777)
			if err != nil {
				return nil, err
			}
			return path, nil
		},
		func(b *Blob, path []string) ([]string, error) {
			path = append(path, b.Path)
			pathStr := strings.Join(path, "/")
			f, err := os.Create(pathStr)
			if err != nil {
				return nil, err
			}
			defer f.Close()

			ft := strings.Split(b.Path, ".")
			if ft[len(ft)-1] == "sh" {
				f.Chmod(0777)
			}
			b.Write(f)
			return path, nil
		})
}

func recurseTreeAction(top Node, metadata []string, treeAction func(*Tree, []string) ([]string, error), blobAction func(*Blob, []string) ([]string, error)) error {
	var newMetaData []string
	var err error
	switch v := top.(type) {
	case *Tree:
		newMetaData, err = treeAction(v, metadata)
		if err != nil {
			return err
		}
	case *Blob:
		newMetaData, err = blobAction(v, metadata)
		if err != nil {
			return err
		}
	}
	for _, child := range top.GetChildren() {
		err := recurseTreeAction(child, newMetaData, treeAction, blobAction)
		if err != nil {
			return err
		}
	}
	return nil
}
