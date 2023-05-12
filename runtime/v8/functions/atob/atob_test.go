package atob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestAtob(t *testing.T) {

	ctx := prepare(t, false, "", nil)
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = atob("ZGVtbzoxMjM0NTY=");
			return {
				result: result,
				__yao_global:__yao_global,
				__yao_sid: __yao_sid,
				__YAO_SU_ROOT: __YAO_SU_ROOT,
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

	assert.Equal(t, "demo:123456", res["result"])
	assert.Equal(t, nil, res["__yao_global"])
	assert.Equal(t, false, res["__YAO_SU_ROOT"])
}

func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("atob", ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)

	var err error
	jsGlobal := v8go.Undefined(ctx.Isolate())
	jsGlobal, err = bridge.JsValue(ctx, global)
	if err != nil {
		t.Fatal(err)
	}

	if err = ctx.Global().Set("__YAO_SU_ROOT", root); err != nil {
		t.Fatal(err)
	}

	if err = ctx.Global().Set("__yao_global", jsGlobal); err != nil {
		t.Fatal(err)
	}
	if err = ctx.Global().Set("__yao_sid", sid); err != nil {
		t.Fatal(err)
	}

	return ctx
}
