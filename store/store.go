package store

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/store/badger"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/gou/store/mongo"
	"github.com/yaoapp/gou/store/redis"
	"github.com/yaoapp/kun/exception"
)

// Pools LRU pools
var Pools = map[string]Store{}

// Load load kv store
func Load(file string, name string) (Store, error) {

	// Check if store is already loaded
	if store, exists := Pools[name]; exists {
		return store, nil
	}

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
	if typ == "lru" || typ == "badger" {
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

// Select Select loaded kv store
func Select(name string) Store {
	store, has := Pools[name]
	if !has {
		exception.New("Store:%s does not load", 500, name).Throw()
	}
	return store
}

// New create a store via connector
func New(c connector.Connector, option Option) (Store, error) {

	if c == nil {
		// Check if this is a badger store request
		if option != nil {
			if path, has := option["path"]; has {
				// This is a badger store
				pathStr := helper.EnvString(path, "./data/badger")
				return badger.New(pathStr)
			}
		}

		// Default to LRU
		size := 10240
		if option != nil {
			if v, has := option["size"]; has {
				size = helper.EnvInt(v, 10240)
			}
		}
		return lru.New(size)
	}

	if c.Is(connector.REDIS) {
		return redis.New(c)
	} else if c.Is(connector.MONGO) {
		return mongo.New(c)
	}

	return nil, fmt.Errorf("the connector does not support")

}
