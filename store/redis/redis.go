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

// Push adds values to the end of a list using Redis RPUSH command
func (store *Store) Push(key string, values ...interface{}) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	// Convert values to strings for Redis
	stringValues := make([]interface{}, len(values))
	for i, value := range values {
		bytes, err := jsoniter.Marshal(value)
		if err != nil {
			log.Error("Store redis Push marshal %s: %s", key, err.Error())
			return err
		}
		stringValues[i] = string(bytes)
	}

	err := store.rdb.RPush(context.Background(), key, stringValues...).Err()
	if err != nil {
		log.Error("Store redis Push %s: %s", key, err.Error())
		return err
	}
	return nil
}

// Pop removes and returns an element from a list
func (store *Store) Pop(key string, position int) (interface{}, error) {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	var val string
	var err error

	if position == 1 { // pop from end
		val, err = store.rdb.RPop(context.Background(), key).Result()
	} else { // pop from beginning
		val, err = store.rdb.LPop(context.Background(), key).Result()
	}

	if err != nil {
		if strings.Contains(err.Error(), "nil") {
			return nil, fmt.Errorf("list is empty or key not found")
		}
		log.Error("Store redis Pop %s: %s", key, err.Error())
		return nil, err
	}

	var value interface{}
	err = jsoniter.Unmarshal([]byte(val), &value)
	if err != nil {
		log.Error("Store redis Pop unmarshal %s: %s", key, err.Error())
		return nil, err
	}

	return value, nil
}

// Pull removes all occurrences of a value from a list using Redis LREM command
func (store *Store) Pull(key string, value interface{}) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("Store redis Pull marshal %s: %s", key, err.Error())
		return err
	}

	// Remove all occurrences (count = 0)
	err = store.rdb.LRem(context.Background(), key, 0, string(bytes)).Err()
	if err != nil {
		log.Error("Store redis Pull %s: %s", key, err.Error())
		return err
	}
	return nil
}

// PullAll removes all occurrences of multiple values from a list
func (store *Store) PullAll(key string, values []interface{}) error {
	for _, value := range values {
		if err := store.Pull(key, value); err != nil {
			return err
		}
	}
	return nil
}

// AddToSet adds values to a list only if they don't already exist (using Lua script)
func (store *Store) AddToSet(key string, values ...interface{}) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	// Lua script to check existence and add only unique values
	luaScript := `
		local key = KEYS[1]
		local values = ARGV
		local result = 0
		
		for i = 1, #values do
			local value = values[i]
			local found = redis.call('LPOS', key, value)
			if not found then
				redis.call('RPUSH', key, value)
				result = result + 1
			end
		end
		
		return result
	`

	// Convert values to strings for Redis
	stringValues := make([]interface{}, len(values))
	for i, value := range values {
		bytes, err := jsoniter.Marshal(value)
		if err != nil {
			log.Error("Store redis AddToSet marshal %s: %s", key, err.Error())
			return err
		}
		stringValues[i] = string(bytes)
	}

	err := store.rdb.Eval(context.Background(), luaScript, []string{key}, stringValues...).Err()
	if err != nil {
		log.Error("Store redis AddToSet %s: %s", key, err.Error())
		return err
	}
	return nil
}

// ArrayLen returns the length of a list using Redis LLEN command
func (store *Store) ArrayLen(key string) int {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	length, err := store.rdb.LLen(context.Background(), key).Result()
	if err != nil {
		log.Error("Store redis ArrayLen %s: %s", key, err.Error())
		return 0
	}
	return int(length)
}

// ArrayGet returns an element at the specified index using Redis LINDEX command
func (store *Store) ArrayGet(key string, index int) (interface{}, error) {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	val, err := store.rdb.LIndex(context.Background(), key, int64(index)).Result()
	if err != nil {
		if strings.Contains(err.Error(), "nil") {
			return nil, fmt.Errorf("index out of range")
		}
		log.Error("Store redis ArrayGet %s[%d]: %s", key, index, err.Error())
		return nil, err
	}

	var value interface{}
	err = jsoniter.Unmarshal([]byte(val), &value)
	if err != nil {
		log.Error("Store redis ArrayGet unmarshal %s[%d]: %s", key, index, err.Error())
		return nil, err
	}

	return value, nil
}

// ArraySet sets an element at the specified index using Redis LSET command
func (store *Store) ArraySet(key string, index int, value interface{}) error {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("Store redis ArraySet marshal %s[%d]: %s", key, index, err.Error())
		return err
	}

	err = store.rdb.LSet(context.Background(), key, int64(index), string(bytes)).Err()
	if err != nil {
		log.Error("Store redis ArraySet %s[%d]: %s", key, index, err.Error())
		return err
	}
	return nil
}

// ArraySlice returns a slice of the list using Redis LRANGE command
func (store *Store) ArraySlice(key string, skip, limit int) ([]interface{}, error) {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	start := int64(skip)
	stop := int64(skip + limit - 1)

	vals, err := store.rdb.LRange(context.Background(), key, start, stop).Result()
	if err != nil {
		log.Error("Store redis ArraySlice %s: %s", key, err.Error())
		return nil, err
	}

	result := make([]interface{}, len(vals))
	for i, val := range vals {
		var value interface{}
		err = jsoniter.Unmarshal([]byte(val), &value)
		if err != nil {
			log.Error("Store redis ArraySlice unmarshal %s[%d]: %s", key, i, err.Error())
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}

// ArrayPage returns a page of the list
func (store *Store) ArrayPage(key string, page, pageSize int) ([]interface{}, error) {
	if page < 1 || pageSize < 1 {
		return []interface{}{}, nil
	}

	skip := (page - 1) * pageSize
	return store.ArraySlice(key, skip, pageSize)
}

// ArrayAll returns all elements in the list using Redis LRANGE command
func (store *Store) ArrayAll(key string) ([]interface{}, error) {
	key = fmt.Sprintf("%s%s", store.Option.Prefix, key)

	vals, err := store.rdb.LRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		log.Error("Store redis ArrayAll %s: %s", key, err.Error())
		return nil, err
	}

	result := make([]interface{}, len(vals))
	for i, val := range vals {
		var value interface{}
		err = jsoniter.Unmarshal([]byte(val), &value)
		if err != nil {
			log.Error("Store redis ArrayAll unmarshal %s[%d]: %s", key, i, err.Error())
			return nil, err
		}
		result[i] = value
	}

	return result, nil
}
