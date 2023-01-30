package widget

import (
	"fmt"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
)

// Load load the widget instances
func (w *Widget) Load() error {

	// Load from the yao app path
	root, err := w.InstanceRoot()
	if err == nil {
		err := Walk(root, w.Extensions, func(root, filename string) error {
			basename := filepath.Base(filename)
			name := InstName(root, basename)
			err := w.LoadInstanceFile(filename, name)
			if err != nil {
				log.Error("[Widget] load %s.%s error: %s file:%s", w.Name, name, err.Error(), filename)
				return err
			}
			log.Info("[Widget] %s.%s loaded", w.Name, name)
			return nil
		})

		if err != nil {
			return err
		}
	}

	// Load from the customize path
	sources, err := w.CompileSource()
	if err != nil {
		log.Error("[Widget] %s compile.js Source: %s ", w.Name, err.Error())
		return err
	}
	for name, source := range sources {
		err := w.LoadInstance(source, name)
		if err != nil {
			log.Error("[Widget] load %s.%s error: %s ", w.Name, name, err.Error())
			continue
		}
		log.Info("[Widget] %s.%s loaded ", w.Name, name)
	}

	return nil
}

// LoadInstanceFile load a instance from a file
func (w *Widget) LoadInstanceFile(file string, name string) error {
	data, err := application.App.Read(file)
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

	// Register modules
	if w.Modules != nil {
		for _, module := range w.Modules {
			err = w.RegisterModule(module, name, dsl)
			if err != nil {
				return err
			}
		}
	}

	// Save instances
	w.Instances[name] = inst

	// Trigger OnLoad Event
	err = w.CompileOnLoad(name, newdsl)
	if err != nil {
		return err
	}

	return nil
}

// RemoveInstance remove the widget instance
func (w *Widget) RemoveInstance(name string) {
	if _, has := w.Instances[name]; !has {
		delete(w.Instances, name)
	}
}

// ReloadInstanceFile reload the widget instance from file
func (w *Widget) ReloadInstanceFile(file string, name string) error {
	return w.LoadInstanceFile(file, name)
}

// ReloadInstance reload the widget instance
func (w *Widget) ReloadInstance(source []byte, name string) error {
	return w.LoadInstance(source, name)
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

	if w.Extensions == nil || len(w.Extensions) == 0 {
		w.Extensions = []string{".yao"}
	}

	return root, nil
}
