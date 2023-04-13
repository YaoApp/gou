package time

import (
	"testing"

	"rogchap.com/v8go"
)

func TestSleep(t *testing.T) {

	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := &Object{}
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	obj.Set("time", ctx)

	_, err := ctx.RunScript(`
		const test = () => time.Sleep(200)
		test()
	`, "")
	if err != nil {
		t.Fatal(err)
	}

}
