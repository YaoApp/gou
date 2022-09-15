package helper

import (
	"bytes"
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
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

// EnvString replace $ENV.xxx with the env
func EnvString(key interface{}, defaults ...string) string {
	k, ok := key.(string)
	if !ok {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return ""
	}

	if ok && strings.HasPrefix(k, "$ENV.") {
		k = strings.TrimPrefix(k, "$ENV.")
		v := os.Getenv(k)
		if v == "" && len(defaults) > 0 {
			return defaults[0]
		}
		return v
	}
	return key.(string)
}

// EnvInt replace $ENV.xxx with the env and cast to the integer
func EnvInt(key interface{}, defaults ...int) int {
	if k, ok := key.(string); ok && strings.HasPrefix(k, "$ENV.") {
		k = strings.TrimPrefix(k, "$ENV.")
		v := os.Getenv(k)
		if v == "" {
			if len(defaults) > 0 {
				return defaults[0]
			}
			return 0
		}
		return any.Of(v).CInt()
	}

	v, ok := key.(int)
	if !ok {
		if len(defaults) > 0 {
			return defaults[0]
		}
		return any.Of(key).CInt()
	}
	return v
}
