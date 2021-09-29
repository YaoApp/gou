package gou

import (
	"fmt"

	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// ValidateArgNums 校验参数数量( args )
func (process *Process) ValidateArgNums(length int) {
	if len(process.Args) < length {
		exception.New(
			fmt.Sprintf("Model:%s%s(args...); 缺少查询参数", process.Class, process.Name),
			400,
		).Throw()
	}
}

// ArgsNotNull 查询参数不能为空
func (process *Process) ArgsNotNull(i int) {
	if process.Args[i] == nil || len(process.Args) <= i {
		exception.New("第%d个查询参数不能为空 %v", 400, i, process.Args[0]).Throw()
	}
}

// ArgsQueryParams 读取参数 QueryParam
func (process *Process) ArgsQueryParams(i int, defaults ...QueryParam) QueryParam {
	param := QueryParam{}
	if len(defaults) > 0 {
		param = defaults[0]
	}

	if process.Args[i] == nil || len(process.Args) <= i {
		return param
	}

	param, ok := process.Args[i].(QueryParam)
	if !ok {
		param, ok = AnyToQueryParam(process.Args[i])
	}

	return param
}

// ArgsInt 读取参数 Int
func (process *Process) ArgsInt(i int, defaults ...int) int {
	value := 0
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if process.Args[i] == nil || len(process.Args) <= i {
		return value
	}

	value, ok = process.Args[i].(int)
	if !ok {
		value = any.Of(process.Args[i]).CInt()
	}

	return value
}

// ArgsString 读取参数 String
func (process *Process) ArgsString(i int, defaults ...string) string {
	value := ""
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if process.Args[i] == nil || len(process.Args) <= i {
		return value
	}

	value, ok = process.Args[i].(string)
	if !ok {
		value = any.Of(process.Args[i]).CString()
	}
	return value
}
