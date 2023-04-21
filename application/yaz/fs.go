package yaz

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
)

// Dir is the public path
type Dir struct {
	path   string
	cipher Cipher
}

// File is the file
type File struct {
	*os.File
	cipher Cipher
	buff   []byte
}

// Root return the root path
func (yaz *Yaz) Root() string {
	return yaz.root
}

// FS returns a http.FileSystem that serves files from the public directory
func (yaz *Yaz) FS(root string) http.FileSystem {
	return Dir{path: filepath.Join(yaz.root, root), cipher: yaz.cipher}
}

// Open implements FileSystem using os.Open, opening files for reading rooted
func (dir Dir) Open(name string) (http.File, error) {

	f, err := os.Open(filepath.Join(dir.path, name))
	if err != nil {
		return nil, err
	}

	// decrypt file
	buff := &bytes.Buffer{}
	if dir.cipher != nil {
		dir.cipher.Decrypt(f, buff)
	}

	return &File{File: f, buff: buff.Bytes(), cipher: dir.cipher}, nil
}

// Read reads up to len(p) bytes into p.
func (file *File) Read(p []byte) (n int, err error) {

	if file.cipher != nil {
		if len(file.buff) == 0 {
			return 0, nil
		}
		n = copy(p, file.buff)
		file.buff = file.buff[n:]
		return n, nil
	}

	return file.Read(p)
}

// Close closes the File, rendering it unusable for I/O.
func (file *File) Close() error {
	file.buff = nil
	return file.File.Close()
}
