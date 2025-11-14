package v8

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

var reFuncHead = regexp.MustCompile(`\s*function\s+(\w+)\s*\(([^)]*)\)\s*\{`)

// Call call the script function
func (context *Context) Call(method string, args ...interface{}) (interface{}, error) {

	// Performance Mode
	if context.Runner != nil {
		return context.Runner.Exec(context.script), nil
	}

	// Set the global data
	global := context.Global()
	err := bridge.SetShareData(context.Context, global, &bridge.Share{
		Sid:    context.Sid,
		Root:   context.Root,
		Global: context.Data,
	})
	if err != nil {
		return nil, err
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New(runtimeOption.ConsoleMode).Set("console", context.Context)
	if err != nil {
		return nil, err
	}

	// Run the method
	jsArgs, err := bridge.JsValues(context.Context, args)
	if err != nil {
		return nil, err
	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
	if err != nil {
		if e, ok := err.(*v8go.JSError); ok {
			PrintException(method, args, e, context.SourceRoots)
		}
		log.Error("%s.%s %s", context.ID, method, err.Error())
		return nil, err
	}
	defer jsRes.Release() // Release the js value

	goRes, err := bridge.GoValue(jsRes, context.Context)
	if err != nil {
		return nil, err
	}

	return goRes, nil
}

// CallAnonymous call the script function with anonymous function
func (context *Context) CallAnonymous(source string, args ...interface{}) (interface{}, error) {

	// Extract function name for debugging, then remove it from source
	name := extractFunctionName(source)
	source = reFuncHead.ReplaceAllString(source, "")

	script, err := context.Isolate.CompileUnboundScript(source, name, v8go.CompileOptions{})
	if err != nil {
		return nil, err
	}

	fn, err := script.Run(context.Context)
	if err != nil {
		return nil, err
	}
	defer fn.Release()

	global := context.Global()
	global.Set(name, fn)
	defer global.Delete(name)

	res, err := context.Call(name, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// extractFunctionName extracts the function name from source code
// If the source contains a named function (e.g., "function foo(arg)"), it returns the name
// Otherwise, returns "<anonymous>"
func extractFunctionName(source string) string {
	matches := reFuncHead.FindStringSubmatch(source)
	if len(matches) > 1 && matches[1] != "" {
		return matches[1]
	}
	return "<anonymous>"
}

// CallAnonymousWith call the script function with anonymous function
func (context *Context) CallAnonymousWith(ctx context.Context, source string, args ...interface{}) (interface{}, error) {

	name := extractFunctionName(source)
	source = reFuncHead.ReplaceAllString(source, "($2) => {")

	script, err := context.Isolate.CompileUnboundScript(source, name, v8go.CompileOptions{})
	if err != nil {
		return nil, err
	}

	fn, err := script.Run(context.Context)
	if err != nil {
		return nil, err
	}
	defer fn.Release()

	global := context.Global()
	global.Set(name, fn)
	defer global.Delete(name)

	res, err := context.CallWith(ctx, name, args...)
	if err != nil {
		color.White("Source:\n")
		lines := strings.Split(source, "\n")
		total := fmt.Sprintf("%d", len(lines))
		for i, line := range lines {
			num := fmt.Sprintf("%d", i+1)
			num = strings.Repeat(" ", len(total)-len(num)) + num
			fmt.Printf("%s: %s\n", num, line)
		}
		return nil, err
	}
	return res, nil
}

// CallWith call the script function
func (context *Context) CallWith(ctx context.Context, method string, args ...interface{}) (interface{}, error) {
	// Performance Mode
	if context.Runner != nil {
		return context.Runner.ExecWithContext(ctx, context.script), nil
	}

	// Set the global data
	global := context.Global()
	err := bridge.SetShareData(context.Context, global, &bridge.Share{
		Sid:    context.Sid,
		Root:   context.Root,
		Global: context.Data,
	})
	if err != nil {
		return nil, err
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New(runtimeOption.ConsoleMode).Set("console", context.Context)
	if err != nil {
		return nil, err
	}

	// Run the method
	jsArgs, err := bridge.JsValues(context.Context, args)
	if err != nil {
		return nil, err
	}
	defer bridge.FreeJsValues(jsArgs)

	// Start a monitoring goroutine to handle context cancellation
	// This goroutine only monitors and calls TerminateExecution, it doesn't execute V8 code
	terminated := false
	terminateLock := sync.Mutex{}
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			// Context cancelled, terminate V8 execution
			terminateLock.Lock()
			terminated = true
			terminateLock.Unlock()
			context.Isolate.TerminateExecution()
		case <-done:
			// Normal completion, do nothing
		}
	}()
	defer close(done)

	// Execute directly in current thread (no new goroutine for V8 execution)
	// This ensures all V8 operations (create, use, dispose) happen on the same thread
	jsRes, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)

	// Check if execution was terminated due to context cancellation
	terminateLock.Lock()
	wasTerminated := terminated
	terminateLock.Unlock()

	if wasTerminated && err != nil {
		return nil, ctx.Err() // Return context error, not V8 error
	}

	if err != nil {
		if e, ok := err.(*v8go.JSError); ok {
			PrintException(method, args, e, context.SourceRoots)
		}
		log.Error("%s.%s %v", context.ID, method, err)
		return nil, err
	}
	defer jsRes.Release() // Release the js value

	goRes, err := bridge.GoValue(jsRes, context.Context)
	if err != nil {
		return nil, err
	}

	return goRes, nil
}

// WithFunction add a function to the context
func (context *Context) WithFunction(name string, cb v8go.FunctionCallback) {
	tmpl := v8go.NewFunctionTemplate(context.Isolate.Isolate, cb)
	context.Global().Set(name, tmpl.GetFunction(context.Context))
}

// WithGlobal add a global variable to the context
func (context *Context) WithGlobal(name string, value interface{}) error {
	switch value.(type) {
	case v8go.Valuer:
		context.Global().Set(name, value)
	default:
		jsValue, err := bridge.JsValue(context.Context, value)
		if err != nil {
			return err
		}
		context.Global().Set(name, jsValue)
	}
	return nil
}

// Close Context
func (context *Context) Close() error {
	// Standard Mode Release Isolate
	if runtimeOption.Mode == "standard" {
		// In standard mode, we must properly manage the v8 Context and Isolate lifecycle
		// The Context must be closed before disposing the Isolate to avoid "isolate entered" errors
		context.UnboundScript = nil
		context.Data = nil

		// Close context first (releases Persistent handle, exits any scopes)
		if context.Context != nil {
			context.Context.Close()
			context.Context = nil
		}

		// Dispose isolate after context is closed
		// This ensures no references remain to the isolate
		if context.Isolate != nil {
			context.Isolate.Dispose()
			context.Isolate = nil
		}

		return nil
	}

	// Performance Mode Release Runner
	if context.Runner != nil {
		context.Runner.Reset()
		context.Context = nil
		context.Data = nil
		context.Runner = nil
		return nil
	}

	return nil
}
