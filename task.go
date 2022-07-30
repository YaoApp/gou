package gou

import (
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// TaskHandlers task process handlers
var TaskHandlers = map[string]ProcessHandler{
	"add":      processTaskAdd,
	"progress": processTaskProgress,
	"get":      processTaskGet,
}

// TaskOption the task option
type TaskOption struct {
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

// LoadTask load task
func LoadTask(source string, name string) (*task.Task, error) {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	config, err := helper.ReadFile(input)
	if err != nil {
		return nil, err
	}

	o := TaskOption{}
	err = jsoniter.Unmarshal(config, &o)
	if err != nil {
		return nil, err
	}

	option := task.Option{
		Name:           name,
		Timeout:        EnvInt(o.Timeout),
		WorkerNums:     EnvInt(o.WorkerNums),
		JobQueueLength: EnvInt(o.Size),
		AttemptAfter:   EnvInt(o.AttemptAfter),
		Attempts:       EnvInt(o.Attempts),
	}

	handlers := taskEventHandlers(name, o)
	t := task.New(handlers, option)
	task.Tasks[name] = t

	return t, nil
}

// SelectTask select task by name
func SelectTask(name string) *task.Task {
	t, has := task.Tasks[name]
	if !has {
		exception.New("Task:%s does not load", 500, name).Throw()
	}
	return t
}

// processTaskAdd
func processTaskAdd(process *Process) interface{} {
	t := SelectTask(process.Class)
	args := []interface{}{}
	if process.NumOfArgs() > 0 {
		args = process.Args
	}
	v, err := t.Add(args...)
	if err != nil {
		exception.New("Task %s Add: %s", 500, process.Class, err).Throw()
	}
	return v
}

// processTaskProgress
func processTaskProgress(process *Process) interface{} {
	process.ValidateArgNums(4)
	id := process.ArgsInt(0)
	curr := process.ArgsInt(1)
	total := process.ArgsInt(2)
	message := process.ArgsString(3)
	err := task.Progress(process.Class, id, curr, total, message)
	if err != nil {
		exception.New("Task %s Progress: %s", 500, process.Class, err).Throw()
	}
	return nil
}

// processTaskGet
func processTaskGet(process *Process) interface{} {
	id := process.ArgsInt(0)
	t := SelectTask(process.Class)

	job, err := t.Get(id)
	if err != nil {
		exception.New("Task %s Progress: %s", 500, process.Class, err).Throw()
	}

	return job
}

func taskEventHandlers(name string, o TaskOption) *task.Handlers {
	handlers := &task.Handlers{
		Exec: func(id int, args ...interface{}) (interface{}, error) {
			args = append([]interface{}{id}, args...)
			return NewProcess(o.Process, args...).Exec()
		},
	}

	if o.Event.Next != "" {
		handlers.NextID = func() (int, error) {
			id, err := NewProcess(o.Event.Next).Exec()
			if err != nil {
				return 0, err
			}
			return any.Of(id).CInt(), nil
		}
	}

	if o.Event.Add != "" {
		handlers.Add = func(id int) {
			_, err := NewProcess(o.Event.Add, id).Exec()
			if err != nil {
				log.Error("[Task] %s event.add %s", name, err.Error())
			}
		}
	}

	if o.Event.Success != "" {
		handlers.Success = func(id int, res interface{}) {
			_, err := NewProcess(o.Event.Success, id, res).Exec()
			if err != nil {
				log.Error("[Task] %s event.success %s", name, err.Error())
			}
		}
	}

	if o.Event.Error != "" {
		handlers.Error = func(id int, res error) {
			_, err := NewProcess(o.Event.Error, id, res.Error()).Exec()
			if err != nil {
				log.Error("[Task] %s event.error %s", name, err.Error())
			}
		}
	}

	if o.Event.Progress != "" {
		handlers.Progress = func(id, curr, total int, message string) {
			_, err := NewProcess(o.Event.Progress, id, curr, total, message).Exec()
			if err != nil {
				log.Error("[Task] %s event.progress %s", name, err.Error())
			}
		}
	}

	return handlers
}
