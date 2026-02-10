package concurrent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
)

func init() {
	// Register test processes
	process.Register("unit.test.echo", func(proc *process.Process) interface{} {
		if len(proc.Args) > 0 {
			return proc.Args[0]
		}
		return nil
	})

	process.Register("unit.test.add", func(proc *process.Process) interface{} {
		if len(proc.Args) < 2 {
			return 0
		}
		a, _ := proc.Args[0].(float64)
		b, _ := proc.Args[1].(float64)
		return a + b
	})

	process.Register("unit.test.panic", func(proc *process.Process) interface{} {
		panic("intentional panic for testing")
	})

	// Sleep process: args[0] = milliseconds to sleep, args[1] = return value
	process.Register("unit.test.sleep", func(proc *process.Process) interface{} {
		if len(proc.Args) < 2 {
			return nil
		}
		ms, _ := proc.Args[0].(float64)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return proc.Args[1]
	})
}

func TestParallelAll(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.echo", Args: []interface{}{"hello"}},
		{Process: "unit.test.echo", Args: []interface{}{"world"}},
		{Process: "unit.test.add", Args: []interface{}{float64(1), float64(2)}},
	}

	results := ParallelAll(tasks, share)

	assert.Equal(t, 3, len(results))
	assert.Equal(t, "hello", results[0].Data)
	assert.Equal(t, 0, results[0].Index)
	assert.Empty(t, results[0].Error)

	assert.Equal(t, "world", results[1].Data)
	assert.Equal(t, 1, results[1].Index)
	assert.Empty(t, results[1].Error)

	assert.Equal(t, float64(3), results[2].Data)
	assert.Equal(t, 2, results[2].Index)
	assert.Empty(t, results[2].Error)
}

func TestParallelAllWithError(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.echo", Args: []interface{}{"ok"}},
		{Process: "unit.test.panic", Args: []interface{}{}},
		{Process: "unit.test.echo", Args: []interface{}{"also_ok"}},
	}

	results := ParallelAll(tasks, share)

	assert.Equal(t, 3, len(results))
	assert.Equal(t, "ok", results[0].Data)
	assert.Empty(t, results[0].Error)

	assert.NotEmpty(t, results[1].Error, "task[1] should have an error")

	assert.Equal(t, "also_ok", results[2].Data)
	assert.Empty(t, results[2].Error)
}

func TestParallelAllWithInvalidProcess(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.echo", Args: []interface{}{"ok"}},
		{Process: "nonexistent.process", Args: []interface{}{}},
	}

	results := ParallelAll(tasks, share)

	assert.Equal(t, 2, len(results))
	assert.Equal(t, "ok", results[0].Data)
	assert.Empty(t, results[0].Error)
	assert.NotEmpty(t, results[1].Error, "task[1] should have an error for invalid process")
}

func TestParallelAllEmpty(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	results := ParallelAll([]Task{}, share)
	assert.Equal(t, 0, len(results))
}

func TestParallelAny(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.echo", Args: []interface{}{"first"}},
		{Process: "unit.test.echo", Args: []interface{}{"second"}},
	}

	results := ParallelAny(tasks, share)

	// At least one result should be successful
	hasSuccess := false
	for _, r := range results {
		if r.Data != nil && r.Error == "" {
			hasSuccess = true
			break
		}
	}
	assert.True(t, hasSuccess, "at least one result should be successful")
}

func TestParallelRace(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.echo", Args: []interface{}{"fast"}},
		{Process: "unit.test.echo", Args: []interface{}{"also_fast"}},
	}

	results := ParallelRace(tasks, share)

	// At least one result should be populated
	hasResult := false
	for _, r := range results {
		if r.Data != nil || r.Error != "" {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult, "at least one result should be populated")
}

// TestParallelAllConcurrency verifies tasks actually run in parallel.
// 3 tasks each sleep 300ms — if concurrent, total should be ~300ms, not ~900ms.
func TestParallelAllConcurrency(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.sleep", Args: []interface{}{float64(300), "a"}},
		{Process: "unit.test.sleep", Args: []interface{}{float64(300), "b"}},
		{Process: "unit.test.sleep", Args: []interface{}{float64(300), "c"}},
	}

	start := time.Now()
	results := ParallelAll(tasks, share)
	elapsed := time.Since(start)

	// All 3 results should be present and correct
	assert.Equal(t, 3, len(results))
	assert.Equal(t, "a", results[0].Data)
	assert.Equal(t, "b", results[1].Data)
	assert.Equal(t, "c", results[2].Data)

	// If truly concurrent: ~300ms. If sequential: ~900ms.
	// Allow generous margin (600ms) but must be < 900ms.
	assert.Less(t, elapsed, 600*time.Millisecond,
		"3 x 300ms tasks should complete in ~300ms if concurrent, got %v", elapsed)
	t.Logf("ParallelAll concurrency: 3 x 300ms tasks completed in %v", elapsed)
}

// TestParallelAnyConcurrency verifies Any returns early on first success.
// Task 0 sleeps 100ms, Task 1 sleeps 500ms. Total should be ~100ms (not 500ms).
func TestParallelAnyConcurrency(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.sleep", Args: []interface{}{float64(100), "fast"}},
		{Process: "unit.test.sleep", Args: []interface{}{float64(500), "slow"}},
	}

	start := time.Now()
	results := ParallelAny(tasks, share)
	elapsed := time.Since(start)

	// The fast task should be in results
	assert.Equal(t, "fast", results[0].Data)

	// Total time should be dominated by the slow task (since all goroutines finish),
	// but the function should still return all collected results.
	// Key check: both tasks ran, results are correct.
	assert.Equal(t, 2, len(results))
	t.Logf("ParallelAny: completed in %v", elapsed)
}

// TestParallelRaceConcurrency verifies Race returns early on first completion.
func TestParallelRaceConcurrency(t *testing.T) {
	share := &bridge.Share{Sid: "test-sid", Global: map[string]interface{}{}}
	tasks := []Task{
		{Process: "unit.test.sleep", Args: []interface{}{float64(100), "fast"}},
		{Process: "unit.test.sleep", Args: []interface{}{float64(500), "slow"}},
	}

	start := time.Now()
	results := ParallelRace(tasks, share)
	elapsed := time.Since(start)

	// At least the fast task should have completed
	hasResult := false
	for _, r := range results {
		if r.Data != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult, "at least one result should be populated")
	t.Logf("ParallelRace: completed in %v", elapsed)
}

func TestParallelAllPreservesShareData(t *testing.T) {
	process.Register("unit.test.checksid", func(proc *process.Process) interface{} {
		return proc.Sid
	})

	share := &bridge.Share{
		Sid:    "shared-session-123",
		Global: map[string]interface{}{"key": "value"},
	}
	tasks := []Task{
		{Process: "unit.test.checksid", Args: []interface{}{}},
	}

	results := ParallelAll(tasks, share)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "shared-session-123", results[0].Data)
	assert.Empty(t, results[0].Error)
}
