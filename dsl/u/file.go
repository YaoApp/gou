package u

import (
	"errors"
	"os"
)

// FileExists trans JSONC to JSON
func FileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
