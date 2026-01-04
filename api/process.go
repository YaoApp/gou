package api

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("api", map[string]process.Handler{
		"list":   processList,
		"reload": processReload,
	})
}

// processList lists all loaded APIs or specific APIs by IDs
// Process: api.List, api.List ["id1", "id2"]
func processList(process *process.Process) interface{} {

	ids := []string{}
	if process.NumOfArgs() > 0 {
		ids = process.ArgsStrings(0)
	}

	// List all
	apis := map[string]*API{}
	if len(ids) == 0 {
		for id, api := range APIs {
			apis[id] = api
		}
		return apis
	}

	// List by ids
	for _, id := range ids {
		if api, has := APIs[id]; has {
			apis[id] = api
		}
	}
	return apis
}

// processReload reloads all API definitions from the specified directory
// Process: api.Reload, api.Reload "apis"
func processReload(process *process.Process) interface{} {
	root := "apis"
	if process.NumOfArgs() > 0 {
		root = process.ArgsString(0)
	}

	err := ReloadAPIs(root)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{
		"success": true,
		"message": "APIs reloaded",
		"count":   len(APIs),
	}
}
