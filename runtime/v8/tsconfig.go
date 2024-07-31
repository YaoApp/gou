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
		if tsconfg.Match(pattern, path) {
			f := tsconfg.ReplacePattern(path, pattern)
			for _, p := range paths {
				matched := false
				dir := filepath.Clean(filepath.Dir(p))
				f = filepath.Join(dir, f)
				err := application.App.Walk(dir, func(root, filename string, isdir bool) error {
					if isdir {
						return nil
					}
					if filename == f {
						matched = true
						return filepath.SkipAll
					}
					return nil
				}, "*.ts")

				if matched {
					return f, true, nil
				}

				if err == nil {
					return path, false, nil
				}
			}
		}
	}
	return path, false, nil
}

// Match match the pattern
func (tsconfg *TSConfig) Match(pattern, path string) bool {
	prefix := strings.Split(pattern, "/*")[0] + string(os.PathSeparator)
	return strings.HasPrefix(path, prefix)
}

// ReplacePattern replace the pattern
func (tsconfg *TSConfig) ReplacePattern(path, pattern string) string {
	prefix := strings.Split(pattern, "/*")[0]
	file := strings.TrimPrefix(path, prefix)
	if strings.HasSuffix(file, ".ts") {
		return file
	}
	return file + ".ts"
}
