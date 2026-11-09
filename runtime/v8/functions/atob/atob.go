package atob

import (
	"encoding/base64"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction function template
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

// exec
func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {

	jsArgs := info.Args()
	if len(jsArgs) < 1 {
		return bridge.JsException(info.Context(), "missing parameters")
	}

	if !jsArgs[0].IsString() {
		return bridge.JsException(info.Context(), "the first parameter should be a string")
	}

	goRes, err := base64.StdEncoding.DecodeString(jsArgs[0].String())
	if err != nil {
		return bridge.JsException(info.Context(), err.Error())
	}

	jsRes, err := bridge.JsValue(info.Context(), string(goRes))
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return jsRes
}
