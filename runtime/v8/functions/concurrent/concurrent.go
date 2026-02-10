package concurrent

import (
	"fmt"
	"sync"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Task represents a single process call to execute concurrently.
type Task struct {
	Process string        // Process name (e.g. "scripts.runtime.basic.Hello")
	Args    []interface{} // Process arguments
}

// TaskResult holds the outcome of a single concurrent task.
type TaskResult struct {
	Data  interface{} `json:"data"`            // Return value on success
	Error string      `json:"error,omitempty"` // Error message on failure
	Index int         `json:"index"`           // Original task index (preserves order)
}

// ParseTasks extracts a []Task from the first JS argument (an array of {process, args} objects).
// Returns nil and an error string if parsing fails.
func ParseTasks(info *v8go.FunctionCallbackInfo) ([]Task, *bridge.Share, error) {
	jsArgs := info.Args()
	if len(jsArgs) < 1 {
		return nil, nil, fmt.Errorf("missing parameters: expected an array of tasks")
	}

	if !jsArgs[0].IsArray() {
		return nil, nil, fmt.Errorf("the first parameter should be an array of tasks")
	}

	share, err := bridge.ShareData(info.Context())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share data: %v", err)
	}

	// Convert JS array to Go slice
	goVal, err := bridge.GoValue(jsArgs[0], info.Context())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse tasks array: %v", err)
	}

	arr, ok := goVal.([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("the first parameter should be an array of tasks")
	}

	tasks := make([]Task, 0, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("task[%d]: expected an object with 'process' field", i)
		}

		procName, ok := m["process"].(string)
		if !ok || procName == "" {
			return nil, nil, fmt.Errorf("task[%d]: 'process' field is required and must be a string", i)
		}

		var args []interface{}
		if rawArgs, exists := m["args"]; exists {
			if argArr, ok := rawArgs.([]interface{}); ok {
				args = argArr
			}
		}

		tasks = append(tasks, Task{Process: procName, Args: args})
	}

	return tasks, share, nil
}

// executeTask runs a single process and returns the result.
// Safe to call from a goroutine — uses process.Of (returns error) instead of process.New (panics).
func executeTask(t Task, share *bridge.Share) TaskResult {
	proc, err := process.Of(t.Process, t.Args...)
	if err != nil {
		return TaskResult{Error: err.Error()}
	}

	proc = proc.WithGlobal(share.Global).WithSID(share.Sid)
	if share.Authorized != nil {
		proc = proc.WithAuthorized(share.Authorized)
	}

	err = proc.Execute()
	if err != nil {
		return TaskResult{Error: err.Error()}
	}

	val := proc.Value()
	proc.Release()
	return TaskResult{Data: val}
}

// ParallelAll executes all tasks concurrently and waits for every task to complete.
// Semantics: Promise.all — all results returned, order preserved.
func ParallelAll(tasks []Task, share *bridge.Share) []TaskResult {
	results := make([]TaskResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[idx] = TaskResult{Error: fmt.Sprint(r), Index: idx}
				}
			}()
			res := executeTask(t, share)
			res.Index = idx
			results[idx] = res
		}(i, task)
	}

	wg.Wait()
	return results
}

// ParallelAny executes all tasks concurrently and returns once the first task succeeds.
// Semantics: Promise.any — first success wins; all tasks still run to completion to avoid leaks.
// A "success" is defined as data != nil and error == "".
func ParallelAny(tasks []Task, share *bridge.Share) []TaskResult {
	results := make([]TaskResult, len(tasks))
	done := make(chan struct{})

	type indexedResult struct {
		idx    int
		result TaskResult
	}
	ch := make(chan indexedResult, len(tasks))

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()

			// Skip work if already done
			select {
			case <-done:
				return
			default:
			}

			defer func() {
				if r := recover(); r != nil {
					ch <- indexedResult{idx, TaskResult{Error: fmt.Sprint(r), Index: idx}}
				}
			}()

			res := executeTask(t, share)
			res.Index = idx
			select {
			case <-done:
			case ch <- indexedResult{idx, res}:
			}
		}(i, task)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results; close done on first success
	foundSuccess := false
	for ir := range ch {
		results[ir.idx] = ir.result
		if !foundSuccess && ir.result.Error == "" && ir.result.Data != nil {
			foundSuccess = true
			close(done)
		}
	}

	return results
}

// ParallelRace executes all tasks concurrently and returns once the first task completes.
// Semantics: Promise.race — first completion wins (success or failure).
func ParallelRace(tasks []Task, share *bridge.Share) []TaskResult {
	results := make([]TaskResult, len(tasks))
	done := make(chan struct{})

	type indexedResult struct {
		idx    int
		result TaskResult
	}
	ch := make(chan indexedResult, len(tasks))

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()

			select {
			case <-done:
				return
			default:
			}

			defer func() {
				if r := recover(); r != nil {
					ch <- indexedResult{idx, TaskResult{Error: fmt.Sprint(r), Index: idx}}
				}
			}()

			res := executeTask(t, share)
			res.Index = idx
			select {
			case <-done:
			case ch <- indexedResult{idx, res}:
			}
		}(i, task)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	gotFirst := false
	for ir := range ch {
		results[ir.idx] = ir.result
		if !gotFirst {
			gotFirst = true
			close(done)
		}
	}

	return results
}
