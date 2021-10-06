package gou

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
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

// NumOfArgs 参数数量
func (process *Process) NumOfArgs() int {
	return len(process.Args)
}

// NumOfArgsIs 参数数量是否等以给等值
func (process *Process) NumOfArgsIs(num int) bool {
	return len(process.Args) == num
}

// ArgsNotNull 查询参数不能为空
func (process *Process) ArgsNotNull(i int) {
	if process.Args[i] == nil || len(process.Args) <= i {
		exception.New("第%d个查询参数不能为空 %v", 400, i, process.Args[0]).Throw()
	}
}

// ArgsURLValue 读取参数
func (process *Process) ArgsURLValue(i int, name string, defaults ...string) string {
	value := ""
	if len(defaults) > 0 {
		value = defaults[0]
	}
	switch process.Args[i].(type) {
	case url.Values:
		if _, has := process.Args[i].(url.Values)[name]; !has {
			return value
		}
		return process.Args[i].(url.Values).Get(name)
	case map[string]string:
		v, has := process.Args[i].(map[string]string)[name]
		if !has {
			return value
		}
		return v
	case map[string]interface{}:
		v, has := process.Args[i].(map[string]interface{})[name]
		if !has {
			return value
		}
		return fmt.Sprintf("%s", v)
	case maps.Map:
		if _, has := process.Args[i].(url.Values)[name]; !has {
			return value
		}
		return fmt.Sprintf("%s", process.Args[i].(maps.Map).Get(name))
	default:
		return value
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

// ArgsMap 读取Map 类型参数
func (process *Process) ArgsMap(i int, defaults ...maps.MapStrAny) maps.MapStrAny {
	value := maps.Map{}
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if process.Args[i] == nil || len(process.Args) <= i {
		return value
	}

	value, ok = process.Args[i].(maps.Map)
	if !ok {
		value = any.Of(process.Args[i]).Map().MapStrAny
	}
	return value
}

// ArgsBool 读取参数 String
func (process *Process) ArgsBool(i int, defaults ...bool) bool {
	value := false
	ok := false
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if process.Args[i] == nil || len(process.Args) <= i {
		return value
	}

	value, ok = process.Args[i].(bool)
	if !ok {
		value = any.Of(process.Args[i]).CBool()
	}
	return value
}
