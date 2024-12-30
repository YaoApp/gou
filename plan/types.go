package plan

import (
	"context"
	"time"
)

// Status represents the status of a plan or task
type Status int

// Status constants for tracking plan and task states
const (
	StatusCreated Status = iota
	StatusRunning
	StatusPaused
	StatusCompleted
	StatusFailed
	StatusDestroyed
)

// Signal represents control signals for tasks
type Signal int

// Signal constants for task control
const (
	SignalPause Signal = iota
	SignalResume
	SignalStop
)

// Config represents plan configuration
type Config struct {
	// StatusCheckInterval is the interval between status checks
	StatusCheckInterval time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		StatusCheckInterval: 10 * time.Millisecond,
	}
}

// Option represents a function that modifies the plan configuration
type Option func(*Config)

// WithStatusCheckInterval sets the status check interval
func WithStatusCheckInterval(d time.Duration) Option {
	return func(c *Config) {
		c.StatusCheckInterval = d
	}
}

// Task represents a unit of work in a plan
type Task struct {
	ID         string
	Order      int
	Fn         TaskFunc
	Status     Status
	Data       interface{}
	Context    context.Context
	Cancel     context.CancelFunc
	SignalChan chan Signal
}

// TaskFunc is the function type that represents a task's execution unit
type TaskFunc func(ctx context.Context, shared SharedSpace, signals <-chan Signal) error

// Plan represents a collection of tasks with shared space
type Plan struct {
	ID          string
	Tasks       map[string]*Task
	SharedSpace SharedSpace
	Status      Status
	Context     context.Context
	Cancel      context.CancelFunc
	Config      *Config
}

// SharedSpace represents the interface for shared storage space
type SharedSpace interface {
	// Set stores a value in the shared space
	Set(key string, value interface{}) error

	// Get retrieves a value from the shared space
	Get(key string) (interface{}, error)

	// Delete removes a value from the shared space
	Delete(key string) error

	// Clear removes all values from the shared space
	Clear() error

	// Subscribe subscribes to changes in the shared space
	Subscribe(key string, callback func(key string, value interface{})) error

	// Unsubscribe removes a subscription
	Unsubscribe(key string) error
}
