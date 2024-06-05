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
