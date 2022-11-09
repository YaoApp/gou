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
func (f *File) WriteFile(file string, data []byte, perm int) (int, error) {
	data, err := f.Fmt(data)
	if err != nil {
		return 0, err
	}
	perm = 0644
	return f.File.WriteFile(file, data, perm)
}

// Allow allow rel path
func (f *File) Allow(patterns ...string) *File {
	f.File.Allow(patterns...)
	return f
}

// Deny deny rel path
func (f *File) Deny(patterns ...string) *File {
	f.File.Deny(patterns...)
	return f
}

// AllowAbs allow abs path
func (f *File) AllowAbs(patterns ...string) *File {
	f.File.AllowAbs(patterns...)
	return f
}

// DenyAbs deny abs path
func (f *File) DenyAbs(patterns ...string) *File {
	f.File.DenyAbs(patterns...)
	return f
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
