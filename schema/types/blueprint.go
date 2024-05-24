package types

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"

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
	data, err := os.ReadFile(file)
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

// NewAny create a Blueprint
func NewAny(data interface{}) (Blueprint, error) {
	blueprint := New()
	bytes, err := json.Marshal(data)
	if err != nil {
		return Blueprint{}, err
	}

	err = jsoniter.Unmarshal(bytes, &blueprint)
	return blueprint, err
}

// NewColumnAny create a Column of Blueprint
func NewColumnAny(data interface{}) (Column, error) {
	column := Column{}
	bytes, err := json.Marshal(data)
	if err != nil {
		return Column{}, err
	}
	err = jsoniter.Unmarshal(bytes, &column)
	return column, err
}

// NewIndexAny create a index of Blueprint
func NewIndexAny(data interface{}) (Index, error) {
	index := Index{}
	bytes, err := json.Marshal(data)
	if err != nil {
		return Index{}, err
	}
	err = jsoniter.Unmarshal(bytes, &index)
	return index, err
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

// ColumnsMapping get the mapping of columns
func (blueprint Blueprint) ColumnsMapping() map[string]Column {
	mapping := map[string]Column{}
	for _, col := range blueprint.Columns {
		mapping[col.Hash()] = col
		mapping[col.Name] = col
	}
	return mapping
}

// IndexesMapping get the mapping of indexes
func (blueprint Blueprint) IndexesMapping() map[string]Index {
	mapping := map[string]Index{}
	for _, idx := range blueprint.Indexes {
		mapping[idx.Name] = idx
		mapping[idx.Hash()] = idx
	}
	return mapping
}

// Hash get the column hash
func (column Column) Hash() string {
	switch column.Type {
	case "enum":
		unique := fmt.Sprintf("%v|%v|%v", column.Type, column.Option, column.Comment)
		return hash(unique)
	default:
		unique := fmt.Sprintf("%v|%v", column.Type, column.Comment)
		return hash(unique)
	}
}

// Hash get the index hash
func (index Index) Hash() string {
	unique := fmt.Sprintf("%v|%v", index.Columns, index.Type)
	return hash(unique)
}

func hash(s string) string {
	h := md5.New()
	return string(h.Sum([]byte(s)))
}
