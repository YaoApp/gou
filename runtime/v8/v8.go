package v8

import (
	"fmt"

	"github.com/yaoapp/gou/application"
)

var runtimeOption = &Option{}

// Start v8 runtime
func Start(option *Option) error {
	option.Validate()
	runtimeOption = option
	chIsoReady = make(chan *Isolate, option.MaxSize)
	for i := 0; i < option.MinSize; i++ {
		_, err := NewIsolate()
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop v8 runtime
func Stop() {
	chIsoReady = make(chan *Isolate, runtimeOption.MaxSize)
	// Remove iso
	isolates.Range(func(iso *Isolate) bool {
		isolates.Remove(iso)
		return true
	})
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
		return nil, fmt.Errorf("script %s not exists", id)
	}

	return script, nil
}
