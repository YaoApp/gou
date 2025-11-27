package store

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	kv "github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// Store Javascript API
type Store struct{}

// New create a new Store object
func New() *Store {
	return &Store{}
}

// ExportObject Export as a Cache Object
func (store *Store) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Set", store.set(iso))
	tmpl.Set("Get", store.get(iso))
	tmpl.Set("GetSet", store.getSet(iso))
	tmpl.Set("GetDel", store.getDel(iso))
	tmpl.Set("Has", store.has(iso))
	tmpl.Set("Del", store.del(iso))
	tmpl.Set("Keys", store.keys(iso))
	tmpl.Set("Len", store.len(iso))
	tmpl.Set("Clear", store.clear(iso))

	// List operations
	tmpl.Set("Push", store.push(iso))
	tmpl.Set("Pop", store.pop(iso))
	tmpl.Set("Pull", store.pull(iso))
	tmpl.Set("PullAll", store.pullAll(iso))
	tmpl.Set("AddToSet", store.addToSet(iso))
	tmpl.Set("ArrayLen", store.arrayLen(iso))
	tmpl.Set("ArrayGet", store.arrayGet(iso))
	tmpl.Set("ArraySet", store.arraySet(iso))
	tmpl.Set("ArraySlice", store.arraySlice(iso))
	tmpl.Set("ArrayPage", store.arrayPage(iso))
	tmpl.Set("ArrayAll", store.arrayAll(iso))
	return tmpl
}

// ExportFunction Export as a javascript Cache function
func (store *Store) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := store.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache args: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		var name = args[0].String()
		if _, has := kv.Pools[name]; !has {
			msg := fmt.Sprintf("Cache %s does not loaded", name)
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			msg := fmt.Sprintf("Cache: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		this.Set("name", name)
		return this.Value
	})
	return tmpl
}

func (store *Store) set(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Set: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Cache Set: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		v, err := bridge.GoValue(args[1], info.Context())
		if err != nil {
			msg := fmt.Sprintf("Cache Set: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		ttl := 0 * time.Second
		if len(args) > 2 {
			ttl = time.Duration(args[2].Integer()) * time.Second
		}

		c.Set(args[0].String(), v, ttl)
		return nil
	})
}

func (store *Store) getSet(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache GetSet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Cache GetSet: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		fn, err := args[1].AsFunction()
		if err != nil {
			msg := fmt.Sprintf("Cache GetSet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		ttl := 0 * time.Second
		if len(args) > 2 {
			ttl = time.Duration(args[2].Integer()) * time.Second
		}

		value, err := c.GetSet(args[0].String(), ttl, func(key string) (interface{}, error) {
			jsKey, err := bridge.JsValue(info.Context(), key)
			if err != nil {
				return nil, err
			}

			recv, _ := v8go.NewValue(iso, "")
			value, err := fn.Call(recv, jsKey)
			if err != nil {
				return nil, err
			}

			return bridge.GoValue(value, info.Context())
		})

		if err != nil {
			msg := fmt.Sprintf("Cache GetSet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		return res
	})
}

func (store *Store) get(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Get: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		value, ok := c.Get(args[0].String())
		if !ok {
			return v8go.Undefined(iso)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		return res
	})
}

func (store *Store) getDel(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Get: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		value, ok := c.GetDel(args[0].String())
		if !ok {
			return v8go.Undefined(iso)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		return res
	})
}

func (store *Store) del(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Del: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Del: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		c.Del(args[0].String())
		return v8go.Undefined(iso)
	})
}

