package dsl

import (
	"path"

	jsoniter "github.com/json-iterator/go"
)

// Tidy scan the source and update workshop.yao then auto-generation the workshop.sum.yao file
func Tidy(root string) error { return nil }

// Format scan the source and format DSL code
func Format(root string) error { return nil }

// OpenWorkshop open and parse the workshop dsl
func OpenWorkshop(root string) (*Workshop, error) {

	file := path.Join(root, "workshop.yao")
	data, err := FileGetJSON(file)
	if err != nil {
		return nil, err
	}

	workshop := &Workshop{}
	err = jsoniter.Unmarshal(data, workshop)
	if err != nil {
		return nil, err
	}

	return workshop, nil
}

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

// Get the repo from the remote repo
func (pkg *Package) Get() {
}
