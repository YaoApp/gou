package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func TestJob(t *testing.T) {

	ctx := prepare(t, false, "", nil)
	defer close(ctx)

	jsRes, err := ctx.RunScript(`
		const test = () => {
			let job = new Job("test.job.run", "http://test.com", "foo=bar");
			let progress = 0
			job.Pending(() => {
				progress ++
			});
			let data = job.Data()
			return {progress:progress, ...data};
		}
		test()
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	goRes, err := bridge.GoValue(jsRes)
	if err != nil {
		t.Fatal(err)
	}

	res, ok := goRes.(map[string]interface{})
	if !ok {
		t.Fatal("result error")
	}

	assert.Equal(t, "hello", res["message"])
	assert.Equal(t, "http://test.com", res["url"])
	assert.Equal(t, "foo=bar", res["payload"])
	assert.Greater(t, res["progress"], float64(0))
}

func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

func prepare(t *testing.T, root bool, sid string, global map[string]interface{}) *v8go.Context {

	iso := v8go.NewIsolate()

	template := v8go.NewObjectTemplate(iso)
	template.Set("Job", New().ExportFunction(iso))

	ctx := v8go.NewContext(iso, template)

	process.Register("test.job.run", func(process *process.Process) interface{} {
		time.Sleep(200 * time.Millisecond)
		url := process.ArgsString(0)
		payload := process.ArgsString(1)
		return map[string]interface{}{"message": "hello", "url": url, "payload": payload}
	})

	return ctx
}
