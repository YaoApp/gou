package dsl

import (
	"bytes"
	"encoding/json"

	"github.com/yaoapp/gou/fs/system"
)

// File DSL File
type File struct{ *system.File }

// New create a new File struct
func New(root ...string) *File {
	systemFile := system.New(root...)
	return &File{File: systemFile}
}

// WriteFile writes data to the named file, creating it if necessary.
//
//	If the file does not exist, WriteFile creates it with permissions perm (before umask); otherwise WriteFile truncates it before writing, without changing permissions.
func (f *File) WriteFile(file string, data []byte, pterm int) (int, error) {
	data, err := f.Fmt(data)
	if err != nil {
		return 0, err
	}
	pterm = 0644
	return f.File.WriteFile(file, data, pterm)
}

// Fmt data
func (f *File) Fmt(data []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, data, "", "  ")
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
