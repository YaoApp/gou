package helper

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

var reVar = regexp.MustCompile("{{[ ]*([^\\s]+)[ ]*}}")                     // {{in.2}}
var reVarStyle2 = regexp.MustCompile("\\?:([^\\s]+)")                       // ?:$in.2
var reFun = regexp.MustCompile("{{[ ]*([0-9a-zA-Z_]+)[ ]*\\((.*)\\)[ ]*}}") // {{pluck($res.users, 'id')}}
var reFunArg = regexp.MustCompile("([^\\s,]+)")                             // $res.users, 'id'

// Bind 绑定数据
func Bind(v interface{}, data maps.Map, vars ...*regexp.Regexp) interface{} {

	if len(vars) == 0 {
		vars = []*regexp.Regexp{reVar, reVarStyle2}
	}

	var res interface{}
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
	} else if valueKind == reflect.String { // String
		input := value.Interface().(string)

		// 替换变量
		for _, reVar := range vars {
			matches := reVar.FindAllStringSubmatch(input, -1)
			length := len(matches)
			if length == 1 { // "{{in.0}}"
				name := matches[0][1]
				res = data[name]
				break
			} else if length > 1 {
				for _, match := range matches {
					val := fmt.Sprintf("%s", data[match[1]])
					input = strings.ReplaceAll(input, match[0], val)
				}
				res = input
				break
			} else {
				res = input
			}
		}
	} else {
		res = v
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
