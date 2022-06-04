package dsl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// Config get the workshop config
func Config() (WorkshopConfig, error) {
	root, err := ConfigRoot()
	if err != nil {
		return nil, err
	}

	file := filepath.Join(root, "workshop.yao")
	exists, _ := FileExists(file)
	if !exists {
		return WorkshopConfig{}, nil
	}

	data, err := FileGetJSON(file)
	if err != nil {
		return nil, err
	}

	cfg := WorkshopConfig{}
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

// Setup github config
func (cfg WorkshopConfig) github() error {
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
