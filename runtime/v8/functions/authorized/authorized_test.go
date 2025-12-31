package authorized

import (
	"testing"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestAuthorizedWithData(t *testing.T) {
	authorized := map[string]interface{}{
		"user_id": "user123",
		"team_id": "team456",
		"scope":   "read write",
		"constraints": map[string]interface{}{
			"team_only": true,
		},
	}

	ctx := prepare(t, true, "test-sid", map[string]interface{}{"hello": "world"}, authorized)
	defer close(ctx)

	// Test Authorized() returns the authorized data
	res, err := ctx.RunScript(`JSON.stringify(Authorized())`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	result := res.String()
	if result == "" || result == "null" {
		t.Fatal("Authorized() should return authorized data")
	}

	// Verify specific fields
	res, err = ctx.RunScript(`Authorized().user_id`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if res.String() != "user123" {
		t.Errorf("Expected user_id to be 'user123', got '%s'", res.String())
	}

	res, err = ctx.RunScript(`Authorized().team_id`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if res.String() != "team456" {
		t.Errorf("Expected team_id to be 'team456', got '%s'", res.String())
	}

	res, err = ctx.RunScript(`Authorized().scope`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if res.String() != "read write" {
		t.Errorf("Expected scope to be 'read write', got '%s'", res.String())
	}

	res, err = ctx.RunScript(`Authorized().constraints.team_only`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Boolean() {
		t.Error("Expected constraints.team_only to be true")
	}
}

func TestAuthorizedWithoutData(t *testing.T) {
	ctx := prepare(t, true, "test-sid", map[string]interface{}{"hello": "world"}, nil)
	defer close(ctx)

	// Test Authorized() returns null when no authorized data
	res, err := ctx.RunScript(`Authorized()`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	if !res.IsNull() {
		t.Errorf("Expected Authorized() to return null when no authorized data, got %v", res)
	}
}

func TestAuthorizedEmptyData(t *testing.T) {
	ctx := prepare(t, true, "test-sid", map[string]interface{}{"hello": "world"}, map[string]interface{}{})
	defer close(ctx)

	// Test Authorized() returns empty object
	res, err := ctx.RunScript(`JSON.stringify(Authorized())`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	if res.String() != "{}" {
		t.Errorf("Expected Authorized() to return empty object, got %s", res.String())
	}
}

func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T, root bool, sid string, global map[string]interface{}, authorized map[string]interface{}) *v8go.Context {
	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Authorized", ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)

	// Set share data with authorized info
	share := &bridge.Share{
		Sid:        sid,
		Root:       root,
		Global:     global,
		Authorized: authorized,
	}

	err := bridge.SetShareData(ctx, ctx.Global(), share)
	if err != nil {
		t.Fatal(err)
	}

	return ctx
}
