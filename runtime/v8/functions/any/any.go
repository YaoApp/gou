package any

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/functions/concurrent"
	"rogchap.com/v8go"
)

// ExportFunction returns the V8 function template for the Any() global function.
// Any() executes multiple processes concurrently and returns once the first succeeds (Promise.any semantics).
// A "success" means data != nil and error is empty.
//
// Usage from JavaScript:
//
//	var results = Any([
//	  { process: "scripts.slow.Fetch", args: ["https://a.com"] },
//	  { process: "scripts.fast.Fetch", args: ["https://b.com"] },
//	]);
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, exec)
}

func exec(info *v8go.FunctionCallbackInfo) *v8go.Value {
	tasks, share, err := concurrent.ParseTasks(info)
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if len(tasks) == 0 {
		jsRes, err := bridge.JsValue(info.Context(), []interface{}{})
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}
		return jsRes
	}

	results := concurrent.ParallelAny(tasks, share)

	jsRes, err := bridge.JsValue(info.Context(), toInterface(results))
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}
	return jsRes
}

// toInterface converts []TaskResult to []interface{} for bridge.JsValue serialization.
func toInterface(results []concurrent.TaskResult) []interface{} {
	out := make([]interface{}, len(results))
	for i, r := range results {
		m := map[string]interface{}{
			"data":  r.Data,
			"index": r.Index,
		}
		if r.Error != "" {
			m["error"] = r.Error
		}
		out[i] = m
	}
	return out
}
