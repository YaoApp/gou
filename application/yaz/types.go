package yaz

import (
	"io"
)

// Package is a type that represents a package.
type Package struct{}

// Cipher is a type that represents a cipher.
type Cipher interface {
	Encrypt(reader io.Reader, writer io.Writer) error
	Decrypt(reader io.Reader, writer io.Writer) error
}
