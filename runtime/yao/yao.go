package yao

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/utils"
	v8 "rogchap.com/v8go"
)

// New create a pure ES6 engine (v8)
func New(numOfContexts int) *Yao {
	iso := v8.NewIsolate()
	global := v8.NewObjectTemplate(iso)

	yao := &Yao{
		iso:             iso,
		template:        global,
		scripts:         map[string]script{},
		objectTemplates: map[string]*v8.ObjectTemplate{},
		numOfContexts:   numOfContexts,
	}

	yao.template.Set("fetch", v8.NewFunctionTemplate(yao.iso, yao.jsFetch))
	return yao
}

// Init initialize (for next version )
func (yao *Yao) Init() error {
	// yao.contexts = NewPool(yao.numOfContexts)
	// yao.ctx = v8.NewContext(yao.iso, yao.template)
	// for i := 0; i < yao.numOfContexts; i++ {
	// 	yao.contexts.Push(v8.NewContext(yao.iso, yao.template))
	// }
	// go yao.contexts.Prepare(200, func() *v8.Context { return v8.NewContext(yao.iso, yao.template) })
	return nil
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

	// Compile
	code := string(source)
	// compiled, err := yao.iso.CompileUnboundScript(code, scriptfile, v8.CompileOptions{}) // compile script to get cached data
	// if err != nil {
	// 	return err
	// }

	// opt
	ctx := v8.NewContext(yao.iso, yao.template)
	ctx.RunScript(code, scriptfile)

	yao.scripts[name] = script{
		name:     name,
		filename: scriptfile,
		source:   code,
		ctx:      ctx,
		// compiled: compiled,
	}
	return nil
}

// Call cal javascript function
func (yao *Yao) Call(data map[string]interface{}, name string, method string, args ...interface{}) (interface{}, error) {
	script, has := yao.scripts[name]
	if !has {
		return nil, fmt.Errorf("The %s does not loaded (%d)", name, len(yao.scripts))
	}

	var err error
	v8ctx := script.ctx
	// v8ctx, err := yao.contexts.Make(func() *v8.Context { return v8.NewContext(yao.iso, yao.template) })
	// if err != nil {
	// 	return nil, err
	// }
	// defer func() { v8ctx.Close() }()
	// v8ctx := yao.ctx

	// _, err = v8ctx.RunScript(script.source, script.filename)
	// // _, err = script.compiled.Run(v8ctx)
	// if err != nil {
	// 	return nil, err
	// }

	global := v8ctx.Global() // get the global object from the context
	if global == nil {
		return nil, fmt.Errorf("global is nil")
	}

	// set global data
	for key, val := range data {
		global.Set(key, MustAnyToValue(v8ctx, val))
	}

	// add global object
	for objectName, template := range yao.objectTemplates {
		object, _ := template.NewInstance(v8ctx)
		global.Set(objectName, object)
	}

	if !global.Has(method) {
		return nil, fmt.Errorf("global %s", method)
	}

	jsArgs, err := ToValuers(v8ctx, args)
	if err != nil {
		return nil, fmt.Errorf("function %s.%s %s", name, method, err.Error())
	}

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

	return ToInterface(value)
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
		anyGlobal, _ := ToInterface(jsGlobal)
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

// ToInterface Convert *v8.Value to Interface
func ToInterface(value *v8.Value) (interface{}, error) {

	if value == nil {
		return nil, nil
	}

	var v interface{} = nil
	if value.IsNull() || value.IsUndefined() {
		return nil, nil
	}

	// fmt.Println("ToInterface:", value.IsArray())

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

// MustAnyToValue Convert any to *v8.Value
func MustAnyToValue(ctx *v8.Context, value interface{}) *v8.Value {
	v, err := AnyToValue(ctx, value)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return v
}

// AnyToValue Convert data to *v8.Value
func AnyToValue(ctx *v8.Context, value interface{}) (*v8.Value, error) {

	switch value.(type) {
	case []byte:
		// Todo: []byte to Uint8Array
		return v8.NewValue(ctx.Isolate(), string(value.([]byte)))
	case string, int32, uint32, int64, uint64, bool, float64, *big.Int:
		return v8.NewValue(ctx.Isolate(), value)
	}

	v, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("AnyToValue error: %s", err)
	}

	return v8.JSONParse(ctx, string(v))
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

// ToValuers Convert any to []v8.Valuer
func ToValuers(ctx *v8.Context, values []interface{}) ([]v8.Valuer, error) {
	res := []v8.Valuer{}
	if ctx == nil {
		return res, fmt.Errorf("Context is nil")
	}

	for i := range values {
		value, err := AnyToValue(ctx, values[i])
		if err != nil {
			log.Error("AnyToValue error: %s", err)
			value, _ = v8.NewValue(ctx.Isolate(), nil)
		}
		res = append(res, value)
	}
	return res, nil
}
