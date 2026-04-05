package task

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	var mu sync.Mutex
	res := map[string]map[int]interface{}{}
	task := New(
		getHandlers(res, &mu),
		Option{
			Name:           "unit-test-task",
			WorkerNums:     100,
			JobQueueLength: 1024,
			Timeout:        5,
			Attempts:       3,
			AttemptAfter:   200,
		},
	)

	tasksMu.Lock()
	Tasks["unit-test-task"] = task
	tasksMu.Unlock()
	defer task.Stop()
	go task.Start()
	for i := 0; i < 9; i++ {
		task.Add("JOB", i)
	}

	time.Sleep(3 * time.Second)
	task.Stop()
	time.Sleep(2 * time.Second)
	fmt.Println("NumGoroutine", runtime.NumGoroutine())

	mu.Lock()
	defer mu.Unlock()
	for i := 2; i <= 18; i = i + 2 {
		assert.Equal(t, fmt.Sprintln("Add: ", i), res["add"][i])
	}

	for i := 2; i <= 14; i = i + 2 {
		assert.Contains(t, res["success"][i], fmt.Sprintf("#%d", i))
	}
	for i := 16; i <= 18; i = i + 2 {
		assert.Contains(t, res["error"][i], fmt.Sprintf("#%d", i))
	}

}

func TestGet(t *testing.T) {
	task := New(
		&Handlers{
			Exec: func(id int, args ...interface{}) (interface{}, error) {
				for i := 1; i < 3; i++ {
					time.Sleep(500 * time.Millisecond)
					Progress("unit-test-task", id, i, 2, fmt.Sprintf("Progress %v/%v", i, 2))
				}
				return nil, nil
			},
		},
		Option{
			Name:           "unit-test-task",
			WorkerNums:     100,
			JobQueueLength: 1024,
			Timeout:        300,
			Attempts:       3,
			AttemptAfter:   200,
		},
	)

	tasksMu.Lock()
	Tasks["unit-test-task"] = task
	tasksMu.Unlock()
	defer task.Stop()
	go task.Start()
	id, err := task.Add()
	if err != nil {
		t.Fatal(err)
	}

	job, err := task.Get(id)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	assert.Equal(t, "WAITING", job["status"])
	assert.Equal(t, 1, job["id"])
	time.Sleep(600 * time.Millisecond)
	job, err = task.Get(id)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	assert.Equal(t, 1, job["current"])
	assert.Equal(t, 2, job["total"])
	assert.Equal(t, "Progress 1/2", job["message"])
	assert.Equal(t, "RUNNING", job["status"])
	assert.Equal(t, 1, job["id"])

	time.Sleep(600 * time.Millisecond)
	_, err = task.Get(id)
	assert.NotNil(t, err)

}

func getHandlers(res map[string]map[int]interface{}, mu *sync.Mutex) *Handlers {
	var idseq = 0
	var idMu sync.Mutex
	return &Handlers{

		NextID: func() (int, error) {
			idMu.Lock()
			idseq = idseq + 2
			id := idseq
			idMu.Unlock()
			return id, nil
		},

		Add: func(id int) {
			mu.Lock()
			defer mu.Unlock()
			if _, has := res["add"]; !has {
				res["add"] = map[int]interface{}{}
			}
			res["add"][id] = fmt.Sprintln("Add: ", id)
		},

		Exec: func(id int, args ...interface{}) (interface{}, error) {
			for i := 1; i < id; i++ {
				time.Sleep(200 * time.Millisecond)
				Progress("unit-test-task", id, i, id, fmt.Sprintf("Progress %v/%v", i, id))
			}
			mu.Lock()
			if _, has := res["exec"]; !has {
				res["exec"] = map[int]interface{}{}
			}
			res["exec"][id] = fmt.Sprintf("%d %v", id, args)
			mu.Unlock()
			return fmt.Sprintf("%d %v", id, args), nil
		},

		Success: func(id int, response interface{}) {
			mu.Lock()
			defer mu.Unlock()
			if _, has := res["success"]; !has {
				res["success"] = map[int]interface{}{}
			}
			res["success"][id] = fmt.Sprintln("Success:", fmt.Sprintf("#%d", id), response)
		},

		Error: func(id int, err error) {
			mu.Lock()
			defer mu.Unlock()
			if _, has := res["error"]; !has {
				res["error"] = map[int]interface{}{}
			}
			res["error"][id] = fmt.Sprintln("Error:", fmt.Sprintf("#%d", id), err)
		},

		Progress: func(id, curr, total int, message string) {
			fmt.Println("Progress:", fmt.Sprintf("#%d", id), curr, total, message)
		},
	}
}
