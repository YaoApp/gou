package eval

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Eval execute an anonymous function

var reFuncHead = regexp.MustCompile(`\s*function\s+(\w+)\s*\(([^)]*)\)\s*\{`)

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

	args := []v8go.Valuer{}
	if len(jsArgs) > 1 {
		for _, arg := range jsArgs[1:] {
			args = append(args, arg)
		}
	}

	source := jsArgs[0].String()
	source = reFuncHead.ReplaceAllString(source, "($2) => {")
	name := fmt.Sprintf("__anonymous_%s", uuid.New().String())

	iso := info.Context().Isolate()

	script, err := iso.CompileUnboundScript(source, name, v8go.CompileOptions{})
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	fn, err := script.Run(info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	defer fn.Release()

	global := info.Context().Global()
	global.Set(name, fn)
	defer global.Delete(name)

	jsRes, err := global.MethodCall(name, args...)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return jsRes
}
