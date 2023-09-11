package v8

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
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

	ctx, err := script.NewContext(process.Sid, process.Global)
	if err != nil {
		message := fmt.Sprintf("scripts.%s failed to create context. %+v", process.ID, err)
		log.Error("[V8] process error. %s", message)
		exception.New(message, 500).Throw()
		return nil
	}
	defer ctx.Close()

	res, err := ctx.Call(process.Method, process.Args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

// processScripts scripts.ID.Method
func processStudio(process *process.Process) interface{} {

	script, err := SelectRoot(process.ID)
	if err != nil {
		exception.New("studio.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	ctx, err := script.NewContext(process.Sid, process.Global)
	if err != nil {
		message := fmt.Sprintf("studio.%s failed to create context. %+v", process.ID, err)
		log.Error("[V8] process error. %s", message)
		exception.New(message, 500).Throw()
		return nil
	}
	defer ctx.Close()

	res, err := ctx.Call(process.Method, process.Args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}
