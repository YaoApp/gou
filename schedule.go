package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/robfig/cron/v3"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// Schedules the registered schedules
var Schedules = map[string]*Schedule{}

// ScheduleHandlers chedule process handlers
var ScheduleHandlers = map[string]ProcessHandler{
	"start": processScheduleStart,
	"stop":  processScheduleStop,
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

// LoadSchedule load schedule
func LoadSchedule(source string, name string) (*Schedule, error) {
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

	sch := &Schedule{name: name}
	err = jsoniter.Unmarshal(config, sch)
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

// SelectSchedule select schedule by name
func SelectSchedule(name string) *Schedule {
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
				args = append(args, EnvString(v))
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
			p, err := ProcessOf(sch.Process, sch.Args...)
			if err != nil {
				log.Error("[Schedule] %s %s %s", sch.name, sch.Process, err)
			}

			_, err = p.Exec()
			if err != nil {
				log.Error("[Schedule] %s %s %s", sch.name, sch.Process, err)
			}
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
func processScheduleStart(process *Process) interface{} {
	sch := SelectSchedule(process.Class)
	sch.Start()
	return map[string]interface{}{"enabled": sch.Enabled}
}

// processScheduleStop
func processScheduleStop(process *Process) interface{} {
	sch := SelectSchedule(process.Class)
	sch.Stop()
	return map[string]interface{}{"enabled": sch.Enabled}
}
