package yaz

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

var ignorePatterns = map[string]bool{"/data": true, "/db": true, "/logs": true}

// Build a package
func Build(root string) (string, error) {
	return compress(root)
}

// BuildSaveTo builds a package and saves it to a file
func BuildSaveTo(root string, output string) error {

	path, err := Build(root)
	if err != nil {
		return err
	}

	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("file %s already exists", output)
	}

	dir := filepath.Dir(output)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	err = os.Rename(path, output)
	if err != nil {
		return err
	}

	return nil
}

// Encrypt a package
func Encrypt(cipher Cipher, reader io.Reader, writer io.Writer) error {
	return cipher.Encrypt(reader, writer)
}

// Decrypt a package
func Decrypt(cipher Cipher, reader io.Reader, writer io.Writer) error {
	return cipher.Decrypt(reader, writer)
}

// EncryptBytes encrypts a byte slice
func EncryptBytes(cipher Cipher, data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	err := Encrypt(cipher, reader, writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

// EncryptFile encrypts a file
func EncryptFile(cipher Cipher, file string, output string) error {

	reader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(output)
	if err != nil {
		return err
	}
	defer writer.Close()

	return Encrypt(cipher, reader, writer)
}

// DecryptBytes decrypts a byte slice
func DecryptBytes(cipher Cipher, data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	err := Decrypt(cipher, reader, writer)
	if err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

// DecryptFile decrypts a file
func DecryptFile(cipher Cipher, file string, output string) error {
	reader, err := os.Open(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(output)
	if err != nil {
		return err
	}

	defer writer.Close()
	return Decrypt(cipher, reader, writer)
}

// compress compresses the package
func compress(root string) (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		return "", err
	}

	tarPath := filepath.Join(tempDir, "application.yaz")
	tarFile, err := os.Create(tarPath)
	if err != nil {
		panic(err)
	}
	defer tarFile.Close()

	gz := gzip.NewWriter(tarFile)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if ignorePatterns[relPath] {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = relPath
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return tarPath, nil
}
