package fs

import (
	"io"
	"time"
)

// FileSystem the filesystem io interface
type FileSystem interface {
	ReadFile(file string) ([]byte, error)
	WriteFile(file string, data []byte, perm uint32) (int, error)
	Write(file string, reader io.Reader, perm uint32) (int, error)

	ReadDir(dir string, recursive bool) ([]string, error)
	Mkdir(dir string, perm uint32) error
	MkdirAll(dir string, perm uint32) error
	MkdirTemp(dir string, pattern string) (string, error)
	Glob(pattern string) ([]string, error)

	Remove(name string) error
	RemoveAll(name string) error

	Exists(name string) (bool, error)
	Size(name string) (int, error)
	Mode(name string) (uint32, error)
	ModTime(name string) (time.Time, error)

	Chmod(name string, mode uint32) error
	IsDir(name string) bool
	IsFile(name string) bool
	IsLink(name string) bool

	Move(oldpath string, newpath string) error
	Copy(src string, dest string) error

	MimeType(name string) (string, error)

	Root() string

	Walk(path string, handler func(root, filename string, isdir bool) error, patterns ...string) error

	List(path string, types []string, page, pageSize int, filter func(string) bool) ([]string, int, int, error)

	Resize(inputPath, outputPath string, width, height uint) error

	CleanCache()
}
