package task

import (
	"context"
	"sync"
	"time"
)

const (
	// WAITING the job is waiting
	WAITING = iota + 1

	// RUNNING the job is running
	RUNNING

	// SUCCESS the job is success
	SUCCESS

	// FAILURE the job is failure
	FAILURE
)

var status = map[int]string{
	WAITING: "WAITING",
	RUNNING: "RUNNING",
	SUCCESS: "SUCCESS",
	FAILURE: "SUCCESS",
}

// Task the task struct
type Task struct {
	name     string
	timeout  int
	handlers *Handlers
	pool     *Pool
	jobs     map[int]*Job
	mutex    sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	Option   Option
}

// Option the task option
type Option struct {
	Name           string
	JobQueueLength int
	WorkerNums     int
	AttemptAfter   int
	Attempts       int
	Timeout        int
}

// Pool the worker pool
type Pool struct {
	size      int
	max       int
	jobque    chan *Job
	workerque chan *Worker
}

// Worker the work struct
type Worker struct {
	job chan *Job
}

// Job the job
type Job struct {
	id       int
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	curr     int
	total    int
	status   int
	message  string
	response interface{}
	args     []interface{}
}

// Handlers the event handlers
type Handlers struct {
	Exec     func(int, ...interface{}) (interface{}, error)
	Progress func(int, int, int, string)
	NextID   func() (int, error)
	Add      func(int)
	Success  func(int, interface{})
	Error    func(int, error)
}
