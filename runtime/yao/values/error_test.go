package values

import (
	"testing"

	"github.com/go-playground/assert/v2"
	"rogchap.com/v8go"
)

func TestError(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	ctx := v8go.NewContext(iso)
	defer ctx.Close()

	v := Error(ctx, "hello")

	obj, err := v.AsObject()
	if err != nil {
		t.Fatal(err)
	}

	message, err := obj.Get("message")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello", message.String())
}

func TestErrorWithoutThrow(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	ctx := v8go.NewContext(iso)
	defer ctx.Close()

	_, err := ctx.RunScript(`
		var ErrorSys = Error
		Error = undefined;
	`, "")

	if err != nil {
		t.Fatal(err)
	}

	v := Error(ctx, "hello")

	obj, err := v.AsObject()
	if err != nil {
		t.Fatal(err)
	}

	message, err := obj.Get("message")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello", message.String())
}
