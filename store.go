package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/exception"
)

// LoadStore load kv store
func LoadStore(source string, name string) (store.Store, error) {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	config, err := helper.ReadFile(input)
	if err != nil {
		return nil, err
	}

	inst := Store{}
	err = jsoniter.Unmarshal(config, &inst)
	if err != nil {
		return nil, err
	}

	typ := strings.ToLower(inst.Type)
	if typ == "lru" {
		stor, err := store.New(nil, inst.Option)
		if err != nil {
			return nil, err
		}
		store.Pools[name] = stor
		return store.Pools[name], nil
	}

	connector, has := connector.Connectors[inst.Connector]
	if !has {
		return nil, fmt.Errorf("Store %s Connector:%s was not loaded", name, inst.Connector)
	}

	stor, err := store.New(connector, inst.Option)
	if err != nil {
		return nil, err
	}

	store.Pools[name] = stor
	return store.Pools[name], nil
}

// SelectStore Select loaded kv store
func SelectStore(name string) store.Store {
	store, has := store.Pools[name]
	if !has {
		exception.New("Store:%s does not load", 500, name).Throw()
	}
	return store
}
