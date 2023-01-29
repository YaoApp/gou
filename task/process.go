package task

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// TaskHandlers task process handlers
var TaskHandlers = map[string]process.Handler{
	"add":      processTaskAdd,
	"progress": processTaskProgress,
	"get":      processTaskGet,
}

func init() {
	process.RegisterGroup("tasks", TaskHandlers)
}

// ProcessOption the task process option
type ProcessOption struct {
	Name         string      `json:"name"`
	Process      string      `json:"process"`
	Size         interface{} `json:"size,omitempty"`
	WorkerNums   interface{} `json:"worker_nums,omitempty"`
	AttemptAfter interface{} `json:"attempt_after,omitempty"`
	Attempts     interface{} `json:"attempts,omitempty"`
	Timeout      interface{} `json:"timeout,omitempty"`
	Event        struct {
		Next     string `json:"next,omitempty"`
		Add      string `json:"add,omitempty"`
		Success  string `json:"success,omitempty"`
		Error    string `json:"error,omitempty"`
		Progress string `json:"progress,omitempty"`
	} `json:"event"`
}

// Load load task
func Load(file string, name string) (*Task, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	o := ProcessOption{}
	err = application.Parse(file, data, &o)
	if err != nil {
		return nil, err
	}

	option := Option{
		Name:           name,
		Timeout:        helper.EnvInt(o.Timeout),
		WorkerNums:     helper.EnvInt(o.WorkerNums),
		JobQueueLength: helper.EnvInt(o.Size),
		AttemptAfter:   helper.EnvInt(o.AttemptAfter),
		Attempts:       helper.EnvInt(o.Attempts),
	}

	handlers := taskEventHandlers(name, o)
	t := New(handlers, option)
	Tasks[name] = t

	return t, nil
}

// Select select task by name
func Select(name string) *Task {
	t, has := Tasks[name]
	if !has {
		exception.New("Task:%s does not load", 500, name).Throw()
	}
	return t
}

// processTaskAdd
func processTaskAdd(process *process.Process) interface{} {
	t := Select(process.ID)
	args := []interface{}{}
	if process.NumOfArgs() > 0 {
		args = process.Args
	}
	v, err := t.Add(args...)
	if err != nil {
		exception.New("Task %s Add: %s", 500, process.ID, err).Throw()
	}
	return v
}

// processTaskProgress
func processTaskProgress(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	id := process.ArgsInt(0)
	curr := process.ArgsInt(1)
	total := process.ArgsInt(2)
	message := process.ArgsString(3)
	err := Progress(process.ID, id, curr, total, message)
	if err != nil {
		exception.New("Task %s Progress: %s", 500, process.ID, err).Throw()
	}
	return nil
}

// processTaskGet
func processTaskGet(process *process.Process) interface{} {
	id := process.ArgsInt(0)
	t := Select(process.ID)

	job, err := t.Get(id)
	if err != nil {
		exception.New("Task %s Progress: %s", 500, process.ID, err).Throw()
	}

	return job
}

func taskEventHandlers(name string, o ProcessOption) *Handlers {
	handlers := &Handlers{
		Exec: func(id int, args ...interface{}) (interface{}, error) {
			args = append([]interface{}{id}, args...)
			return process.New(o.Process, args...).Exec()
		},
	}

	if o.Event.Next != "" {
		handlers.NextID = func() (int, error) {
			id, err := process.New(o.Event.Next).Exec()
			if err != nil {
				return 0, err
			}
			return any.Of(id).CInt(), nil
		}
	}

	if o.Event.Add != "" {
		handlers.Add = func(id int) {
			_, err := process.New(o.Event.Add, id).Exec()
			if err != nil {
				log.Error("[Task] %s event.add %s", name, err.Error())
			}
		}
	}

	if o.Event.Success != "" {
		handlers.Success = func(id int, res interface{}) {
			_, err := process.New(o.Event.Success, id, res).Exec()
			if err != nil {
				log.Error("[Task] %s event.success %s", name, err.Error())
			}
		}
	}

	if o.Event.Error != "" {
		handlers.Error = func(id int, res error) {
			_, err := process.New(o.Event.Error, id, res.Error()).Exec()
			if err != nil {
				log.Error("[Task] %s event.error %s", name, err.Error())
			}
		}
	}

	if o.Event.Progress != "" {
		handlers.Progress = func(id, curr, total int, message string) {
			_, err := process.New(o.Event.Progress, id, curr, total, message).Exec()
			if err != nil {
				log.Error("[Task] %s event.progress %s", name, err.Error())
			}
		}
	}

	return handlers
}
