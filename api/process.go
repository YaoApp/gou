package api

import "github.com/yaoapp/gou/process"

func init() {
	process.RegisterGroup("api", map[string]process.Handler{
		"list": processList,
	})
}

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
