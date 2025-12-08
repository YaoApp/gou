package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestProcess(t *testing.T) {

	ctx := prepare(t, false, "", nil)
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Process("unit.test.process", "foo", 99, 0.618);
			return {
				...result,
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

	assert.Equal(t, res["Sid"], res["__yao_sid"])
	assert.Equal(t, nil, res["__yao_global"])
	assert.Equal(t, false, res["__YAO_SU_ROOT"])
	assert.Equal(t, "unit.test.process", res["Name"])
	assert.Equal(t, []interface{}{"foo", float64(99), 0.618}, res["Args"])
}

func TestProcessWithData(t *testing.T) {

	ctx := prepare(t, true, "SID-0101", map[string]interface{}{"hello": "world"})
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Process("unit.test.process", "foo", 99, 0.618);
			return {
				...result,
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

	assert.Equal(t, res["Sid"], res["__yao_sid"])
	assert.Equal(t, res["Global"], res["__yao_global"])
	assert.Equal(t, "SID-0101", res["__yao_sid"])
	assert.Equal(t, map[string]interface{}{"hello": "world"}, res["__yao_global"])
	assert.Equal(t, true, res["__YAO_SU_ROOT"])
	assert.Equal(t, "unit.test.process", res["Name"])
	assert.Equal(t, []interface{}{"foo", float64(99), 0.618}, res["Args"])
}

func TestProcessWithAuthorized(t *testing.T) {

	authorized := map[string]interface{}{
		"user_id": "user123",
		"team_id": "team456",
		"scope":   "read write",
		"constraints": map[string]interface{}{
			"team_only": true,
		},
	}

	ctx := prepareWithAuthorized(t, true, "SID-AUTH-001", map[string]interface{}{"hello": "world"}, authorized)
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			const result = Process("unit.test.process", "foo", 99, 0.618);
			return {
				...result,
				__yao_global:__yao_data["DATA"],
				__yao_sid:__yao_data["SID"],
				__YAO_SU_ROOT:__yao_data["ROOT"],
				__yao_authorized:__yao_data["AUTHORIZED"],
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

	assert.Equal(t, res["Sid"], res["__yao_sid"])
	assert.Equal(t, res["Global"], res["__yao_global"])
	assert.Equal(t, "SID-AUTH-001", res["__yao_sid"])
	assert.Equal(t, map[string]interface{}{"hello": "world"}, res["__yao_global"])
	assert.Equal(t, true, res["__YAO_SU_ROOT"])
	assert.Equal(t, "unit.test.process", res["Name"])
	assert.Equal(t, []interface{}{"foo", float64(99), 0.618}, res["Args"])

	// Check Authorized information
	authInfo := res["Authorized"]
	assert.NotNil(t, authInfo, "Authorized should not be nil")

	authMap, ok := authInfo.(map[string]interface{})
	assert.True(t, ok, "Authorized should be map[string]interface{}")

	assert.Equal(t, "user123", authMap["user_id"])
	assert.Equal(t, "team456", authMap["team_id"])
	assert.Equal(t, "read write", authMap["scope"])

	constraints, ok := authMap["constraints"].(map[string]interface{})
	assert.True(t, ok, "constraints should be map[string]interface{}")
	assert.Equal(t, true, constraints["team_only"])
}

func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {
	return prepareWithAuthorized(t, root, sid, global, nil)
}

func prepareWithAuthorized(t *testing.T, root bool, sid string, global map[string]interface{}, authorized map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Process", ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)

	var err error

	// Set share data with authorized info
	share := &bridge.Share{
		Sid:        sid,
		Root:       root,
		Global:     global,
		Authorized: authorized,
	}

	err = bridge.SetShareData(ctx, ctx.Global(), share)
	if err != nil {
		t.Fatal(err)
	}

	// Also set __yao_data for test verification
	goData := map[string]interface{}{
		"SID":        sid,
		"ROOT":       root,
		"DATA":       global,
		"AUTHORIZED": authorized,
	}

	jsData, err := bridge.JsValue(ctx, goData)
	if err != nil {
		t.Fatal(err)
	}

	if err = ctx.Global().Set("__yao_data", jsData); err != nil {
		t.Fatal(err)
	}

	process.Register("unit.test.process", func(proc *process.Process) interface{} {
		// Return process with Authorized converted to map for easier testing
		result := map[string]interface{}{
			"Name":   proc.Name,
			"Args":   proc.Args,
			"Sid":    proc.Sid,
			"Global": proc.Global,
		}

		if proc.Authorized != nil {
			result["Authorized"] = proc.Authorized.AuthorizedToMap()
		} else {
			result["Authorized"] = map[string]interface{}{}
		}

		return result
	})
	return ctx
}
