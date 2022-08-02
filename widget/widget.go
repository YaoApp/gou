package widget

import (
	"io/ioutil"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime"
	"github.com/yaoapp/kun/log"
)

// Widgets the registered widgets
var Widgets = map[string]*Widget{}

// Load load a widget
func Load(path string, runtime *runtime.Runtime) (*Widget, error) {
	w := &Widget{Name: filepath.Base(path), Path: path, Runtime: runtime}

	data, err := ioutil.ReadFile(filepath.Join(path, "widget.json"))
	if err != nil {
		log.Error("[Widget] open widget.json error: %s", err.Error())
		return nil, err
	}

	err = jsoniter.Unmarshal(data, w)
	if err != nil {
		log.Error("[Widget] parse widget.json error: %s", err.Error())
		return nil, err
	}

	err = w.loadScripts()
	if err != nil {
		log.Error("[Widget] load widget scirpts error: %s", err.Error())
		return nil, err
	}

	// Register the widget
	Widgets[w.Name] = w
	return w, nil
}

// Load load the widget instances
func (w *Widget) Load(name string) error {

	// Compile DSL

	// Trigger the load event

	// Load Models

	// Load Flows

	// Load Tables

	// Load Tasks

	// Load Schedules

	return nil
}

// Migrate the migrate
func (w *Widget) Migrate() error {
	return nil
}
