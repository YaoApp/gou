package yaz

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
	"golang.org/x/crypto/md4"
)

var defaultPatterns = []string{"*.yao", "*.json", "*.jsonc", "*.yaml", "*.so", "*.dll", "*.js", "*.py", "*.ts", "*.wasm"}

// Open opens a package file.
func Open(reader io.Reader, file string, cipher Cipher, cache ...bool) (*Yaz, error) {

	// uncompress
	path, err := uncompressYaz(reader, file, cache...)
	if err != nil {
		return nil, err
	}

	yaz := &Yaz{
		cipher: cipher,
		root:   path,
	}

	return yaz, nil
}

// OpenCache opens a package file from cache
func OpenCache(file string, cipher Cipher) (*Yaz, error) {

	path, err := cachePath(file)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		yaz := &Yaz{
			cipher: cipher,
			root:   path,
		}
		return yaz, nil
	}

	return nil, fmt.Errorf("%s not found cache", file)
}

// OpenFile opens a package file.
func OpenFile(file string, cipher Cipher, cache ...bool) (*Yaz, error) {

	// uncompress
	path, err := uncompressYazFile(file, cache...)
	if err != nil {
		return nil, err
	}

	yaz := &Yaz{
		cipher: cipher,
		root:   path,
	}

	return yaz, nil
}

// Glob searches for files in the package.
func (yaz *Yaz) Glob(pattern string) (matches []string, err error) {
	patternAbs, err := yaz.abs(pattern)
	if err != nil {
		return nil, err
	}
	matches, err = filepath.Glob(patternAbs)
	if err != nil {
		return nil, err
	}
	for i, match := range matches {
		matches[i] = strings.TrimPrefix(match, yaz.root)
	}
	return matches, nil
}

// Walk walks the package file.
func (yaz *Yaz) Walk(root string, handler func(root, filename string, isdir bool) error, patterns ...string) error {

	rootAbs, err := yaz.abs(root)
	if err != nil {
		return err
	}

	if patterns == nil {
		patterns = defaultPatterns
	}

	return filepath.Walk(rootAbs, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("[yaz.Walk] %s %s", filename, err.Error())
			return err
		}

		isdir := info.IsDir()
		if patterns != nil && !isdir && len(patterns) > 0 && patterns[0] != "-" {
			notmatched := true
			basname := filepath.Base(filename)
			for _, pattern := range patterns {
				if matched, _ := filepath.Match(pattern, basname); matched {
					notmatched = false
					break
				}
			}

			if notmatched {
				return nil
			}
		}

		name := strings.TrimPrefix(filename, rootAbs)
		if name == "" && isdir {
			name = string(os.PathSeparator)
		}

		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "/.") || strings.HasPrefix(name, "\\.") {
			return nil
		}

		if !isdir {
			name = filepath.Join(root, name)
		}

		err = handler(root, name, isdir)
		if filepath.SkipDir == err || filepath.SkipAll == err {
			return err
		}

		if err != nil {
			log.Error("[yaz.Walk] %s %s", filename, err.Error())
			return err
		}

		return nil
	})
}

// Read reads a file from the package.
func (yaz *Yaz) Read(name string) ([]byte, error) {

	file, err := yaz.abs(name)
	if err != nil {
		return nil, err
	}

	// decrypt file
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	if !yaz.cipherIsNull() && encryptFiles[ext] {
		reader, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		buff := &bytes.Buffer{}
		err = yaz.cipher.Decrypt(reader, buff)
		if err != nil {
			return nil, err
		}
		return buff.Bytes(), nil
	}

	return os.ReadFile(file)
}

// Write writes a file to the package.
func (yaz *Yaz) Write(name string, content []byte) error {
	return fmt.Errorf("yaz is a read only filesystem")
}

// Remove removes a file from the package.
func (yaz *Yaz) Remove(name string) error {
	return fmt.Errorf("yaz is a read only filesystem")
}

// Exists checks if a file exists in the package.
func (yaz *Yaz) Exists(name string) (bool, error) {

	file, err := yaz.abs(name)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Watch watches the package file.
func (yaz *Yaz) Watch(handler func(event string, name string), interrupt chan uint8) error {
	return fmt.Errorf("yaz does not support watch")
}

// abs returns the absolute path of the file.
func (yaz *Yaz) abs(root string) (string, error) {
	root = filepath.Join(yaz.root, root)
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return root, nil
}

func uncompressYaz(reader io.Reader, file string, cache ...bool) (string, error) {

	loadCache := false

	if len(cache) == 0 {
		loadCache = true
	}

	// load from cache
	if len(cache) > 0 {
		loadCache = cache[0]
	}

	if loadCache {

		path, err := cachePath(file)
		if err != nil {
			return Uncompress(reader)
		}

		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo.IsDir() {
			return path, nil
		}

		err = UncompressTo(reader, path)
		if err != nil {
			return "", err
		}

		return path, nil
	}

	// uncompress
	return Uncompress(reader)
}

func uncompressYazFile(file string, cache ...bool) (string, error) {

	loadCache := false

	if len(cache) == 0 {
		loadCache = true
	}

	// load from cache
	if len(cache) > 0 {
		loadCache = cache[0]
	}

	if loadCache {

		path, err := cachePath(file)
		if err != nil {
			return UncompressFile(file)
		}

		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo.IsDir() {
			return path, nil
		}

		err = UncompressFileTo(file, path)
		if err != nil {
			return "", err
		}

		return path, nil
	}

	// uncompress
	return UncompressFile(file)
}

func cachePath(file string) (string, error) {

	// get file info
	fileInfo, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	modTime := fileInfo.ModTime()

	// cache
	hash := md4.New()
	data := fmt.Sprintf("%s.%d", file, modTime.UnixMilli())

	io.WriteString(hash, data)
	name := fmt.Sprintf("%x", hash.Sum(nil))
	cacheRoot, err := os.UserHomeDir()
	if err != nil {
		cacheRoot = os.TempDir()
	}

	return filepath.Join(cacheRoot, ".yaoapps", "cache", name), nil
}

func (yaz *Yaz) cipherIsNull() bool {
	return cipherIsNull(yaz.cipher)
}
