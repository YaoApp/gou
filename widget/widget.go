package widget

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// Widgets the registered widgets
var Widgets = map[string]*Widget{}

// Load load a widget
func Load(path string, processRegister ProcessRegister, moduleRegister ModuleRegister) (*Widget, error) {

	w := &Widget{
		Name:            filepath.Base(path),
		Path:            path,
		Instances:       map[string]*Instance{},
		ProcessRegister: processRegister,
		ModuleRegister:  moduleRegister,
	}

	setting := filepath.Join(path, "widget.yao")
	data, err := application.App.Read(setting)
	if err != nil {
		log.Error("[Widget] open widget.yao error: %s", err.Error())
		return nil, err
	}

	err = application.Parse(setting, data, w)
	if err != nil {
		log.Error("[Widget] parse widget.yao error: %s", err.Error())
		return nil, err
	}

	err = w.loadScripts()
	if err != nil {
		log.Error("[Widget] load widget scirpts error: %s", err.Error())
		return nil, err
	}

	// Register the process
	err = w.RegisterProcess()
	if err != nil {
		return nil, err
	}

	// Register the api
	err = w.RegisterAPI()
	if err != nil {
		return nil, err
	}

	process.Register(fmt.Sprintf("widgets.%s.reload", w.Name), processReloadWidgetInstance)

	// Register the widget
	Widgets[w.Name] = w
	return w, nil
}

// Migrate the migrate
func (w *Widget) Migrate() error {
	return nil
}
