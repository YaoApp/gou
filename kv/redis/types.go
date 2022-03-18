package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache redis store
type Cache struct {
	rdb    *redis.Client
	Option *Option
}

// Option redis option
type Option struct {
	Timeout time.Duration
	Prefix  string
	Redis   *redis.Options
}
