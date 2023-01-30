package plugin

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("plugins", processPlugins)
}

// processPlugins
func processPlugins(process *process.Process) interface{} {

	plugin, err := Select(process.ID)
	if err != nil {
		exception.New("plugins.%s not loaded", 404, process.ID).Throw()
		return nil
	}
	res, err := plugin.Exec(process.Method, process.Args...)
	if err != nil {
		exception.Err(err, 500).Throw()
	}

	return res.MustValue()
}
