package gou

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/utils"
)

func TestLoadTask(t *testing.T) {
	taskLoad(t)
	assert.Equal(t, 1, len(task.Tasks))
	assert.NotPanics(t, func() { SelectTask("mail") })
}

func TestTaskProcess(t *testing.T) {
	mail := taskLoad(t)
	go mail.Start()
	defer mail.Stop()
	id, err := NewProcess("tasks.mail.Add", "max@iqka.com").Exec()
	if err != nil {
		t.Fatal(err)
	}

	id2, err := NewProcess("tasks.mail.Add", "max@iqka.com", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// time.Sleep(200 * time.Millisecond)
	s1, err := NewProcess("tasks.mail.Get", id).Exec()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	time.Sleep(50 * time.Millisecond)
	s2, err := NewProcess("tasks.mail.Get", id).Exec()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// time.Sleep(500 * time.Millisecond)

	utils.Dump(s1, s2, id, id2)
	assert.Equal(t, 1025, id)
	assert.Equal(t, 1026, id2)
	// assert.Equal(t, "RUNNING", s1.(map[string]interface{})["status"])
	// assert.Equal(t, "RUNNING", s2.(map[string]interface{})["status"])
	// assert.Equal(t, 3, s2.(map[string]interface{})["total"])
	// assert.Equal(t, "unit-test", s2.(map[string]interface{})["message"])

	// waitting
	time.Sleep(3000 * time.Millisecond)
}

func taskLoad(t *testing.T) *task.Task {

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
	return SelectTask("mail")
}
