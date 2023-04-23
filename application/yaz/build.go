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
	"reflect"
	"strings"

	"github.com/yaoapp/gou/application/ignore"
)

var ignores = []string{"data", "db", "logs", "tmp", "vendor", "dist", ".github", "plugins", "wasms", ".git", ".gitignore", ".gitmodules", ".gitattributes", ".gitkeep", ".gitlab-ci.yml"}
var encryptFiles = map[string]bool{"js": true, "yao": true, "jsonc": true, "json": true, "html": true, "htm": true, "css": true, "txt": true, "md": true, "go": true, "yml": true, "yaml": true, "xml": true, "conf": true, "ini": true, "toml": true, "sql": true, "tpl": true, "tmpl": true, "tmpl.html": true, "tmpl.js": true, "tmpl.css": true, "tmpl.txt": true, "tmpl.yml": true, "tmpl.yaml": true, "tmpl.xml": true, "tmpl.conf": true, "tmpl.ini": true, "tmpl.toml": true, "tmpl.sql": true, "tmpl.tpl": true, "tmpl.tmpl": true, "tmpl.tmpl.html": true, "tmpl.tmpl.js": true, "tmpl.tmpl.css": true, "tmpl.tmpl.txt": true, "tmpl.tmpl.yml": true, "tmpl.tmpl.yaml": true, "tmpl.tmpl.xml": true, "tmpl.tmpl.conf": true, "tmpl.tmpl.ini": true, "tmpl.tmpl.toml": true, "tmpl.tmpl.sql": true, "tmpl.tmpl.tpl": true, "tmpl.tmpl.tmpl": true}

// Pack a package
func Pack(root string, cipher Cipher) (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "pack-*")
	if err != nil {
		return "", err
	}

	target := filepath.Join(tempDir, "application.yaz")
	if err := compress(root, target, cipher); err != nil {
		return "", err
	}

	return target, nil
}

// PackTo packs a package and saves it to a file
func PackTo(root string, output string, cipher Cipher) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", output)
	}
	return compress(root, output, cipher)
}

// Unpack a package
func Unpack(file string, cipher Cipher) (string, error) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "unpack-*")
	if err != nil {
		return "", err
	}

	if err := uncompressFile(file, tempDir, cipher); err != nil {
		return "", err
	}
	return tempDir, nil
}

// UnpackTo unpacks a package to a directory
func UnpackTo(file string, output string, cipher Cipher) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", output)
	}
	return uncompressFile(file, output, cipher)
}

// Compress a package
func Compress(root string) (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "unpack-*")
	if err != nil {
		return "", err
	}

	target := filepath.Join(tempDir, "application.yaz")
	if err := compress(root, target, nil); err != nil {
		return "", err
	}

	return target, nil
}

// CompressTo builds a package and saves it to a file
func CompressTo(root string, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", output)
	}
	if err := compress(root, output, nil); err != nil {
		return err
	}
	return nil
}

// UncompressFile a package
func UncompressFile(file string) (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "uncompress-*")
	if err != nil {
		return "", err
	}

	if err := uncompressFile(file, tempDir, nil); err != nil {
		return "", err
	}
	return tempDir, nil
}

// UncompressFileTo uncompress a package to a specified directory.
func UncompressFileTo(file string, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", output)
	}
	return uncompressFile(file, output, nil)
}

// UncompressTo uncompress a package to a specified directory.
func UncompressTo(reader io.Reader, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return fmt.Errorf("directory %s already exists", output)
	}
	return uncompress(reader, output, nil)
}

// Uncompress a package
func Uncompress(reader io.Reader) (string, error) {

	tempDir, err := ioutil.TempDir(os.TempDir(), "uncompress-*")
	if err != nil {
		return "", err
	}

	if err := uncompress(reader, tempDir, nil); err != nil {
		return "", err
	}
	return tempDir, nil
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
func compress(root string, target string, cipher Cipher) error {

	tarFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gz := gzip.NewWriter(tarFile)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	// add gitignore patterns
	ignorePatterns := ignores
	gitignore := ignore.Compile(filepath.Join(root, ".gitignore"), ignorePatterns...)

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if gitignore.MatchesPath(relPath) {
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

			// Encrypt the file
			ext := strings.TrimPrefix(filepath.Ext(path), ".")

			if !cipherIsNull(cipher) && encryptFiles[ext] {

				dir := filepath.Dir(target)
				encryptFile := filepath.Join(dir, filepath.Base(path)+".enc")
				encryptWriter, err := os.Create(encryptFile)
				if err != nil {
					return err
				}
				defer encryptWriter.Close()
				defer os.Remove(encryptFile)

				err = cipher.Encrypt(file, encryptWriter)
				if err != nil {
					return err
				}

				encryptReader, err := os.Open(encryptFile)
				if err != nil {
					return err
				}
				defer encryptReader.Close()

				_, err = io.Copy(tw, encryptReader)
				return err
			}

			_, err = io.Copy(tw, file)
			return err
		}

		return nil
	})

	return err
}

func uncompressFile(file string, dest string, cipher Cipher) error {

	tarFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	return uncompress(tarFile, dest, cipher)
}

func uncompress(tarFile io.Reader, dest string, cipher Cipher) error {

	gzr, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}

		case tar.TypeReg:

			fileWriter, err := os.Create(target)
			if err != nil {
				return err
			}
			defer fileWriter.Close()

			ext := strings.TrimPrefix(filepath.Ext(target), ".")
			if !cipherIsNull(cipher) && encryptFiles[ext] {
				decryptFile := filepath.Join(dest, filepath.Base(target)+".dec")
				decryptWriter, err := os.Create(decryptFile)
				if err != nil {
					return err
				}
				defer decryptWriter.Close()
				defer os.Remove(decryptFile)

				if err := cipher.Decrypt(tarReader, decryptWriter); err != nil {
					return err
				}

				decryptReader, err := os.Open(decryptFile)
				if err != nil {
					return err
				}
				defer decryptReader.Close()

				if _, err = io.Copy(fileWriter, decryptReader); err != nil {
					return err
				}
				continue
			}

			if _, err := io.Copy(fileWriter, tarReader); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown type %d in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

func cipherIsNull(cipher Cipher) bool {

	if cipher == nil {
		return true
	}

	switch reflect.TypeOf(cipher).Kind() {
	case reflect.Ptr:
		return reflect.ValueOf(cipher).IsNil()
	}

	return false
}
