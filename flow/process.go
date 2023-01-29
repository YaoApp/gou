package flow

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("flows", processFlows)
}

// processScripts
func processFlows(process *process.Process) interface{} {

	flow, err := Select(process.ID)
	if err != nil {
		exception.New("flows.%s not loaded", 404, process.ID).Throw()
		return nil
	}

	flow.WithGlobal(process.Global).WithSID(process.Sid)

	res, err := flow.Exec(process.Args...)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}
