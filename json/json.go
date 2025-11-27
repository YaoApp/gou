package json

import (
	jsoniter "github.com/json-iterator/go"
)

// Encode encodes the given value to JSON string
func Encode(v interface{}) (string, error) {
	res, err := jsoniter.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// Decode decodes JSON string to interface{}
func Decode(data string) (interface{}, error) {
	var res interface{}
	err := jsoniter.UnmarshalFromString(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// DecodeTyped decodes JSON string to the given typed pointer
func DecodeTyped(data string, v interface{}) error {
	return jsoniter.UnmarshalFromString(data, v)
}
