package studio

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction function template
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

// exec
func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {

	share, err := bridge.ShareData(info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !share.Root {
		return bridge.JsException(info.Context(), "function is not allowed")
	}

	jsArgs := info.Args()
	if len(jsArgs) < 1 {
		return bridge.JsException(info.Context(), "missing parameters")
	}

	if !jsArgs[0].IsString() {
		return bridge.JsException(info.Context(), "the first parameter should be a string")
	}

	goArgs := []interface{}{}
	if len(jsArgs) > 1 {
		goArgs, err = bridge.GoValues(jsArgs[1:], info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
	}

	name := fmt.Sprintf("studio.%s", strings.TrimPrefix(jsArgs[0].String(), "studio."))
	goRes, err := process.New(name, goArgs...).
		WithGlobal(share.Global).
		WithSID(share.Sid).
		Exec()

	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	jsRes, err := bridge.JsValue(info.Context(), goRes)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return jsRes
}
