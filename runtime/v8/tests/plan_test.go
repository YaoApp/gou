package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/utils"
)

func TestPlan(t *testing.T) {
	option := v8.Option{}
	prepare(t, &option)
	defer v8.Stop()

	script, err := v8.Select("runtime.api.plan")
	if err != nil {
		t.Fatal(err)
	}

	p := process.New("scripts.runtime.api.plan.Test")
	res := script.Exec(p)
	utils.Dump(res)
}

func prepare(t *testing.T, option *v8.Option) {
	root := os.Getenv("GOU_TEST_APPLICATION")

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	// application scripts
	scripts := map[string]string{
		"runtime.api.plan": filepath.Join("scripts", "runtime", "api", "plan.ts"),
	}

	for id, file := range scripts {
		_, err := v8.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	prepareSetup(t, option)
}

func prepareSetup(t *testing.T, option *v8.Option) {
	v8.EnablePrecompile()
	v8.Start(option)
}
