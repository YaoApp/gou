package store

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/gou/store/mongo"
	"github.com/yaoapp/gou/store/redis"
)

// Pools LRU pools
var Pools = map[string]Store{}

// New create a store via connector
func New(c connector.Connector, option Option) (Store, error) {

	if c == nil {
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
