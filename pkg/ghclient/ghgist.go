package ghclient

// File represents file object in github gist
type File struct {
	Content string `json:"content"`
}

// Gist github gist object. Contains collection of files
type Gist struct {
	Description string          `json:"description"`
	Public      bool            `json:"public"`
	Files       map[string]File `json:"files"`
}

// NewGist gist factory
func NewGist() Gist {
	return Gist{
		Public: true,
		Files:  make(map[string]File),
	}
}

// AddFile add file to gist
func (g *Gist) AddFile(name, content string) {
	g.Files[name] = File{
		Content: content,
	}
}
