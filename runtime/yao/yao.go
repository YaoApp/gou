package yao

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/utils"
	v8 "rogchap.com/v8go"
)

// New create a pure ES6 engine (v8)
func New() *Yao {
	iso := v8.NewIsolate()
	global := v8.NewObjectTemplate(iso)

	yao := &Yao{
		iso:             iso,
		template:        global,
		scripts:         map[string]script{},
		objectTemplates: map[string]*v8.ObjectTemplate{},
	}

	yao.template.Set("fetch", v8.NewFunctionTemplate(yao.iso, yao.jsFetch))
	return yao
}

// Load load and compile script
func (yao *Yao) Load(filename string, name string) error {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return yao.LoadReader(file, name, filename)
}

// LoadReader load and compile script
func (yao *Yao) LoadReader(reader io.Reader, name string, filename ...string) error {
	source, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	scriptfile := fmt.Sprintf("%s.js", name)
	if len(filename) > 0 {
		scriptfile = filename[0]
	}
	iso := v8.NewIsolate()
	defer iso.Close()

	// Compile
	code := string(source)
	compiled, err := yao.iso.CompileUnboundScript(code, scriptfile, v8.CompileOptions{}) // compile script to get cached data
	if err != nil {
		return err
	}

	yao.scripts[name] = script{
		name:     name,
		filename: scriptfile,
		source:   code,
		compiled: compiled,
	}
	return nil
}

// Call cal javascript function
func (yao *Yao) Call(data map[string]interface{}, name string, method string, args ...interface{}) (interface{}, error) {
	script, has := yao.scripts[name]
	if !has {
		return nil, fmt.Errorf("The %s does not loaded (%d)", name, len(yao.scripts))
	}

	v8ctx := v8.NewContext(yao.iso, yao.template) // new context within the VM
	defer v8ctx.Close()

	_, err := script.compiled.Run(v8ctx)
	if err != nil {
		return nil, err
	}

	global := v8ctx.Global() // get the global object from the context

	// set global data
	for key, val := range data {
		global.Set(key, interfaceToValuer(v8ctx, val))
	}

	// add global object
	for objectName, template := range yao.objectTemplates {
		object, _ := template.NewInstance(v8ctx)
		global.Set(objectName, object)
	}

	jsArgs := arrayToValuers(v8ctx, args)
	value, err := global.MethodCall(method, jsArgs...)
	if err != nil {
		return nil, fmt.Errorf("function %s.%s %s", name, method, err.Error())
	}

	if value.IsPromise() { // wait for the promise to resolve
		promise, err := value.AsPromise()
		if err != nil {
			return nil, fmt.Errorf("function execute error. %s.%s %s", name, method, err.Error())
		}
		for promise.State() == v8.Pending {
			continue
		}
		value = promise.Result()
	}

	return valueToInterface(value)
}

// Has check the given name of the script is load
func (yao *Yao) Has(name string) bool {
	_, has := yao.scripts[name]
	return has
}

// AddFunction add a global function
func (yao *Yao) AddFunction(name string, fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) error {
	jsFun := yao.goFunTemplate(fn)
	yao.template.Set(name, jsFun)
	return nil
}

func (yao *Yao) goFunTemplate(fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *v8.FunctionTemplate {
	return v8.NewFunctionTemplate(yao.iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
		global := map[string]interface{}{}
		jsGlobal, _ := info.Context().Global().Get("__yao_global")
		anyGlobal, _ := valueToInterface(jsGlobal)
		if iGlobal, ok := anyGlobal.(map[string]interface{}); ok {
			global = iGlobal
		}
		jsSid, _ := info.Context().Global().Get("__yao_sid")
		args := valuesToArray(info.Args())
		res := fn(global, jsSid.String(), args...)
		return interfaceToValue(info.Context(), res)
	})
}

// AddObject add a global object
func (yao *Yao) AddObject(name string, methods map[string]func(global map[string]interface{}, sid string, args ...interface{}) interface{}) error {
	object := v8.NewObjectTemplate(yao.iso)
	for method, fn := range methods {
		jsFun := yao.goFunTemplate(fn)
		object.Set(method, jsFun)
	}
	yao.objectTemplates[name] = object
	return nil
}

func (yao *Yao) jsLog(info *v8.FunctionCallbackInfo) *v8.Value {
	values := valuesToArray(info.Args())
	utils.Dump(values)
	return nil
}

func (yao *Yao) jsFetch(info *v8.FunctionCallbackInfo) *v8.Value {
	args := info.Args()
	url := args[0].String()
	resolver, _ := v8.NewPromiseResolver(info.Context())
	go func() {
		res, _ := http.Get(url)
		body, _ := ioutil.ReadAll(res.Body)
		val, _ := v8.NewValue(yao.iso, string(body))
		resolver.Resolve(val)
	}()
	return resolver.GetPromise().Value
}

func valueToInterface(value *v8.Value) (interface{}, error) {
	var v interface{} = nil
	if value.IsNull() || value.IsUndefined() {
		return nil, nil
	}

	content, err := value.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = jsoniter.Unmarshal([]byte(content), &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func interfaceToValuer(ctx *v8.Context, value interface{}) v8.Valuer {
	var valuer v8.Valuer
	if value == nil {
		valuer, _ = v8.NewValue(ctx.Isolate(), value)
		return valuer
	}
	v, _ := jsoniter.Marshal(value)
	valuer, _ = v8.JSONParse(ctx, string(v))
	return valuer
}

func interfaceToValue(ctx *v8.Context, value interface{}) *v8.Value {
	var valuer *v8.Value
	if value == nil {
		valuer, _ = v8.NewValue(ctx.Isolate(), value)
		return valuer
	}
	v, _ := jsoniter.Marshal(value)
	valuer, _ = v8.JSONParse(ctx, string(v))
	return valuer
}

func valuesToArray(values []*v8.Value) []interface{} {
	res := []interface{}{}
	for i := range values {
		var v interface{} = nil
		if values[i].IsNull() || values[i].IsUndefined() {
			res = append(res, nil)
			continue
		}

		content, _ := values[i].MarshalJSON()
		jsoniter.Unmarshal([]byte(content), &v)
		res = append(res, v)
	}
	return res
}

func arrayToValuers(ctx *v8.Context, values []interface{}) []v8.Valuer {
	res := []v8.Valuer{}
	for i := range values {
		value, _ := jsoniter.Marshal(values[i])
		valuer, _ := v8.JSONParse(ctx, string(value))
		res = append(res, valuer)
	}
	return res
}
