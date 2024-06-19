package yaml

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"gopkg.in/yaml.v3"
)

// ProcessEncode json Encode
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	res, err := yaml.Marshal(process.Args[0])
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
	err := yaml.Unmarshal(data, &res)
	if err != nil {
		exception.New("YAML decode error: %s", 500, err).Throw()
	}
	return res
}
