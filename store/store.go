package store

import (
	"fmt"
	"strings"
	"sync"

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
var rwlock sync.RWMutex // Use RWMutex for better concurrency

// LoadSync load store sync
func LoadSync(file string, name string) (Store, error) {
	rwlock.Lock()
	defer rwlock.Unlock()
	return Load(file, name)
}

// LoadSourceSync load store from source sync
func LoadSourceSync(data []byte, id string, file string) (Store, error) {
	rwlock.Lock()
	defer rwlock.Unlock()
	return LoadSource(data, id, file)
}

// Load load kv store
func Load(file string, name string) (Store, error) {
	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return LoadSource(data, name, file)
}

// LoadSource load store from source
func LoadSource(data []byte, id string, file string) (Store, error) {
	inst := Instance{}
	err := application.Parse(file, data, &inst)
	if err != nil {
		return nil, err
	}

	typ := strings.ToLower(inst.Type)
	if typ == "lru" || typ == "badger" {
		stor, err := New(nil, inst.Option)
		if err != nil {
			return nil, err
		}
		Pools[id] = stor
		return Pools[id], nil
	}

	connector, has := connector.Connectors[inst.Connector]
	if !has {
		return nil, fmt.Errorf("Store %s Connector:%s was not loaded", id, inst.Connector)
	}

	stor, err := New(connector, inst.Option)
	if err != nil {
		return nil, err
	}

	Pools[id] = stor
	return Pools[id], nil
}

// Select Select loaded kv store
func Select(name string) Store {
	store, has := Pools[name]
	if !has {
		exception.New("Store:%s does not load", 500, name).Throw()
	}
	return store
}

// Get Get the store from the pool
func Get(name string) (Store, error) {
	store, has := Pools[name]
	if !has {
		return nil, fmt.Errorf("Store:%s does not load", name)
	}
	return store, nil
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
