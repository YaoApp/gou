package session

import "time"

// Manager Session 管理器
type Manager interface {
	Init()
	Set(id string, key string, value interface{}, expired time.Duration) error
	Get(id string, key string) (interface{}, error)
	Del(id string, key string) error
	Dump(id string) (map[string]interface{}, error)
}

// Session 数据结构
type Session struct {
	id      string
	name    string
	timeout time.Duration
	Manager Manager
}
