package widget

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
)

// Walk the path
func Walk(root string, typeName string, cb func(root, filename string) error) error {
	root = path.Join(root, "/")
	err := filepath.Walk(root, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			log.With(log.F{"root": root, "type": typeName, "filename": filename}).Error(err.Error())
			return err
		}
		if strings.HasSuffix(filename, typeName) {
			cb(root, filename)
		}
		return nil
	})
	return err
}

// InstName   root: "/tests/apis"  file: "/tests/apis/foo/bar.http.json"
func InstName(root string, file string) string {
	filename := strings.TrimPrefix(file, root+"/") // "foo/bar.http.json"
	namer := strings.Split(filename, ".")          // ["foo/bar", "http", "json"]
	nametypes := strings.Split(namer[0], "/")      // ["foo", "bar"]
	name := strings.Join(nametypes, ".")           // "foo.bar"
	return name
}

// DirNotExists validate the folder
func DirNotExists(dir string) bool {
	dir = strings.TrimPrefix(dir, "fs://")
	dir = strings.TrimPrefix(dir, "file://")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return true
	}
	return false
}
