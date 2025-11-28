package use

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestUseBasic(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			return Use(MockResource, "test-id", (obj) => {
				return {
					id: obj.id,
					data: obj.getData()
				};
			});
		}
		test()
	`, "test.js")

	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := goRes.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test-id", result["id"])
	assert.Equal(t, "data: test-id", result["data"])
}

func TestUseWithError(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	_, err := ctx.RunScript(`
		const test = () => {
			Use(MockResource, "test-id", (obj) => {
				throw new Error("Test error");
			});
		}
		test()
	`, "test.js")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Test error")
}

func TestUseNested(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			return Use(MockResource, "outer", (outer) => {
				return Use(MockResource, "inner", (inner) => {
					return {
						outerId: outer.id,
						innerId: inner.id,
						outerData: outer.getData(),
						innerData: inner.getData()
					};
				});
			});
		}
		test()
	`, "test.js")

	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := goRes.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "outer", result["outerId"])
	assert.Equal(t, "inner", result["innerId"])
	assert.Equal(t, "data: outer", result["outerData"])
	assert.Equal(t, "data: inner", result["innerData"])
}

func TestUseInvalidConstructor(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	_, err := ctx.RunScript(`
		const test = () => {
			Use("not a constructor", (obj) => {
				return obj;
			});
		}
		test()
	`, "test.js")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "constructor function")
}

func TestUseNoCallback(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	_, err := ctx.RunScript(`
		const test = () => {
			Use(MockResource, "test-id");
		}
		test()
	`, "test.js")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback function")
}

func TestUseMultipleArgs(t *testing.T) {
	ctx := prepare(t)
	defer closeCtx(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			return Use(MockResourceWithArgs, "id1", "arg2", 123, (obj) => {
				return {
					id: obj.id,
					arg2: obj.arg2,
					num: obj.num
				};
			});
		}
		test()
	`, "test.js")

	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, ok := goRes.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "id1", result["id"])
	assert.Equal(t, "arg2", result["arg2"])
	assert.Equal(t, float64(123), result["num"])
}

// Helper functions

func closeCtx(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T) *v8go.Context {
	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Use", ExportFunction(iso))

	// Create mock constructors for testing
	template.Set("MockResource", createMockResourceConstructor(iso))
	template.Set("MockResourceWithArgs", createMockResourceWithArgsConstructor(iso))

	ctx := v8go.NewContext(iso, template)
	return ctx
}

// createMockResourceConstructor creates a mock constructor that simulates a resource with Release method
func createMockResourceConstructor(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(ctx, "MockResource requires id as first argument")
		}

		id := args[0].String()

		// Create a JavaScript object
		jsCode := `
			(function(id) {
				return {
					id: id,
					released: false,
					getData: function() {
						return "data: " + this.id;
					},
					Release: function() {
						this.released = true;
					}
				};
			})
		`

		script, err := iso.CompileUnboundScript(jsCode, "mock.js", v8go.CompileOptions{})
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		fn, err := script.Run(ctx)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}
		defer fn.Release()

		fnObj, err := fn.AsFunction()
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		idVal, err := v8go.NewValue(iso, id)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		result, err := fnObj.Call(v8go.Undefined(iso), idVal)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return result
	})
}

// createMockResourceWithArgsConstructor creates a mock constructor that accepts multiple arguments
func createMockResourceWithArgsConstructor(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 3 {
			return bridge.JsException(ctx, "MockResourceWithArgs requires 3 arguments")
		}

		id := args[0].String()
		arg2 := args[1].String()
		num := args[2].Number()

		// Create a JavaScript object
		jsCode := `
			(function(id, arg2, num) {
				return {
					id: id,
					arg2: arg2,
					num: num,
					Release: function() {
						this.released = true;
					}
				};
			})
		`

		script, err := iso.CompileUnboundScript(jsCode, "mock.js", v8go.CompileOptions{})
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		fn, err := script.Run(ctx)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}
		defer fn.Release()

		fnObj, err := fn.AsFunction()
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		idVal, _ := v8go.NewValue(iso, id)
		arg2Val, _ := v8go.NewValue(iso, arg2)
		numVal, _ := v8go.NewValue(iso, num)

		result, err := fnObj.Call(v8go.Undefined(iso), idVal, arg2Val, numVal)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return result
	})
}
