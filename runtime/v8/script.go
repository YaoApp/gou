package v8

import (
	"fmt"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/runtime/v8/objects/console"
	"rogchap.com/v8go"
)

// Scripts loaded scripts
var Scripts = map[string]*Script{}

// RootScripts the scripts for studio
var RootScripts = map[string]*Script{}

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
	script.Source = string(source)
	script.Root = true
	RootScripts[id] = script
	return script, nil
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

// Compile the javascript
func (script *Script) Compile(iso *Isolate, timeout time.Duration) (*v8go.Context, error) {

	if iso.Isolate == nil {
		return nil, fmt.Errorf("isolate was removed")
	}

	if timeout == 0 {
		timeout = time.Second * 5
	}

	ctx := v8go.NewContext(iso.Isolate, iso.template)
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		return nil, err
	}

	// console.log("foo", "bar", 1, 2, 3, 4)
	err = console.New().Set("console", ctx)
	if err != nil {
		return nil, err
	}

	_, err = instance.Run(ctx)
	if err != nil {
		return nil, err
	}

	iso.contexts[script] = ctx // cache
	return ctx, nil
}
