package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/redis"
	"github.com/yaoapp/kun/log"
)

// New create a new store via connector
func New(c connector.Connector) (*Store, error) {
	redis, ok := c.(*redis.Connector)
	if !ok {
		return nil, fmt.Errorf("the connector was not a *redis.Connector")
	}
	return &Store{rdb: redis.Rdb, Option: Option{Prefix: fmt.Sprintf("%s:", redis.Name)}}, nil
}

// Get looks up a key's value from the store.
func (store *Store) Get(key string) (value interface{}, ok bool) {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)
	val, err := store.rdb.Get(context.Background(), key).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "nil") {
			log.Error("Store redis Get %s: %s", key, err.Error())
		}
		return nil, false
	}

	err = jsoniter.Unmarshal([]byte(val), &value)
	if err != nil {
		log.Error("Store redis Get %s: %s val: %s", key, err.Error(), val)
		return nil, false
	}

	return value, true
}

// Set adds a value to the store.
func (store *Store) Set(key string, value interface{}, ttl time.Duration) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("Store redis Set %s: %s", key, err.Error())
		return err
	}

	err = store.rdb.Set(context.Background(), key, bytes, ttl).Err()
	if err != nil {
		log.Error("Store redis Set %s: %s", key, err.Error())
		return err
	}
	return nil
}

// Del remove is used to purge a key from the store
func (store *Store) Del(key string) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)
	err := store.rdb.Del(context.Background(), key).Err()
	if err != nil {
		return err
	}
	return nil
}

// Has check if the store is exist ( without updating recency or frequency )
func (store *Store) Has(key string) bool {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)
	cmd := store.rdb.Exists(context.Background(), key)
	v, _ := cmd.Result()
	return int(v) == 1
}

// Len returns the number of stored entries (**not O(1)**)
func (store *Store) Len() int {
	script := fmt.Sprintf("return #redis.pcall('keys', '%s*')", store.Option.Prefix)
	cmd := store.rdb.Eval(context.Background(), script, []string{})
	v, err := cmd.Result()
	if err != nil {
		log.Error("Store redis Len: %s", err.Error())
		return 0
	}

	if int64v, ok := v.(int64); ok {
		return int(int64v)
	}

	intv, err := strconv.Atoi(fmt.Sprintf("%s", v))
	if err != nil {
		log.Error("Store redis Len: %s", err.Error())
		return 0
	}
	return intv
}

// Keys returns all the cached keys
func (store *Store) Keys() []string {
	prefix := store.Option.Prefix
	keys, err := store.rdb.Keys(context.Background(), prefix+"*").Result()
	if err != nil {
		log.Error("Store redis Keys:%s", err.Error())
		return []string{}
	}

	for i := range keys {
		keys[i] = strings.TrimPrefix(keys[i], prefix)
	}

	return keys
}

// Clear is used to clear the cache
func (store *Store) Clear() {
	keys := store.Keys()
	for _, key := range keys {
		store.Del(key)
	}
}

// GetSet looks up a key's value from the cache. if does not exist add to the cache
func (store *Store) GetSet(key string, ttl time.Duration, getValue func(key string) (interface{}, error)) (interface{}, error) {
	value, ok := store.Get(key)
	if !ok {
		var err error
		value, err = getValue(key)
		if err != nil {
			return nil, err
		}
		store.Set(key, value, ttl)
	}
	return value, nil
}

// GetDel looks up a key's value from the cache, then remove it.
func (store *Store) GetDel(key string) (value interface{}, ok bool) {
	value, ok = store.Get(key)
	if !ok {
		return nil, false
	}
	err := store.Del(key)
	if err != nil {
		return value, false
	}
	return value, true
}

// GetMulti mulit get values
func (store *Store) GetMulti(keys []string) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, _ := store.Get(key)
		values[key] = value
	}
	return values
}

// SetMulti mulit set values
func (store *Store) SetMulti(values map[string]interface{}, ttl time.Duration) {
	for key, value := range values {
		store.Set(key, value, ttl)
	}
}

// DelMulti mulit remove values
func (store *Store) DelMulti(keys []string) {
	for _, key := range keys {
		store.Del(key)
	}
}

// GetSetMulti mulit get values, if does not exist add to the cache
func (store *Store) GetSetMulti(keys []string, ttl time.Duration, getValue func(key string) (interface{}, error)) map[string]interface{} {
	values := map[string]interface{}{}
	for _, key := range keys {
		value, ok := store.Get(key)
		if !ok {
			var err error
			value, err = getValue(key)
			if err != nil {
				log.Error("GetSetMulti Set %s: %s", key, err.Error())
			}
		}
		values[key] = value
	}
	return values
}
