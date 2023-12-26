package v8

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
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

	if runtimeOption.Mode == "performance" {
		return nil, fmt.Errorf("performance mode is not supported yet")
	}

	iso, err := SelectIsoStandard(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		return nil, err
	}

	ctx := v8go.NewContext(iso, iso.Template)

	// Create instance of the script
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		return nil, fmt.Errorf("scripts.%s %s", script.ID, err.Error())
	}
	v, err := instance.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("scripts.%s %s", script.ID, err.Error())
	}
	defer v.Release()

	return &Context{
		ID:            script.ID,
		Sid:           sid,
		Data:          global,
		Root:          script.Root,
		Timeout:       timeout,
		Isolate:       iso,
		Context:       ctx,
		UnboundScript: instance,
	}, nil
}

// Exec execute the script
// the default mode is "standard" and the other value is "performance".
// the "standard" mode save memory but will run slower. can be used in most cases, especially in arm64 device.
// the "performance" mode need more memory but will run faster. can be used in high concurrency and large script.
func (script *Script) Exec(process *process.Process) interface{} {
	if runtimeOption.Mode == "performance" {
		return script.execPerformance(process)
	}
	return script.execStandard(process)
}

// execPerformance execute the script in performance mode
func (script *Script) execPerformance(process *process.Process) interface{} {

	iso, err := SelectIsoPerformance(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		return err
	}
	defer Unlock(iso)

	return "Performance Mode is not supported yet"

	// iso, ctx, err := MakeContext(script)
	// if err != nil {
	// 	exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
	// 	return nil
	// }
	// defer Unlock(iso)
	// defer ctx.Context.Close()

	// // Set the global data
	// global := ctx.Context.Global()
	// err = bridge.SetShareData(ctx.Context, global, &bridge.Share{
	// 	Sid:    process.Sid,
	// 	Root:   script.Root,
	// 	Global: process.Global,
	// })
	// if err != nil {
	// 	exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
	// 	return nil
	// }

	// // Run the method
	// jsArgs, err := bridge.JsValues(ctx.Context, process.Args)
	// if err != nil {
	// 	return fmt.Errorf("%s.%s %s", script.ID, process.Method, err.Error())
	// }
	// defer bridge.FreeJsValues(jsArgs)

	// jsRes, err := global.MethodCall(process.Method, bridge.Valuers(jsArgs)...)
	// if err != nil {
	// 	return fmt.Errorf("%s.%s %+v", script.ID, process.Method, err)
	// }

	// goRes, err := bridge.GoValue(jsRes, ctx.Context)
	// if err != nil {
	// 	return fmt.Errorf("%s.%s %s", script.ID, process.Method, err.Error())
	// }

	// return goRes
}

// execStandard execute the script in standard mode
func (script *Script) execStandard(process *process.Process) interface{} {

	iso, err := SelectIsoStandard(time.Duration(runtimeOption.DefaultTimeout) * time.Millisecond)
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}
	defer iso.Dispose()

	ctx := v8go.NewContext(iso, iso.Template)
	defer ctx.Close()

	// Next Version will support this, snapshot will be used in the next version
	// ctx, err := iso.Context()
	// if err != nil {
	// 	exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
	// 	return nil
	// }

	// Create instance of the script
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}
	v, err := instance.Run(ctx)
	if err != nil {
		return err
	}
	defer v.Release()

	// Set the global data
	global := ctx.Global()
	err = bridge.SetShareData(ctx, global, &bridge.Share{
		Sid:    process.Sid,
		Root:   script.Root,
		Global: process.Global,
	})
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New("scripts.%s.%s %s", 500, script.ID, process.Method, err.Error()).Throw()
		return nil
	}

	// Run the method
	jsArgs, err := bridge.JsValues(ctx, process.Args)
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil

	}
	defer bridge.FreeJsValues(jsArgs)

	jsRes, err := global.MethodCall(process.Method, bridge.Valuers(jsArgs)...)
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	goRes, err := bridge.GoValue(jsRes, ctx)
	if err != nil {
		log.Error("scripts.%s.%s %s", script.ID, process.Method, err.Error())
		exception.New(err.Error(), 500).Throw()
		return nil
	}

	return goRes
}

// ContextTimeout get the context timeout
func (script *Script) ContextTimeout() time.Duration {
	if script.Timeout > 0 {
		return script.Timeout
	}
	return time.Duration(runtimeOption.ContextTimeout) * time.Millisecond
}
