package gou

import (
	"time"

	"github.com/yaoapp/kun/log"
)

// StoreHandlers store process handlers
var StoreHandlers = map[string]ProcessHandler{
	"get":    processStoreGet,
	"set":    processStoreSet,
	"has":    processStoreHas,
	"del":    processStoreDel,
	"getdel": processStoreGetDel,
	"len":    processStoreLen,
	"keys":   processStoreKeys,
	"clear":  processStoreClear,
}

// processStoreGet stores.<name>.Get
func processStoreGet(process *Process) interface{} {
	process.ValidateArgNums(1)
	store := SelectStore(process.Class)
	value, ok := store.Get(process.ArgsString(0))
	if !ok {
		log.Error("store %s Get return false", process.Class)
		return nil
	}
	return value
}

// processStoreSet stores.<name>.Set
func processStoreSet(process *Process) interface{} {
	process.ValidateArgNums(2)
	store := SelectStore(process.Class)
	duration := process.ArgsInt(2, 0)
	store.Set(process.ArgsString(0), process.Args[1], time.Duration(duration)*time.Second)
	return nil
}

// processStoreHas stores.<name>.Has
func processStoreHas(process *Process) interface{} {
	process.ValidateArgNums(1)
	store := SelectStore(process.Class)
	return store.Has(process.ArgsString(0))
}

// processStoreDel stores.<name>.Del
func processStoreDel(process *Process) interface{} {
	process.ValidateArgNums(1)
	store := SelectStore(process.Class)
	store.Del(process.ArgsString(0))
	return nil
}

// processStoreGetDel stores.<name>.GetDel
func processStoreGetDel(process *Process) interface{} {
	process.ValidateArgNums(1)
	store := SelectStore(process.Class)
	value, ok := store.GetDel(process.ArgsString(0))
	if !ok {
		log.Error("store %s GetDel return false", process.Class)
		return nil
	}
	return value
}

// processStoreLen stores.<name>.Len
func processStoreLen(process *Process) interface{} {
	store := SelectStore(process.Class)
	return store.Len()
}

// processStoreKeys stores.<name>.Keys
func processStoreKeys(process *Process) interface{} {
	store := SelectStore(process.Class)
	return store.Keys()
}

// processStoreClear stores.<name>.Keys
func processStoreClear(process *Process) interface{} {
	store := SelectStore(process.Class)
	store.Clear()
	return nil
}
