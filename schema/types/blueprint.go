package types

import (
	"io/ioutil"

	jsoniter "github.com/json-iterator/go"
)

// New  create a Blueprint
func New() Blueprint {
	return Blueprint{
		Columns: []Column{},
		Indexes: []Index{},
		Option:  BlueprintOption{},
	}
}

// NewFile create a Blueprint by File
func NewFile(file string) (Blueprint, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return Blueprint{}, nil
	}
	return NewJSON(data)
}

// NewJSON create a Blueprint by JSON text
func NewJSON(data []byte) (Blueprint, error) {
	blueprint := New()
	err := jsoniter.Unmarshal(data, &blueprint)
	return blueprint, err
}

// NewMap create a Blueprint by map string
func NewMap(data map[string]interface{}) (Blueprint, error) {
	return New(), nil
}

// NewDiff create a new Diff
func NewDiff() Diff {
	diff := Diff{}
	diff.Columns.Add = []Column{}
	diff.Columns.Del = []Column{}
	diff.Columns.Alt = []Column{}
	diff.Indexes.Add = []Index{}
	diff.Indexes.Del = []Index{}
	diff.Indexes.Alt = []Index{}
	diff.Option = map[string]bool{}
	return diff
}
