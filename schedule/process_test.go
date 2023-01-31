package schedule

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/task"
)

func TestLoadSchedule(t *testing.T) {
	scheduleLoad(t)
	assert.Equal(t, 2, len(Schedules))
	assert.NotPanics(t, func() { Select("sendmail") })
	assert.NotPanics(t, func() { Select("mail") })
}

func TestScheduleUseTask(t *testing.T) {
	// scheduleLoad(t)
	// go SelectTask("mail").Start()
	// defer SelectTask("mail").Stop()

	// sch := SelectSchedule("sendmail")
	// sch.Start()
	// fmt.Println("Start", sch.Enabled)
	// time.Sleep(80 * time.Second)
	// sch.Stop()
	// fmt.Println("Stop", sch.Enabled)
	// time.Sleep(80 * time.Second)
}

func TestScheduleUseProcess(t *testing.T) {
	// scheduleLoad(t)
	// sch := SelectSchedule("mail")
	// sch.Start()
	// fmt.Println("Start", sch.Enabled)
	// time.Sleep(80 * time.Second)
	// sch.Stop()
	// fmt.Println("Stop", sch.Enabled)
	// time.Sleep(80 * time.Second)
}

func TestScheduleProcesses(t *testing.T) {
	scheduleLoad(t)
	res, err := process.New("schedules.mail.Start").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, res.(map[string]interface{})["enabled"].(bool))

	res, err = process.New("schedules.mail.Stop").Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, res.(map[string]interface{})["enabled"].(bool))
}

func scheduleLoad(t *testing.T) {

	taskLoad(t)

	_, err := Load(path.Join("schedules", "sendmail.sch.yao"), "sendmail")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(path.Join("schedules", "mail.sch.yao"), "mail")
	if err != nil {
		t.Fatal(err)
	}

}

func taskLoad(t *testing.T) {
	loadApp(t)
	loadScripts(t)

	process.Register("xiang.system.Sleep", func(process *process.Process) interface{} {
		process.ValidateArgNums(1)
		ms := process.ArgsInt(0)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil
	})

	_, err := task.Load(path.Join("tasks", "mail.task.yao"), "mail")
	if err != nil {
		t.Fatal(err)
	}

}

func loadApp(t *testing.T) {

	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := application.OpenFromDisk(root) // Load app
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)

}

func loadScripts(t *testing.T) {

	scripts := map[string]string{
		"tests.task.mail": filepath.Join("scripts", "tests", "task", "mail.js"),
	}

	for id, file := range scripts {
		_, err := v8.Load(file, id)
		if err != nil {
			t.Fatal(err)
		}
	}

	err := v8.Start(&v8.Option{})
	if err != nil {
		t.Fatal(err)
	}

}
