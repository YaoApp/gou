package log

import (
	"testing"

	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

func TestLogObject(t *testing.T) {

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := &Object{}
	global := v8go.NewObjectTemplate(iso)
	global.Set("log", obj.ExportObject(iso))

	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	log.SetLevel(log.TraceLevel)

	// ===== Trace
	_, err := ctx.RunScript(`log.Trace("Trace: %s %v %#v", "hello world", ["foo", "bar"], {"foo":"bar"})`, "")
	if err != nil {
		t.Fatal(err)
	}

	// ===== Debug
	_, err = ctx.RunScript(`log.Debug("Debug: %s %v %#v", "hello world", ["foo", "bar"], {"foo":"bar"})`, "")
	if err != nil {
		t.Fatal(err)
	}

	// ===== Info
	_, err = ctx.RunScript(`log.Info("Info: %s %v %#v", "hello world", ["foo", "bar"], {"foo":"bar"})`, "")
	if err != nil {
		t.Fatal(err)
	}

	// ===== Error
	_, err = ctx.RunScript(`log.Error("Error: %s %v %#v", "hello world", ["foo", "bar"], {"foo":"bar"})`, "")
	if err != nil {
		t.Fatal(err)
	}
}
