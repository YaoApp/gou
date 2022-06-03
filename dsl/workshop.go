package dsl

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
)

// Tidy scan the source and update workshop.yao then auto-generation the workshop.sum.yao file
func Tidy(root string) error { return nil }

// Format scan the source and format DSL code
func Format(root string) error { return nil }

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

// WorkshopRoot get the workshop root in the local disk
func WorkshopRoot() (string, error) {
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

// OpenWorkshop open and parse the workshop dsl
func OpenWorkshop(root string) (*Workshop, error) {

	file := path.Join(root, "workshop.yao")
	data, err := FileGetJSON(file)
	if err != nil {
		return nil, err
	}

	workshop := &Workshop{Mapping: map[string]Package{}}
	err = jsoniter.Unmarshal(data, workshop)
	if err != nil {
		return nil, err
	}

	err = workshop.SetMapping()
	if err != nil {
		return nil, err
	}

	return workshop, nil
}

// SetMapping mapping alias and package
func (workshop *Workshop) SetMapping() error {

	for i, pkg := range workshop.Require {

		// Check name
		if _, has := workshop.Mapping[pkg.Alias]; has {
			return fmt.Errorf(
				"\"%s\" and \"%s\" has the same name \"%s\", please change it",
				workshop.Mapping[pkg.Alias].URL, pkg.URL, pkg.Alias,
			)
		}

		// Replace
		pkgpath := filepath.Join(pkg.Repo, pkg.Path)
		if path, has := workshop.Replace[pkgpath]; has {

			localpath, err := filepath.Abs(path)
			if err != nil {
				return err
			}

			if _, err = os.Stat(localpath); err != nil {
				return err
			}

			if _, err = os.Stat(filepath.Join(localpath, "app.yao")); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%s is not YAO application", localpath)
				}
				return err
			}
			workshop.Require[i].Replaced = true
			workshop.Require[i].LocalPath = localpath
		}
		workshop.Mapping[pkg.Alias] = workshop.Require[i]
	}
	return nil
}

// Map workshop to map[string]interface{}
func (workshop Workshop) Map() map[string]interface{} {
	require := []map[string]interface{}{}
	for _, pkg := range workshop.Require {
		require = append(require, pkg.Map())
	}
	return map[string]interface{}{
		"mapping": workshop.Mapping,
		"replace": workshop.Replace,
		"require": require,
	}
}

// Validate the packages
func (workshop *Workshop) Validate() {}

// Add add a repo to workshop.ayo
func (workshop *Workshop) Add(repo string, alias string) error {
	return nil
}

// Del delete a repo from workshop.yao
func (workshop *Workshop) Del(repo string) error {
	return nil
}

// Get the repo from the given remote repo
func Get(repo string) {}
