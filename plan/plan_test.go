package plan

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// checkGoroutineLeaks runs a test function and checks for goroutine leaks
func checkGoroutineLeaks(t *testing.T, testFunc func()) {
	initialGoroutines := runtime.NumGoroutine()
	testFunc()

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	if finalGoroutines > initialGoroutines {
		buf := make([]byte, 4096)
		runtime.Stack(buf, true)
		t.Errorf("Goroutine leak: had %d, now have %d goroutines\nStack trace:\n%s",
			initialGoroutines, finalGoroutines, string(buf))
	}
}

func TestPlanBasicOperations(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		ctx := context.Background()
		shared := NewMemorySharedSpace()
		plan := NewPlan(ctx, "test-plan", shared, WithStatusCheckInterval(5*time.Millisecond))

		if plan.Status != StatusCreated {
			t.Errorf("Expected initial status to be StatusCreated, got %v", plan.Status)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		// Test adding tasks
		err := plan.AddTask("task1", 1, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			defer wg.Done()
			return shared.Set("task1", "completed")
		})
		if err != nil {
			t.Fatalf("Failed to add task1: %v", err)
		}

		err = plan.AddTask("task2", 2, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			defer wg.Done()
			return shared.Set("task2", "completed")
		})
		if err != nil {
			t.Fatalf("Failed to add task2: %v", err)
		}

		// Test starting plan
		err = plan.Start()
		if err != nil {
			t.Fatalf("Failed to start plan: %v", err)
		}

		// Wait for tasks to complete
		wg.Wait()

		// Verify task results
		val1, err := shared.Get("task1")
		if err != nil {
			t.Errorf("Failed to get task1 result: %v", err)
		}
		if val1 != "completed" {
			t.Errorf("Task1 failed to complete properly, got %v", val1)
		}

		val2, err := shared.Get("task2")
		if err != nil {
			t.Errorf("Failed to get task2 result: %v", err)
		}
		if val2 != "completed" {
			t.Errorf("Task2 failed to complete properly, got %v", val2)
		}

		// Check final status
		if plan.Status != StatusCompleted {
			t.Errorf("Expected plan status to be StatusCompleted, got %v", plan.Status)
		}

		// Clean up
		plan.Stop()
	})
}

func TestPlanParallelExecution(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		ctx := context.Background()
		shared := NewMemorySharedSpace()
		plan := NewPlan(ctx, "parallel-plan", shared, WithStatusCheckInterval(5*time.Millisecond))

		var wg sync.WaitGroup
		wg.Add(2)

		// Add two tasks with same order for parallel execution
		err := plan.AddTask("parallel1", 1, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			return shared.Set("parallel1", "completed")
		})
		if err != nil {
			t.Errorf("Failed to add parallel1: %v", err)
		}

		err = plan.AddTask("parallel2", 1, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			return shared.Set("parallel2", "completed")
		})
		if err != nil {
			t.Errorf("Failed to add parallel2: %v", err)
		}

		start := time.Now()
		err = plan.Start()
		if err != nil {
			t.Errorf("Failed to start plan: %v", err)
		}

		wg.Wait()
		duration := time.Since(start)

		// If tasks ran in parallel, duration should be ~50ms, not ~100ms
		if duration >= 90*time.Millisecond {
			t.Errorf("Tasks did not execute in parallel, took %v", duration)
		}

		// Clean up
		plan.Stop()
	})
}

func TestPlanCancellation(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		ctx := context.Background()
		shared := NewMemorySharedSpace()
		plan := NewPlan(ctx, "cancel-plan", shared, WithStatusCheckInterval(5*time.Millisecond))

		taskStarted := make(chan struct{})
		taskCompleted := make(chan struct{})

		err := plan.AddTask("long-running", 1, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			close(taskStarted)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
				close(taskCompleted)
				return nil
			case sig := <-signals:
				if sig == SignalStop {
					return context.Canceled
				}
				return nil
			}
		})
		if err != nil {
			t.Errorf("Failed to add task: %v", err)
		}

		// Start the plan in a goroutine
		go func() {
			err := plan.Start()
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled error, got %v", err)
			}
		}()

		// Wait for task to start
		<-taskStarted

		// Stop the plan
		err = plan.Stop()
		if err != nil {
			t.Errorf("Failed to stop plan: %v", err)
		}

		// Verify task was cancelled
		select {
		case <-taskCompleted:
			t.Error("Task completed despite cancellation")
		case <-time.After(100 * time.Millisecond):
			// Expected behavior - task was cancelled
		}
	})
}

