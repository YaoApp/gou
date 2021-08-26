package helper

import (
	"bytes"
	"io"

	jsoniter "github.com/json-iterator/go"
)

// 常用函数

// UnmarshalFile JSON Unmarshal
func UnmarshalFile(file io.Reader, v interface{}) error {
	content, err := ReadFile(file)
	if err != nil {
		return err
	}

	return jsoniter.Unmarshal(content, v)
}

// ReadFile 读取文件内容
func ReadFile(file io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadFileString 读取文件内容, 返回String
func ReadFileString(file io.Reader) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(file)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
