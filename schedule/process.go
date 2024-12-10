package schedule

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// Schedules the registered schedules
var Schedules = map[string]*Schedule{}

// ScheduleHandlers chedule process handlers
var ScheduleHandlers = map[string]process.Handler{
	"start": processScheduleStart,
	"stop":  processScheduleStop,
}

func init() {
	process.RegisterGroup("schedules", ScheduleHandlers)
}

// Schedule the schedule struct
type Schedule struct {
	name     string
	Name     string        `json:"name"`
	Process  string        `json:"process,omitempty"`
	Schedule string        `json:"schedule"`
	TaskName string        `json:"task,omitempty"`
	Args     []interface{} `json:"args,omitempty"`
	id       cron.EntryID
	Enabled  bool
	cron     *cron.Cron
}

// Load load schedule
func Load(file string, name string) (*Schedule, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	sch := &Schedule{name: name}
	err = application.Parse(file, data, sch)
	if err != nil {
		return nil, err
	}

	handler, err := sch.handler()
	if err != nil {
		return nil, err
	}

	c := cron.New()
	id, err := c.AddFunc(sch.Schedule, handler)

	if err != nil {
		return nil, err
	}

	sch.cron = c
	sch.id = id
	Schedules[name] = sch
	return sch, nil
}

// Select select schedule by name
func Select(name string) *Schedule {
	sch, has := Schedules[name]
	if !has {
		exception.New("Schedule:%s does not load", 500, name).Throw()
	}
	return sch
}

// parse Args
func (sch *Schedule) parseArgs() {
	args := []interface{}{}
	if sch.Args != nil {
		for _, arg := range sch.Args {
			if v, ok := arg.(string); ok {
				args = append(args, helper.EnvString(v))
				continue
			}
			args = append(args, arg)
		}
	}
	sch.Args = args
}

// handler task or process
func (sch *Schedule) handler() (func(), error) {
	sch.parseArgs()

	if sch.TaskName != "" {
		_, has := task.Tasks[sch.TaskName]
		if !has {
			return nil, fmt.Errorf("%s was not loaded", sch.TaskName)
		}
		return func() { task.Tasks[sch.TaskName].Add(sch.Args...) }, nil
	} else if sch.Process != "" {
		return func() {
			p, err := process.Of(sch.Process, sch.Args...)
			if err != nil {
				log.Error("[Schedule] %s %s %s", sch.name, sch.Process, err)
			}

			err = p.Execute()
			if err != nil {
				log.Error("[Schedule] %s %s %s", sch.name, sch.Process, err)
			}
			defer p.Release()
		}, nil
	}

	return nil, fmt.Errorf("process or task is required")
}

// Start start the schedule
func (sch *Schedule) Start() {
	sch.Enabled = true
	sch.cron.Start()
}

// Stop start the schedule
func (sch *Schedule) Stop() {
	sch.Enabled = false
	sch.cron.Stop()
}

// processScheduleStart
func processScheduleStart(process *process.Process) interface{} {
	sch := Select(process.ID)
	sch.Start()
	return map[string]interface{}{"enabled": sch.Enabled}
}

// processScheduleStop
func processScheduleStop(process *process.Process) interface{} {
	sch := Select(process.ID)
	sch.Stop()
	return map[string]interface{}{"enabled": sch.Enabled}
}
