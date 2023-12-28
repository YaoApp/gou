package v8

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("scripts", processScripts)
	process.Register("studio", processStudio)
}

// processScripts
func processScripts(process *process.Process) interface{} {

	script, err := Select(process.ID)
	if err != nil {
		exception.New("scripts.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	return script.Exec(process)
}

// processScripts scripts.ID.Method
func processStudio(process *process.Process) interface{} {

	script, err := SelectRoot(process.ID)
	if err != nil {
		exception.New("studio.%s not loaded", 404, process.ID).Throw()
		return nil
	}
	return script.Exec(process)

}
