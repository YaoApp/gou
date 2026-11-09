package lang

import (
	"strings"

	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction function template
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, replace)
}

// replace
func replace(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) == 0 {
		return v8go.Undefined(info.Context().Isolate())
	}

	if !args[0].IsString() {
		return args[0]
	}

	value := strings.TrimPrefix(args[0].String(), "::")
	lang.Replace(&value)

	jsValue, err := bridge.JsValue(info.Context(), value)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return jsValue
}
