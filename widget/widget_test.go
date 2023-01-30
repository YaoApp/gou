package widget

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
)

func TestLoad(t *testing.T) {
	w := load(t)
	v, err := w.ScriptExec("helper", "Foo", "Hello")
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, "dyform", w.Name)
	assert.Equal(t, "Dynamic Form", w.Label)
	assert.Equal(t, "A form widget. users can design forms online", w.Description)
	assert.Equal(t, "0.1.0", w.Version)
	assert.Equal(t, "Hello World", v)
}

func TestInstanceLoad(t *testing.T) {
	w := load(t)
	err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
}

func load(t *testing.T) *Widget {

	root := os.Getenv("GOU_TEST_APPLICATION")

	// Load app
	app, err := application.OpenFromDisk(root)
	if err != nil {
		t.Fatal(err)
	}
	application.Load(app)
	v8.Start(&v8.Option{})

	path := filepath.Join("widgets", "dyform")
	widget, err := Load(path, processRegister(), moduleRegister())
	if err != nil {
		t.Fatal(err)
	}
	return widget
}

func moduleRegister() ModuleRegister {
	return ModuleRegister{
		"Models": func(name string, source []byte) error {
			fmt.Printf("Model %s Registered\n", name)
			return nil
		},
		"Flows": func(name string, source []byte) error {
			fmt.Printf("Flow %s Registered\n", name)
			return nil
		},
		"Apis": func(name string, source []byte) error {
			fmt.Printf("API %s Registered\n", name)
			return nil
		},
	}
}

func processRegister() ProcessRegister {
	return func(widget, name string, p func(args ...interface{}) interface{}) error {
		fmt.Printf("PROCESS: widgets.%s.%s Registered\n", widget, name)
		processName := strings.ToLower(fmt.Sprintf("widgets.%s.%s", widget, name))
		process.Register(processName, func(process *process.Process) interface{} {
			return map[string]interface{}{"instance": "pad", "payload": "foo"}
		})
		return nil
	}
}
