package task

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
	"github.com/yaoapp/kun/utils"
)

func TestLoadTask(t *testing.T) {
	taskLoad(t)
	assert.Equal(t, 1, len(Tasks))
	assert.NotPanics(t, func() { Select("mail") })
}

func TestTaskProcess(t *testing.T) {
	mail := taskLoad(t)
	go mail.Start()
	defer mail.Stop()
	id, err := process.New("tasks.mail.Add", "max@iqka.com").Exec()
	if err != nil {
		t.Fatal(err)
	}

	id2, err := process.New("tasks.mail.Add", "max@iqka.com", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// time.Sleep(200 * time.Millisecond)
	s1, err := process.New("tasks.mail.Get", id).Exec()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	time.Sleep(100 * time.Millisecond)
	s2, err := process.New("tasks.mail.Get", id).Exec()
	if err != nil {
		t.Fatal(err)
	}

	// time.Sleep(100 * time.Millisecond)

	utils.Dump(s1, s2, id, id2)
	// assert.Equal(t, 1025, id)
	// assert.Equal(t, 1026, id2)
	// assert.Equal(t, "RUNNING", s1.(map[string]interface{})["status"])
	// assert.Equal(t, "RUNNING", s2.(map[string]interface{})["status"])
	// assert.Equal(t, 3, s2.(map[string]interface{})["total"])
	// assert.Equal(t, "unit-test", s2.(map[string]interface{})["message"])

	// waitting
	time.Sleep(3000 * time.Millisecond)
}

func taskLoad(t *testing.T) *Task {
	loadApp(t)
	loadScripts(t)

	process.Register("xiang.system.Sleep", func(process *process.Process) interface{} {
		process.ValidateArgNums(1)
		ms := process.ArgsInt(0)
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil
	})

	_, err := Load(path.Join("tasks", "mail.task.yao"), "mail")
	if err != nil {
		t.Fatal(err)
	}

	return Select("mail")
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
