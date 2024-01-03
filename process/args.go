package process

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"

	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/gou/types"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// ValidateArgNums Validate the parameters numbers
func (process *Process) ValidateArgNums(length int) {
	if len(process.Args) < length {
		exception.New("%s Missing parameters, %d parameters are required", 400, process.Name, length).Throw()
	}
}

// NumOfArgs get the parameters numbers
func (process *Process) NumOfArgs() int {
	return len(process.Args)
}

// NumOfArgsIs check if the number of parameters is the given value
func (process *Process) NumOfArgsIs(num int) bool {
	return len(process.Args) == num
}

// ArgsNotNull parameters should be not null
func (process *Process) ArgsNotNull(i int) {
	if len(process.Args) <= i || process.Args[i] == nil {
		exception.New("%s The %d parameter cannot be empty", 400, process.Name, i).Throw()
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
func (process *Process) ArgsQueryParams(i int, defaults ...types.QueryParam) types.QueryParam {
	param := types.QueryParam{}
	if len(defaults) > 0 {
		param = defaults[0]
	}

	if len(process.Args) <= i || process.Args[i] == nil {
		return param
	}

	param, ok := process.Args[i].(types.QueryParam)
	if !ok {
		param, ok = types.AnyToQueryParam(process.Args[i])
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

	if len(process.Args) <= i || process.Args[i] == nil {
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

// ArgsUint32 get Uint32
func (process *Process) ArgsUint32(i int, defaults ...uint32) uint32 {
	value := uint32(0)
	if len(defaults) > 0 {
		value = defaults[0]
	}

	if len(process.Args) <= i || process.Args[i] == nil {
		return value
	}

	switch v := process.Args[i].(type) {

	case uint32:
		return v

	case string:
		val, err := strconv.ParseInt(v, 0, 32)
		if err != nil {
			return value
		}
		return uint32(val)

	case int:
		return uint32(v)

	case int8:
		return uint32(v)

	case int16:
		return uint32(v)

	case int32:
		return uint32(v)

	case int64:
		return uint32(v)

	case uint:
		return uint32(v)

	case uint8:
		return uint32(v)

	case uint16:
		return uint32(v)

	case uint64:
		return uint32(v)

	case float32:
		return uint32(v)

	case float64:
		return uint32(v)
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

	if len(process.Args) <= i || process.Args[i] == nil {
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

	if len(process.Args) <= i || process.Args[i] == nil {
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
	switch values := columnsAny.(type) {
	case []interface{}:
		for _, v := range values {
			value, ok := v.(string)
			if ok {
				columns = append(columns, value)
				continue
			}
			exception.New("参数错误: 第%d个参数不是字符串数组", 400, index+1).Ctx(process.Args[index]).Throw()
		}
		break

	case []string:
		columns = values
		break

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
