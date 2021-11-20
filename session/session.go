package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"time"

	"github.com/yaoapp/kun/exception"
)

// Managers 已注册会话管理器
var Managers = map[string]Manager{}

// 注册默认的会话管理器
func init() {
	Register("memory", &Memory{})
}

// Register 注册会话管理器
func Register(name string, manger Manager) {
	manger.Init()
	Managers[name] = manger
}

// Use 选用会话管理器
func Use(name string) *Session {
	expiredAt := 3600 * time.Second
	if manager, has := Managers[name]; has {
		return &Session{Manager: manager, expiredAt: expiredAt}
	}
	return &Session{Manager: Managers["memory"], expiredAt: expiredAt}
}

// ID 生成SessionID
func ID() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		log.Fatalf("Can't create session id")
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Expire 设定过期时间
func (session *Session) Expire(expiredAt time.Duration) *Session {
	session.expiredAt = expiredAt
	return session
}

// ID 选择指定 Session ID
func (session *Session) ID(id string) *Session {
	session.id = id
	return session
}

// Make 生成新的 Session ID
func (session *Session) Make() *Session {
	session.id = ID()
	return session
}

// GetID 读取 Session ID
func (session *Session) GetID() string {
	return session.id
}

// Set 设置数值
func (session *Session) Set(key string, value interface{}) error {
	return session.Manager.Set(session.id, key, value, session.expiredAt)
}

// MustSet 设置数值
func (session *Session) MustSet(key string, value interface{}) {
	err := session.Set(key, value)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
}

// SetWithEx 设置数值
func (session *Session) SetWithEx(key string, value interface{}, expiredAt time.Duration) error {
	return session.Manager.Set(session.id, key, value, expiredAt)
}

// MustSetWithEx 设置数值
func (session *Session) MustSetWithEx(key string, value interface{}, expiredAt time.Duration) {
	err := session.SetWithEx(key, value, expiredAt)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
}

// Get 读取数值
func (session *Session) Get(key string) (interface{}, error) {
	return session.Manager.Get(session.id, key)
}

// MustGet 读取数值
func (session *Session) MustGet(key string) interface{} {
	value, err := session.Get(key)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return value
}

// Dump 导出所有数据
func (session *Session) Dump() (map[string]interface{}, error) {
	return session.Manager.Dump(session.id)
}

// MustDump 导出所有数据
func (session *Session) MustDump() map[string]interface{} {
	value, err := session.Dump()
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return value
}

// // Cookie 从Cookie中读取 Session ID
// func (session *Session) Cookie(name string) {}

// // QueryString 从QueryString中读取 Session ID
// func (session *Session) QueryString(name string) {}

// // Header 从Header中读取
// func (session *Session) Header(name string) {}

// // Map 从Map中读取 SessionID
// func (session *Session) Map(name string) {}
