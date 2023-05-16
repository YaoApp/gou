package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/gou/runtime/v8/functions/studio"
	"rogchap.com/v8go"
)

func TestStudio(t *testing.T) {
	prepare(t)

	ctx := prepareStudio(t, true, "", nil)
	defer closeStudio(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Studio("studio.runtime.basic.Hello", "foo", 99, 0.618);
			const result2 = Studio("runtime.basic.Hello", "foo", 99, 0.618);
			return {
				result: result,
				result2: result2,
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

	assert.Equal(t, "", res["__yao_sid"])
	assert.Equal(t, nil, res["__yao_global"])
	assert.Equal(t, true, res["__YAO_SU_ROOT"])
	assert.Equal(t, "foo", res["result"])
	assert.Equal(t, "foo", res["result2"])
}

func TestStudioWithData(t *testing.T) {
	prepare(t)

	ctx := prepareStudio(t, true, "SID-0101", map[string]interface{}{"hello": "world"})
	defer closeStudio(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Studio("studio.runtime.basic.Hello", "foo", 99, 0.618);
			return {
				result: result,
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

	assert.Equal(t, "SID-0101", res["__yao_sid"])
	assert.Equal(t, "SID-0101", res["__yao_sid"])
	assert.Equal(t, map[string]interface{}{"hello": "world"}, res["__yao_global"])
	assert.Equal(t, true, res["__YAO_SU_ROOT"])
	assert.Equal(t, "foo", res["result"])
}

func TestStudioNotRoot(t *testing.T) {
	prepare(t)

	ctx := prepareStudio(t, false, "", nil)
	defer closeStudio(ctx)

	_, err := ctx.RunScript(`
		const test = () => {
			const result = Studio("studio.runtime.basic.Hello", "foo", 99, 0.618);
			return {
				...result,
				__yao_global:__yao_data["DATA"],
				__yao_sid:__yao_data["SID"],
				__YAO_SU_ROOT:__yao_data["ROOT"],
			}
		}
		test()
	`, "")

	assert.Equal(t, "Error: function is not allowed", err.Error())
}

func closeStudio(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepareStudio(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Studio", studio.ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)
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
