package task

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/yaoapp/kun/log"
)

// Tasks the registered tasks
var Tasks = map[string]*Task{}

// New create new task
func New(handlers *Handlers, option Option) *Task {

	if option.WorkerNums == 0 {
		option.WorkerNums = 1
	}

	if option.AttemptAfter == 0 {
		option.AttemptAfter = 200
	}

	if option.Timeout == 0 {
		option.Timeout = 300
	}

	if option.JobQueueLength == 0 {
		option.JobQueueLength = 1024
	}

	pool := &Pool{
		size:      option.WorkerNums,
		max:       option.JobQueueLength,
		jobque:    make(chan *Job, option.JobQueueLength),
		workerque: make(chan *Worker, option.WorkerNums),
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Task{
		name:     option.Name,
		handlers: handlers,
		jobs:     map[int]*Job{},
		mutex:    sync.Mutex{},
		ctx:      ctx,
		cancel:   cancel,
		pool:     pool,
		timeout:  option.Timeout,
		Option:   option,
	}
}

// Start start the worker pool
func (t *Task) Start() {
	defer t.cancel()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	for i := 0; i < t.pool.size; i++ {
		w := createWorker()
		t.startWorker(w)
	}

	for {
		select {
		case job := <-t.pool.jobque:
			worker := <-t.pool.workerque
			worker.job <- job
		case <-interrupt:
			return
		case <-t.ctx.Done():
			return
		}
	}
}

// Stop the task
func (t *Task) Stop() {
	t.cancel()
}

// Add a job to the job queue
func (t *Task) Add(args ...interface{}) (int, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	id := t.nextID()
	if len(t.pool.jobque) > t.pool.max {
		return 0, fmt.Errorf("[TASK] %s reached the limit of jobs queue", t.name)
	}

	timeout := time.Duration(t.timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	job := &Job{
		id:      id,
		args:    args,
		timeout: timeout,
		ctx:     ctx,
		cancel:  cancel,
	}

	t.jobs[id] = job
	t.add(job)
	t.pool.jobque <- job
	return id, nil
}

// Progress set the progress of the job
func Progress(name string, id, curr, total int, message string) error {

	t, has := Tasks[name]
	if !has {
		return fmt.Errorf("task %s does not exist", name)
	}

	job, has := t.jobs[id]
	if !has {
		return fmt.Errorf("job %d does not exist or was completed", id)
	}

	job.curr = curr
	job.total = total
	job.message = message
	t.progress(job, curr, total, message)
	return nil
}

// Get get job by job id
func (t *Task) Get(id int) (map[string]interface{}, error) {
	job, has := t.jobs[id]
	if !has {
		return nil, fmt.Errorf("job %d does not exist or was completed", id)
	}
	return map[string]interface{}{
		"id":       job.id,
		"status":   status[job.status],
		"current":  job.curr,
		"total":    job.total,
		"message":  job.message,
		"response": job.response,
	}, nil
}

// createWorker create a new worker
func createWorker() *Worker {
	return &Worker{job: make(chan *Job)}
}

// startWorker the worker
func (t *Task) startWorker(w *Worker) {
	go func() {
		for {
			t.pool.workerque <- w
			select {
			case job := <-w.job:
				t.start(job)
			case <-t.ctx.Done(): // quit worker when the task is canceled
				return
			}
		}
	}()
}

// start start running job
func (t *Task) start(job *Job) {

	defer job.cancel()
	defer delete(t.jobs, job.id)

	ch := make(chan interface{}, 1) // the result channel
	chError := make(chan error, 1)  // the error channel

	go func() {
		defer func() {
			if err := recover(); err != nil {
				chError <- fmt.Errorf("TASK: %v Job:%v %v", t.name, job.id, err)
			}
		}()
		log.Trace("[TASK] %s #%d RUNNING", t.name, job.id)
		res, err := t.exec(job)
		if err != nil {
			chError <- err
			return
		}

		ch <- res
	}()

	select {
	case <-t.ctx.Done():
		log.Error("[TASK] %s Job:%v the task was canceled (%v)", t.name, job.id, t.ctx.Err())
		t.failure(job, t.ctx.Err())
		return

	case <-job.ctx.Done():
		log.Error("[TASK] %s Job:%v the job was canceled (%v)", t.name, job.id, job.ctx.Err())
		t.failure(job, job.ctx.Err())
		return

	case err := <-chError:
		log.Error("[TASK] %s Job:%v  %v", t.name, job.id, err.Error())
		t.failure(job, err)
		return

	case res := <-ch:
		log.Trace("[TASK] %s Job:%v  %v", t.name, job.id, res)
		t.success(job, res)
		return
	}
}

func (t *Task) nextID() int {
	if t.handlers.NextID == nil {
		return len(t.pool.jobque) + 1
	}

	id, err := t.handlers.NextID()
	if err != nil {
		log.Error("[TASK] %s can't get next id (%s)", t.name, err.Error())
		return len(t.pool.jobque) + 1
	}
	return id
}

// exec excute the job
// @todo:
//  1. The goroutine will be running until the handler completed, it should be killed.
//  2. Should retry if the handler is error or panic
func (t *Task) exec(job *Job) (interface{}, error) {
	job.status = RUNNING
	if t.handlers.Exec == nil {
		err := fmt.Errorf("[TASK] %s Job:%v, is not set the execute handler", t.name, job.id)
		return nil, err
	}
	return t.handlers.Exec(job.id, job.args...)
}

func (t *Task) failure(job *Job, err error) {
	job.status = FAILURE
	job.response = err.Error()
	if t.handlers.Error == nil {
		return
	}
	t.handlers.Error(job.id, err)
}

func (t *Task) success(job *Job, response interface{}) {
	job.status = SUCCESS
	job.response = response
	if t.handlers.Success == nil {
		return
	}
	t.handlers.Success(job.id, response)
}

func (t *Task) add(job *Job) {
	job.status = WAITING
	if t.handlers.Add == nil {
		return
	}
	t.handlers.Add(job.id)
}

func (t *Task) progress(job *Job, curr, total int, message string) {
	if t.handlers.Progress == nil {
		return
	}
	t.handlers.Progress(job.id, curr, total, message)
}
