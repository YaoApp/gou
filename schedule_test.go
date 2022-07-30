package gou

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadSchedule(t *testing.T) {
	scheduleLoad(t)
	assert.Equal(t, 2, len(Schedules))
	assert.NotPanics(t, func() { SelectSchedule("sendmail") })
	assert.NotPanics(t, func() { SelectSchedule("mail") })
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
	res, err := NewProcess("schedules.mail.Start").Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, res.(map[string]interface{})["enabled"].(bool))

	res, err = NewProcess("schedules.mail.Stop").Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, res.(map[string]interface{})["enabled"].(bool))
}

func scheduleLoad(t *testing.T) {

	RegisterProcessHandler("xiang.system.Sleep", func(process *Process) interface{} {
		process.ValidateArgNums(1)
		ms := process.ArgsInt(0)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil
	})

	root := os.Getenv("GOU_TEST_APP_ROOT")
	_, err := LoadTask("file://"+path.Join(root, "tasks", "mail.task.json"), "mail")
	if err != nil {
		t.Fatal(err)
	}

	err = Yao.Load(path.Join(root, "scripts", "mail.js"), "mail")
	if err != nil {
		fmt.Println(err)
		t.Fatal(err)
	}

	_, err = LoadSchedule("file://"+path.Join(root, "schedules", "sendmail.sch.json"), "sendmail")
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadSchedule("file://"+path.Join(root, "schedules", "mail.sch.json"), "mail")
	if err != nil {
		t.Fatal(err)
	}

}
