package yaz

import "io"

// Yaz is a type that represents a package.
type Yaz struct {
	root   string // the app root (temp dir)
	cipher Cipher // the cipher interface
}

// Cipher is a type that represents a cipher.
type Cipher interface {
	Encrypt(reader io.Reader, writer io.Writer) error
	Decrypt(reader io.Reader, writer io.Writer) error
}
