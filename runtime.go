package gou

import (
	"strconv"
	"strings"

	"github.com/yaoapp/gou/runtime"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/utils"
)

// Yao JavaScript 运行环境
var Yao *runtime.Runtime

// LoadRuntime load runtime
func LoadRuntime(option runtime.Option) {
	Yao = runtime.Yao(option)
	Yao.
		AddFunction("Process", func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
			if len(args) < 1 {
				return map[string]interface{}{"code": 400, "message": "缺少处理器名称"}
			}
			name, ok := args[0].(string)
			if !ok {
				return map[string]interface{}{"code": 400, "message": "处理器参数不正确"}
			}

			in := []interface{}{}
			if len(args) > 1 {
				in = args[1:]
			}

			value, err := NewProcess(name, in...).WithGlobal(global).WithSID(sid).Exec()
			if err != nil {
				return map[string]interface{}{"code": 500, "message": err.Error()}
			}
			return value
		}).
		AddRootFunction("Studio", func(global map[string]interface{}, sid string, args ...interface{}) interface{} {
			if len(args) < 1 {
				return map[string]interface{}{"code": 400, "message": "缺少处理器名称"}
			}

			name, ok := args[0].(string)
			if !ok {
				return map[string]interface{}{"code": 400, "message": "处理器参数不正确"}
			}

			in := []interface{}{}
			if len(args) > 1 {
				in = args[1:]
			}

			namer := strings.Split(name, ".")
			last := len(namer) - 1
			service := strings.ToLower(strings.Join(namer[0:last], "."))
			method := namer[last]
			res, err := Yao.New(service, method).
				WithGlobal(global).
				WithSid(sid).
				RootCall(in...)

			if err != nil {
				message := err.Error()

				// JS Exception
				if strings.HasPrefix(message, "Exception|") {
					message = strings.Replace(message, "Exception|", "", -1)
					values := strings.Split(message, ":")
					if len(values) == 2 {
						code := 500
						if v, err := strconv.Atoi(values[0]); err == nil {
							code = v
						}
						message = strings.TrimSpace(values[1])
						exception.New(message, code).Throw()
					}
				}

				// Other
				code := 500
				values := strings.Split(message, "|")
				if len(values) == 2 {
					if v, err := strconv.Atoi(values[0]); err == nil {
						code = v
					}
					message = values[0]
				}

				exception.New(message, code).Throw()
			}
			return res
		}).
		AddObject("console", map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}{
			"log": func(_ map[string]interface{}, _ string, args ...interface{}) interface{} {
				utils.Dump(args...)
				return nil
			},
		}).
		Init()
}
