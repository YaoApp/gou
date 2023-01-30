package widget

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
)

// Walk the path
func Walk(root string, extensions []string, cb func(root, filename string) error) error {

	exts := []string{}
	for _, ext := range extensions {
		exts = append(exts, fmt.Sprintf("*.%s", ext))
	}

	err := application.App.Walk(root, func(root, filename string, isdir bool) error {
		if isdir {
			return nil
		}
		cb(root, filename)
		return nil
	}, exts...)
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
