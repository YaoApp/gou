package query

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/query/share"
)

// Engines registered engines
var Engines = map[string]share.DSL{}

// Register Register a Query Engine
func Register(name string, engine share.DSL) {
	name = strings.ToLower(name)
	Engines[name] = engine
}

// Unregister Unregister a Query Engine
func Unregister(name string) {
	name = strings.ToLower(name)
	delete(Engines, name)
}

// Alias set the Engine alias
func Alias(name, alias string) {
	name = strings.ToLower(name)
	alias = strings.ToLower(alias)
	if _, has := Engines[name]; has {
		Engines[alias] = Engines[name]
	}
}

// Select choose one engine
func Select(name string) (share.DSL, error) {
	engine, has := Engines[name]
	if !has {
		return nil, fmt.Errorf("%s not found", name)
	}
	return engine, nil
}
