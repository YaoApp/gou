package plan

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
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
		id = plan.id;
		plan.Release();
		return id;	
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
		plan.Add("task-1", 1, "unit.test.task11");
		status = plan.Status();
		plan.Release();
		return status;
	}

	function test2() {
		const plan = new Plan("test2-plan");
		plan.Add("task-2", 1, "unit.test.task21");
		status = plan.Status();
		plan.Release();
		return status;
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
		plan.Add("task-1", 1, "unit.test.task11");
		status = plan.TaskStatus("task-1");
		plan.Release();
		return status;
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

func TestPlanRun(t *testing.T) {
	ctx := prepare()
	defer close(ctx)

	_, err := ctx.RunScript(`
	function task1(task, shared) {
		print("hello");
	}
	function test() {
		const plan = new Plan("test-plan");
		plan.Add("task-1", 1, "unit.test.task11");
		plan.Add("task-2", 1, "unit.test.task12");
		plan.Add("task-3", 2, "unit.test.task21");
		plan.Add("task-4", 2, "unit.test.task22");
		plan.Run();
		plan.Release(); // release the plan
	}
	test();
	`, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPlanEvents(t *testing.T) {
	ctx := prepare()
	defer close(ctx)

	v, err := ctx.RunScript(`
	function test() {
		const plan = new Plan("test-plan");
		plan.Subscribe("TaskStarted", "unit.test.subscribe");
		plan.Subscribe("TaskCompleted", "unit.test.subscribe", "hi");
		plan.Add("task-1", 1, "unit.test.task11");
		plan.Add("task-2", 1, "unit.test.task12");
		plan.Add("task-3", 2, "unit.test.task21");
		plan.Add("task-4", 2, "unit.test.task22");
		plan.Run();

		// Get Plan Status
		const plan2 = new Plan("test-plan");
		status = plan2.Status();
		plan2.Release();
		return status;
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

	dot := maps.Of(res.(map[string]interface{})).Dot()
	assert.Equal(t, "completed", dot.Get("plan"))
	assert.Equal(t, "completed", dot.Get("tasks.task-1"))
	assert.Equal(t, "completed", dot.Get("tasks.task-2"))
	assert.Equal(t, "completed", dot.Get("tasks.task-3"))
	assert.Equal(t, "completed", dot.Get("tasks.task-4"))
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
	template.Set("print", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		fmt.Println(info.Args()[0].String())
		return v8go.Undefined(iso)
	}))
	ctx := v8go.NewContext(iso, template)

	process.Register("unit.test.task11", func(process *process.Process) interface{} {
		fmt.Println(" unit.test.task11:", process.Args)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	process.Register("unit.test.task12", func(process *process.Process) interface{} {
		fmt.Println(" unit.test.task12:", process.Args)
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	process.Register("unit.test.task21", func(process *process.Process) interface{} {
		fmt.Println(" unit.test.task21:", process.Args)
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	process.Register("unit.test.task22", func(process *process.Process) interface{} {
		fmt.Println(" unit.test.task22:", process.Args)
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	process.Register("unit.test.subscribe", func(process *process.Process) interface{} {
		plan := process.ArgsString(0)
		key := process.ArgsString(1)
		value := process.Args[2]
		if value != nil {
			restArgs := make([]interface{}, 0)
			for i := 3; i < len(process.Args); i++ {
				restArgs = append(restArgs, process.Args[i])
			}

			fmt.Println(" process.subscribe:", plan, key, value, restArgs)
		}

		return nil
	})

	return ctx
}
