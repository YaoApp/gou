package widget

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// LoadWidget load widgets
func LoadWidget(path string, name string, register ModuleRegister) (*Widget, error) {
	_, err := Load(path, customProcessRegister(), register)
	if err != nil {
		return nil, err
	}
	return Widgets[name], nil
}

func customProcessRegister() ProcessRegister {

	return func(widget, name string, handler func(args ...interface{}) interface{}) error {

		widget = strings.ToLower(widget)
		name = strings.ToLower(name)
		log.Info("[Widget] Register Process widgets.%s.%s", widget, name)
		processName := strings.ToLower(fmt.Sprintf("widgets.%s.%s", widget, name))
		process.Register(processName, func(process *process.Process) interface{} {
			return handler(process.Args...)
		})

		return nil
	}
}

func processReloadWidgetInstance(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	widgetName := process.ID
	name := process.ArgsString(0)
	source := process.ArgsString(1)
	w, has := Widgets[widgetName]
	if !has {
		exception.New("widget %s does not load", 400, widgetName).Throw()
		return nil
	}
	return w.ReloadInstance([]byte(source), name)
}
