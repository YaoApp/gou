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
	"github.com/yaoapp/gou/runtime/yao/objects"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/utils"
	"rogchap.com/v8go"
)

// New create a pure ES6 engine (v8go)
func New(numOfContexts int) *Yao {
	iso := v8go.NewIsolate()
	global := v8go.NewObjectTemplate(iso)

	yao := &Yao{
		iso:             iso,
		template:        global,
		scripts:         map[string]script{},
		objectTemplates: map[string]*v8go.ObjectTemplate{},
		numOfContexts:   numOfContexts,
	}

	yao.template.Set("fetch", v8go.NewFunctionTemplate(yao.iso, yao.jsFetch))
	yao.AddFunctionTemplates(map[string]*v8go.FunctionTemplate{
		"Exception": objects.NewException().ExportFunction(yao.iso),
		"WebSocket": objects.NewWebSocket().ExportFunction(yao.iso),
		"Store":     objects.NewStore().ExportFunction(yao.iso),
	})
	return yao
}

// Init initialize (for next version )
func (yao *Yao) Init() error {
	// yao.contexts = NewPool(yao.numOfContexts)
	// yao.ctx = v8go.NewContext(yao.iso, yao.template)
	// for i := 0; i < yao.numOfContexts; i++ {
	// 	yao.contexts.Push(v8go.NewContext(yao.iso, yao.template))
	// }
	// go yao.contexts.Prepare(200, func() *v8go.Context { return v8go.NewContext(yao.iso, yao.template) })
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
	// compiled, err := yao.iso.CompileUnboundScript(code, scriptfile, v8go.CompileOptions{}) // compile script to get cached data
	// if err != nil {
	// 	return err
	// }

	// opt
	ctx := v8go.NewContext(yao.iso, yao.template)
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
	// v8ctx, err := yao.contexts.Make(func() *v8go.Context { return v8go.NewContext(yao.iso, yao.template) })
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
		log.Error("function %s.%s %s", name, method, err.Error())
		return nil, err
	}

	value, err := global.MethodCall(method, jsArgs...)
	if err != nil {
		log.Error("function %s.%s %s", name, method, err.Error())
		return nil, err
	}

	if value.IsPromise() { // wait for the promise to resolve
		promise, err := value.AsPromise()
		if err != nil {
			log.Error("function execute error. %s.%s %s", name, method, err.Error())
			return nil, err
		}
		for promise.State() == v8go.Pending {
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

func (yao *Yao) goFunTemplate(fn func(global map[string]interface{}, sid string, args ...interface{}) interface{}) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(yao.iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
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
	object := v8go.NewObjectTemplate(yao.iso)
	for method, fn := range methods {
		jsFun := yao.goFunTemplate(fn)
		object.Set(method, jsFun)
	}
	yao.objectTemplates[name] = object
	return nil
}

// AddFunctionTemplates add function templates to global
func (yao *Yao) AddFunctionTemplates(tmpls map[string]*v8go.FunctionTemplate) error {
	for name, tmpl := range tmpls {
		yao.template.Set(name, tmpl)
	}
	return nil
}

func (yao *Yao) jsLog(info *v8go.FunctionCallbackInfo) *v8go.Value {
	values := valuesToArray(info.Args())
	utils.Dump(values)
	return nil
}

func (yao *Yao) jsFetch(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	url := args[0].String()
	resolver, _ := v8go.NewPromiseResolver(info.Context())
	go func() {
		res, _ := http.Get(url)
		body, _ := ioutil.ReadAll(res.Body)
		val, _ := v8go.NewValue(yao.iso, string(body))
		resolver.Resolve(val)
	}()
	return resolver.GetPromise().Value
}

// ToInterface Convert *v8go.Value to Interface
func ToInterface(value *v8go.Value) (interface{}, error) {

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

// MustAnyToValue Convert any to *v8go.Value
func MustAnyToValue(ctx *v8go.Context, value interface{}) *v8go.Value {
	v, err := AnyToValue(ctx, value)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return v
}

// AnyToValue Convert data to *v8go.Value
func AnyToValue(ctx *v8go.Context, value interface{}) (*v8go.Value, error) {

	switch value.(type) {
	case []byte:
		// Todo: []byte to Uint8Array
		return v8go.NewValue(ctx.Isolate(), string(value.([]byte)))
	case string, int32, uint32, int64, uint64, bool, float64, *big.Int:
		return v8go.NewValue(ctx.Isolate(), value)
	}

	v, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("AnyToValue error: %s", err)
	}

	return v8go.JSONParse(ctx, string(v))
}

func interfaceToValue(ctx *v8go.Context, value interface{}) *v8go.Value {
	var valuer *v8go.Value
	if value == nil {
		valuer, _ = v8go.NewValue(ctx.Isolate(), value)
		return valuer
	}
	v, _ := jsoniter.Marshal(value)
	valuer, _ = v8go.JSONParse(ctx, string(v))
	return valuer
}

func valuesToArray(values []*v8go.Value) []interface{} {
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

// ToValuers Convert any to []v8go.Valuer
func ToValuers(ctx *v8go.Context, values []interface{}) ([]v8go.Valuer, error) {
	res := []v8go.Valuer{}
	if ctx == nil {
		return res, fmt.Errorf("Context is nil")
	}

	for i := range values {
		value, err := AnyToValue(ctx, values[i])
		if err != nil {
			log.Error("AnyToValue error: %s", err)
			value, _ = v8go.NewValue(ctx.Isolate(), nil)
		}
		res = append(res, value)
	}
	return res, nil
}
