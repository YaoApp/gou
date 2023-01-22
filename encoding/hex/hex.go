package hex

import (
	"encoding/hex"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessEncode hex Encode
func ProcessEncode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := []byte(fmt.Sprintf("%v", process.Args[0]))
	return hex.EncodeToString(data)
}

// ProcessDecode hex Decode
func ProcessDecode(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	str := process.ArgsString(0)
	data, err := hex.DecodeString(str)
	if err != nil {
		exception.New("Base64 decode error: %s", 500, err).Throw()
	}
	return string(data)
}
