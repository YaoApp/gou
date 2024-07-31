package disk

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/log"
)

func TestOpen(t *testing.T) {
	root := os.Getenv("GOU_TEST_APPLICATION")
	_, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Open("/path/not-exists")
	assert.NotNil(t, err)
}

func TestWalk(t *testing.T) {
	app := prepare(t)

	files := []string{}
	err := app.Walk("models", func(root, filename string, isdir bool) error {
		files = append(files, filepath.Join(filename))
		assert.IsType(t, true, isdir)
		assert.IsType(t, "string", filename)
		assert.Equal(t, "models", root)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(files), 1)
}

func TestGlob(t *testing.T) {
	app := prepare(t)

	matches, err := app.Glob("models/*.mod.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(matches), 1)

	matches, err = app.Glob("/models/*.mod.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(matches), 1)

	matches, err = app.Glob("/models/*.tab.yao")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, matches)
}

func TestWalkWithPatterns(t *testing.T) {
	app := prepare(t)

	files := []string{}
	err := app.Walk("scripts", func(root, filename string, isdir bool) error {
		files = append(files, filepath.Join(filename))
		assert.IsType(t, true, isdir)
		assert.IsType(t, "string", filename)
		assert.Equal(t, "scripts", root)
		if !isdir {
			ext := filepath.Ext(filename)
			assert.True(t, ext == ".ts" || ext == ".js")
		}
		return nil
	}, "*.js", "*.ts")

	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(files), 1)
}

func TestRead(t *testing.T) {
	app := prepare(t)
	data, err := app.Read(filepath.Join("models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)

	data, err = app.Read(filepath.Join("/", "models", "user.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(data), 1)

	_, err = app.Read(filepath.Join("/", "models", "user.mod.yao-not-exists"))
	assert.NotNil(t, err)
}

func TestWrite(t *testing.T) {
	app := prepare(t)
	data := []byte(`{"name":"test"}`)
	err := app.Write(filepath.Join("models", "temp.mod.yao"), data)
	if err != nil {
		t.Fatal(err)
	}

	exists, _ := app.Exists(filepath.Join("models", "temp.mod.yao"))
	assert.True(t, exists)

	err = app.Remove(filepath.Join("models", "temp.mod.yao"))
	if err != nil {
		t.Fatal(err)
	}

	exists, _ = app.Exists(filepath.Join("models", "temp.mod.yao"))
	assert.False(t, exists)
}

func TestWatch(t *testing.T) {
	app := prepare(t)
	interrupt := make(chan uint8, 1)
	done := make(chan bool, 1)

	// recive interrupt signal
	onInterrupt := make(chan os.Signal, 1)
	signal.Notify(onInterrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// trace := sync.Map{}
	go func() {
		err := app.Watch(func(event string, name string) {
			if event == "CHMOD" {
				return
			}
			log.Info("[Watch] UnitTests %s %s", event, name)
			// trace.Store(event, name)
		}, interrupt)
		if err != nil {
			done <- true
			return
		}

		done <- true
	}()

	// CHECK
	exists, _ := app.Exists(filepath.Join("models", "tmp", "temp.mod.yao"))
	log.Info("CHECK: %v", exists)

	time.Sleep(2 * time.Second)

	// CREATE
	err := app.Write(filepath.Join("models", "tmp", "temp.mod.yao"), []byte("{}"))
	if err != nil {
		interrupt <- 0
		t.Fatal(err)
	}

	data, _ := app.Read(filepath.Join("models", "tmp", "temp.mod.yao"))
	log.Info("DATA AFTER CREATE: %s", data)

	// CHANGE
	app.Write(filepath.Join("models", "tmp", "temp.mod.yao"), []byte(`{"foo":"bar"}`))
	if err != nil {
		interrupt <- 0
		t.Fatal(err)
	}

	data, _ = app.Read(filepath.Join("models", "tmp", "temp.mod.yao"))
	log.Info("DATA AFTER WRITE: %s", data)

	// REMOVE
	app.Remove(filepath.Join("models", "tmp", "temp.mod.yao"))
	if err != nil {
		interrupt <- 0
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	// CREATE, _ := trace.Load("CREATE")
	// WRITE, _ := trace.Load("WRITE")
	// REMOVE, _ := trace.Load("REMOVE")

	// assert.Equal(t, "/models/tmp/temp.mod.yao", CREATE)
	// assert.Equal(t, "/models/tmp/temp.mod.yao", WRITE)
	// assert.Equal(t, "/models/tmp/temp.mod.yao", REMOVE)

	interrupt <- 0

	for {
		select {
		case <-done:
			return

		case <-onInterrupt:
			interrupt <- 0
			break
		}
	}
}

func prepare(t *testing.T) *Disk {
	root := os.Getenv("GOU_TEST_APPLICATION")
	app, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return app
}