func TestSharedSpaceSubscription(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		shared := NewMemorySharedSpace()
		notifications := make(chan string, 1)

		err := shared.Subscribe("test-key", func(key string, value interface{}) {
			if value != nil {
				notifications <- value.(string)
			}
		})
		if err != nil {
			t.Errorf("Failed to subscribe: %v", err)
		}

		err = shared.Set("test-key", "test-value")
		if err != nil {
			t.Errorf("Failed to set value: %v", err)
		}

		select {
		case value := <-notifications:
			if value != "test-value" {
				t.Errorf("Expected notification with 'test-value', got %v", value)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Did not receive notification in time")
		}

		// Clean up
		shared.Unsubscribe("test-key")
	})
}

func TestTaskSignalHandling(t *testing.T) {
	checkGoroutineLeaks(t, func() {
		ctx := context.Background()
		shared := NewMemorySharedSpace()
		// Create plan with custom status check interval
		plan := NewPlan(ctx, "signal-test-plan", shared, WithStatusCheckInterval(5*time.Millisecond))

		taskStatus := make(chan string, 10)
		done := make(chan struct{})

		// Add a long-running task that handles signals
		err := plan.AddTask("signal-task", 1, func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error {
			defer close(done)

			// Signal started
			select {
			case taskStatus <- "started":
			case <-ctx.Done():
				return ctx.Err()
			}

			// Main task loop
			for {
				select {
				case <-ctx.Done():
					taskStatus <- "stopped"
					return ctx.Err()
				case sig := <-signals:
					switch sig {
					case SignalPause:
						taskStatus <- "paused"
						// Update task status and wait for resume
						task := plan.Tasks["signal-task"]
						task.Status = StatusPaused
						// Wait for resume signal
						for sig := range signals {
							if sig == SignalResume {
								task.Status = StatusRunning
								taskStatus <- "resumed"
								break
							}
							if sig == SignalStop {
								taskStatus <- "stopped"
								task.Status = StatusCompleted
								return nil
							}
						}
					case SignalResume:
						taskStatus <- "resumed"
					case SignalStop:
						taskStatus <- "stopped"
						task := plan.Tasks["signal-task"]
						task.Status = StatusCompleted
						return nil
					}
				default:
					// Simulate some work
					time.Sleep(plan.Config.StatusCheckInterval)
				}
			}
		})
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}

		// Start the plan
		go func() {
			if err := plan.Start(); err != nil && err != context.Canceled {
				t.Errorf("Unexpected error: %v", err)
			}
		}()

		// Helper function to check status with timeout
		checkStatus := func(expected string) error {
			select {
			case status := <-taskStatus:
				if status != expected {
					return fmt.Errorf("expected status %s, got %s", expected, status)
				}
				return nil
			case <-time.After(100 * time.Millisecond):
				return fmt.Errorf("timeout waiting for status %s", expected)
			}
		}

		// Wait for task to start
		if err := checkStatus("started"); err != nil {
			t.Fatal(err)
		}

		// Test pause
		if err := plan.Pause(); err != nil {
			t.Fatalf("Failed to pause plan: %v", err)
		}
		if err := checkStatus("paused"); err != nil {
			t.Fatal(err)
		}

		// Test resume
		if err := plan.Resume(); err != nil {
			t.Fatalf("Failed to resume plan: %v", err)
		}
		if err := checkStatus("resumed"); err != nil {
			t.Fatal(err)
		}

		// Test stop
		if err := plan.Stop(); err != nil {
			t.Fatalf("Failed to stop plan: %v", err)
		}
		if err := checkStatus("stopped"); err != nil {
			t.Fatal(err)
		}

		// Wait for task to complete
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for task to complete")
		}
	})
}
