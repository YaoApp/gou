package redis

import (
	"time"

	"github.com/go-redis/redis/v8"
)

// Store redis store
type Store struct {
	rdb    *redis.Client
	Option Option
}

// Option redis option
type Option struct {
	Timeout time.Duration
	Prefix  string
}
