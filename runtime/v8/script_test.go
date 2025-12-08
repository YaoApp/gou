package v8

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
)

// func TestLoad(t *testing.T) {
// 	prepare(t)
// 	time.Sleep(20 * time.Millisecond)
// 	assert.Equal(t, 3, len(Scripts))
// 	assert.Equal(t, 1, len(RootScripts))
// 	assert.Equal(t, 2, len(chIsoReady))
// }

func TestTransformTS(t *testing.T) {

	option := option()
	option.Mode = "standard"
	option.Import = true
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	files := map[string]string{
		"app.ts":       filepath.Join("scripts", "runtime", "ts", "app.ts"),
		"lib.hello.ts": filepath.Join("scripts", "runtime", "ts", "lib", "hello.ts"),
	}

	app, err := application.App.Read(files["app.ts"])
	if err != nil {
		t.Fatal(err)
	}

	appSource, err := TransformTS(files["app.ts"], app)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, appSource)
	imports := ImportMap[files["app.ts"]]
	assert.Len(t, imports, 3)
	for _, im := range imports {
		module, has := Modules[im.AbsPath]
		assert.True(t, has)
		assert.NotEmpty(t, module.Source)
	}
}

func TestTransformTSWithTSConfig(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.Import = true
	option.HeapSizeLimit = 4294967296

	// add tsconfig
	tsconfig := &TSConfig{
		CompilerOptions: &TSConfigCompilerOptions{
			Paths: map[string][]string{
				"@yao/*": {"./scripts/.types/*"},
				"@lib/*": {"./scripts/runtime/ts/lib/*"},
			},
		},
	}
	option.TSConfig = tsconfig

	prepare(t, option)
	defer Stop()

	files := map[string]string{
		"page.ts":      filepath.Join("scripts", "runtime", "ts", "page.ts"),
		"lib.hello.ts": filepath.Join("scripts", "runtime", "ts", "lib", "hello.ts"),
	}

	page, err := application.App.Read(files["page.ts"])
	if err != nil {
		t.Fatal(err)
	}

	pageSource, err := TransformTS(files["page.ts"], page)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, pageSource)
	imports := ImportMap[files["page.ts"]]
	assert.Len(t, imports, 2)
	for _, im := range imports {
		module, has := Modules[im.AbsPath]
		assert.True(t, has)
		assert.NotEmpty(t, module.Source)
	}
}

func TestExecStandard(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.Import = true
	option.HeapSizeLimit = 4294967296
	prepare(t, option)
	defer Stop()

	Load(filepath.Join("scripts", "runtime", "ts", "app.ts"), "runtime.ts.app")
	script, err := Select("runtime.ts.app")
	if err != nil {
		t.Fatal(err)
	}

	p := process.New("scripts.runtime.ts.app.FooBar")
	res := script.Exec(p)
	data, ok := res.([]interface{})
	if !ok {
		t.Fatal("result error")
	}

	assert.Len(t, data, 3)
	assert.Contains(t, data[0], "Hello")
}

func TestContextWithAuthorized(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Create a test script
	script := &Script{
		ID:   "test.context.authorized",
		Root: true,
	}

	// Create context with authorized information
	authorized := map[string]interface{}{
		"user_id": "user123",
		"team_id": "team456",
		"scope":   "read write",
		"constraints": map[string]interface{}{
			"team_only": true,
		},
	}

	ctx, err := script.NewContext("test-session", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	// Set authorized info
	ctx.WithAuthorized(authorized)

	// Verify authorized is set
	assert.NotNil(t, ctx.Authorized)
	assert.Equal(t, "user123", ctx.Authorized["user_id"])
	assert.Equal(t, "team456", ctx.Authorized["team_id"])
	assert.Equal(t, "read write", ctx.Authorized["scope"])

	constraints, ok := ctx.Authorized["constraints"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, constraints["team_only"])
}

func TestContextCloseCleanupAuthorized(t *testing.T) {
	option := option()
	option.Mode = "standard"
	prepare(t, option)
	defer Stop()

	// Create a test script
	script := &Script{
		ID:   "test.context.cleanup",
		Root: false,
	}

	// Create context with authorized information
	authorized := map[string]interface{}{
		"user_id": "user999",
	}

	ctx, err := script.NewContext("test-session-2", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}

	ctx.WithAuthorized(authorized)

	// Verify authorized is set
	assert.NotNil(t, ctx.Authorized)
	assert.Equal(t, "user999", ctx.Authorized["user_id"])

	// Close context
	err = ctx.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Verify authorized is cleaned up
	assert.Nil(t, ctx.Authorized)
	assert.Nil(t, ctx.Data)
}
