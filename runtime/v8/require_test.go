package v8

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestRequre(t *testing.T) {

	prepare(t)

	ctx := requrePrepare(t, false, "", nil)
	defer requireClose(ctx)

	jsRes, err := ctx.RunScript(`
		const lib = Require("runtime.lib");
		const { Foo } = Require("runtime.lib");
		function Hello() {
			return {
			  "lib.Foo": lib.Foo(),
			  "Foo": Foo()
			};
		}
		Hello()
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

	assert.Equal(t, "bar", res["lib.Foo"])
	assert.Equal(t, "bar", res["Foo"])
}

func requireClose(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func requrePrepare(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Require", Require(iso))

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
