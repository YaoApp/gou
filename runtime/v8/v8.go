package v8

import (
	"fmt"
)

// Load load the script
func Load(file string, id string) (*Script, error) {
	script := NewScript(file, id)

	Scripts[id] = script
	return script, nil
}

// LoadRoot load the script with root privileges
func LoadRoot(file string, id string) (*Script, error) {
	script := NewScript(file, id)
	RootScripts[id] = script
	return script, nil
}

// Select a warm intance
func Select(id string) (*Script, error) {
	script, has := Scripts[id]
	if !has {
		return nil, fmt.Errorf("[V8] script %s does not exist", id)
	}
	return script, nil
}
