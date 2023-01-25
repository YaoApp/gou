package v8

import (
	"time"

	"github.com/yaoapp/gou/application"
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

// NewContent create a new content
func (script *Script) NewContent(sid string, global map[string]interface{}) (*Context, error) {

	timeout := script.Timeout
	if timeout == 0 {
		timeout = 100 * time.Millisecond
	}

	iso, err := SelectIso(timeout)
	if err != nil {
		return nil, err
	}

	ctx := v8go.NewContext(iso)
	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{CachedData: script.Cache}) // compile script in new isolate with cached data
	if err != nil {
		return nil, err
	}

	_, err = instance.Run(ctx)
	if err != nil {
		return nil, err
	}

	return &Context{
		Context: ctx,
		SID:     sid,
		Data:    global,
		Iso:     iso,
	}, nil
}

// Compile the javascript
func (script *Script) Compile(timeout time.Duration) error {

	source, err := application.App.Read(script.File)
	if err != nil {
		return err
	}

	if timeout == 0 {
		timeout = time.Second * 5
	}

	iso, err := SelectIso(timeout)
	if err != nil {
		return err
	}
	defer iso.Unlock()

	ctx := v8go.NewContext(iso)
	defer ctx.Close()

	script.Source = string(source)

	data, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		return err
	}

	script.Cache = data.CreateCodeCache()
	return nil
}
