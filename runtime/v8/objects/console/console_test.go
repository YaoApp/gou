package console

import (
	"testing"

	"rogchap.com/v8go"
)

func TestConsoleInDevelopmentMode(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := New("development")
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	err := obj.Set("console", ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("console.log", func(t *testing.T) {
		_, err := ctx.RunScript(`console.log("hello world", ["foo", "bar"], {"foo":"bar"})`, "")
		if err != nil {
			t.Fatal(err)
		}
		// Only verify that no error occurs, as we cannot reliably capture the output
	})

	t.Run("console.info", func(t *testing.T) {
		_, err := ctx.RunScript(`console.info("info message", 123)`, "")
		if err != nil {
			t.Fatal(err)
		}
		// Only verify that no error occurs
	})

	t.Run("console.warn", func(t *testing.T) {
		_, err := ctx.RunScript(`console.warn("warning message", true)`, "")
		if err != nil {
			t.Fatal(err)
		}
		// Only verify that no error occurs
	})

	t.Run("console.error", func(t *testing.T) {
		_, err := ctx.RunScript(`console.error("error message", null)`, "")
		if err != nil {
			t.Fatal(err)
		}
		// Only verify that no error occurs
	})
}

func TestConsoleInProductionMode(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := New("production")
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	err := obj.Set("console", ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("console.log should not output in production", func(t *testing.T) {
		// Cannot capture output, but console.log should not cause errors in production mode
		_, err := ctx.RunScript(`console.log("hello world")`, "")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("console.info, warn, error should still output in production", func(t *testing.T) {
		_, err := ctx.RunScript(`console.info("info in production")`, "")
		if err != nil {
			t.Fatal(err)
		}
		// Only verify that no error occurs
	})
}

func TestEmptyArgs(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := New("development")
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	err := obj.Set("console", ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ctx.RunScript(`console.log()`, "")
	if err == nil {
		t.Error("Expected error when calling console.log with no arguments")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestExportObject(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := New("development")
	template := obj.ExportObject(iso)

	if template == nil {
		t.Fatal("ExportObject should return a valid object template")
	}

	ctx := v8go.NewContext(iso)
	defer ctx.Close()

	consoleObj, err := template.NewInstance(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = ctx.Global().Set("console", consoleObj)
	if err != nil {
		t.Fatal(err)
	}

	// Only test if execution succeeds
	_, err = ctx.RunScript(`console.log("exported object test")`, "")
	if err != nil {
		t.Fatal(err)
	}
	// Since we cannot reliably capture the output, we only verify no errors occur
}
