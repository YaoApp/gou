package disk

import (
	"net/http"
	"os"
	"path/filepath"
)

// Dir is the public path
type Dir string

// Root return the root path
func (disk *Disk) Root() string {
	return disk.root
}

// FS returns a http.FileSystem that serves files from the public directory
func (disk *Disk) FS(root string) http.FileSystem {
	return Dir(filepath.Join(disk.root, root))
}

// Open implements FileSystem using os.Open, opening files for reading rooted
func (dir Dir) Open(name string) (http.File, error) {
	return os.Open(filepath.Join(string(dir), name))
}
