package store

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/exception"
)

// Load load kv store
func Load(file string, name string) (Store, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	inst := Instance{}
	err = application.Parse(file, data, &inst)
	if err != nil {
		return nil, err
	}

	typ := strings.ToLower(inst.Type)
	if typ == "lru" {
		stor, err := New(nil, inst.Option)
		if err != nil {
			return nil, err
		}
		Pools[name] = stor
		return Pools[name], nil
	}

	connector, has := connector.Connectors[inst.Connector]
	if !has {
		return nil, fmt.Errorf("Store %s Connector:%s was not loaded", name, inst.Connector)
	}

	stor, err := New(connector, inst.Option)
	if err != nil {
		return nil, err
	}

	Pools[name] = stor
	return Pools[name], nil
}

// SelectStore Select loaded kv store
func SelectStore(name string) Store {
	store, has := Pools[name]
	if !has {
		exception.New("Store:%s does not load", 500, name).Throw()
	}
	return store
}
