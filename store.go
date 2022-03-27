package gou

import (
	"fmt"
	"io"
	"os"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/kv"
	"github.com/yaoapp/gou/kv/lru"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
)

// LoadStore load kv store
func LoadStore(source string, name string) (kv.Store, error) {
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

	store := Store{}
	err = jsoniter.Unmarshal(config, &store)
	if err != nil {
		return nil, err
	}

	typ := strings.ToLower(store.Type)
	switch typ {
	case "lru":
		s := 1024
		if size, ok := store.Option["size"]; ok {
			switch size.(type) {
			case string:
				size = os.Getenv(size.(string))
			}
			s = any.Of(size).CInt()
			if s == 0 {
				s = 1024
			}
		}
		cache, err := lru.New(s)
		if err != nil {
			return nil, err
		}
		kv.Pools[name] = cache
		return cache, err

	default:
		return nil, fmt.Errorf("Store %s Type:%s does not support yet", name, typ)
	}
}

// SelectStore Select loaded kv store
func SelectStore(name string) kv.Store {
	store, has := kv.Pools[name]
	if !has {
		exception.New("Store:%s does not load", 500, name).Throw()
	}
	return store
}
