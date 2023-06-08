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
func Bind(v interface{}, data map[string]interface{}, vars ...*regexp.Regexp) interface{} {
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

	switch valueKind {
	case reflect.Slice, reflect.Array: // Slice || Array
		val := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			val[i] = Bind(value.Index(i).Interface(), data)
		}
		res = val
	case reflect.Map: // Map
		val := make(map[string]interface{})
		for _, key := range value.MapKeys() {
			k := fmt.Sprintf("%s", key)
			val[k] = Bind(value.MapIndex(key).Interface(), data)
		}
		res = val
	case reflect.String: // String
		input := value.Interface().(string)

		// 替换变量
		for _, re := range vars {
			matches := re.FindAllStringSubmatchIndex(input, -1)
			length := len(matches)
			if length == 1 { // "{{in.0}}"
				name := input[matches[0][2]:matches[0][3]]
				res = data[name]
				break
			} else if length > 1 {
				var sb strings.Builder
				lastIndex := 0
				for _, match := range matches {
					val := fmt.Sprintf("%s", data[input[match[2]:match[3]]])
					sb.WriteString(input[lastIndex:match[0]])
					sb.WriteString(val)
					lastIndex = match[1]
				}
				sb.WriteString(input[lastIndex:])
				res = sb.String()
				break
			} else {
				res = input
			}
		}
	default:
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
