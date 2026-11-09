package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
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

func TestAfter(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := &Object{}
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	obj.Set("time", ctx)

	testArgs := []interface{}{}

	process.Register("unit.test.time", func(process *process.Process) interface{} {
		testArgs = process.Args
		return nil
	})

	_, err := ctx.RunScript(`
		time.After(200, "unit.test.time", "foo", "bar")
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, []interface{}{"foo", "bar"}, testArgs)

}

func TestAfterError(t *testing.T) {
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	obj := &Object{}
	ctx := v8go.NewContext(iso)
	defer ctx.Close()
	obj.Set("time", ctx)

	testArgs := []interface{}{}

	process.Register("unit.test.time", func(process *process.Process) interface{} {
		testArgs = process.Args
		exception.New("unit.test.time", 200).Throw()
		return nil
	})

	_, err := ctx.RunScript(`
		time.After(200, "unit.test.time", "foo", "bar")
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, []interface{}{"foo", "bar"}, testArgs)

}
