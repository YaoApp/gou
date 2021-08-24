package helper

import (
	"bytes"
	"encoding/json"
	"io/fs"
)

// 常用函数

// UnmarshalFile JSON Unmarshal
func UnmarshalFile(file fs.File, v interface{}) error {
	content, err := ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, v)
}

// ReadFile 读取文件内容
func ReadFile(file fs.File) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadFileString 读取文件内容, 返回String
func ReadFileString(file fs.File) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
