package objects

import (
	"fmt"

	"github.com/yaoapp/gou/query"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/gou/runtime/yao/bridge"
	"github.com/yaoapp/gou/runtime/yao/values"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"rogchap.com/v8go"
)

// QueryOBJ Javascript API
type QueryOBJ struct{}

// NewQuery create a new Query Object
func NewQuery() *QueryOBJ {
	return &QueryOBJ{}
}

// ExportObject Export as a Cache Object
// var query = new Query("engine")
// query.Get({"select":["id"], "from":"user", "limit":1})
// query.Paginate({"select":["id"], "from":"user"})
// query.First({"select":["id"], "from":"user"})
// query.Run({"stmt":"show version"})
func (obj *QueryOBJ) ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	tmpl := v8go.NewObjectTemplate(iso)
	tmpl.Set("Get", obj.get(iso))
	tmpl.Set("Run", obj.run(iso))
	tmpl.Set("Paginate", obj.paginate(iso))
	tmpl.Set("First", obj.first(iso))
	return tmpl
}

// ExportFunction Export as a javascript Cache function
// var query = new Query("engine")
func (obj *QueryOBJ) ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	object := obj.ExportObject(iso)
	tmpl := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {

		var name = "default"
		args := info.Args()
		if len(args) > 0 {
			name = args[0].String()
		}

		if _, has := query.Engines[name]; !has {
			msg := fmt.Sprintf("Query Engine %s does not loaded", name)
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		this, err := object.NewInstance(info.Context())
		if err != nil {
			msg := fmt.Sprintf("Query Engine: %s", err.Error())
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		this.Set("name", name)
		return this.Value
	})
	return tmpl
}

func (obj *QueryOBJ) get(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Query: %s", "Missing parameters")
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		data, err := obj.runQueryGet(iso, info, args[0])
		if err != nil {
			msg := fmt.Sprintf("Query: %s", err.Error())
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		return obj.response(iso, info, data)
	})
}

func (obj *QueryOBJ) first(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Query: %s", "Missing parameters")
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		data, err := obj.runQueryFirst(iso, info, args[0])
		if err != nil {
			msg := fmt.Sprintf("Query: %s", err.Error())
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		return obj.response(iso, info, data)
	})
}

func (obj *QueryOBJ) paginate(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Query: %s", "Missing parameters")
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		data, err := obj.runQueryPaginate(iso, info, args[0])
		if err != nil {
			msg := fmt.Sprintf("Query: %s", err.Error())
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		return obj.response(iso, info, data)
	})
}

func (obj *QueryOBJ) run(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			msg := fmt.Sprintf("Query: %s", "Missing parameters")
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		data, err := obj.runQueryRun(iso, info, args[0])
		if err != nil {
			msg := fmt.Sprintf("Query: %s", err.Error())
			log.Error(msg)
			return iso.ThrowException(values.Error(info.Context(), msg))
		}

		return obj.response(iso, info, data)
	})
}

func (obj *QueryOBJ) runQueryGet(iso *v8go.Isolate, info *v8go.FunctionCallbackInfo, param *v8go.Value) (data interface{}, err error) {
	defer func() { err = exception.Catch(recover()) }()
	dsl, input, err := obj.getQueryDSL(info, param)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return nil, err
	}
	data = dsl.Get(input)
	return obj.response(iso, info, data), err
}

func (obj *QueryOBJ) runQueryPaginate(iso *v8go.Isolate, info *v8go.FunctionCallbackInfo, param *v8go.Value) (data interface{}, err error) {
	defer func() { err = exception.Catch(recover()) }()
	dsl, input, err := obj.getQueryDSL(info, param)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return nil, err
	}
	data = dsl.Paginate(input)
	return obj.response(iso, info, data), err
}

func (obj *QueryOBJ) runQueryFirst(iso *v8go.Isolate, info *v8go.FunctionCallbackInfo, param *v8go.Value) (data interface{}, err error) {
	defer func() { err = exception.Catch(recover()) }()
	dsl, input, err := obj.getQueryDSL(info, param)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return nil, err
	}
	data = dsl.First(input)
	return obj.response(iso, info, data), err
}

func (obj *QueryOBJ) runQueryRun(iso *v8go.Isolate, info *v8go.FunctionCallbackInfo, param *v8go.Value) (data interface{}, err error) {
	defer func() { err = exception.Catch(recover()) }()
	dsl, input, err := obj.getQueryDSL(info, param)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return nil, err
	}
	data = dsl.Run(input)
	return obj.response(iso, info, data), err
}

func (obj *QueryOBJ) response(iso *v8go.Isolate, info *v8go.FunctionCallbackInfo, data interface{}) *v8go.Value {
	res, err := bridge.AnyToValue(info.Context(), data)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return iso.ThrowException(values.Error(info.Context(), msg))
	}
	return res
}

func (obj *QueryOBJ) getEngine(info *v8go.FunctionCallbackInfo) (share.DSL, error) {
	name, err := info.This().Get("name")
	if err != nil {
		return nil, err
	}
	return query.Select(name.String())
}

func (obj *QueryOBJ) getQueryDSL(info *v8go.FunctionCallbackInfo, param *v8go.Value) (share.DSL, maps.MapStrAny, error) {

	engine, err := obj.getEngine(info)
	if err != nil {
		msg := fmt.Sprintf("Query: %s", err.Error())
		log.Error(msg)
		return nil, nil, err
	}

	v, err := bridge.ToInterface(param)
	if err != nil {
		return nil, nil, err
	}

	switch v.(type) {
	case map[string]interface{}:
		var params = maps.Of(v.(map[string]interface{}))
		dsl, err := engine.Load(params) // should be cached
		if err != nil {
			return nil, nil, err
		}
		return dsl, params, nil
	}
	return nil, nil, fmt.Errorf("Query: %s", "parameters fomart error")
}
