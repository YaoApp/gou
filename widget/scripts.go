package widget

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/kun/log"
)

func (w *Widget) loadScripts() error {
	err := w.loadWidgetScirpts()
	if err != nil {
		return err
	}
	return w.loadHelperScripts()
}

// CompileExec Execute the compile script
func (w *Widget) CompileExec(method string, args ...interface{}) (interface{}, error) {
	return w.Exec("compile", method, args...)
}

// ExportExec Execute the export script
func (w *Widget) ExportExec(method string, args ...interface{}) (interface{}, error) {
	return w.Exec("export", method, args...)
}

// ProcessExec Execute the process script
func (w *Widget) ProcessExec(method string, args ...interface{}) (interface{}, error) {
	return w.Exec("process", method, args...)
}

// ScriptExec Execute the other script
func (w *Widget) ScriptExec(scriptName, method string, args ...interface{}) (interface{}, error) {

	name := fmt.Sprintf("%s.scripts.%s", w.Name, scriptName)
	inst, err := v8.SelectWidget(name)
	if err != nil {
		return nil, err
	}

	ctx, err := inst.NewContext("", nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	return ctx.Call(method, args...)
}

// Exec Execute the export script
func (w *Widget) Exec(name, method string, args ...interface{}) (interface{}, error) {

	name = fmt.Sprintf("%s.%s", w.Name, name)
	inst, err := v8.SelectWidget(name)
	if err != nil {
		return nil, err
	}

	ctx, err := inst.NewContext("", nil)
	if err != nil {
		return nil, err
	}
	defer ctx.Close()

	return ctx.Call(method, args...)
}

// loadWidgetScirpts load the compile, export, process script
func (w *Widget) loadWidgetScirpts() error {

	file := filepath.Join(w.Path, "compile.js")
	_, err := v8.LoadWidget(file, fmt.Sprintf("%s.compile", w.Name))
	if err != nil {
		log.Error("[Widget] load compile.js error: %s", err.Error())
		return err
	}

	file = filepath.Join(w.Path, "export.js")
	_, err = v8.LoadWidget(file, fmt.Sprintf("%s.export", w.Name))
	if err != nil {
		log.Error("[Widget] load export.js error: %s", err.Error())
		return err
	}

	file = filepath.Join(w.Path, "process.js")
	_, err = v8.LoadWidget(file, fmt.Sprintf("%s.process", w.Name))
	if err != nil {
		log.Warn("[Widget] load process.js error: %s", err.Error())
	}

	return nil
}

// loadHelperScripts
func (w *Widget) loadHelperScripts() error {

	return application.App.Walk(w.Path, func(root, filename string, isdir bool) error {

		if isdir {
			return nil
		}

		basename := filepath.Base(filename)
		if basename == "process.js" || basename == "export.js" || basename == "compile.js" {
			return nil
		}

		name := fmt.Sprintf("%s.scripts.%s", w.Name, InstName(root, basename))
		_, err := v8.LoadWidget(filename, name)
		if err != nil {
			log.Warn("[Widget] load script %s error: %s", InstName(root, basename), err.Error())
			return err
		}
		return nil
	}, "*.js")

}
