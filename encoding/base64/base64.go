package base64

import (
	"encoding/base64"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// Package base64 implements base64 encoding as specified by RFC 4648.

// ProcessEncode base64 Encode
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := []byte(fmt.Sprintf("%v", process.Args[0]))
	return base64.StdEncoding.EncodeToString(data)
}

// ProcessDecode base64 Decode
func ProcessDecode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	str := process.ArgsString(0)
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		exception.New("Base64 decode error: %s", 500, err).Throw()
	}
	return string(data)
}

// processDecodeBinary
func processDecodeBinary(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	str := process.ArgsString(0)
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		exception.New("Base64 decode error: %s", 500, err).Throw()
	}
	return data
}
