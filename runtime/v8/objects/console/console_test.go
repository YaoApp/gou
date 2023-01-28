package console

import (
	"testing"

	"rogchap.com/v8go"
)

func TestLog(t *testing.T) {

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := &Object{}
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	obj.Set("console", ctx)

	_, err := ctx.RunScript(`
		const test = () => console.log("hello world", ["foo", "bar"], {"foo":"bar"})
		test()
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Printf("%#v\n", v.IsUndefined())
}
