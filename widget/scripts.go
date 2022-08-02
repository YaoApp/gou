package widget

import (
	"fmt"
	"path/filepath"

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
func (w *Widget) ScriptExec(script, method string, args ...interface{}) (interface{}, error) {
	return w.Exec(fmt.Sprintf("scripts.%s", script), method, args...)
}

// Exec Execute the export script
func (w *Widget) Exec(name, method string, args ...interface{}) (interface{}, error) {
	return w.Runtime.New(fmt.Sprintf("%s.%s", w.Name, name), method).Call(args...)
}

// loadWidgetScirpts load the compile, export, process script
func (w *Widget) loadWidgetScirpts() error {
	if w.Runtime == nil {
		return fmt.Errorf("Javascript runtime is not set")
	}

	file := filepath.Join(w.Path, "compile.js")
	err := w.Runtime.Load(file, fmt.Sprintf("%s.compile", w.Name))
	if err != nil {
		log.Error("[Widget] load compile.js error: %s", err.Error())
		return err
	}

	file = filepath.Join(w.Path, "export.js")
	err = w.Runtime.Load(file, fmt.Sprintf("%s.export", w.Name))
	if err != nil {
		log.Error("[Widget] load export.js error: %s", err.Error())
		return err
	}

	file = filepath.Join(w.Path, "process.js")
	err = w.Runtime.Load(file, fmt.Sprintf("%s.process", w.Name))
	if err != nil {
		log.Warn("[Widget] load process.js error: %s", err.Error())
	}

	return nil
}

// loadHelperScripts
func (w *Widget) loadHelperScripts() error {
	if DirNotExists(w.Path) {
		return fmt.Errorf("%s does not exists", w.Path)
	}
	return Walk(w.Path, ".js", func(root, filename string) error {

		basename := filepath.Base(filename)
		if basename == "process.js" || basename == "export.js" || basename == "compile.js" {
			return nil
		}

		name := fmt.Sprintf("%s.scripts.%s", w.Name, InstName(root, basename))
		err := w.Runtime.Load(filename, name)
		if err != nil {
			log.Warn("[Widget] load script %s error: %s", InstName(root, basename), err.Error())
			return err
		}
		return nil
	})
}
