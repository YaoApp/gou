package workshop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/dsl/u"
)

// GetConfig get the workshop config
func GetConfig() (Config, error) {
	root, err := ConfigRoot()
	if err != nil {
		return nil, err
	}

	file := filepath.Join(root, "workshop.yao")
	exists, _ := u.FileExists(file)
	if !exists {
		return Config{}, nil
	}

	data, err := u.FileGetJSON(file)
	if err != nil {
		return nil, err
	}

	cfg := Config{}
	err = jsoniter.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.github()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// LocalRoot get the root path in the local disk
// the default is: ~/yao/
func LocalRoot() (string, error) {
	root := os.Getenv(RootEnvName)
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, "yao")
	}
	return root, nil
}

// Root get the workshop root in the local disk
func Root() (string, error) {
	root, err := LocalRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "workshop"), nil
}

// ConfigRoot get the workshop root in the local disk
func ConfigRoot() (string, error) {
	root, err := LocalRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config"), nil
}

// Setup github config
func (cfg Config) github() error {
	if _, has := cfg["github.com"]; !has {
		return nil
	}

	if _, has := cfg["github.com"]["token"]; !has {
		return nil
	}

	token, ok := cfg["github.com"]["token"].(string)
	if !ok {
		return fmt.Errorf("token should be a filepath, but got %v", token)
	}

	if strings.HasPrefix(token, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		token = fmt.Sprintf("%s%s", home, strings.TrimLeft(token, "~"))
	}

	file, err := filepath.Abs(token)
	if err != nil {
		return err
	}

	info, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("the token file does not found. %v", token)
	}

	if err != nil {
		return err
	}

	mode := info.Mode().String()
	if mode != "-r--------" && mode != "-rw-------" {
		return fmt.Errorf("the token file can be read by other users. \ntry: chmod 600 %v", token)
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return fmt.Errorf("the token %s is empty", token)
	}

	token = strings.Trim(string(content), "\n")
	cfg["github.com"]["token"] = token
	return nil
}