func (store *Store) has(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Has: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Has: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		has := c.Has(args[0].String())

		res, err := v8go.NewValue(info.Context().Isolate(), has)
		if err != nil {
			msg := fmt.Sprintf("Cache Has: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) len(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Len: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		len := c.Len()
		res, err := v8go.NewValue(info.Context().Isolate(), int32(len))
		if err != nil {
			msg := fmt.Sprintf("Cache Len: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) keys(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Keys: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		keys := c.Keys()
		res, err := bridge.JsValue(info.Context(), keys)
		if err != nil {
			msg := fmt.Sprintf("Cache Keys: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) clear(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Cache Clear: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		c.Clear()
		return v8go.Undefined(iso)
	})
}

func (store *Store) getLRU(info *v8go.FunctionCallbackInfo) (kv.Store, error) {
	name, err := info.This().Get("name")
	if err != nil {
		return nil, err
	}
	c, has := kv.Pools[name.String()]
	if !has {
		return nil, fmt.Errorf("%s does not load", name)
	}
	return c, nil
}

// List operations implementations

func (store *Store) push(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store Push: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store Push: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		values := make([]interface{}, len(args)-1)
		for i, arg := range args[1:] {
			v, err := bridge.GoValue(arg, info.Context())
			if err != nil {
				msg := fmt.Sprintf("Store Push: %s", err.Error())
				log.Error("%s", msg)
				return bridge.JsException(info.Context(), msg)
			}
			values[i] = v
		}

		err = c.Push(key, values...)
		if err != nil {
			msg := fmt.Sprintf("Store Push: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return v8go.Undefined(iso)
	})
}

func (store *Store) pop(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store Pop: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store Pop: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		position := int(args[1].Integer())

		value, err := c.Pop(key, position)
		if err != nil {
			msg := fmt.Sprintf("Store Pop: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Store Pop: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) pull(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store Pull: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store Pull: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		value, err := bridge.GoValue(args[1], info.Context())
		if err != nil {
			msg := fmt.Sprintf("Store Pull: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		err = c.Pull(key, value)
		if err != nil {
			msg := fmt.Sprintf("Store Pull: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return v8go.Undefined(iso)
	})
}

func (store *Store) pullAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store PullAll: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store PullAll: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		values := make([]interface{}, len(args)-1)
		for i, arg := range args[1:] {
			v, err := bridge.GoValue(arg, info.Context())
			if err != nil {
				msg := fmt.Sprintf("Store PullAll: %s", err.Error())
				log.Error("%s", msg)
				return bridge.JsException(info.Context(), msg)
			}
			values[i] = v
		}

		err = c.PullAll(key, values)
		if err != nil {
			msg := fmt.Sprintf("Store PullAll: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return v8go.Undefined(iso)
	})
}

func (store *Store) addToSet(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store AddToSet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store AddToSet: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		values := make([]interface{}, len(args)-1)
		for i, arg := range args[1:] {
			v, err := bridge.GoValue(arg, info.Context())
			if err != nil {
				msg := fmt.Sprintf("Store AddToSet: %s", err.Error())
				log.Error("%s", msg)
				return bridge.JsException(info.Context(), msg)
			}
			values[i] = v
		}

		err = c.AddToSet(key, values...)
		if err != nil {
			msg := fmt.Sprintf("Store AddToSet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return v8go.Undefined(iso)
	})
}

func (store *Store) arrayLen(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayLen: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Store ArrayLen: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		length := c.ArrayLen(key)

		res, err := v8go.NewValue(iso, int32(length))
		if err != nil {
			msg := fmt.Sprintf("Store ArrayLen: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) arrayGet(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayGet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Store ArrayGet: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		index := int(args[1].Integer())

		value, err := c.ArrayGet(key, index)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayGet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayGet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) arraySet(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArraySet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 3 {
			msg := fmt.Sprintf("Store ArraySet: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		index := int(args[1].Integer())
		value, err := bridge.GoValue(args[2], info.Context())
		if err != nil {
			msg := fmt.Sprintf("Store ArraySet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		err = c.ArraySet(key, index, value)
		if err != nil {
			msg := fmt.Sprintf("Store ArraySet: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return v8go.Undefined(iso)
	})
}

func (store *Store) arraySlice(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArraySlice: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 3 {
			msg := fmt.Sprintf("Store ArraySlice: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		skip := int(args[1].Integer())
		limit := int(args[2].Integer())

		values, err := c.ArraySlice(key, skip, limit)
		if err != nil {
			msg := fmt.Sprintf("Store ArraySlice: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), values)
		if err != nil {
			msg := fmt.Sprintf("Store ArraySlice: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) arrayPage(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayPage: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 3 {
			msg := fmt.Sprintf("Store ArrayPage: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		page := int(args[1].Integer())
		pageSize := int(args[2].Integer())

		values, err := c.ArrayPage(key, page, pageSize)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayPage: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), values)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayPage: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}

func (store *Store) arrayAll(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		c, err := store.getLRU(info)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayAll: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Store ArrayAll: %s", "Missing parameters")
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		key := args[0].String()
		values, err := c.ArrayAll(key)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayAll: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), values)
		if err != nil {
			msg := fmt.Sprintf("Store ArrayAll: %s", err.Error())
			log.Error("%s", msg)
			return bridge.JsException(info.Context(), msg)
		}
		return res
	})
}
