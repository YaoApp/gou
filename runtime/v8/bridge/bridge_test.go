package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application"
	"rogchap.com/v8go"
)

func call(ctx *v8go.Context, method string, args ...interface{}) (interface{}, error) {

	global := ctx.Global()
	jsArgs, err := JsValues(ctx, args)
	if err != nil {
		return nil, err
	}
	defer FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(method, Valuers(jsArgs)...)
	if err != nil {
		return nil, err
	}

	goRes, err := GoValue(jsRes, ctx)
	if err != nil {
		return nil, err
	}

	return goRes, nil
}

func prepare(t *testing.T) *v8go.Context {

	root := os.Getenv("GOU_TEST_APPLICATION")

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	file := filepath.Join("scripts", "runtime", "bridge.js")
	source, err := app.Read(file)
	if err != nil {
		t.Fatal(err)
	}

	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	_, err = ctx.RunScript(string(source), file)
	if err != nil {
		t.Fatal(err)
	}

	return ctx
}

func closeContext(ctx *v8go.Context) {
	ctx.Close()
	ctx.Isolate().Dispose()
}

func TestShareData(t *testing.T) {
	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	defer closeContext(ctx)

	tests := []struct {
		name     string
		share    *Share
		validate func(t *testing.T, result *Share)
	}{
		{
			name: "Basic share data",
			share: &Share{
				Sid:    "session123",
				Root:   true,
				Global: map[string]interface{}{"key": "value"},
				Iso:    "isolate001",
			},
			validate: func(t *testing.T, result *Share) {
				if result.Sid != "session123" {
					t.Errorf("Sid = %v, want session123", result.Sid)
				}
				if result.Root != true {
					t.Errorf("Root = %v, want true", result.Root)
				}
				if result.Global["key"] != "value" {
					t.Errorf("Global[key] = %v, want value", result.Global["key"])
				}
				if result.Iso != "isolate001" {
					t.Errorf("Iso = %v, want isolate001", result.Iso)
				}
				if result.Authorized != nil {
					t.Errorf("Authorized = %v, want nil", result.Authorized)
				}
			},
		},
		{
			name: "Share data with Authorized",
			share: &Share{
				Sid:    "session456",
				Root:   false,
				Global: map[string]interface{}{"hello": "world"},
				Authorized: map[string]interface{}{
					"user_id": "user123",
					"team_id": "team456",
					"scope":   "read write",
					"constraints": map[string]interface{}{
						"team_only": true,
						"extra": map[string]interface{}{
							"department": "engineering",
						},
					},
				},
			},
			validate: func(t *testing.T, result *Share) {
				if result.Sid != "session456" {
					t.Errorf("Sid = %v, want session456", result.Sid)
				}
				if result.Root != false {
					t.Errorf("Root = %v, want false", result.Root)
				}
				if result.Global["hello"] != "world" {
					t.Errorf("Global[hello] = %v, want world", result.Global["hello"])
				}

				if result.Authorized == nil {
					t.Fatal("Authorized should not be nil")
				}

				if result.Authorized["user_id"] != "user123" {
					t.Errorf("Authorized[user_id] = %v, want user123", result.Authorized["user_id"])
				}
				if result.Authorized["team_id"] != "team456" {
					t.Errorf("Authorized[team_id] = %v, want team456", result.Authorized["team_id"])
				}
				if result.Authorized["scope"] != "read write" {
					t.Errorf("Authorized[scope] = %v, want read write", result.Authorized["scope"])
				}

				constraints, ok := result.Authorized["constraints"].(map[string]interface{})
				if !ok {
					t.Fatal("Authorized[constraints] should be map[string]interface{}")
				}
				if constraints["team_only"] != true {
					t.Errorf("constraints[team_only] = %v, want true", constraints["team_only"])
				}
			},
		},
		{
			name: "Empty share data",
			share: &Share{
				Sid:    "",
				Root:   false,
				Global: map[string]interface{}{},
			},
			validate: func(t *testing.T, result *Share) {
				if result.Sid != "" {
					t.Errorf("Sid = %v, want empty", result.Sid)
				}
				if result.Root != false {
					t.Errorf("Root = %v, want false", result.Root)
				}
				if len(result.Global) != 0 {
					t.Errorf("Global length = %v, want 0", len(result.Global))
				}
				if result.Authorized != nil {
					t.Errorf("Authorized = %v, want nil", result.Authorized)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set share data
			err := SetShareData(ctx, ctx.Global(), tt.share)
			if err != nil {
				t.Fatalf("SetShareData failed: %v", err)
			}

			// Retrieve share data
			result, err := ShareData(ctx)
			if err != nil {
				t.Fatalf("ShareData failed: %v", err)
			}

			// Validate
			tt.validate(t, result)
		})
	}
}

func TestShareDataRoundTrip(t *testing.T) {
	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	defer closeContext(ctx)

	original := &Share{
		Sid:  "roundtrip-session",
		Root: true,
		Global: map[string]interface{}{
			"string": "test",
			"number": float64(42),
			"bool":   true,
			"nested": map[string]interface{}{
				"key": "value",
			},
		},
		Iso: "roundtrip-iso",
		Authorized: map[string]interface{}{
			"user_id":    "user999",
			"team_id":    "team888",
			"scope":      "admin",
			"session_id": "sess777",
			"constraints": map[string]interface{}{
				"owner_only":   true,
				"creator_only": false,
				"team_only":    true,
				"extra": map[string]interface{}{
					"region":     "us-west",
					"department": "sales",
				},
			},
		},
	}

	// Set share data
	err := SetShareData(ctx, ctx.Global(), original)
	if err != nil {
		t.Fatalf("SetShareData failed: %v", err)
	}

	// Retrieve share data
	retrieved, err := ShareData(ctx)
	if err != nil {
		t.Fatalf("ShareData failed: %v", err)
	}

	// Compare all fields
	if retrieved.Sid != original.Sid {
		t.Errorf("Sid mismatch: got %v, want %v", retrieved.Sid, original.Sid)
	}
	if retrieved.Root != original.Root {
		t.Errorf("Root mismatch: got %v, want %v", retrieved.Root, original.Root)
	}
	if retrieved.Iso != original.Iso {
		t.Errorf("Iso mismatch: got %v, want %v", retrieved.Iso, original.Iso)
	}

	// Compare Global
	if retrieved.Global["string"] != original.Global["string"] {
		t.Errorf("Global[string] mismatch: got %v, want %v", retrieved.Global["string"], original.Global["string"])
	}
	if retrieved.Global["number"] != original.Global["number"] {
		t.Errorf("Global[number] mismatch: got %v, want %v", retrieved.Global["number"], original.Global["number"])
	}
	if retrieved.Global["bool"] != original.Global["bool"] {
		t.Errorf("Global[bool] mismatch: got %v, want %v", retrieved.Global["bool"], original.Global["bool"])
	}

	// Compare Authorized
	if retrieved.Authorized == nil {
		t.Fatal("Authorized should not be nil after round trip")
	}

	if retrieved.Authorized["user_id"] != original.Authorized["user_id"] {
		t.Errorf("Authorized[user_id] mismatch: got %v, want %v", retrieved.Authorized["user_id"], original.Authorized["user_id"])
	}
	if retrieved.Authorized["team_id"] != original.Authorized["team_id"] {
		t.Errorf("Authorized[team_id] mismatch: got %v, want %v", retrieved.Authorized["team_id"], original.Authorized["team_id"])
	}
	if retrieved.Authorized["scope"] != original.Authorized["scope"] {
		t.Errorf("Authorized[scope] mismatch: got %v, want %v", retrieved.Authorized["scope"], original.Authorized["scope"])
	}

	// Compare constraints
	retrievedConstraints, ok := retrieved.Authorized["constraints"].(map[string]interface{})
	if !ok {
		t.Fatal("Retrieved constraints should be map[string]interface{}")
	}
	originalConstraints, _ := original.Authorized["constraints"].(map[string]interface{})

	if retrievedConstraints["owner_only"] != originalConstraints["owner_only"] {
		t.Errorf("constraints[owner_only] mismatch: got %v, want %v", retrievedConstraints["owner_only"], originalConstraints["owner_only"])
	}
	if retrievedConstraints["team_only"] != originalConstraints["team_only"] {
		t.Errorf("constraints[team_only] mismatch: got %v, want %v", retrievedConstraints["team_only"], originalConstraints["team_only"])
	}

	// Compare extra
	retrievedExtra, ok := retrievedConstraints["extra"].(map[string]interface{})
	if !ok {
		t.Fatal("Retrieved extra should be map[string]interface{}")
	}
	originalExtra, _ := originalConstraints["extra"].(map[string]interface{})

	if retrievedExtra["region"] != originalExtra["region"] {
		t.Errorf("extra[region] mismatch: got %v, want %v", retrievedExtra["region"], originalExtra["region"])
	}
	if retrievedExtra["department"] != originalExtra["department"] {
		t.Errorf("extra[department] mismatch: got %v, want %v", retrievedExtra["department"], originalExtra["department"])
	}
}

func TestShareDataWithNilAuthorized(t *testing.T) {
	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	defer closeContext(ctx)

	share := &Share{
		Sid:        "test-session",
		Root:       false,
		Global:     map[string]interface{}{"key": "value"},
		Authorized: nil,
	}

	// Set share data
	err := SetShareData(ctx, ctx.Global(), share)
	if err != nil {
		t.Fatalf("SetShareData failed: %v", err)
	}

	// Retrieve share data
	result, err := ShareData(ctx)
	if err != nil {
		t.Fatalf("ShareData failed: %v", err)
	}

	// Authorized should be nil when not set
	if result.Authorized != nil {
		t.Errorf("Authorized should be nil when not set, got: %v", result.Authorized)
	}
}
