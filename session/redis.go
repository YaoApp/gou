package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
)

// Redis session store
type Redis struct {
	timeout time.Duration
	options *redis.Options
	rdb     *redis.Client
}

// NewRedis create a new redis instance
// host string, port int, db int, password string, username string, timeout int
func NewRedis(host string, options ...string) (*Redis, error) {

	inst := &Redis{
		timeout: 5 * time.Second,
		options: &redis.Options{},
		rdb:     nil,
	}

	port := 6379
	if len(options) > 0 {
		port = any.Of(options[0]).CInt()
	}

	if len(options) > 1 {
		inst.options.DB = any.Of(options[1]).CInt()
	}

	if len(options) > 2 {
		inst.options.Password = options[2]
	}

	if len(options) > 3 {
		inst.options.Username = options[3]
	}

	if len(options) > 4 {
		inst.timeout = time.Duration(any.Of(options[4]).CInt()) * time.Second
	}

	inst.options.Addr = fmt.Sprintf("%s:%d", host, port)

	client := redis.NewClient(inst.options).WithTimeout(inst.timeout)
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Error("Session redis Ping: %s host: %s options: %v", err.Error(), host, options)
		return nil, err
	}

	inst.rdb = client
	return inst, nil
}

// Init initialization
func (redis *Redis) Init() {}

// Set session value
func (redis *Redis) Set(id string, key string, value interface{}, timeout time.Duration) error {
	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		log.Error("Session redis Set: %s key %s", err.Error(), skey)
		return err
	}

	log.Debug("Session redis Set: %s KEY: %s VALUE: %v TS: %#v", skey, key, value, timeout)
	err = redis.rdb.Set(context.Background(), skey, bytes, timeout).Err()
	if err != nil {
		log.Error("Session redis Set: %s", err.Error())
		return err
	}
	return nil
}

// Get session value
func (redis *Redis) Get(id string, key string) (interface{}, error) {

	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	val, err := redis.rdb.Get(context.Background(), skey).Result()
	if err != nil {
		if "redis: nil" == err.Error() {
			return nil, nil
		}

		log.Error("Session redis Get: %s ERROR:%s", skey, err.Error())
		return nil, err
	}

	var value interface{}
	err = jsoniter.Unmarshal([]byte(val), &value)
	if err != nil {
		log.Error("Session redis Get JSON: %s val: %s ERROR:%s", skey, val, err.Error())
		return nil, err
	}

	return value, nil
}

// Del session value
func (redis *Redis) Del(id string, key string) error {
	skey := fmt.Sprintf("%s:%s:%s", "yao:session", id, key)
	log.Debug("Session redis Del: %s", skey)
	err := redis.rdb.Del(context.Background(), skey).Err()
	if err != nil {
		log.Error("Session redis Del: %s", err.Error())
		return err
	}
	return nil
}

// Dump session data
func (redis *Redis) Dump(id string) (map[string]interface{}, error) {
	prefix := fmt.Sprintf("%s:%s:", "yao:session", id)
	res := map[string]interface{}{}
	keys, err := redis.rdb.Keys(context.Background(), prefix+"*").Result()
	if err != nil {
		log.Error("Session redis Dump %s ERROR:%s", id, err.Error())
		return res, err
	}

	for _, key := range keys {
		key = strings.TrimPrefix(key, prefix)
		val, err := redis.Get(id, key)
		if err != nil {
			res[key] = nil
			continue
		}
		res[key] = val
	}
	return res, nil
}
