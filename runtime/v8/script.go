package v8

import (
	"fmt"
	"sync"
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
		Context: sync.Map{},
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

	ctx, ok := script.Context.Load(iso)
	if !ok {
		return nil, fmt.Errorf("[v8] get content error")
	}

	return &Context{
		Context: ctx.(*v8go.Context),
		SID:     sid,
		Data:    global,
		Iso:     iso,
	}, nil

}

// Compile the javascript
func (script *Script) Compile(iso *Isolate, timeout time.Duration) error {

	source, err := application.App.Read(script.File)
	if err != nil {
		return err
	}

	if timeout == 0 {
		timeout = time.Second * 5
	}

	ctx := v8go.NewContext(iso)
	script.Source = string(source)

	instance, err := iso.CompileUnboundScript(script.Source, script.File, v8go.CompileOptions{})
	if err != nil {
		return err
	}

	_, err = instance.Run(ctx)
	if err != nil {
		return err
	}

	script.Context.Store(iso, ctx)
	return nil
}
