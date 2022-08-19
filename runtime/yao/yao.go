package yao

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/lang"
	"github.com/yaoapp/gou/runtime/yao/bridge"
	"github.com/yaoapp/gou/runtime/yao/objects"
	"github.com/yaoapp/gou/runtime/yao/values"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
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

	yao.template.Set("LL", v8.NewFunctionTemplate(yao.iso, yao.jsLang))
	yao.AddObjectTemplate("log", objects.NewLog().ExportObject(yao.iso))
	yao.AddFunctionTemplates(map[string]*v8.FunctionTemplate{
		"Exception": objects.NewException().ExportFunction(yao.iso),
		"WebSocket": objects.NewWebSocket().ExportFunction(yao.iso),
		"Store":     objects.NewStore().ExportFunction(yao.iso),
		"Query":     objects.NewQuery().ExportFunction(yao.iso),
	})
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
		log.Error("[Runtime] load %s %s error: %s", filename, name, err.Error())
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
	_, err = ctx.RunScript(code, scriptfile)
	if err != nil {
		return err
	}

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
	if data == nil {
		data = map[string]interface{}{}
	}

	for key, val := range data {
		global.Set(key, bridge.MustAnyToValue(v8ctx, val))
	}

	// add global object
	for objectName, template := range yao.objectTemplates {
		object, _ := template.NewInstance(v8ctx)
		global.Set(objectName, object)
	}

	if !global.Has(method) {
		return nil, fmt.Errorf("function %s does not exists", method)
	}

	jsArgs, err := bridge.ArrayToValuers(v8ctx, args)
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
		for promise.State() == v8.Pending {
			continue
		}
		value = promise.Result()
	}

	return bridge.ToInterface(value)
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
		anyGlobal, _ := bridge.ToInterface(jsGlobal)
		if iGlobal, ok := anyGlobal.(map[string]interface{}); ok {
			global = iGlobal
		}
		jsSid, _ := info.Context().Global().Get("__yao_sid")
		args := bridge.ValuesToArray(info.Args())
		res := fn(global, jsSid.String(), args...)
		return bridge.MustAnyToValue(info.Context(), res)
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

// AddObjectTemplate add a global object template
func (yao *Yao) AddObjectTemplate(name string, object *v8go.ObjectTemplate) error {
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

func (yao *Yao) jsLang(info *v8.FunctionCallbackInfo) *v8.Value {
	args := info.Args()
	if len(args) == 0 {
		return v8.Undefined(info.Context().Isolate())
	}

	if !args[0].IsString() {
		return args[0]
	}

	value := args[0].String()
	lang.Replace(&value)

	iso := info.Context().Isolate()
	v, err := v8.NewValue(iso, value)
	if err != nil {
		return iso.ThrowException(values.Error(info.Context(), err.Error()))
	}

	return v
}

// func (yao *Yao) jsLog(info *v8.FunctionCallbackInfo) *v8.Value {
// 	values := bridge.ValuesToArray(info.Args())
// 	utils.Dump(values)
// 	return nil
// }

// func (yao *Yao) jsFetch(info *v8.FunctionCallbackInfo) *v8.Value {
// 	args := info.Args()
// 	url := args[0].String()
// 	resolver, _ := v8.NewPromiseResolver(info.Context())
// 	go func() {
// 		res, _ := http.Get(url)
// 		if res.Body == nil {
// 			resolver.Resolve(v8.Undefined(yao.iso))
// 			return
// 		}
// 		body, _ := ioutil.ReadAll(res.Body)
// 		val, _ := v8.NewValue(yao.iso, string(body))
// 		resolver.Resolve(val)
// 	}()
// 	return resolver.GetPromise().Value
// }
