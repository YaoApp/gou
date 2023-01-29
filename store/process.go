package store

import (
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

// StoreHandlers store process handlers
var StoreHandlers = map[string]process.Handler{
	"get":    processStoreGet,
	"set":    processStoreSet,
	"has":    processStoreHas,
	"del":    processStoreDel,
	"getdel": processStoreGetDel,
	"len":    processStoreLen,
	"keys":   processStoreKeys,
	"clear":  processStoreClear,
}

func init() {
	process.RegisterGroup("stores", StoreHandlers)
}

// processStoreGet stores.<name>.Get
func processStoreGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	value, ok := store.Get(process.ArgsString(0))
	if !ok {
		log.Error("store %s Get return false", process.ID)
		return nil
	}
	return value
}

// processStoreSet stores.<name>.Set
func processStoreSet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	duration := process.ArgsInt(2, 0)
	store.Set(process.ArgsString(0), process.Args[1], time.Duration(duration)*time.Second)
	return nil
}

// processStoreHas stores.<name>.Has
func processStoreHas(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	return store.Has(process.ArgsString(0))
}

// processStoreDel stores.<name>.Del
func processStoreDel(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	store.Del(process.ArgsString(0))
	return nil
}

// processStoreGetDel stores.<name>.GetDel
func processStoreGetDel(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	value, ok := store.GetDel(process.ArgsString(0))
	if !ok {
		log.Error("store %s GetDel return false", process.ID)
		return nil
	}
	return value
}

// processStoreLen stores.<name>.Len
func processStoreLen(process *process.Process) interface{} {
	store := Select(process.ID)
	return store.Len()
}

// processStoreKeys stores.<name>.Keys
func processStoreKeys(process *process.Process) interface{} {
	store := Select(process.ID)
	return store.Keys()
}

// processStoreClear stores.<name>.Keys
func processStoreClear(process *process.Process) interface{} {
	store := Select(process.ID)
	store.Clear()
	return nil
}
