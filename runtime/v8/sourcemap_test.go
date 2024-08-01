package v8

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"rogchap.com/v8go"
)

func TestStackTrace(t *testing.T) {
	option := option()
	option.Mode = "standard"
	option.Import = true
	option.HeapSizeLimit = 4294967296
	option.Debug = true

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

	source, err := application.App.Read(files["page.ts"])
	if err != nil {
		t.Fatal(err)
	}

	script, err := MakeScript(source, files["page.ts"], 5*time.Second)

	ctx, err := script.NewContext(uuid.New().String(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	global := ctx.Global()
	_, err = global.MethodCall("SomethingError")
	if err == nil {
		t.Fatal("error expected but got nil")
	}

	e, ok := err.(*v8go.JSError)
	if !ok {
		t.Fatal("error is not a JSError")
	}

	trace := StackTrace(e, nil)
	assert.NotEmpty(t, trace)
	assert.Contains(t, trace, "Exception|400: Error occurred")
	assert.Contains(t, trace, "/scripts/runtime/ts/page.ts:12:2")
	assert.Contains(t, trace, "/scripts/runtime/ts/lib/bar.ts:7:2")
	assert.Contains(t, trace, "/scripts/runtime/ts/lib/err.ts:8:10")

	// with source root
	trace = StackTrace(e, map[string]string{"/scripts": "/iscripts"})
	assert.NotEmpty(t, trace)
	assert.Contains(t, trace, "Exception|400: Error occurred")
	assert.Contains(t, trace, "/iscripts/runtime/ts/page.ts:12:2")
	assert.Contains(t, trace, "/iscripts/runtime/ts/lib/bar.ts:7:2")
	assert.Contains(t, trace, "/iscripts/runtime/ts/lib/err.ts:8:10")

	// with source root function
	replace := func(file string) string {
		if strings.HasPrefix(file, "/scripts") {
			return strings.Replace(file, "/scripts", "/fscripts", 1)
		}
		return file
	}

	trace = StackTrace(e, replace)
	assert.NotEmpty(t, trace)
	assert.Contains(t, trace, "Exception|400: Error occurred")
	assert.Contains(t, trace, "/fscripts/runtime/ts/page.ts:12:2")
	assert.Contains(t, trace, "/fscripts/runtime/ts/lib/bar.ts:7:2")
	assert.Contains(t, trace, "/fscripts/runtime/ts/lib/err.ts:8:10")
}
