package eval

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestEval(t *testing.T) {

	ctx := prepare(t, false, "", nil)
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Eval("function add(a, b) { return a + b; }", 5, 3);
			return {
				result,
				__yao_global:__yao_data["DATA"],
				__yao_sid:__yao_data["SID"],
				__YAO_SU_ROOT:__yao_data["ROOT"],
			}
		}
		test()
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		t.Fatal(err)
	}

	res, ok := goRes.(map[string]interface{})
	if !ok {
		t.Fatal("result error")
	}

	assert.Equal(t, float64(8), res["result"])
	assert.Equal(t, nil, res["__yao_global"])
	assert.Equal(t, "", res["__yao_sid"])
	assert.Equal(t, false, res["__YAO_SU_ROOT"])
}

func TestEvalWithData(t *testing.T) {

	ctx := prepare(t, true, "SID-0101", map[string]interface{}{"hello": "world"})
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Eval("function greet(name) { return 'Hello, ' + name + '! ' + __yao_data.DATA.hello; }", "User");
			return {
				result,
				__yao_global:__yao_data["DATA"],
				__yao_sid:__yao_data["SID"],
				__YAO_SU_ROOT:__yao_data["ROOT"],
			}
		}
		test()
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		t.Fatal(err)
	}

	res, ok := goRes.(map[string]interface{})
	if !ok {
		t.Fatal("result error")
	}

	assert.Equal(t, "Hello, User! world", res["result"])
	assert.Equal(t, map[string]interface{}{"hello": "world"}, res["__yao_global"])
	assert.Equal(t, "SID-0101", res["__yao_sid"])
	assert.Equal(t, true, res["__YAO_SU_ROOT"])
}

func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Eval", ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)

	var err error
	goData := map[string]interface{}{
		"SID":  sid,
		"ROOT": root,
		"DATA": global,
	}

	jsData, err := bridge.JsValue(ctx, goData)
	if err != nil {
		t.Fatal(err)
	}

	if err = ctx.Global().Set("__yao_data", jsData); err != nil {
		t.Fatal(err)
	}

	return ctx
}
