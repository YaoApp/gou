package json

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/watchfultele/jsonrepair"
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
	data := process.ArgsString(0)
	var res interface{}
	err := jsoniter.UnmarshalFromString(data, &res)
	if err != nil {
		exception.New("JSON decode error: %s", 500, err).Throw()
	}
	return res
}

// ProcessRepair json Repair
func ProcessRepair(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsString(0)
	repaired, err := jsonrepair.JSONRepair(data)
	if err != nil {
		exception.New("JSON repair error: %s", 500, err).Throw()
	}
	return string(repaired)
}

// ProcessParse json Parse
func ProcessParse(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsString(0)
	var res interface{}
	err := jsoniter.UnmarshalFromString(data, &res)
	if err != nil {
		repaired, errRepair := jsonrepair.JSONRepair(data)
		if errRepair != nil {
			exception.New("JSON parse error: %s", 500, err).Throw()
		}

		// Retry
		errRepair = jsoniter.UnmarshalFromString(repaired, &res)
		if errRepair != nil {
			exception.New("JSON parse error: %s", 500, err).Throw()
		}
	}
	return res
}
