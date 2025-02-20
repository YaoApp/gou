package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/maps"
	"rogchap.com/v8go"
)

func TestNewPlan(t *testing.T) {
	ctx := prepare()
	defer close(ctx)

	v, err := ctx.RunScript(`
	function test() {
		const plan = new Plan("test-plan");
		return plan.id;
	}
	test();
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "test-plan", res)
}

func TestPlanAdd(t *testing.T) {
	ctx := prepare()
	defer close(ctx)

	v, err := ctx.RunScript(`
	function test1() {
		const plan = new Plan("test-plan");
		plan.Add("task-1", 1, function(task, shared) {
			shared.Set("foo", "bar");
		});
		return plan.Status();
	}

	function test2() {
		const plan = new Plan("test2-plan");
		plan.Add("task-2", 1, function(task, shared) {
			shared.Set("foo", "bar");
		});
		return plan.Status();
	}

	function test() {
		return {
			test1: test1(),
			test2: test2(),
		}
	}
	test();
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}

	resMap, ok := res.(map[string]interface{})
	if !ok {
		t.Fatal("res is not a map")
	}

	dot := maps.Of(resMap).Dot()
	assert.Equal(t, "created", dot.Get("test1.plan"))
	assert.Equal(t, "created", dot.Get("test2.plan"))
	assert.Equal(t, "created", dot.Get("test1.tasks.task-1"))
	assert.Equal(t, "created", dot.Get("test2.tasks.task-2"))
}

func TestPlanTaskStatus(t *testing.T) {
	ctx := prepare()
	defer close(ctx)

	v, err := ctx.RunScript(`
	function test() {
		const plan = new Plan("test-plan");
		plan.Add("task-1", 1, function(task, shared) {
			shared.Set("foo", "bar");
		});
		return plan.TaskStatus("task-1");
	}
	test();
	`, "")
	if err != nil {
		t.Fatal(err)
	}

	res, err := bridge.GoValue(v, ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "created", res)
}

// close close the v8go context
func close(ctx *v8go.Context) {
	ctx.Isolate().Dispose()
}

// prepare prepare the v8go context
func prepare() *v8go.Context {
	iso := v8go.NewIsolate()
	template := v8go.NewObjectTemplate(iso)
	template.Set("Plan", New().ExportFunction(iso))
	ctx := v8go.NewContext(iso, template)
	return ctx
}
