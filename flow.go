package gou

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
)

// Flows 已加载工作流列表
var Flows = map[string]*Flow{}

// LoadFlow 载入数据接口
func LoadFlow(source string, name string) *Flow {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	flow := Flow{
		Name:         name,
		Source:       source,
		Scripts:      map[string]string{},
		ScriptSource: map[string]string{},
	}
	err := helper.UnmarshalFile(input, &flow)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	Flows[name] = &flow
	return Flows[name]
}

// LoadScript 载入脚本
func (flow *Flow) LoadScript(source string, name string) *Flow {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	content, err := ioutil.ReadAll(input)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	flow.Scripts[name] = string(content)
	flow.ScriptSource[name] = source
	return flow
}

// Reload 重新载入API
func (flow *Flow) Reload() *Flow {
	new := LoadFlow(flow.Source, flow.Name)
	for name, source := range flow.ScriptSource {
		new.LoadScript(source, name)
	}
	flow = new
	Flows[flow.Name] = new
	return flow
}

// SelectFlow 读取已加载Flow
func SelectFlow(name string) *Flow {
	flow, has := Flows[name]
	if !has {
		exception.New(
			fmt.Sprintf("Flow:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return flow
}

// Bind 绑定数据
func Bind(v interface{}, data maps.Map) interface{} {

	var res interface{} = v
	value := reflect.ValueOf(v)
	value = reflect.Indirect(value)
	valueKind := value.Kind()
	if valueKind == reflect.Interface {
		value = value.Elem()
		valueKind = value.Kind()
	}

	if valueKind == reflect.Slice || valueKind == reflect.Array { // Slice || Array
		val := []interface{}{}
		for i := 0; i < value.Len(); i++ {
			val = append(val, Bind(value.Index(i).Interface(), data))
		}
		res = val
	} else if valueKind == reflect.Map { // Map
		val := map[string]interface{}{}
		for _, key := range value.MapKeys() {
			k := fmt.Sprintf("%s", key)
			val[k] = Bind(value.MapIndex(key).Interface(), data)
		}
		res = val
	} else if valueKind == reflect.String { // 绑定数据
		input := value.Interface().(string)

		// ReplaceVar
		matches := reVar.FindAllStringSubmatch(input, -1)
		length := len(matches)
		if length == 1 { // "{{in.0}}"
			name := matches[0][1]
			res = data[name]
		} else if length > 1 {
			for _, match := range matches {
				val := fmt.Sprintf("%s", data[match[1]])
				input = strings.ReplaceAll(input, match[0], val)
			}
		}

		// ReplaceFilters
		matches = reFun.FindAllStringSubmatch(input, -1)
		length = len(matches)
		if length == 1 {
			name := matches[0][1]
			if method, has := Filters[name]; has {
				args := extraFunArgs(matches[0][2], data)
				res = method(args...)
			} else {
				res = nil
			}
		}
	}
	return res
}

func extraFunArgs(input string, data maps.Map) []interface{} {
	args := []interface{}{}
	matches := reFunArg.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		key := match[1]
		keyAny := any.Of(key)
		if strings.HasPrefix(key, ":") {
			key = key[1:]
			args = append(args, data[key])
		} else if strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'") {
			args = append(args, strings.Trim(key, "'"))
		} else if strings.Contains(key, ".") {
			args = append(args, keyAny.CFloat())
		} else {
			args = append(args, keyAny.CInt())
		}
	}
	return args
}
