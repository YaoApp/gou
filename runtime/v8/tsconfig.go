package v8

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
)

// GetFileName get the file name from the tsconfig
func (tsconfg *TSConfig) GetFileName(path string) (string, bool, error) {
	if tsconfg == nil {
		return path, false, nil
	}

	if tsconfg.CompilerOptions == nil || tsconfg.CompilerOptions.Paths == nil {
		return path, false, nil
	}

	for pattern, paths := range tsconfg.CompilerOptions.Paths {

		match, err := filepath.Match(pattern, path)
		if err != nil {
			return path, false, nil
		}

		if match {
			f := tsconfg.ReplacePattern(path, pattern)
			for _, p := range paths {
				matches, err := application.App.Glob(p)
				if err != nil {
					return path, false, err
				}
				for _, file := range matches {
					if strings.HasSuffix(file, f) {
						return file, true, nil
					}
				}
			}
		}

	}

	return path, false, nil
}

// ReplacePattern replace the pattern
func (tsconfg *TSConfig) ReplacePattern(path, pattern string) string {
	dir := filepath.Clean(filepath.Dir(path)) + string(os.PathSeparator)
	file := strings.TrimLeft(path, dir)
	if strings.HasSuffix(file, ".ts") {
		return file
	}
	return file + ".ts"
}
