package dsl

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/dsl/u"
)

// New create a new YAO DSL
func New() *YAO {
	return &YAO{
		Head:    NewHead(),
		Content: map[string]interface{}{},
	}
}

// Open open YAO DSL file
func (yao *YAO) Open(file string) error {
	bytes, err := u.FileGetJSON(file)
	if err != nil {
		return err
	}
	err = yao.Head.SetFile(file)
	if err != nil {
		return err
	}

	err = yao.Source(bytes)
	if err != nil {
		return err
	}
	return nil
}

// Source load DSL from source
func (yao *YAO) Source(source []byte) error {

	if yao.Head.Type == 0 {
		return fmt.Errorf("please set the type first")
	}

	err := jsoniter.Unmarshal(source, yao)
	if err != nil {
		return err
	}

	return nil
}

// Save export jsonc text and save to file
func (yao *YAO) Save() error { return nil }

// SaveAs export jsonc text and save to file
func (yao *YAO) SaveAs(file string) error { return nil }

// Bytes to bytes
func (yao *YAO) Bytes() ([]byte, error) { return []byte{}, nil }

// Download download the JSON file from workshop to vendor
func (yao *YAO) Download() {}

// UnmarshalJSON for json
func (yao *YAO) UnmarshalJSON(data []byte) error {
	content, err := u.ToMap(data)
	if err != nil {
		return err
	}

	yao.Content = content

	if yao.Head == nil {
		yao.Head = NewHead()
	}

	yao.Head.SetFrom(content["FROM"])

	err = yao.Head.SetLang(content["LANG"])
	if err != nil {
		return err
	}

	err = yao.Head.SetVersion(content["VERSION"])
	if err != nil {
		return err
	}

	err = yao.Head.SetCommand(content["RUN"])
	if err != nil {
		return err
	}

	return nil
}
