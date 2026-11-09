package race

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/functions/concurrent"
	"rogchap.com/v8go"
)

// ExportFunction returns the V8 function template for the Race() global function.
// Race() executes multiple processes concurrently and returns once the first completes (Promise.race semantics).
// Unlike Any(), Race returns on the first completion regardless of success or failure.
//
// Usage from JavaScript:
//
//	var results = Race([
//	  { process: "scripts.slow.Process", args: [] },
//	  { process: "scripts.fast.Process", args: [] },
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

	results := concurrent.ParallelRace(tasks, share)

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
