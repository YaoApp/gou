package helper

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime/v8/bridge"
)

// Dump The Dump function dumps the given variables:
func Dump(values ...interface{}) {

	f := NewFormatter()
	f.Indent = 4
	f.RawStrings = true
	for _, v := range values {

		if err, ok := v.(error); ok {
			color.Red(err.Error())
			continue
		}

		switch value := v.(type) {

		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			color.Cyan(fmt.Sprintf("%v", v))
			continue

		case string, []byte:
			color.Green(fmt.Sprintf("%s", v))
			continue

		case bridge.UndefinedT:
			color.Magenta(value.String())
			continue

		case bridge.FunctionT:
			color.Cyan(value.String())
			continue

		case bridge.PromiseT:
			color.Cyan("Promise { " + value.String() + " }")
			continue

		default:
			var res interface{}
			txt, err := jsoniter.Marshal(v)
			if err != nil {
				color.Red(err.Error())
				continue
			}

			jsoniter.Unmarshal(txt, &res)
			bytes, _ := f.Marshal(res)
			fmt.Println(string(bytes))
		}
	}
}

// ToString returns a formatted string representation of the given variables
func ToString(values ...interface{}) string {
	var result strings.Builder
	f := NewFormatter()
	f.Indent = 4
	f.RawStrings = true
	f.DisabledColor = true

	for _, v := range values {
		if err, ok := v.(error); ok {
			result.WriteString(err.Error() + "\n")
			continue
		}

		switch value := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			result.WriteString(fmt.Sprintf("%v\n", v))
			continue

		case string, []byte:
			result.WriteString(fmt.Sprintf("%s\n", v))
			continue

		case bridge.UndefinedT:
			result.WriteString(value.String() + "\n")
			continue

		case bridge.FunctionT:
			result.WriteString(value.String() + "\n")
			continue

		case bridge.PromiseT:
			result.WriteString("Promise { " + value.String() + " }\n")
			continue

		default:
			var res interface{}
			txt, err := jsoniter.Marshal(v)
			if err != nil {
				result.WriteString(err.Error() + "\n")
				continue
			}

			jsoniter.Unmarshal(txt, &res)
			bytes, _ := f.Marshal(res)
			result.WriteString(string(bytes) + "\n")
		}
	}

	return result.String()
}

// DumpError dumps the given variables in red color
func DumpError(values ...interface{}) {
	str := ToString(values...)
	color.Red(str)
}

// DumpWarn dumps the given variables in yellow color
func DumpWarn(values ...interface{}) {
	str := ToString(values...)
	color.Yellow(str)
}

// DumpInfo dumps the given variables in blue color
func DumpInfo(values ...interface{}) {
	str := ToString(values...)
	color.Blue(str)
}
