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
	return tmpl
}

// ExportFunction Export as a javascript Cache function
func (store *Store) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := store.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache args: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		var name = args[0].String()
		if _, has := kv.Pools[name]; !has {
			msg := fmt.Sprintf("Cache %s does not loaded", name)
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			msg := fmt.Sprintf("Cache: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Cache Set: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		v, err := bridge.GoValue(args[1])
		if err != nil {
			msg := fmt.Sprintf("Cache Set: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 2 {
			msg := fmt.Sprintf("Cache GetSet: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		fn, err := args[1].AsFunction()
		if err != nil {
			msg := fmt.Sprintf("Cache GetSet: %s", err.Error())
			log.Error(msg)
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

			return bridge.GoValue(value)
		})

		if err != nil {
			msg := fmt.Sprintf("Cache GetSet: %s", err.Error())
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Get: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		value, ok := c.Get(args[0].String())
		if !ok {
			return v8go.Undefined(iso)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Get: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		value, ok := c.GetDel(args[0].String())
		if !ok {
			return v8go.Undefined(iso)
		}

		res, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			msg := fmt.Sprintf("Cache Get: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Del: %s", "Missing parameters")
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Cache Has: %s", "Missing parameters")
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		has := c.Has(args[0].String())

		res, err := v8go.NewValue(info.Context().Isolate(), has)
		if err != nil {
			msg := fmt.Sprintf("Cache Has: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		len := c.Len()
		res, err := v8go.NewValue(info.Context().Isolate(), int32(len))
		if err != nil {
			msg := fmt.Sprintf("Cache Len: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
			return bridge.JsException(info.Context(), msg)
		}

		keys := c.Keys()
		res, err := bridge.JsValue(info.Context(), keys)
		if err != nil {
			msg := fmt.Sprintf("Cache Keys: %s", err.Error())
			log.Error(msg)
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
			log.Error(msg)
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
