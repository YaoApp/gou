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

	// List operations
	"push":       processStorePush,
	"pop":        processStorePop,
	"pull":       processStorePull,
	"pullall":    processStorePullAll,
	"addtoset":   processStoreAddToSet,
	"arraylen":   processStoreArrayLen,
	"arrayget":   processStoreArrayGet,
	"arrayset":   processStoreArraySet,
	"arrayslice": processStoreArraySlice,
	"arraypage":  processStoreArrayPage,
	"arrayall":   processStoreArrayAll,
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

// processStoreClear stores.<name>.Clear
func processStoreClear(process *process.Process) interface{} {
	store := Select(process.ID)
	store.Clear()
	return nil
}

// processStorePush stores.<name>.Push
func processStorePush(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)
	values := process.Args[1:]
	err := store.Push(key, values...)
	if err != nil {
		log.Error("store %s Push error: %v", process.ID, err)
		return err
	}
	return nil
}

// processStorePop stores.<name>.Pop
func processStorePop(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)
	position := process.ArgsInt(1)
	value, err := store.Pop(key, position)
	if err != nil {
		log.Error("store %s Pop error: %v", process.ID, err)
		return nil
	}
	return value
}

// processStorePull stores.<name>.Pull
func processStorePull(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)
	value := process.Args[1]
	err := store.Pull(key, value)
	if err != nil {
		log.Error("store %s Pull error: %v", process.ID, err)
		return err
	}
	return nil
}

// processStorePullAll stores.<name>.PullAll
func processStorePullAll(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)

	var values []interface{}

	// If only one argument after key, check if it's a slice
	if len(process.Args) == 2 {
		valuesArg := process.Args[1]
		switch v := valuesArg.(type) {
		case []interface{}:
			values = v
		case []string:
			values = make([]interface{}, len(v))
			for i, s := range v {
				values[i] = s
			}
		default:
			values = []interface{}{v}
		}
	} else {
		// Multiple arguments - use all arguments after key
		values = process.Args[1:]
	}

	err := store.PullAll(key, values)
	if err != nil {
		log.Error("store %s PullAll error: %v", process.ID, err)
		return err
	}
	return nil
}

// processStoreAddToSet stores.<name>.AddToSet
func processStoreAddToSet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)
	values := process.Args[1:]
	err := store.AddToSet(key, values...)
	if err != nil {
		log.Error("store %s AddToSet error: %v", process.ID, err)
		return err
	}
	return nil
}

// processStoreArrayLen stores.<name>.ArrayLen
func processStoreArrayLen(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	key := process.ArgsString(0)
	return store.ArrayLen(key)
}

// processStoreArrayGet stores.<name>.ArrayGet
func processStoreArrayGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	store := Select(process.ID)
	key := process.ArgsString(0)
	index := process.ArgsInt(1)
	value, err := store.ArrayGet(key, index)
	if err != nil {
		log.Error("store %s ArrayGet error: %v", process.ID, err)
		return nil
	}
	return value
}

// processStoreArraySet stores.<name>.ArraySet
func processStoreArraySet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	store := Select(process.ID)
	key := process.ArgsString(0)
	index := process.ArgsInt(1)
	value := process.Args[2]
	err := store.ArraySet(key, index, value)
	if err != nil {
		log.Error("store %s ArraySet error: %v", process.ID, err)
		return err
	}
	return nil
}

// processStoreArraySlice stores.<name>.ArraySlice
func processStoreArraySlice(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	store := Select(process.ID)
	key := process.ArgsString(0)
	skip := process.ArgsInt(1)
	limit := process.ArgsInt(2)
	values, err := store.ArraySlice(key, skip, limit)
	if err != nil {
		log.Error("store %s ArraySlice error: %v", process.ID, err)
		return nil
	}
	return values
}

// processStoreArrayPage stores.<name>.ArrayPage
func processStoreArrayPage(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	store := Select(process.ID)
	key := process.ArgsString(0)
	page := process.ArgsInt(1)
	pageSize := process.ArgsInt(2)
	values, err := store.ArrayPage(key, page, pageSize)
	if err != nil {
		log.Error("store %s ArrayPage error: %v", process.ID, err)
		return nil
	}
	return values
}

// processStoreArrayAll stores.<name>.ArrayAll
func processStoreArrayAll(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	store := Select(process.ID)
	key := process.ArgsString(0)
	values, err := store.ArrayAll(key)
	if err != nil {
		log.Error("store %s ArrayAll error: %v", process.ID, err)
		return nil
	}
	return values
}
