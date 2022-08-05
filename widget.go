package gou

import (
	"strings"

	"github.com/yaoapp/gou/widget"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// WidgetHandlers widget process handlers
var WidgetHandlers = map[string]ProcessHandler{
	"reload": processReloadWidgetInstance,
}

// WidgetCustomHandlers custom widget handlers
var WidgetCustomHandlers = map[string]map[string]ProcessHandler{}

// LoadWidget load widgets
func LoadWidget(path string, name string, register widget.ModuleRegister) (*widget.Widget, error) {
	_, err := widget.Load(path, Yao, customProcessRegister(), register)
	if err != nil {
		return nil, err
	}
	return widget.Widgets[name], nil
}

func customProcessRegister() widget.ProcessRegister {
	return func(widget, name string, handler func(args ...interface{}) interface{}) error {
		widget = strings.ToLower(widget)
		name = strings.ToLower(name)
		log.Info("[Widget] Register Process widgets.%s.%s", widget, name)
		if _, has := WidgetCustomHandlers[widget]; !has {
			WidgetCustomHandlers[widget] = map[string]ProcessHandler{}
		}

		WidgetCustomHandlers[widget][name] = func(process *Process) interface{} {
			return handler(process.Args...)
		}
		return nil
	}
}

func processReloadWidgetInstance(process *Process) interface{} {
	process.ValidateArgNums(2)
	widgetName := process.Class
	name := process.ArgsString(0)
	source := process.ArgsString(1)
	w, has := widget.Widgets[widgetName]
	if !has {
		exception.New("widget %s does not load", 400, widgetName).Throw()
		return nil
	}
	return w.ReloadInstance([]byte(source), name)
}
