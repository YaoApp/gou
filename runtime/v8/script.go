package v8

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/application"
)

// Scripts loaded scripts
var Scripts = map[string]*Script{}

// RootScripts the scripts for studio
var RootScripts = map[string]*Script{}

var importRe = regexp.MustCompile(`import\s+.*;`)

// NewScript create a new script
func NewScript(file string, id string, timeout ...time.Duration) *Script {

	t := time.Duration(0)
	if len(timeout) > 0 {
		t = timeout[0]
	}

	return &Script{
		ID:      id,
		File:    file,
		Timeout: t,
	}
}

// Load load the script
func Load(file string, id string) (*Script, error) {
	script := NewScript(file, id)
	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(file, ".ts") {
		source, err = TransformTS(source)
		if err != nil {
			return nil, err
		}
	}

	script.Source = string(source)
	script.Root = false
	Scripts[id] = script
	return script, nil
}

// LoadRoot load the script with root privileges
func LoadRoot(file string, id string) (*Script, error) {
	script := NewScript(file, id)
	source, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(file, ".ts") {
		source, err = TransformTS(source)
		if err != nil {
			return nil, err
		}
	}

	script.Source = string(source)
	script.Root = true
	RootScripts[id] = script
	return script, nil
}

// TransformTS transform the typescript
func TransformTS(source []byte) ([]byte, error) {

	// @todo import supoort
	jsCode := importRe.ReplaceAllString(string(source), "")
	result := api.Transform(jsCode, api.TransformOptions{
		Loader: api.LoaderTS,
		Target: api.ESNext,
	})

	if len(result.Errors) > 0 {
		errors := []string{}
		for _, err := range result.Errors {
			errors = append(errors, fmt.Sprintf("%s", err.Text))
		}
		return nil, fmt.Errorf("transform ts code error: %v", strings.Join(errors, "\n"))
	}

	return result.Code, nil
}

// Transform the javascript
func Transform(source string, globalName string) string {
	result := api.Transform(source, api.TransformOptions{
		Loader:     api.LoaderJS,
		Format:     api.FormatIIFE,
		GlobalName: globalName,
	})
	return string(result.Code)
}

// Select a script
func Select(id string) (*Script, error) {
	script, has := Scripts[id]
	if !has {
		return nil, fmt.Errorf("script %s not exists", id)
	}
	return script, nil
}

// SelectRoot a script with root privileges
func SelectRoot(id string) (*Script, error) {

	script, has := RootScripts[id]
	if has {
		return script, nil
	}

	script, has = Scripts[id]
	if !has {
		return nil, fmt.Errorf("script(root) %s not exists", id)
	}

	return script, nil
}

// NewContext create a new context
func (script *Script) NewContext(sid string, global map[string]interface{}) (*Context, error) {

	timeout := script.Timeout
	if timeout == 0 {
		timeout = time.Duration(runtimeOption.ContextTimeout) * time.Millisecond
	}

	iso, err := SelectIso(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		return nil, err
	}

	context, err := iso.SelectContext(script, timeout)
	if err != nil {
		return nil, err
	}

	return &Context{
		ID:      script.ID,
		Context: context,
		SID:     sid,
		Data:    global,
		Root:    script.Root,
		Iso:     iso,
	}, nil
}

// Compile the javascript
// func (script *Script) Compile(iso *Isolate, timeout time.Duration) (*v8go.Context, error) {

// 	if iso.Isolate == nil {
// 		return nil, fmt.Errorf("isolate was removed")
// 	}

// 	if timeout == 0 {
// 		timeout = time.Second * 5
// 	}

// 	ctx := v8go.NewContext(iso.Isolate, iso.template)
// 	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// console.log("foo", "bar", 1, 2, 3, 4)
// 	err = console.New().Set("console", ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	_, err = instance.Run(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// iso.contexts[script] = ctx // cache
// 	return ctx, nil
// }

// debug : debug the script
// func (script *Script) debug(sid string, data map[string]interface{}, method string, args ...interface{}) (interface{}, error) {

// 	timeout := script.Timeout
// 	if timeout == 0 {
// 		timeout = 100 * time.Millisecond
// 	}

// 	iso, err := SelectIso(timeout)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer iso.Unlock()

// 	ctx := v8go.NewContext(iso.Isolate, iso.template)
// 	defer ctx.Close()

// 	instance, err := iso.Isolate.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
// 	if err != nil {
// 		return nil, err
// 	}

// 	_, err = instance.Run(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	global := ctx.Global()

// 	jsArgs, err := bridge.JsValues(ctx, args)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s.%s %s", script.ID, method, err.Error())
// 	}
// 	defer bridge.FreeJsValues(jsArgs)

// 	jsData, err := bridge.JsValue(ctx, map[string]interface{}{
// 		"SID":  sid,
// 		"ROOT": script.Root,
// 		"DATA": data,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer func() {
// 		if !jsData.IsNull() && !jsData.IsUndefined() {
// 			jsData.Release()
// 		}
// 	}()

// 	err = global.Set("__yao_data", jsData)
// 	if err != nil {
// 		return nil, err
// 	}

// 	res, err := global.MethodCall(method, bridge.Valuers(jsArgs)...)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s.%s %+v", script.ID, method, err)
// 	}

// 	goRes, err := bridge.GoValue(res, ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s.%s %s", script.ID, method, err.Error())
// 	}

// 	return goRes, nil
// }
