package application

import "net/http"

// Application the application interface
type Application interface {
	Walk(path string, handler func(root, filename string, isdir bool) error, patterns ...string) error
	Read(name string) ([]byte, error)
	Write(name string, content []byte) error
	Remove(name string) error
	Exists(name string) (bool, error)
	Watch(handler func(event string, name string), interrupt chan uint8) error
	Glob(pattern string) (matches []string, err error)

	Root() string
	FS(root string) http.FileSystem
}

// Pack the application pack
type Pack struct {
	Name         string            `json:"name,omitempty"`
	Environments map[string]string `json:"environments,omitempty"`
}
