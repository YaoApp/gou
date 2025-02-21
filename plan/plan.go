package plan

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NewPlan creates a new Plan instance
func NewPlan(ctx context.Context, id string, shared SharedSpace, opts ...Option) *Plan {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	planCtx, cancel := context.WithCancel(ctx)
	return &Plan{
		ID:          id,
		Tasks:       make(map[string]*Task),
		SharedSpace: shared,
		Status:      StatusCreated,
		Context:     planCtx,
		Cancel:      cancel,
		Config:      config,
	}
}

// AddTask adds a new task to the plan
func (p *Plan) AddTask(id string, order int, fn TaskFunc) error {
	if _, exists := p.Tasks[id]; exists {
		return fmt.Errorf("task with ID %s already exists", id)
	}

	taskCtx, cancel := context.WithCancel(p.Context)
	task := &Task{
		ID:         id,
		Order:      order,
		Fn:         fn,
		Status:     StatusCreated,
		Context:    taskCtx,
		Cancel:     cancel,
		SignalChan: make(chan Signal, 1), // Buffer of 1 to prevent blocking
	}

	p.Tasks[id] = task
	return nil
}

// RemoveTask removes a task from the plan
func (p *Plan) RemoveTask(id string) error {
	task, exists := p.Tasks[id]
	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	if task.Status == StatusRunning {
		task.SignalChan <- SignalStop
		task.Cancel()
	}
	close(task.SignalChan)
	delete(p.Tasks, id)
	return nil
}

// Start begins execution of the plan
func (p *Plan) Start() error {
	if p.Status == StatusRunning {
		return fmt.Errorf("plan is already running")
	}

	p.Status = StatusRunning

	// Find max order
	maxOrder := -1
	for _, task := range p.Tasks {
		if task.Order > maxOrder {
			maxOrder = task.Order
		}
	}

	// Execute tasks in order
	for order := 0; order <= maxOrder; order++ {
		var tasksAtOrder []*Task
		for _, task := range p.Tasks {
			if task.Order == order {
				tasksAtOrder = append(tasksAtOrder, task)
			}
		}

		if len(tasksAtOrder) == 0 {
			continue
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(tasksAtOrder))

		for _, task := range tasksAtOrder {
			wg.Add(1)
			t := task // Create new variable for goroutine
			go func() {
				defer wg.Done()
				t.Status = StatusRunning
				if err := t.Fn(t.Context, p.SharedSpace, t.SignalChan); err != nil {
					t.Status = StatusFailed
					t.Data = err
					errChan <- err
				} else {
					t.Status = StatusCompleted
				}
			}()
		}

		// Wait for all tasks at this order to complete
		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			if err != nil {
				p.Status = StatusFailed
				return err
			}
		}

		// Check if context was cancelled
		if p.Context.Err() != nil {
			p.Status = StatusFailed
			return p.Context.Err()
		}
	}

	p.Status = StatusCompleted
	return nil
}

// Pause temporarily halts execution of the plan
func (p *Plan) Pause() error {
	if p.Status != StatusRunning {
		return fmt.Errorf("plan is not running")
	}

	// Send pause signal to all running tasks and wait for them to pause
	var wg sync.WaitGroup
	for _, task := range p.Tasks {
		if task.Status == StatusRunning {
			wg.Add(1)
			go func(t *Task) {
				defer wg.Done()
				t.SignalChan <- SignalPause
				// Wait for task to update its status
				for t.Status == StatusRunning {
					time.Sleep(p.Config.StatusCheckInterval)
				}
			}(task)
		}
	}
	wg.Wait()

	p.Status = StatusPaused
	return nil
}

// Resume continues execution of a paused plan
func (p *Plan) Resume() error {
	if p.Status != StatusPaused {
		return fmt.Errorf("plan is not paused")
	}

	// Send resume signal to all paused tasks and wait for them to resume
	var wg sync.WaitGroup
	for _, task := range p.Tasks {
		if task.Status == StatusPaused {
			wg.Add(1)
			go func(t *Task) {
				defer wg.Done()
				t.SignalChan <- SignalResume
				// Wait for task to update its status
				for t.Status == StatusPaused {
					time.Sleep(p.Config.StatusCheckInterval)
				}
			}(task)
		}
	}
	wg.Wait()

	p.Status = StatusRunning
	return nil
}

// Stop terminates execution of the plan
func (p *Plan) Stop() error {
	// Send stop signal to all tasks and wait for them to stop
	var wg sync.WaitGroup
	for _, task := range p.Tasks {
		if task.Status == StatusRunning || task.Status == StatusPaused {
			wg.Add(1)
			go func(t *Task) {
				defer wg.Done()
				t.SignalChan <- SignalStop
				// Wait for task to update its status
				for t.Status != StatusCompleted && t.Status != StatusFailed {
					time.Sleep(p.Config.StatusCheckInterval)
				}
			}(task)
		}
	}
	wg.Wait()

	p.Cancel() // This will trigger context cancellation for all tasks
	p.Status = StatusDestroyed

	// Clean up signal channels
	for _, task := range p.Tasks {
		close(task.SignalChan)
	}

	return nil
}

// Release releases the plan
func (p *Plan) Release() {
	p.SharedSpace.Clear()
	p.SharedSpace = nil
	p.Tasks = nil
	p = nil
}

// Trigger triggers an event on the plan
func (p *Plan) Trigger(event string, data interface{}) {
	p.SharedSpace.Set(event, data)
}

// GetStatus returns the current status of the plan and its tasks
func (p *Plan) GetStatus() (Status, map[string]Status) {
	taskStatuses := make(map[string]Status)
	for id, task := range p.Tasks {
		taskStatuses[id] = task.Status
	}
	return p.Status, taskStatuses
}

// GetTaskStatus returns the status of a specific task
func (p *Plan) GetTaskStatus(taskID string) (Status, error) {
	task, exists := p.Tasks[taskID]
	if !exists {
		return StatusUnknown, fmt.Errorf("task with ID %s not found", taskID)
	}
	return task.Status, nil
}

// GetTaskData returns the data associated with a specific task
func (p *Plan) GetTaskData(taskID string) (interface{}, error) {
	task, exists := p.Tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", taskID)
	}
	return task.Data, nil
}
