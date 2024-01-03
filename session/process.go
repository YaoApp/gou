package session

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// SessionHandlers 模型运行器
var SessionHandlers = map[string]process.Handler{
	"id":      processID,
	"get":     processGet,
	"del":     processDel,
	"set":     processSet,
	"dump":    processDump,
	"getmany": processGetMany,
	"setmany": processSetMany,
	"delmany": processDelMany,
}

func init() {
	process.RegisterGroup("session", SessionHandlers)
}

// Lang get user language setting
func Lang(process *process.Process, defaults ...string) string {
	if process.Sid != "" {
		ss := Global().ID(process.Sid)
		v, _ := ss.Get("__yao_lang")
		if len(defaults) > 0 && v == nil {
			return defaults[0]
		}

		lang := ""
		if v != nil {
			lang = fmt.Sprintf("%v", v)
		}
		return lang
	}

	if len(defaults) > 0 {
		return defaults[0]
	}

	return ""
}

// processID
func processID(process *process.Process) interface{} {
	setSession(process)
	return process.Sid
}

// processGet
func processGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	ss := setSession(process)
	if process.NumOfArgs() == 2 {
		ss = Global().ID(process.ArgsString(1))
		return ss.MustGet(process.ArgsString(0))
	}
	return ss.MustGet(process.ArgsString(0))
}

// processDel
func processDel(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	ss := setSession(process)
	if process.NumOfArgs() == 2 {
		ss = Global().ID(process.ArgsString(1))
		ss.MustDel(process.ArgsString(0))
		return nil
	}
	ss.MustDel(process.ArgsString(0))
	return nil
}

// processSet
func processSet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	ss := setSession(process)
	if process.NumOfArgs() == 3 {
		log.Debug("set sessio ttl KEY: %s, VALUE: %v, TS: %d", process.ArgsString(0), process.Args[1], process.ArgsInt(2))
		ss.MustSetWithEx(process.ArgsString(0), process.Args[1], time.Duration(process.ArgsInt(2))*time.Second)
		return nil

	} else if process.NumOfArgs() == 4 {
		log.Debug("set session id & ttl ID: %s KEY: %s, VALUE: %v, TS: %d", process.ArgsString(3), process.ArgsString(0), process.Args[1], process.ArgsInt(2))
		ss = Global().ID(process.ArgsString(3))
		ss.MustSetWithEx(process.ArgsString(0), process.Args[1], time.Duration(process.ArgsInt(2))*time.Second)
		return nil
	}

	log.Debug("set session KEY: %s, VALUE: %v", process.ArgsString(0), process.Args[1])
	ss.MustSet(process.ArgsString(0), process.Args[1])
	return nil
}

// processDump
func processDump(process *process.Process) interface{} {
	ss := setSession(process)
	if process.NumOfArgs() == 1 {
		ss = Global().ID(process.ArgsString(0))
		return ss.MustDump()
	}
	return ss.MustDump()
}

// processGetMany
func processGetMany(process *process.Process) interface{} {
	ss := setSession(process)
	process.ValidateArgNums(1)
	keys := process.ArgsStrings(0)
	res := map[string]interface{}{}
	if process.NumOfArgs() == 2 {
		ss = Global().ID(process.ArgsString(1))
		for _, key := range keys {
			res[key] = ss.MustGet(key)
		}
		return res
	}

	for _, key := range keys {
		res[key] = ss.MustGet(key)
	}
	return res
}

// processDelMany
func processDelMany(process *process.Process) interface{} {
	ss := setSession(process)
	process.ValidateArgNums(1)
	keys := process.ArgsStrings(0)
	if process.NumOfArgs() == 2 {
		ss = Global().ID(process.ArgsString(1))
		for _, key := range keys {
			ss.MustDel(key)
		}
		return nil
	}
	for _, key := range keys {
		ss.MustDel(key)
	}
	return nil
}

// processSetMany
func processSetMany(process *process.Process) interface{} {
	ss := setSession(process)
	process.ValidateArgNums(1)
	if process.NumOfArgs() == 2 {
		ss.MustSetManyWithEx(process.ArgsMap(0), time.Duration(process.ArgsInt(1))*time.Second)
		return nil

	} else if process.NumOfArgs() == 3 {
		ss = Global().ID(process.ArgsString(2))
		ss.MustSetManyWithEx(process.ArgsMap(0), time.Duration(process.ArgsInt(1))*time.Second)
		return nil
	}

	ss.MustSetMany(process.ArgsMap(0))
	return nil
}

func setSession(process *process.Process) *Session {
	ss := Global()
	if process.Sid != "" {
		ss.ID(process.Sid)
	}
	return ss
}
