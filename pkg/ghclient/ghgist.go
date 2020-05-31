package ghclient

import "strings"

// File represents file object in github gist
type File struct {
	Content string `json:"content"`
}

// Gist github gist object. Contains collection of files
type Gist struct {
	Description string           `json:"description"`
	Public      bool             `json:"public"`
	Files       map[string]*File `json:"files"`
}

// NewGist gist factory
func NewGist() Gist {
	return Gist{
		Public: true,
		Files:  make(map[string]*File),
	}
}

// WriteFile appends content to gist file. Create new
// file if it does not already exist
func (g *Gist) WriteFile(name, content string) {
	var sb strings.Builder
	if f, found := g.Files[name]; found {
		sb.WriteString(f.Content)
		sb.WriteString(content)
		f.Content = sb.String()
		return
	}

	g.Files[name] = &File{
		Content: content,
	}
}
