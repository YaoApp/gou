package json

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessEncode json Encode
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	res, err := jsoniter.Marshal(process.Args[0])
	if err != nil {
		exception.New("JSON decode error: %s", 500, err).Throw()
	}
	return string(res)
}

// ProcessDecode json Decode
func ProcessDecode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := []byte(process.ArgsString(0))
	var res interface{}
	err := jsoniter.Unmarshal(data, &res)
	if err != nil {
		exception.New("Base64 decode error: %s", 500, err).Throw()
	}
	return res
}
