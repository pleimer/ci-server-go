package ghclient

type file struct {
	Content string `json:"content"`
}

// Gist github gist object. Contains collection of files
type Gist struct {
	Description string          `json:"description"`
	Public      bool            `json:"public"`
	Files       map[string]file `json:"files"`
}

// NewGist gist factory
func NewGist() Gist {
	return Gist{
		Public: true,
		Files:  make(map[string]file),
	}
}

// AddFile add file to gist
func (g *Gist) AddFile(name, content string) {
	g.Files[name] = file{
		Content: content,
	}
}
