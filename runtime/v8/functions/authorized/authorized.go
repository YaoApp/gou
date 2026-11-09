package authorized

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// ExportFunction function template
// Usage: Authorized() - returns the authorized information
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

// exec returns the authorized information from the current context
func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {
	share, err := bridge.ShareData(info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Return nil if no authorized info
	if share.Authorized == nil {
		jsRes, err := bridge.JsValue(info.Context(), nil)
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
		return jsRes
	}

	// Return the authorized information
	jsRes, err := bridge.JsValue(info.Context(), share.Authorized)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	return jsRes
}
