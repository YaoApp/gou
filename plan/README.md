# Plan Component

Plan is a Go package that provides a flexible task orchestration system with shared state management and signal control capabilities.

## Features

- Task ordering and parallel execution
- Shared state management between tasks
- Task lifecycle control (pause/resume/stop)
- Signal-based task communication
- Configurable status check intervals
- Thread-safe operations
- Resource cleanup management

## Installation

```bash
go get github.com/yaoapp/gou/plan
```

## Quick Start

```go
package main

import (
    "context"
    "time"
    "github.com/yaoapp/gou/plan"
)

func main() {
    // Create a shared space
    shared := plan.NewMemorySharedSpace()

    // Create a plan with custom status check interval
    p := plan.NewPlan(
        context.Background(),
        "example-plan",
        shared,
        plan.WithStatusCheckInterval(5*time.Millisecond),
    )

    // Add tasks
    p.AddTask("task1", 1, func(ctx context.Context, shared plan.SharedSpace, signals <-chan plan.Signal) error {
        // Task implementation
        return shared.Set("key1", "value1")
    })

    p.AddTask("task2", 2, func(ctx context.Context, shared plan.SharedSpace, signals <-chan plan.Signal) error {
        // Task implementation with signal handling
        select {
        case sig := <-signals:
            switch sig {
            case plan.SignalPause:
                // Handle pause
            case plan.SignalResume:
                // Handle resume
            case plan.SignalStop:
                return nil
            }
        case <-ctx.Done():
            return ctx.Err()
        }
        return nil
    })

    // Start the plan
    if err := p.Start(); err != nil {
        panic(err)
    }
}
```

## Core Concepts

### Plan

A Plan is a collection of tasks with a shared state space. It manages task execution order and lifecycle.

### Task

A Task is a unit of work that can:

- Access shared state
- Respond to control signals
- Execute in parallel with other tasks of the same order
- Report its status and store task-specific data

### Shared Space

SharedSpace provides a thread-safe storage mechanism for tasks to share data and communicate.

## API Reference

### Plan Creation

```go
func NewPlan(ctx context.Context, id string, shared SharedSpace, opts ...Option) *Plan
```

Options:

- `WithStatusCheckInterval(duration)`: Set the interval for status checks

### Task Management

```go
func (p *Plan) AddTask(id string, order int, fn TaskFunc) error
func (p *Plan) RemoveTask(id string) error
```

### Plan Control

```go
func (p *Plan) Start() error
func (p *Plan) Pause() error
func (p *Plan) Resume() error
func (p *Plan) Stop() error
```

### Status and Data

```go
func (p *Plan) GetStatus() (Status, map[string]Status)
func (p *Plan) GetTaskData(taskID string) (interface{}, error)
```

### Shared Space Operations

```go
func (s SharedSpace) Set(key string, value interface{}) error
func (s SharedSpace) Get(key string) (interface{}, error)
func (s SharedSpace) Subscribe(key string, callback func(key string, value interface{})) error
func (s SharedSpace) Unsubscribe(key string) error
```

## Task States

- `StatusCreated`: Initial state
- `StatusRunning`: Task is executing
- `StatusPaused`: Task is temporarily halted
- `StatusCompleted`: Task finished successfully
- `StatusFailed`: Task encountered an error
- `StatusDestroyed`: Task was terminated

## Best Practices

1. **Error Handling**: Always check for errors returned by plan operations.
2. **Signal Handling**: Implement proper signal handling in long-running tasks.
3. **Resource Cleanup**: Use `Stop()` to properly clean up resources.
4. **Context Usage**: Respect context cancellation in task implementations.
5. **Shared State**: Use shared space for task communication rather than external variables.

## Thread Safety

The Plan component is designed to be thread-safe:

- All plan operations are synchronized
- Shared space operations are protected by mutexes
- Signal channels are buffered to prevent blocking
