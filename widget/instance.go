package widget

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// Load load the widget instances
func (w *Widget) Load() error {

	// Load from the yao app path
	root, err := w.InstanceRoot()
	if err == nil {
		err := Walk(root, w.Extension, func(root, filename string) error {
			basename := filepath.Base(filename)
			name := InstName(root, basename)
			return w.LoadInstanceFile(filename, name)
		})

		if err != nil {
			return err
		}
	}

	// Load from the customize path
	sources, err := w.CompileSource()
	if err != nil {
		log.Error("[Widget] compile.js Source: %s ", err.Error())
		return err
	}
	for name, source := range sources {
		err := w.LoadInstance(source, name)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadInstanceFile load a instance from a file
func (w *Widget) LoadInstanceFile(file string, name string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return w.LoadInstance(data, name)
}

// LoadInstance load a instance
func (w *Widget) LoadInstance(source []byte, name string) error {

	dsl := map[string]interface{}{}
	err := jsoniter.Unmarshal(source, &dsl)
	if err != nil {
		return err
	}

	// Compile DSL
	newdsl, err := w.CompileCompile(name, dsl)
	if err != nil {
		return err
	}

	inst := &Instance{Name: name, DSL: newdsl}

	// Register Models

	// Register Flows

	// Register Tables

	// Register Tasks

	// Register Schedules

	w.Instances[name] = inst

	// Trigger OnLoad Event
	err = w.CompileOnLoad(name, newdsl)
	if err != nil {
		return err
	}

	return nil
}

// InstanceRoot get the instance root path
func (w *Widget) InstanceRoot() (string, error) {
	// walk roots
	var root = ""
	if w.Root != "" {
		root = filepath.Join(w.Path, "..", "..", w.Root)
	}

	if root == "" {
		err := fmt.Errorf("widget %s instance root missing", w.Name)
		log.Warn("[Widget] %s ", err.Error())
		return "", err
	}

	if DirNotExists(root) {
		err := fmt.Errorf("widget %s %s dose not exists", w.Name, root)
		log.Warn("[Widget] %s ", err.Error())
		return root, err
	}

	if w.Extension == "" {
		w.Extension = ".json"
	}

	return root, nil
}
