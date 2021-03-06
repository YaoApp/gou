package gou

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/yaoapp/gou/query/share"
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

	// 空值返回
	v, ok := process.Args[i].(string)
	if ok && v == "" {
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

	if len(process.Args) <= i || process.Args[i] == nil {
		return value
	}

	if values, ok := process.Args[i].(url.Values); ok {
		res := maps.Map{}
		for key, val := range values {
			if len(val) <= 1 {
				res[key] = values.Get(key)
				continue
			}
			res[key] = val
		}
		return res
	}

	if value, ok = process.Args[i].(maps.Map); ok {
		return value
	}

	return any.Of(process.Args[i]).Map().MapStrAny
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

// ArgsRecords 读取 []map[string]interface{} 参数值
func (process *Process) ArgsRecords(index int) []map[string]interface{} {
	process.ValidateArgNums(index + 1)
	records := []map[string]interface{}{}
	args := process.Args[index]

	if args == nil {
		exception.New("参数错误: 第%d个参数不能为空值", 400, index+1).Ctx(fmt.Sprintf("%#v", process.Args[index])).Throw()
	}

	switch args.(type) {
	case []interface{}:
		for _, v := range args.([]interface{}) {
			value, ok := v.(map[string]interface{})
			if ok {
				records = append(records, value)
				continue
			} else if value, ok := v.(maps.MapStrAny); ok {
				records = append(records, value)
				continue
			}
			exception.New("参数错误: 第%d个参数不是数组", 400, index+1).Ctx(fmt.Sprintf("%#v", process.Args[index])).Throw()
		}
		break
	case []maps.MapStrAny:
		for _, v := range args.([]maps.MapStrAny) {
			records = append(records, v)
		}
		break
	case []share.Record:
		for _, v := range args.([]share.Record) {
			records = append(records, v)
		}
		break
	case []map[string]interface{}:
		records = args.([]map[string]interface{})
		break
	default:
		fmt.Printf("%#v %s\n", args, reflect.TypeOf(args).Kind())
		exception.New("参数错误: 第%d个参数不是数组", 400, index+1).Ctx(fmt.Sprintf("%#v", process.Args[index])).Throw()
		break
	}
	return records
}

// ArgsStrings 读取 []string 参数值
func (process *Process) ArgsStrings(index int) []string {
	process.ValidateArgNums(index + 1)
	columnsAny := process.Args[index]
	columns := []string{}
	switch columnsAny.(type) {
	case []interface{}:
		for _, v := range columnsAny.([]interface{}) {
			value, ok := v.(string)
			if ok {
				columns = append(columns, value)
				continue
			}
			exception.New("参数错误: 第%d个参数不是字符串数组", 400, index+1).Ctx(process.Args[index]).Throw()
		}
	case []string:
	default:
		exception.New("参数错误: 第%d个参数不是字符串数组", 400, index+1).Ctx(process.Args[index]).Throw()
		break
	}
	return columns
}

// ArgsArray 读取 []interface{} 参数值
func (process *Process) ArgsArray(index int) []interface{} {
	process.ValidateArgNums(index + 1)
	return any.Of(process.Args[index]).CArray()
}
