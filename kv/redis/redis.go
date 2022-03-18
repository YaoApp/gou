package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// New create a new LRU cache
func New(option *Option) (*Cache, error) {
	cache := &Cache{Option: option}
	rdb := redis.
		NewClient(cache.Option.Redis).
		WithTimeout(cache.Option.Timeout)
	cache.rdb = rdb
	return cache, nil
}

// Get looks up a key's value from the cache.
func (cache *Cache) Get(key string) (value interface{}, ok bool) {
	key = fmt.Sprintf("%s%s", cache.Option.Prefix, key)
	val, err := cache.rdb.Get(context.Background(), key).Result()
	if err != nil {
		log.Error("KV redis Get: %s", err.Error())
		return nil, false
	}

	err = jsoniter.Unmarshal([]byte(val), &value)
	if err != nil {
		log.Error("KV redis Get: %s val: %s", err.Error(), val)
		return nil, false
	}

	return value, true
}

// Set adds a value to the cache.
func (cache *Cache) Set(key string, value interface{}, expiration time.Duration) error {
	key = fmt.Sprintf("%s%s", cache.Option.Prefix, key)
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		return err
	}

	err = cache.rdb.Set(context.Background(), key, bytes, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

// Del remove is used to purge a key from the cache
func (cache *Cache) Del(key string) error {
	key = fmt.Sprintf("%s%s", cache.Option.Prefix, key)
	err := cache.rdb.Del(context.Background(), key).Err()
	if err != nil {
		return err
	}
	return nil
}

// Has check if the cache is exist ( without updating recency or frequency )
func (cache *Cache) Has(key string) bool {
	key = fmt.Sprintf("%s%s", cache.Option.Prefix, key)
	cmd := cache.rdb.Exists(context.Background(), key)
	v, _ := cmd.Result()
	return int(v) == 0
}

// Len returns the number of cached entries (**not O(1)**)
func (cache *Cache) Len() int {
	script := fmt.Sprintf("return #redis.pcall('keys', '%s*')", cache.Option.Prefix)
	cmd := cache.rdb.Eval(context.Background(), script, []string{})
	v, err := cmd.Result()
	if err != nil {
		log.Error("KV redis Len: %s", err.Error())
		return 0
	}

	intv, err := strconv.Atoi(fmt.Sprintf("%s", v))
	if err != nil {
		log.Error("KV redis Len: %s", err.Error())
		return 0
	}
	return intv
}
