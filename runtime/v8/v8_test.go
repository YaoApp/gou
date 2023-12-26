package v8

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaoapp/gou/application"
)

func option() *Option {
	option := &Option{}
	option.Validate()
	return option
}

func prepare(t *testing.T, option *Option) {
	root := os.Getenv("GOU_TEST_APPLICATION")

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

	// application scripts
	scripts := map[string]string{
		"runtime.basic":      filepath.Join("scripts", "runtime", "basic.js"),
		"runtime.lib":        filepath.Join("scripts", "runtime", "lib.js"),
		"runtime.typescript": filepath.Join("scripts", "runtime", "typescript.ts"),
	}

	for id, file := range scripts {
		_, err := Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	// root scripts
	rootScripts := map[string]string{
		"runtime.basic": filepath.Join("studio", "runtime", "basic.js"),
	}

	for id, file := range rootScripts {
		_, err := LoadRoot(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	prepareSetup(t, option)
}

func prepareSetup(t *testing.T, option *Option) {
	EnablePrecompile()
	Start(option)
}
