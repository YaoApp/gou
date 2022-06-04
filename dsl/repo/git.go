package repo

import "fmt"

// Git API
type Git struct{}

// Content Get file Content via Github API
func (git *Git) Content(file string) ([]byte, error) {
	return nil, fmt.Errorf("self-host git repo not supported yet, using GitHub instead")
}

// Dir Get folders
func (git *Git) Dir(path string) ([]string, error) {
	return nil, fmt.Errorf("self-host git repo not supported yet, using GitHub instead")
}
